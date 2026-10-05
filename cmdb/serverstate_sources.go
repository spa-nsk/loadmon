package cmdb

import (
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"

	"loadmon/grpcclient"
	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
)

// Источники для единого правила Stat/PowerState/Active (2026-10-02, см.
// ReconcileServerSprRecordsTypeAndState и kontext/03-data-model.md).
//
// Вместо прямых запросов к vCenter/OneView на каждую запись справочника
// (раньше - GetVMByKeyDirect/GetHostByKeyDirect, по новому govmomi-
// соединению на каждый вызов) один раз за прогон строятся индексы по уже
// собранным данным процесса metrics_grpc:
//   - VMware - пулы vspheremetrics (VMMapList/HostMapList), в т.ч. хосты вне
//     кластеров (одиночные/горячий резерв: в пуле у них Cluster = имя хоста);
//     достоверность - отметка успешной сверки vCenter (vspheremetrics.
//     VCenterSyncedSince);
//   - HP OneView - Infrastructure (ServerBySN/ServerByVirtualSN/ServerByName);
//     достоверность - LastError == nil и свежий LastRefresh;
//   - Zabbix и SCOM - свежие снимки server_metric: метрики за последние
//     минуты = агент доступен = сервер включён. Отсутствие метрик ничего не
//     доказывает (агент мог упасть) - эти источники дают только "найден и
//     включён"; если сборщик целиком не пишет данные, записи с привязкой к
//     нему не помечаются "не найденными".

const (
	vcenterSyncMaxAge = time.Hour        // vCenter опрашивается раз в Wait секунд; час без успешной сверки - источник недостоверен
	oneViewMaxAge     = 7 * time.Hour    // OneView перечитывается раз в 6 часов
	zabbixFreshWindow = 15 * time.Minute // zabbixmetrics пишет раз в минуту
	scomFreshWindow   = 30 * time.Minute // scommetrics - раз в IntervalSec (по умолчанию 5 минут)
)

type serverStateSources struct {
	localVC     map[string]bool // vCenter, которые опрашивает этот процесс
	syncedVC    map[string]bool // ... и недавно успешно сверены
	allVCSynced bool            // есть хотя бы один локальный vCenter и все сверены

	vmByKey    map[string]*vsphere.VMPoint // ключ VMMapList: vcenter/datacenter/cluster/name
	vmByVcUUID map[string]*vsphere.VMPoint // vcenter/uuid
	vmByVcName map[string]*vsphere.VMPoint // vcenter/имя в нижнем регистре
	hostBySN   map[string]*vsphere.HostInfo
	hostByName map[string]*vsphere.HostInfo // полное и короткое имя в нижнем регистре

	ovCertain bool // OneView не настроен или последний сбор успешен и свеж
	ovBySN    map[string]*oneview.Server
	ovByVSN   map[string]*oneview.Server
	ovByName  map[string]*oneview.Server // короткое имя в нижнем регистре

	zabbixAlive bool
	zabbixOn    map[int64]bool // ZabbixHost.Id со свежими метриками
	scomAlive   bool
	scomOn      map[int64]bool // ScomHost.Id со свежими метриками

	vmSpr map[int64]*models.VMSprRecord
	hw    map[int64]*models.ServerHardware
}

func loadServerStateSources(o orm.Ormer) *serverStateSources {
	s := &serverStateSources{
		localVC: map[string]bool{}, syncedVC: map[string]bool{},
		vmByKey: map[string]*vsphere.VMPoint{}, vmByVcUUID: map[string]*vsphere.VMPoint{}, vmByVcName: map[string]*vsphere.VMPoint{},
		hostBySN: map[string]*vsphere.HostInfo{}, hostByName: map[string]*vsphere.HostInfo{},
		ovBySN: map[string]*oneview.Server{}, ovByVSN: map[string]*oneview.Server{}, ovByName: map[string]*oneview.Server{},
		vmSpr: map[int64]*models.VMSprRecord{}, hw: map[int64]*models.ServerHardware{},
	}

	// --- VMware: пулы в памяти ---
	if infra := vsphere.GetGlobalVMInfrastructure(); infra != nil {
		for _, vc := range infra.VCenterMapList.Keys() {
			s.localVC[vc] = true
			if vsphere.VCenterSyncedSince(vc, vcenterSyncMaxAge) {
				s.syncedVC[vc] = true
			}
		}
		s.allVCSynced = len(s.localVC) > 0 && len(s.syncedVC) == len(s.localVC)
		for _, key := range infra.VMMapList.Keys() {
			vm, ok := infra.VMMapList.Get(key)
			if !ok || vm == nil {
				continue
			}
			s.vmByKey[key] = vm
			if vm.UUID != "" {
				s.vmByVcUUID[vm.Vcenter+"/"+vm.UUID] = vm
			}
			if vm.Name != "" {
				s.vmByVcName[vm.Vcenter+"/"+strings.ToLower(vm.Name)] = vm
			}
		}
		for _, key := range infra.HostMapList.Keys() {
			h, ok := infra.HostMapList.Get(key)
			if !ok || h == nil {
				continue
			}
			if sn := strings.TrimSpace(h.SerialNumber); sn != "" {
				s.hostBySN[sn] = h
			}
			for _, n := range nameVariants(h.Name) {
				s.hostByName[n] = h
			}
		}
	}

	// --- HP OneView ---
	ov := oneview.GetGlobalOVInfrastructure()
	if ov == nil || len(ov.EndpointList()) == 0 {
		s.ovCertain = true // OneView не настроен - отсутствие в нём ничего не меняет
	} else {
		ov.WaitUnLockGetData()
		s.ovCertain = ov.LastError == nil && !ov.LastRefresh.IsZero() && time.Since(ov.LastRefresh) <= oneViewMaxAge
		for sn, srv := range ov.ServerBySN {
			s.ovBySN[sn] = srv
		}
		for vsn, srv := range ov.ServerByVirtualSN {
			s.ovByVSN[vsn] = srv
		}
		for _, name := range ov.ServerByName.Keys() {
			if srv, ok := ov.ServerByName.Get(name); ok && srv != nil {
				s.ovByName[shortLower(name)] = srv
			}
		}
	}

	// --- Zabbix / SCOM: свежие снимки server_metric ---
	s.zabbixOn, s.zabbixAlive = freshMetricHosts(o, models.MetricSourceZabbix, zabbixFreshWindow)
	s.scomOn, s.scomAlive = freshMetricHosts(o, models.MetricSourceSCOM, scomFreshWindow)

	// --- справочники, на которые ссылаются записи ---
	var vms []*models.VMSprRecord
	if _, err := o.QueryTable(new(models.VMSprRecord)).All(&vms, "Id", "Vcenter", "Datacenter", "Cluster", "ClusterId", "Vm_Name", "Vm_Uuid", "Vm_PowerState"); err != nil {
		log.Println("serverstate: чтение vm_spr_record:", err)
	}
	for _, v := range vms {
		s.vmSpr[v.Id] = v
	}
	var hws []*models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).All(&hws, "Id", "PowerState", "Active", "TimeActiveState", "ActivatorUser", "TimeDeActiveState", "DeActivatorUser", "SourceUpdateType"); err != nil {
		log.Println("serverstate: чтение server_hardware:", err)
	}
	for _, h := range hws {
		s.hw[h.Id] = h
	}

	log.Printf("serverstate: vCenter локальных %d, сверены %d; VM %d, хостов %d; OneView серверов %d (достоверен=%v); Zabbix свежих %d, SCOM свежих %d",
		len(s.localVC), len(s.syncedVC), len(s.vmByKey), len(s.hostBySN), len(s.ovBySN), s.ovCertain, len(s.zabbixOn), len(s.scomOn))
	return s
}

func freshMetricHosts(o orm.Ormer, source int, window time.Duration) (map[int64]bool, bool) {
	var ids []int64
	if _, err := o.Raw("SELECT DISTINCT host_id FROM server_metric WHERE source = ? AND collected_at >= ?", source, time.Now().Add(-window)).QueryRows(&ids); err != nil {
		log.Printf("serverstate: чтение свежих server_metric (source=%d): %v", source, err)
		return map[int64]bool{}, false
	}
	m := make(map[int64]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m, len(ids) > 0
}

// serverEvidence - что известно о записи из источников в этом прогоне.
type serverEvidence struct {
	host     *vsphere.HostInfo
	vm       *vsphere.VMPoint
	vmSpr    *models.VMSprRecord
	ovSrv    *oneview.Server
	zabbixOn bool
	scomOn   bool
	// certainAbsent - все источники, где запись могла бы найтись, отработали
	// достоверно; только тогда "не найден" можно применять.
	certainAbsent bool
}

func (e serverEvidence) found() bool {
	return e.host != nil || e.vm != nil || e.ovSrv != nil || e.zabbixOn || e.scomOn
}

func (e serverEvidence) poweredOn() bool {
	return (e.host != nil && e.host.PowerState) || (e.vm != nil && e.vm.PowerState) ||
		powerStateFromOneView(e.ovSrv) == 1 || e.zabbixOn || e.scomOn
}

// auditSource - код системы для аудита Active (models.UserSource*) по
// источнику, в котором запись найдена; для ненайденной - по типу записи.
func (e serverEvidence) auditSource(srvType int) int64 {
	switch {
	case e.host != nil || e.vm != nil:
		return models.UserSourceVMware
	case e.ovSrv != nil:
		return models.UserSourceOneView
	case e.zabbixOn:
		return models.UserSourceZabbix
	case e.scomOn:
		return models.UserSourceSCOM
	}
	return activeAuditSource(srvType)
}

func (s *serverStateSources) evidence(srv *models.ServerSprRecords) serverEvidence {
	ev := serverEvidence{certainAbsent: true}

	// VMware: ВМ по связанной записи vm_spr_record - точный ключ, затем
	// UUID и имя в том же vCenter (ВМ могла переехать в другой кластер
	// между 6-часовыми обновлениями vm_spr_record).
	if srv.VMSprId > 0 {
		if vs := s.vmSpr[srv.VMSprId]; vs != nil {
			ev.vmSpr = vs
			vm := s.vmByKey[vs.Vcenter+"/"+vs.Datacenter+"/"+vs.Cluster+"/"+vs.Vm_Name]
			if vm == nil && vs.Vm_Uuid != "" {
				vm = s.vmByVcUUID[vs.Vcenter+"/"+vs.Vm_Uuid]
			}
			if vm == nil && vs.Vm_Name != "" {
				vm = s.vmByVcName[vs.Vcenter+"/"+strings.ToLower(vs.Vm_Name)]
			}
			if vm == nil && !s.localVC[vs.Vcenter] && vs.Vcenter != "" && vs.Cluster != "" && vs.Vm_Name != "" {
				// vCenter опрашивает другой экземпляр metrics_grpc - спрашиваем его
				if p, err := grpcclient.GetVMByKeyDirect(vs.Vcenter, vs.Datacenter, vs.Cluster, vs.Vm_Name); err == nil {
					vm = &p
				}
			}
			ev.vm = vm
			if vm == nil && !s.syncedVC[vs.Vcenter] {
				ev.certainAbsent = false
			}
		}
	}

	// VMware: хост (гипервизор, в т.ч. вне кластера) - по серийному номеру,
	// для не-ВМ ещё и по имени (во всех локальных vCenter).
	sn := serverSprSerial(srv)
	if sn != "" {
		ev.host = s.hostBySN[sn]
	}
	if ev.host == nil && srv.Type != 1 {
		for _, n := range recordNames(srv) {
			if h := s.hostByName[n]; h != nil {
				ev.host = h
				break
			}
		}
	}
	if !s.allVCSynced {
		ev.certainAbsent = false
	}

	// HP OneView
	if sn != "" {
		if ev.ovSrv = s.ovBySN[sn]; ev.ovSrv == nil {
			ev.ovSrv = s.ovByVSN[sn]
		}
	}
	if ev.ovSrv == nil && srv.Type != 1 {
		for _, n := range recordNames(srv) {
			if hw := s.ovByName[n]; hw != nil {
				ev.ovSrv = hw
				break
			}
		}
	}
	if !s.ovCertain {
		ev.certainAbsent = false
	}

	// Zabbix / SCOM - только "найден и включён"
	if srv.ZabbixHostId > 0 {
		ev.zabbixOn = s.zabbixOn[srv.ZabbixHostId]
		if !s.zabbixAlive {
			ev.certainAbsent = false
		}
	}
	if srv.ScomHostId > 0 {
		ev.scomOn = s.scomOn[srv.ScomHostId]
		if !s.scomAlive {
			ev.certainAbsent = false
		}
	}
	return ev
}

// isManual - запись внесена вручную или импортирована из Итиллиум:
// SourceUpdateType = SourceUpdateManual у самой записи или у связанного
// ServerHardware (по HWSprId). Правило (2026-10-02): если объект такой
// записи не найден ни в одном источнике, автоматика не меняет её
// Stat/PowerState/Active и связанный ServerHardware; если найден - запись
// обновляется по источнику как обычно.
func (s *serverStateSources) isManual(srv *models.ServerSprRecords) bool {
	if srv.SourceUpdateType == models.SourceUpdateManual {
		return true
	}
	if h := s.hw[srv.HWSprId]; h != nil {
		return h.SourceUpdateType == models.SourceUpdateManual
	}
	return false
}

func nameVariants(name string) []string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return nil
	}
	if sh := shortLower(n); sh != n {
		return []string{n, sh}
	}
	return []string{n}
}

func shortLower(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if i := strings.Index(n, "."); i > 0 {
		return n[:i]
	}
	return n
}

func recordNames(srv *models.ServerSprRecords) []string {
	var out []string
	out = append(out, nameVariants(srv.Name)...)
	out = append(out, nameVariants(srv.NameFQDN)...)
	return out
}

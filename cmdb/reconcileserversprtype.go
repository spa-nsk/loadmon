package cmdb

import (
	"log"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"

	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
)

// ReconcileServerSprRecordsTypeAndState - ЕДИНОЕ правило Stat/PowerState/
// Active для models.ServerSprRecords и связанного models.ServerHardware
// (2026-10-02, по решению владельца проекта). Вызывается после обновления
// инвентаризации (раз в 6 часов) и раз в час (UpdatePowerStateForServers).
//
// Источники (см. serverstate_sources.go): VMware (ВМ и хосты, включая
// хосты вне кластеров и в горячем резерве, подключённые к vCenter), HP
// OneView, Zabbix и SCOM (свежие метрики = агент доступен = включён).
// Найден хотя бы в одном источнике - найден; хотя бы один говорит
// "включён" - включён.
//
//   - найден и включён:  Stat=1, PowerState=1, Active=1;
//   - найден, выключен:  Stat=1, PowerState=2, Active=1, НО ручное Active=2
//     (вывод из эксплуатации человеком или импортом Итиллиум) сохраняется -
//     см. desiredActive;
//   - не найден:         Stat=1, PowerState=2, Active=2 - только если все
//     источники, где запись могла бы найтись, отработали достоверно
//     (недоступный vCenter/OneView/сборщик метрик - запись не трогается);
//     ручные записи и данные из Итиллиум, не найденные в источниках, не
//     трогаются никогда (serverStateSources.isManual).
//
// Записи со Stat=0 (историческая запись) и Stat=2 (помечено для удаления)
// ставит только человек - автоматика их не меняет; Stat=666 исключены из
// выборки вообще.
//
// Для найденных записей также уточняются Type (2 - хост VMware, 1 - ВМ,
// 0 - есть только в OneView) и место размещения, как и раньше.
func ReconcileServerSprRecordsTypeAndState() {
	o := orm.NewOrm()
	src := loadServerStateSources(o)

	var records []*models.ServerSprRecords
	if _, err := models.ServerSprQS(o).OrderBy("Name").All(&records); err != nil {
		log.Println("ReconcileServerSprRecordsTypeAndState: ошибка чтения справочника:", err)
		return
	}

	var skippedUncertain, skippedManual, updated int
	for _, rec := range records {
		if rec == nil || models.IsServerSprDestroyed(rec) {
			continue
		}
		switch reconcileServerSprRecord(o, src, rec) {
		case reconcileSkippedUncertain:
			skippedUncertain++
		case reconcileSkippedManual:
			skippedManual++
		case reconcileUpdated:
			updated++
		}
	}
	log.Printf("ReconcileServerSprRecordsTypeAndState: записей %d, изменено %d, пропущено (источник недостоверен) %d, пропущено (ручные/Итиллиум, нет в источниках) %d",
		len(records), updated, skippedUncertain, skippedManual)
}

type reconcileResult int

const (
	reconcileNoChange reconcileResult = iota
	reconcileUpdated
	reconcileSkippedUncertain
	reconcileSkippedManual
	reconcileSkippedManualStat
)

func reconcileServerSprRecord(o orm.Ormer, src *serverStateSources, srv *models.ServerSprRecords) reconcileResult {
	if srv.Stat == models.ServerSprStatHistory || srv.Stat == models.ServerSprStatMarkedForDelete {
		return reconcileSkippedManualStat // Stat 0/2 ставит только человек
	}

	ev := src.evidence(srv)
	found := ev.found()
	on := found && ev.poweredOn()

	if !found {
		if !ev.certainAbsent {
			return reconcileSkippedUncertain
		}
		if src.isManual(srv) {
			return reconcileSkippedManual
		}
	}

	var fields []string

	// Тип и место размещения - только для найденных.
	if found {
		expectedType := srv.Type
		switch {
		case ev.host != nil:
			expectedType = 2
		case ev.vm != nil:
			expectedType = 1
		case ev.ovSrv != nil:
			expectedType = 0
		}
		if srv.Type != expectedType {
			srv.Type = expectedType
			fields = append(fields, "Type")
		}
		switch expectedType {
		case 2:
			fields = appendHypervisorFields(srv, ev.host, fields)
		case 1:
			fields = appendVMFields(srv, vmPlacementFrom(ev), fields)
		case 0:
			if ev.ovSrv != nil && ev.host == nil {
				clearHypervisorPlacement(srv, &fields)
			}
		}
	}

	newPower := powerStateFromBool(on)
	if srv.PowerState != newPower {
		srv.PowerState = newPower
		fields = append(fields, "PowerState")
	}
	if srv.Stat != models.ServerSprStatUsed {
		srv.Stat = models.ServerSprStatUsed
		fields = append(fields, "Stat")
	}

	newActive := desiredActive(found, on, srv.Active, srv.DeActivatorUser)
	if srv.Type == 1 && vmNameMarkedForRemoval(srv.Name) {
		newActive = 2 // ВМ переименована перед удалением (_OFF/_DEL) - прежнее правило сохранено
	}
	auditSrc := ev.auditSource(srv.Type)
	if srv.Active != newActive {
		models.SetActiveWithAudit(&srv.Active, &srv.TimeActiveState, &srv.ActivatorUser, &srv.TimeDeActiveState, &srv.DeActivatorUser, newActive, auditSrc)
		fields = append(fields, "Active", "TimeActiveState", "ActivatorUser", "TimeDeActiveState", "DeActivatorUser")
	}

	result := reconcileNoChange
	if fields = uniqueFields(fields); len(fields) > 0 {
		if _, err := o.Update(srv, fields...); err != nil {
			log.Printf("ReconcileServerSprRecordsTypeAndState: ошибка обновления %q (Id=%d): %v", srv.Name, srv.Id, err)
		} else {
			result = reconcileUpdated
		}
	}

	// Справочник ВМ: состояние питания по VMware.
	if ev.vmSpr != nil && srv.Type == 1 {
		vmOn := ev.vm != nil && ev.vm.PowerState
		if ev.vmSpr.Vm_PowerState != vmOn {
			ev.vmSpr.Vm_PowerState = vmOn
			if _, err := o.Update(ev.vmSpr, "Vm_PowerState"); err != nil {
				log.Printf("ReconcileServerSprRecordsTypeAndState: Vm_PowerState %q: %v", srv.Name, err)
			}
		}
	}

	// ServerHardware - то же правило: PowerState как у карточки, Active по
	// desiredActive с учётом собственного ручного вывода из эксплуатации.
	if hw := src.hw[srv.HWSprId]; hw != nil && srv.HWSprId > 0 {
		var hwCols []string
		if hw.PowerState != newPower {
			hw.PowerState = newPower
			hwCols = append(hwCols, "PowerState")
		}
		if hwActive := desiredActive(found, on, hw.Active, hw.DeActivatorUser); hw.Active != hwActive {
			models.SetActiveWithAudit(&hw.Active, &hw.TimeActiveState, &hw.ActivatorUser, &hw.TimeDeActiveState, &hw.DeActivatorUser, hwActive, auditSrc)
			hwCols = append(hwCols, "Active", "TimeActiveState", "ActivatorUser", "TimeDeActiveState", "DeActivatorUser")
		}
		if len(hwCols) > 0 {
			if _, err := o.Update(hw, hwCols...); err != nil {
				log.Printf("ReconcileServerSprRecordsTypeAndState: ServerHardware Id=%d: %v", hw.Id, err)
			}
		}
	}
	return result
}

// desiredActive - Active по правилу: найден и включён -> 1; найден и
// выключен -> 1, кроме ручного вывода из эксплуатации (Active=2, поставленного
// человеком или импортом Итиллиум - см. isAutoDeactivation); не найден -> 2.
func desiredActive(found, on bool, cur int, deActivator int64) int {
	switch {
	case found && on:
		return 1
	case found:
		if cur == 2 && !isAutoDeactivation(deActivator) {
			return 2
		}
		return 1
	default:
		return 2
	}
}

// isAutoDeactivation - Active=2 поставила автоматика (VMware/OneView/Zabbix/
// SCOM). Реальный пользователь, models.UserSourceManual (ручной ввод/импорт
// Итиллиум) и 0 (автор не записан - деактивация до появления аудита,
// 2026-09-22; трактуется как ручная, чтобы не потерять решение человека)
// считаются ручным выводом.
func isAutoDeactivation(user int64) bool {
	switch user {
	case models.UserSourceVMware, models.UserSourceOneView, models.UserSourceZabbix, models.UserSourceSCOM:
		return true
	}
	return false
}

func vmNameMarkedForRemoval(name string) bool {
	up := strings.ToUpper(name)
	return strings.Contains(up, "_OFF") || strings.Contains(up, "_DEL")
}

// vmPlacement - место размещения ВМ для заполнения Vcenter/Datacenter/
// Cluster/ClusterId записи Type=1 (по живым данным VMware, иначе по
// vm_spr_record).
type vmPlacement struct {
	vcenter    string
	datacenter string
	cluster    string
	clusterId  string
}

func vmPlacementFrom(ev serverEvidence) *vmPlacement {
	if ev.vm != nil {
		return &vmPlacement{vcenter: ev.vm.Vcenter, datacenter: ev.vm.Datacenter, cluster: ev.vm.Cluster, clusterId: ev.vm.ClusterID}
	}
	if ev.vmSpr != nil {
		return &vmPlacement{vcenter: ev.vmSpr.Vcenter, datacenter: ev.vmSpr.Datacenter, cluster: ev.vmSpr.Cluster, clusterId: ev.vmSpr.ClusterId}
	}
	return nil
}

// activeAuditSource - код системы для аудита Active по типу записи (0 -
// HP OneView, 1/2 - VMware); используется для ненайденных записей.
func activeAuditSource(srvType int) int64 {
	if srvType == 0 {
		return models.UserSourceOneView
	}
	return models.UserSourceVMware
}

func serverSprSerial(srv *models.ServerSprRecords) string {
	if s := strings.TrimSpace(srv.SerialNumber); s != "" {
		return s
	}
	return strings.TrimSpace(GetSNFromServerSprRecords(srv))
}

func appendHypervisorFields(srv *models.ServerSprRecords, host *vsphere.HostInfo, fields []string) []string {
	if host == nil {
		return fields
	}
	if srv.Vcenter != host.Vcenter {
		srv.Vcenter = host.Vcenter
		fields = append(fields, "Vcenter")
	}
	// заглушки vsphere.NoDatacenter/NoCluster (хост без определённого
	// размещения, 2026-10-02) пишутся пустой строкой
	if dc := vsphere.PlacementValue(host.Datacenter); srv.Datacenter != dc {
		srv.Datacenter = dc
		fields = append(fields, "Datacenter")
	}
	if cl := vsphere.PlacementValue(host.Cluster); srv.Cluster != cl {
		srv.Cluster = cl
		fields = append(fields, "Cluster")
	}
	if srv.ClusterId != host.ClusterID {
		srv.ClusterId = host.ClusterID
		fields = append(fields, "ClusterId")
	}
	if sn := strings.TrimSpace(host.SerialNumber); sn != "" && srv.SerialNumber != sn {
		srv.SerialNumber = sn
		fields = append(fields, "SerialNumber")
	}
	if host.ProductFullName != "" && srv.OS != host.ProductFullName {
		srv.OS = host.ProductFullName
		fields = append(fields, "OS")
	}
	return fields
}

// appendVMFields заполняет Vcenter/Datacenter/Cluster/ClusterId записи
// ServerSprRecords (Type=1) местом размещения ВМ (vmPlacementFrom: живые
// данные VMware, иначе vm_spr_record) - аналог appendHypervisorFields для
// гипервизоров. До 2026-09-04 вместо этого вызывался
// clearHypervisorPlacement, который эти поля затирал.
func appendVMFields(srv *models.ServerSprRecords, vm *vmPlacement, fields []string) []string {
	if vm == nil {
		return fields
	}
	if srv.Vcenter != vm.vcenter {
		srv.Vcenter = vm.vcenter
		fields = append(fields, "Vcenter")
	}
	if srv.Datacenter != vm.datacenter {
		srv.Datacenter = vm.datacenter
		fields = append(fields, "Datacenter")
	}
	if srv.Cluster != vm.cluster {
		srv.Cluster = vm.cluster
		fields = append(fields, "Cluster")
	}
	if srv.ClusterId != vm.clusterId {
		srv.ClusterId = vm.clusterId
		fields = append(fields, "ClusterId")
	}
	return fields
}

func clearHypervisorPlacement(srv *models.ServerSprRecords, fields *[]string) {
	if srv.Vcenter != "" {
		srv.Vcenter = ""
		*fields = append(*fields, "Vcenter")
	}
	if srv.Datacenter != "" {
		srv.Datacenter = ""
		*fields = append(*fields, "Datacenter")
	}
	if srv.Cluster != "" {
		srv.Cluster = ""
		*fields = append(*fields, "Cluster")
	}
	if srv.ClusterId != "" {
		srv.ClusterId = ""
		*fields = append(*fields, "ClusterId")
	}
}

func updateVMSprPowerState(o orm.Ormer, srv *models.ServerSprRecords, poweredOn bool) {
	if srv.VMSprId <= 0 {
		return
	}
	vmSpr := models.VMSprRecord{Id: srv.VMSprId}
	if err := o.Read(&vmSpr); err != nil {
		return
	}
	if vmSpr.Vm_PowerState == poweredOn {
		return
	}
	vmSpr.Vm_PowerState = poweredOn
	if _, err := o.Update(&vmSpr, "Vm_PowerState"); err != nil {
		log.Printf("ReconcileServerSprRecordsTypeAndState: Vm_PowerState %q: %v", srv.Name, err)
	}
}

func powerStateFromBool(on bool) int {
	if on {
		return 1
	}
	return 2
}

func powerStateFromOneView(hw *oneview.Server) int {
	if hw == nil {
		return 2
	}
	ps := strings.TrimSpace(hw.Identity.PowerState)
	if strings.EqualFold(ps, "On") {
		return 1
	}
	if ps != "" {
		return 2
	}
	return 0
}

func uniqueFields(fields []string) []string {
	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

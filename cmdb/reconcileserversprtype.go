package cmdb

import (
	"loadmon/grpcclient"
	"loadmon/models"
	"log"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
	vsphere "loadmon/vspheremetrics"
)

// ReconcileServerSprRecordsTypeAndState обходит models.ServerSprRecords и
// сверяет Type/PowerState с фактическим наличием объекта в VMware и HP OneView
// (oneview2):
//
//   - Type 1 — виртуальная машина (найдена в vSphere как VM);
//   - Type 2 — аппаратный сервер-гипервизор (приоритет: найден в vSphere как
//     ESXi-хост, даже если также есть в OneView);
//   - Type 0 — аппаратный сервер без роли гипервизора (есть в OneView, но не
//     хост в vSphere; хосты других вендоров могут быть только в VMware);
//   - если сервер не найден ни в VMware, ни в OneView — Stat=0, PowerState=2,
//     Active=2 (выведен из эксплуатации), кроме записей с SourceUpdateType=1
//     (вручную) в ServerSprRecords или связанном ServerHardware — их статус
//     автоматической сверкой не затирается.
//
// Для проверки наличия используются прямые gRPC-запросы к metrics_grpc
// (GetVMByKeyDirect, GetHostByKeyDirect, GetHostBySN) и индекс OneView
// (GetHostData / FindServerHardwareSN), а не только закэшированные пулы
// vSphere — см. комментарии к CheckServerSprRecordsExistence.
func ReconcileServerSprRecordsTypeAndState() {
	o := orm.NewOrm()
	ov := oneview.GetGlobalOVInfrastructure()

	var records []*models.ServerSprRecords
	if _, err := o.QueryTable("server_spr_records").OrderBy("Name").All(&records); err != nil {
		log.Println("ReconcileServerSprRecordsTypeAndState: ошибка чтения справочника:", err)
		return
	}

	for _, rec := range records {
		if rec == nil || rec.Stat == 666 {
			continue
		}
		reconcileServerSprRecord(o, ov, rec)
	}
}

func reconcileServerSprRecord(o orm.Ormer, ov *oneview.Infrastructure, srv *models.ServerSprRecords) {
	host := lookupVMwareHost(srv)
	vmFound, vmPower := lookupVMwareVM(o, srv)
	hw := lookupOneViewServer(ov, srv)

	inVMwareHost := host != nil
	inVMwareVM := vmFound
	inOneView := hw != nil

	if !inVMwareHost && !inVMwareVM && !inOneView {
		if skipAbsentMarkDueToManualSource(o, srv) {
			return
		}
		markServerSprAbsent(o, srv)
		return
	}

	expectedType := expectedServerSprType(inVMwareHost, inVMwareVM)
	fields := applyExpectedServerSprState(o, srv, expectedType, host, vmPower, hw)
	if len(fields) == 0 {
		return
	}
	if _, err := o.Update(srv, fields...); err != nil {
		log.Printf("ReconcileServerSprRecordsTypeAndState: ошибка обновления %q (Id=%d): %v",
			srv.Name, srv.Id, err)
	}
}

func expectedServerSprType(inVMwareHost, inVMwareVM bool) int {
	if inVMwareHost {
		return 2
	}
	if inVMwareVM {
		return 1
	}
	return 0
}

func lookupVMwareHost(srv *models.ServerSprRecords) *vsphere.HostInfo {
	sn := serverSprSerial(srv)
	if sn != "" {
		if hs := grpcclient.GetHostBySN(sn); hs != nil {
			return hs
		}
	}
	if srv.Vcenter != "" && srv.Cluster != "" && srv.Name != "" {
		if hs, err := grpcclient.GetHostByKeyDirect(srv.Vcenter, srv.Datacenter, srv.Cluster, srv.Name); err == nil {
			return &hs
		}
	}
	if srv.Vcenter != "" && srv.Cluster != "" && srv.NameFQDN != "" && srv.NameFQDN != srv.Name {
		if hs, err := grpcclient.GetHostByKeyDirect(srv.Vcenter, srv.Datacenter, srv.Cluster, srv.NameFQDN); err == nil {
			return &hs
		}
	}
	return nil
}

func lookupVMwareVM(o orm.Ormer, srv *models.ServerSprRecords) (found bool, poweredOn bool) {
	if srv.VMSprId <= 0 {
		return false, false
	}
	vmSpr := models.VMSprRecord{Id: srv.VMSprId}
	if err := o.Read(&vmSpr); err != nil {
		return false, false
	}
	if vmSpr.Vcenter == "" || vmSpr.Cluster == "" || vmSpr.Vm_Name == "" {
		return false, false
	}
	vm, err := grpcclient.GetVMByKeyDirect(vmSpr.Vcenter, vmSpr.Datacenter, vmSpr.Cluster, vmSpr.Vm_Name)
	if err != nil {
		return false, false
	}
	return true, vm.PowerState
}

func lookupOneViewServer(ov *oneview.Infrastructure, srv *models.ServerSprRecords) *oneview.Server {
	sn := serverSprSerial(srv)
	if sn != "" {
		if hw, _, _ := GetHostData(sn); hw != nil {
			return hw
		}
		if ov != nil {
			if hw, err := ov.FindServerHardwareSN(sn); err == nil && hw != nil {
				return hw
			}
		}
	}
	for _, name := range []string{strings.TrimSpace(srv.NameFQDN), strings.TrimSpace(srv.Name)} {
		if name == "" {
			continue
		}
		if hw, ok := grpcclient.GetServerHardwareByName(name); ok && hw != nil {
			return hw
		}
	}
	return nil
}

func serverSprSerial(srv *models.ServerSprRecords) string {
	if s := strings.TrimSpace(srv.SerialNumber); s != "" {
		return s
	}
	return strings.TrimSpace(GetSNFromServerSprRecords(srv))
}

// skipAbsentMarkDueToManualSource сообщает, что запись не должна получать
// Stat=0/PowerState=2/Active=2 при отсутствии в VMware и OneView: источник
// последнего обновления — ручной ввод (SourceUpdateType=1) у самой записи
// server_spr_records или у связанного server_hardware (HWSprId).
func skipAbsentMarkDueToManualSource(o orm.Ormer, srv *models.ServerSprRecords) bool {
	if srv.SourceUpdateType == models.SourceUpdateManual {
		return true
	}
	if srv.HWSprId <= 0 {
		return false
	}
	hw := models.ServerHardware{Id: srv.HWSprId}
	if err := o.Read(&hw, "Id"); err != nil {
		return false
	}
	return hw.SourceUpdateType == models.SourceUpdateManual
}

func markServerSprAbsent(o orm.Ormer, srv *models.ServerSprRecords) {
	changed := srv.Stat != 0 || srv.PowerState != 2 || srv.Active != 2
	srv.Stat = 0
	srv.PowerState = 2
	srv.Active = 2
	if changed {
		if _, err := o.Update(srv, "Stat", "PowerState", "Active"); err != nil {
			log.Printf("ReconcileServerSprRecordsTypeAndState: ошибка пометки отсутствующего %q: %v", srv.Name, err)
		}
	}
	if srv.Type == 1 && srv.VMSprId > 0 {
		vmSpr := models.VMSprRecord{Id: srv.VMSprId}
		if err := o.Read(&vmSpr); err == nil && vmSpr.Vm_PowerState {
			vmSpr.Vm_PowerState = false
			if _, err := o.Update(&vmSpr, "Vm_PowerState"); err != nil {
				log.Printf("ReconcileServerSprRecordsTypeAndState: Vm_PowerState для %q: %v", srv.Name, err)
			}
		}
	}
}

func applyExpectedServerSprState(
	o orm.Ormer,
	srv *models.ServerSprRecords,
	expectedType int,
	host *vsphere.HostInfo,
	vmPoweredOn bool,
	hw *oneview.Server,
) []string {
	var fields []string

	if srv.Type != expectedType {
		srv.Type = expectedType
		fields = append(fields, "Type")
	}

	oldPower := srv.PowerState

	switch expectedType {
	case 2:
		fields = appendHypervisorFields(srv, host, fields)
		srv.PowerState = powerStateFromBool(host != nil && host.PowerState)
	case 1:
		clearHypervisorPlacement(srv, &fields)
		srv.PowerState = powerStateFromBool(vmPoweredOn)
		updateVMSprPowerState(o, srv, vmPoweredOn)
	case 0:
		clearHypervisorPlacement(srv, &fields)
		srv.PowerState = powerStateFromOneView(hw)
	}

	if srv.PowerState != oldPower {
		fields = append(fields, "PowerState")
	}

	return uniqueFields(fields)
}

func appendHypervisorFields(srv *models.ServerSprRecords, host *vsphere.HostInfo, fields []string) []string {
	if host == nil {
		return fields
	}
	if srv.Vcenter != host.Vcenter {
		srv.Vcenter = host.Vcenter
		fields = append(fields, "Vcenter")
	}
	if srv.Datacenter != host.Datacenter {
		srv.Datacenter = host.Datacenter
		fields = append(fields, "Datacenter")
	}
	if srv.Cluster != host.Cluster {
		srv.Cluster = host.Cluster
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

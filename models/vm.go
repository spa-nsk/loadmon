package models

import (
	vsphere "loadmon/vspheremetrics"
)

// VmReq описывает параметры запроса списка виртуальных машин (по идентификатору
// или по паре vCenter/Datacenter/Cluster/имя ВМ).
type VmReq struct {
	Id         int64
	Vcenter    string
	Datacenter string
	Cluster    string
	Vm_Name    string
}

// GetVmList возвращает список ВМ (vspheremetrics.VMPoint) из глобальной инфраструктуры:
// либо одну ВМ по идентификатору id, либо результат фильтрации по vcenter/datacenter/cluster/vmname.
func GetVmList(id string, vcenter string, datacenter string, cluster string, vmname string) ([]vsphere.VMPoint, bool) {
	infra := vsphere.GetGlobalVMInfrastructure()
	if id != "" {
		vm, ok := infra.VMIDMapList.Get(id)
		if ok {
			return []vsphere.VMPoint{*vm}, true
		} else {
			return nil, false
		}
	}
	vms, err := infra.GetVMListFilter(vcenter, datacenter, cluster, vmname)
	if err == nil {
		return vms, true
	}
	return vms, false
}

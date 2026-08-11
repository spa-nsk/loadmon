package vspheremetrics

import (
	"strings"

	"sort"

	"errors"
)

func (infra *VMInfrastructure) GetHostList() ([]HostInfo, error) {
	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var hostList []HostInfo
	hostList = make([]HostInfo, 0)

	for _, key := range infra.HostMapList.Keys() {
		hi, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		hiN := *hi
		hiN.Perf = make([]*HSPerf, len(hi.Perf))
		for i, perfI := range hi.Perf {
			perf := HSPerf{}
			perf = *perfI
			hiN.Perf[i] = &perf
		}
		//copy(hiN.Perf, hi.Perf)
		hostList = append(hostList, hiN)
	}

	sort.Sort(HSListID(hostList))
	return hostList, nil
}

// GetHostListFilter возвращает список хостов ESXi, отфильтрованный по vcenter/datacenter/cluster/hostname
// (значение "all" отключает фильтрацию по соответствующему полю).
func (infra *VMInfrastructure) GetHostListFilter(vcenter, datacenter, cluster, hostname string) ([]HostInfo, error) {

	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var hostList []HostInfo
	hostList = make([]HostInfo, 0)
	for _, key := range infra.HostMapList.Keys() {
		hi, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		if vcenter != "all" && hi.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && hi.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && hi.Cluster != cluster {
			continue
		}
		if hostname != "all" && hi.Name != hostname {
			continue
		}
		hiN := *hi
		hiN.Perf = make([]*HSPerf, len(hi.Perf))
		copy(hiN.Perf, hi.Perf)
		hostList = append(hostList, hiN)
	}

	sort.Sort(HSListID(hostList))

	return hostList, nil
}

// GetVMList возвращает список всех виртуальных машин из глобальной инфраструктуры.
func (infra *VMInfrastructure) GetVMList() ([]VMPoint, error) {
	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var vmList []VMPoint
	vmList = make([]VMPoint, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}
		vmN := *vm
		vmN.Perf = make([]*VMPerf, len(vm.Perf))
		copy(vmN.Perf, vm.Perf)
		vmList = append(vmList, vmN)
	}

	sort.Sort(VMListID(vmList))
	return vmList, nil
}

// GetVMListFilter возвращает список виртуальных машин, отфильтрованный по vcenter/datacenter/cluster
// и подстроке имени vmname (значения "all" или "" отключают фильтрацию по соответствующему полю).
func (infra *VMInfrastructure) GetVMListFilter(vcenter, datacenter, cluster string, vmname string) ([]VMPoint, error) {
	if infra == nil {
		return nil, errors.New(vminfraNotInitialised)
	}

	var vmList []VMPoint
	vmList = make([]VMPoint, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vcenter != "all" && vcenter != "" && vm.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && datacenter != "" && vm.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && cluster != "" && vm.Cluster != cluster {
			continue
		}
		if vmname != "all" && vmname != "" && !strings.Contains(vm.Name, vmname) {
			continue
		}

		vmN := *vm
		vmN.Perf = make([]*VMPerf, len(vm.Perf))
		copy(vmN.Perf, vm.Perf)
		vmList = append(vmList, vmN)
	}
	sort.Sort(VMListID(vmList))
	return vmList, nil
}

// GetVM возвращает виртуальную машину по точному ключу vcenter/datacenter/cluster/vmname.
func (infra *VMInfrastructure) GetVM(vcenter, datacenter, cluster string, vmname string) (VMPoint, error) {
	if infra == nil {
		return VMPoint{}, errors.New(vminfraNotInitialised)
	}
	//блокируем  и снимаем при выходе из функции

	key := vcenter + "/" + datacenter + "/" + cluster + "/" + vmname
	vm, ok := infra.VMMapList.Get(key)

	if ok {
		return *vm, nil
	}
	return VMPoint{}, errors.New("VM не найдена")
}

// GetVMKey возвращает виртуальную машину по её составному ключу (KeyName) в карте VMMapList.
func (infra *VMInfrastructure) GetVMKey(key string) (VMPoint, error) {
	if infra == nil {
		return VMPoint{}, errors.New(vminfraNotInitialised)
	}
	vm, ok := infra.VMMapList.Get(key)

	if ok {
		return *vm, nil
	}
	return VMPoint{}, errors.New("VM не найдена")
}

// GetVMPerfomance возвращает копию виртуальной машины с историей метрик по её числовому идентификатору (ID).
func (infra *VMInfrastructure) GetVMPerfomance(id int64) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vm.ID == id {
			vmp = *vm

			vmp.Perf = make([]*VMPerf, len(vm.Perf))
			copy(vmp.Perf, vm.Perf)
			return &vmp
		}
	}
	return &vmp
}

// GetVMPerfomanceName возвращает копию виртуальной машины с историей метрик,
// у которой HostName содержит подстроку name.
func (infra *VMInfrastructure) GetVMPerfomanceName(name string) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if strings.Contains(vm.HostName, name) {
			vmp = *vm

			vmp.Perf = make([]*VMPerf, len(vm.Perf))
			copy(vmp.Perf, vm.Perf)
			return &vmp
		}
	}
	return &vmp
}

// GetVMPerfomanceKey возвращает копию виртуальной машины с историей метрик по ключу vcenter/datacenter/cluster/vmname.
func (infra *VMInfrastructure) GetVMPerfomanceKey(vcenter string, datacenter string, cluster string, vmname string) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)

	key := vcenter + "/" + datacenter + "/" + cluster + "/" + vmname
	vm, ok := infra.VMMapList.Get(key)
	if ok {
		vmp = *vm
		vmp.Perf = make([]*VMPerf, len(vm.Perf))
		copy(vmp.Perf, vm.Perf)
		return &vmp
	}
	return &vmp
}

// GetVMPerfomanceID возвращает копию виртуальной машины с историей метрик по числовому идентификатору (ID).
func (infra *VMInfrastructure) GetVMPerfomanceID(id int64) *VMPoint {

	if infra == nil {
		return nil
	}

	vmp := VMPoint{}
	vmp.Perf = make([]*VMPerf, 0)
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}
		if vm.ID == id {
			vmp = *vm

			vmp.Perf = make([]*VMPerf, len(vm.Perf))
			copy(vmp.Perf, vm.Perf)
			return &vmp
		}
	}
	return &vmp
}

// normalizeVCenterHost убирает схему и хвост /sdk — в ConfigVCenters.Hostname
// должен быть только host[:port], но иногда указывают полный URL.

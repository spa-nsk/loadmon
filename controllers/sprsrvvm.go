package controllers

import (
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"
	"sort"
)

// SprSrvVM отображает список виртуальных машин с каскадным фильтром
// vcenter -> datacenter -> cluster -> vm. Значение "all" (или пустая
// строка в URL) снимает ограничение фильтра на соответствующем уровне -
// это тот же принцип, что уже используется в gRPC-сервере (grpcserver)
// для фильтрации кластеров/хостов/ВМ.
// Маршрут: /sprsrv/vm/:Vcenter/:Datacenter/:Cluster/:VM
func (this *MainController) SprSrvVM() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")
	vmname := this.Ctx.Input.Param(":VM")

	if vcenter == "" {
		vcenter = "all"
	}
	if datacenter == "" {
		datacenter = "all"
	}
	if cluster == "" {
		cluster = "all"
	}
	if vmname == "" {
		vmname = "all"
	}

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_vm_"+vcenter+"_"+datacenter+"_"+cluster+"_"+vmname, 302)
		return
	}

	this.activeContent("sprsrv/vmlist")

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster
	this.Data["CurVM"] = vmname

	// верхний уровень фильтра: список всех известных vcenter
	vcenters := grpcclient.LoadVCenterList("")
	vcenterNames := make([]string, 0, len(vcenters))
	for _, vc := range vcenters {
		if vc == nil {
			continue
		}
		vcenterNames = append(vcenterNames, vc.Hostname)
	}
	sort.Strings(vcenterNames)
	this.Data["VCenterList"] = vcenterNames

	// датацентры в рамках выбранного vcenter (или все, если vcenter == "all")
	datacenters := grpcclient.LoadDatacenterList(vcenter)
	datacenterNames := make([]string, 0, len(datacenters))
	for _, dc := range datacenters {
		if dc == nil {
			continue
		}
		datacenterNames = append(datacenterNames, dc.Name)
	}
	sort.Strings(datacenterNames)
	this.Data["DatacenterList"] = datacenterNames

	// кластеры в рамках выбранных vcenter/datacenter
	clusters := grpcclient.LoadClusterList(vcenter, datacenter)
	clusterNames := make([]string, 0, len(clusters))
	for _, cl := range clusters {
		if cl == nil {
			continue
		}
		clusterNames = append(clusterNames, cl.Name)
	}
	sort.Strings(clusterNames)
	this.Data["ClusterList"] = clusterNames

	// список ВМ в рамках выбранных vcenter/datacenter/cluster (для выпадающего
	// списка ":vm" и одновременно, если vmname == "all", как итоговые данные
	// таблицы - повторный запрос не требуется)
	vmSource := grpcclient.LoadVMList(vcenter, datacenter, cluster, "all")

	vmNameSet := make(map[string]struct{}, len(vmSource))
	vmNames := make([]string, 0, len(vmSource))
	for _, vm := range vmSource {
		if vm == nil {
			continue
		}
		if _, ok := vmNameSet[vm.Name]; ok {
			continue
		}
		vmNameSet[vm.Name] = struct{}{}
		vmNames = append(vmNames, vm.Name)
	}
	sort.Strings(vmNames)
	this.Data["VMNameList"] = vmNames

	var vmList []*vsphere.VMPoint
	if vmname == "all" {
		vmList = vmSource
	} else {
		vmList = grpcclient.LoadVMList(vcenter, datacenter, cluster, vmname)
	}

	sort.Slice(vmList, func(i, j int) bool {
		if vmList[i] == nil || vmList[j] == nil {
			return vmList[j] == nil && vmList[i] != nil
		}
		return vmList[i].Name < vmList[j].Name
	})

	this.Data["VMList"] = vmList
	this.Data["VMCount"] = len(vmList)
}

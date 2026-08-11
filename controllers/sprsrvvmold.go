package controllers

import (
	"loadmon/grpcclient"
	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
)

// SprSrvVMOld — предыдущая версия SprSrvVM (восстановлено из скриншотов).
// Использует шаблон sprsrv/vm и сохраняет фильтр в сессии (models.LoadFilter.Map).
func (this *MainController) SprSrvVMOld() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")
	vmname := this.Ctx.Input.Param(":VM")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_vm"+"_"+vcenter+"_"+datacenter+"_"+cluster+"_"+vmname, 302)
		return
	}

	this.activeContent("sprsrv/vm")

	this.Data["ViewOn"] = 0

	m := sess.(models.LoadFilter).Map

	VcenterId := "vcenterId"
	DatacenterId := "datacenterId"
	ClusterId := "clusterId"
	VmNameId := "vmnameId"

	vcList := make([]*vsphere.VCenter, 0)
	dcList := make([]*vsphere.Datacenter, 0)
	clList := make([]*vsphere.ClusterShort, 0)

	switch this.Ctx.Input.Method() {
	case "GET":
		if s, ok := m[VcenterId]; ok {
			vcenter = s.(string)
		}
		if s, ok := m[DatacenterId]; ok {
			datacenter = s.(string)
		}
		if s, ok := m[ClusterId]; ok {
			cluster = s.(string)
		}
		if s, ok := m[VmNameId]; ok {
			vmname = s.(string)
		}
	case "POST":
		s1 := this.GetString(VcenterId)
		s2 := this.GetString(DatacenterId)
		s3 := this.GetString(ClusterId)
		s4 := this.GetString(VmNameId)

		delete(m, VcenterId)
		delete(m, DatacenterId)
		delete(m, ClusterId)
		delete(m, VmNameId)

		if s1 != "" {
			m[VcenterId] = s1
		}
		if s2 != "" {
			m[DatacenterId] = s2
		}
		if s3 != "" {
			m[ClusterId] = s3
		}
		if s4 != "" {
			m[VmNameId] = s4
		}

		vcenter = s1
		datacenter = s2
		cluster = s3
		vmname = s4
		this.Data["ViewOn"] = 1
	}
	vcList = grpcclient.LoadVCenterList("all")
	dcList = grpcclient.LoadDatacenterList(vcenter)
	clList = grpcclient.LoadClusterList(vcenter, datacenter)

	this.Data[VcenterId] = vcenter
	this.Data[DatacenterId] = datacenter
	this.Data[ClusterId] = cluster
	this.Data[VmNameId] = vmname
	this.Data["VCenterList"] = vcList
	this.Data["DatacenterList"] = dcList
	this.Data["ClusterList"] = clList
	listCluster := make([]*vsphere.Cluster, 0)
	this.Data["ListCluster"] = listCluster

	if this.Data["ViewOn"] == 0 { //если фильтр не заполнялся
		return
	}

	vm_list := grpcclient.LoadVMList(vcenter, datacenter, cluster, vmname)
	//sort.Sort(vsphere.VMPerfCPUorMEM(vm_list))

	this.Data["VMList"] = vm_list
}

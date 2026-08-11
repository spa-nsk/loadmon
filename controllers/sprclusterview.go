package controllers

import (
	"loadmon/grpcclient"
	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
	"sort"
)

// SprCluster — просмотр кластеров и одиночных хостов (восстановлено из скриншотов).
// Использует шаблон sprsrv/cluster и сохраняет фильтр в сессии (models.LoadFilter.Map).
func (this *MainController) SprCluster() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_cluster"+"_"+vcenter+"_"+datacenter+"_"+cluster, 302)
		return
	}

	this.activeContent("sprsrv/cluster")

	this.Data["ViewOn"] = 0

	VcenterId := "vcenterId"
	DatacenterId := "datacenterId"
	ClusterId := "clusterId"

	vcList := make([]*vsphere.VCenter, 0)
	dcList := make([]*vsphere.Datacenter, 0)
	clList := make([]*vsphere.ClusterShort, 0)

	m := sess.(models.LoadFilter).Map

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
	case "POST":
		s1 := this.GetString(VcenterId)
		s2 := this.GetString(DatacenterId)
		s3 := this.GetString(ClusterId)

		delete(m, VcenterId)
		delete(m, DatacenterId)
		delete(m, ClusterId)

		if s1 != "" {
			m[VcenterId] = s1
		}
		if s2 != "" {
			m[DatacenterId] = s2
		}
		if s3 != "" {
			m[ClusterId] = s3
		}

		vcenter = s1
		datacenter = s2
		cluster = s3
		this.Data["ViewOn"] = 1
	}

	this.Data[VcenterId] = vcenter
	this.Data[DatacenterId] = datacenter
	this.Data[ClusterId] = cluster

	vcList = grpcclient.LoadVCenterList("all")
	dcList = grpcclient.LoadDatacenterList(vcenter)
	clList = grpcclient.LoadClusterList(vcenter, datacenter)

	this.Data["VcenterList"] = vcList
	this.Data["DatacenterList"] = dcList
	this.Data["ClusterList2"] = clList

	if this.Data["ViewOn"] == 0 {
		return
	}

	vmList := grpcclient.LoadVMList(vcenter, datacenter, cluster, "all")
	hsList := grpcclient.LoadHostListFull(vcenter, datacenter, cluster, "all")
	clusterList := grpcclient.LoadClusterListFull(vcenter, datacenter, cluster)

	grpcclient.LinkVMToCluster(vmList, clusterList)
	grpcclient.LoadVMListToClusterStorage(clusterList)
	grpcclient.LinkHostToCluster(hsList, clusterList)
	sort.Sort(vsphere.ClusterVcenter(clusterList))

	this.Data["ClusterList"] = clusterList
}

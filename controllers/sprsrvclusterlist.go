package controllers

import (
	"loadmon/grpcclient"
	"sort"
)

// SprSrvListClusterView отображает список кластеров vSphere по всем
// подключённым vCenter, по аналогии со списком виртуальных машин
// (controllers/sprsrvvm.go, views/sprsrv/vmlist.tpl). Данные берутся из
// vspheremetrics.Cluster (агрегированные метрики CPU/память/диск/сеть,
// состав хостов/ВМ/хранилищ).
//
// В отличие от /sprsrv/vm, реальное количество кластеров невелико (десятки,
// не тысячи), поэтому здесь не требуется двухшаговая загрузка по ключам -
// используется grpcclient.LoadClusterListFull, который через потоковый
// GetClusterFull уже поддерживает фильтр "all" на каждом уровне.
//
// Фильтр vcenter/datacenter/cluster передаётся через GET-параметры (а не
// сегменты пути), чтобы не ломать уже используемую в меню (header.tpl)
// ссылку "/sprsrv/listclusterview" без параметров.
//
// Маршрут: /sprsrv/listclusterview
func (this *MainController) SprSrvListClusterView() {

	vcenter := this.GetString("vcenter")
	datacenter := this.GetString("datacenter")
	cluster := this.GetString("cluster")

	if vcenter == "" {
		vcenter = "all"
	}
	if datacenter == "" {
		datacenter = "all"
	}
	if cluster == "" {
		cluster = "all"
	}

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_listclusterview", 302)
		return
	}

	this.activeContent("sprsrv/clusterlist")

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster

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

	// список кластеров в рамках выбранных vcenter/datacenter/cluster,
	// с полным набором данных (агрегированные метрики, счётчики хостов/ВМ)
	clusterList := grpcclient.LoadClusterListFull(vcenter, datacenter, cluster)

	clusterNameSet := make(map[string]struct{}, len(clusterList))
	clusterNames := make([]string, 0, len(clusterList))
	for _, cl := range clusterList {
		if cl == nil {
			continue
		}
		if _, ok := clusterNameSet[cl.Name]; ok {
			continue
		}
		clusterNameSet[cl.Name] = struct{}{}
		clusterNames = append(clusterNames, cl.Name)
	}
	sort.Strings(clusterNames)
	this.Data["ClusterList"] = clusterNames

	sort.Slice(clusterList, func(i, j int) bool {
		if clusterList[i] == nil || clusterList[j] == nil {
			return clusterList[j] == nil && clusterList[i] != nil
		}
		if clusterList[i].Vcenter != clusterList[j].Vcenter {
			return clusterList[i].Vcenter < clusterList[j].Vcenter
		}
		if clusterList[i].Datacenter != clusterList[j].Datacenter {
			return clusterList[i].Datacenter < clusterList[j].Datacenter
		}
		return clusterList[i].Name < clusterList[j].Name
	})

	for _, cl := range clusterList {
		if cl == nil {
			continue
		}
		cl.CalcPerfomance()
	}

	this.Data["ClusterFullList"] = clusterList
	this.Data["ClusterCount"] = len(clusterList)
}

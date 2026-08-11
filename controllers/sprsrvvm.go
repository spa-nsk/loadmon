package controllers

import (
	"loadmon/grpcclient"
	pb "loadmon/protocol"
	"sort"
	"strconv"
)

// SprSrvVM отображает список виртуальных машин с каскадным фильтром
// vcenter -> datacenter -> cluster -> vm. Значение "all" (или пустая
// строка в URL) снимает ограничение фильтра на соответствующем уровне -
// это тот же принцип, что уже используется в gRPC-сервере (grpcserver)
// для фильтрации кластеров/хостов/ВМ.
//
// Набор отображаемых данных и структура строки таблицы повторяют старый
// шаблон views/sprsrv/vm.tpl (см. controllers/sprsrvvmold.go), но с
// исправленными именами полей (vsphere.VMPoint, а не устаревший
// models.VMSprRecord) и оформлением "классической таблицы", как в
// /sprsrv/view2 (тема light/dark, LED-статус, пагинация).
//
// Реальные объёмы - до 30000 ВМ, поэтому список грузится в два шага, без
// изменений в protocol/*.proto (существующих сообщений достаточно):
//  1. grpcclient.LoadVMListKeys -> GetVMListKey возвращает только ключи
//     (VMName+Key) всех ВМ, подходящих под фильтр - лёгкие объекты,
//     без Perf/Disk/Files/UsageDatastore. По ним же строится список для
//     выпадающего меню "Виртуальная машина" и считается общее количество
//     (VMCount), без создания тяжёлых VMPoint.
//  2. grpcclient.LoadVMByKeyRecs -> GetVMStreamByKeyList догружает полные
//     VMPoint (со всеми дисками/снапшотами/RDM) только для ключей текущей
//     страницы (не более PageSize штук), а не для всей выборки.
//
// Таким образом при фильтре "все" по всем уровням сеть и сервер больше не
// строят и не передают 30000 полных объектов ради отображения одной
// страницы - тяжёлые данные запрашиваются только для видимых записей.
//
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

	// ключи ВМ в рамках выбранных vcenter/datacenter/cluster (для выпадающего
	// списка ":vm" и для подсчёта общего количества/пагинации) - лёгкий
	// запрос, полные объекты VMPoint здесь не строятся
	keySource := grpcclient.LoadVMListKeys(vcenter, datacenter, cluster, "all")

	vmNameSet := make(map[string]struct{}, len(keySource))
	vmNames := make([]string, 0, len(keySource))
	for _, k := range keySource {
		if k == nil {
			continue
		}
		if _, ok := vmNameSet[k.VMName]; ok {
			continue
		}
		vmNameSet[k.VMName] = struct{}{}
		vmNames = append(vmNames, k.VMName)
	}
	sort.Strings(vmNames)
	this.Data["VMNameList"] = vmNames

	var keyList []*pb.VMKeyRec
	if vmname == "all" {
		keyList = keySource
	} else {
		keyList = make([]*pb.VMKeyRec, 0, len(keySource))
		for _, k := range keySource {
			if k != nil && k.VMName == vmname {
				keyList = append(keyList, k)
			}
		}
	}

	sort.Slice(keyList, func(i, j int) bool {
		if keyList[i] == nil || keyList[j] == nil {
			return keyList[j] == nil && keyList[i] != nil
		}
		return keyList[i].VMName < keyList[j].VMName
	})

	// пагинация на уровне ключей (как в /sprsrv/view2, но раньше по потоку
	// данных) - полные данные (Disk/Perf/Files/UsageDatastore) запрашиваются
	// только для ключей текущей страницы, а не для всей выборки целиком;
	// именно это устраняет задержку при фильтре "все" на объёмах до 30000 ВМ
	pageSize, _ := strconv.Atoi(this.GetString("pagesize"))
	switch pageSize {
	case 25, 50, 100, 200:
	default:
		pageSize = 50
	}
	page, _ := strconv.Atoi(this.GetString("page"))
	if page < 1 {
		page = 1
	}
	totalRecords := len(keyList)
	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalRecords {
		end = totalRecords
	}
	pagedKeys := make([]*pb.VMKeyRec, 0)
	if offset < end {
		pagedKeys = keyList[offset:end]
	}

	pagedList := grpcclient.LoadVMByKeyRecs(pagedKeys)
	sort.Slice(pagedList, func(i, j int) bool {
		if pagedList[i] == nil || pagedList[j] == nil {
			return pagedList[j] == nil && pagedList[i] != nil
		}
		return pagedList[i].Name < pagedList[j].Name
	})

	this.Data["VMList"] = pagedList
	this.Data["VMCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = 0
	if totalRecords > 0 {
		this.Data["PageFrom"] = offset + 1
	}
	this.Data["PageTo"] = end
}

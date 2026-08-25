package controllers

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"loadmon/grpcclient"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
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

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport) // выгрузка списка ВМ в XLSX
	} else {
		RegisterRequest(this, LogTypeViewReference) // просмотр списка виртуальных машин
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
		// Поиск по частично введённому имени ВМ (регистронезависимо, без учёта
		// позиции подстроки) - раньше значение ":VM" бралось из выпадающего
		// списка точных имён, теперь это поле свободного ввода на форме.
		needle := strings.ToLower(vmname)
		keyList = make([]*pb.VMKeyRec, 0, len(keySource))
		for _, k := range keySource {
			if k != nil && strings.Contains(strings.ToLower(k.VMName), needle) {
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

	// Экспорт в XLSX отдаёт ВЕСЬ найденный по фильтру список (все ключи из
	// keyList), а не только текущую страницу - тот же принцип, что и в
	// /sprsrv/hwviewlist. Набор столбцов повторяет то, что показывает
	// html-таблица этой страницы (views/sprsrv/vmlist.tpl): столбцы
	// Id/vCenter/Датацентр/Кластер/Имя/Хост/CPU/Память/Сеть/IP + сводка по
	// дискам/хранилищам/снапшотам, вложенным в ячейку строки ВМ в html.
	if this.GetString("export") == "xlsx" {
		fullList := grpcclient.LoadVMByKeyRecs(keyList)
		sort.Slice(fullList, func(i, j int) bool {
			if fullList[i] == nil || fullList[j] == nil {
				return fullList[j] == nil && fullList[i] != nil
			}
			return fullList[i].Name < fullList[j].Name
		})
		flash := beego.NewFlash()
		ExportVMListXLSX(this, flash, fullList)
		return
	}

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

// vmListXLSXHeader - заголовок листа XLSX-экспорта /sprsrv/vm/... Порядок и
// состав столбцов повторяют то, что показывает html-таблица этой страницы
// (views/sprsrv/vmlist.tpl): сводные столбцы CPU/память/сеть/IP как есть, а
// вложенные в ячейку строки ВМ html-подтаблицы (диски, хранилища и RDM,
// снапшоты) сведены в текстовые столбцы с разделителем "; " на запись.
var vmListXLSXHeader = []string{
	"Id", "vCenter", "Датацентр", "Кластер", "Имя", "Хост (ESXi)",
	"Рабочая станция", "Питание",
	"Процессоры", "Ядра", "CPU макс, %",
	"Память, ГБ", "Память макс, %",
	"Сеть", "IP адрес",
	"Путь", "Диски всего, ГБ", "Занято, %", "Свободно, ГБ",
	"Диски (детально)", "Хранилища и RDM", "Снапшоты",
}

// vmListXLSXRow формирует одну строку XLSX-экспорта из vsphere.VMPoint -
// см. комментарий к vmListXLSXHeader про соответствие html-таблице.
func vmListXLSXRow(vm *vsphere.VMPoint) []string {
	if vm == nil {
		return make([]string, len(vmListXLSXHeader))
	}

	workStation := "нет"
	if vm.WorkStation {
		workStation = "да"
	}
	powerState := "Выключена"
	if vm.PowerState {
		powerState = "Включена"
	}

	disks := make([]string, 0, len(vm.Disk))
	for _, d := range vm.Disk {
		disks = append(disks, fmt.Sprintf("%s: %.2f ГБ (%.2f%%, %.2f ГБ своб.)", d.DiskPath, d.CapacityGb, d.PercentUsed, d.FreeSpaceGb))
	}

	storages := make([]string, 0, len(vm.UsageDatastore))
	for _, u := range vm.UsageDatastore {
		if u.PercentAllocateStorage > 0 {
			storages = append(storages, fmt.Sprintf("%s: %.2f ГБ (%.2f%%)", u.StorageName, u.CommittedGb, u.PercentAllocateStorage))
		} else {
			storages = append(storages, fmt.Sprintf("%s %s: %.2f ГБ", u.StorageName, u.Datastore, u.CommittedGb))
		}
	}

	snapshots := make([]string, 0)
	for _, f := range vm.Files {
		if f.Type == "snapshotData" {
			snapshots = append(snapshots, fmt.Sprintf("%s: %.2f КБ", f.Name, f.Size))
		}
	}

	return []string{
		strconv.FormatInt(vm.ID, 10),
		vm.Vcenter,
		vm.Datacenter,
		vm.Cluster,
		vm.Name,
		vm.HostName,
		workStation,
		powerState,
		strconv.FormatInt(int64(vm.NumCPU), 10),
		strconv.FormatInt(int64(vm.NumCores), 10),
		fmt.Sprintf("%.2f", vm.CPUUsageMaximum),
		fmt.Sprintf("%.2f", vm.MemorySizeGB),
		fmt.Sprintf("%.2f", vm.MemUsageMaximum),
		vm.Network,
		vm.IPAddress,
		vm.PathName,
		fmt.Sprintf("%.2f", vm.CapacityGb),
		fmt.Sprintf("%.2f", vm.PercentUsed),
		fmt.Sprintf("%.2f", vm.FreeSpaceGb),
		strings.Join(disks, "; "),
		strings.Join(storages, "; "),
		strings.Join(snapshots, "; "),
	}
}

// ExportVMListXLSX отдаёт найденный по фильтру список виртуальных машин
// (весь, а не только текущую страницу) файлом XLSX на скачивание -
// формирование листа аналогично экспорту /sprsrv/hwviewlist.
func ExportVMListXLSX(this *MainController, flash *beego.FlashData, vmList []*vsphere.VMPoint) {
	if len(vmList) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("vm%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Виртуальные машины"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &vmListXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, vm := range vmList {
		row := vmListXLSXRow(vm)
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" ", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		return
	}

	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	this.Ctx.Output.Header("Content-Disposition", "attachment; filename=\""+shortName+"\"")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")
	this.Ctx.Output.Header("Content-Length", fmt.Sprint(fileInfo.Size()))

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}

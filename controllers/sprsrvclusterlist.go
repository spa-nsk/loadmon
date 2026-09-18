package controllers

import (
	"fmt"
	"loadmon/grpcclient"
	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
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

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport) // выгрузка списка кластеров в XLSX
	} else {
		RegisterRequest(this, LogTypeViewReference) // просмотр списка кластеров
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

	// Справочник кластеров (models.ClusterSprRecords) - используется для
	// отображения дополнительных колонок (тип среды/статус/примечание) и
	// для экспорта в XLSX. Поиск ведётся по тому же составному ключу
	// "Vcenter/Datacenter/Name", что и в cmdb (см. cmdb/savedata.go,
	// grpcclient.LoadClusterSpr) - именно Name (наименование кластера), а не
	// ClusterId.
	clusterSpr, err := grpcclient.LoadClusterSpr(vcenter, datacenter, cluster)
	if err != nil || clusterSpr == nil {
		clusterSpr = make(map[string]*models.ClusterSprRecords)
	}
	this.Data["ClusterSprMap"] = clusterSpr

	// экспорт в XLSX отдаёт весь найденный по фильтру список, а не только
	// текущую страницу - тот же приём, что и в /sprsrv/vm и /sprsrv/hwviewlist
	if this.GetString("export") == "xlsx" {
		flash := beego.NewFlash()
		ExportClusterListXLSX(this, flash, clusterList, clusterSpr)
		return
	}

	// пагинация - тот же приём и те же параметры (page/pagesize), что и в
	// /sprsrv/vm (controllers/sprsrvvm.go, views/sprsrv/vmlist.tpl); в
	// отличие от /sprsrv/vm здесь не требуется двухшаговая загрузка по
	// ключам (кластеров на порядок меньше, чем ВМ), поэтому нарезка страницы
	// выполняется прямо по уже полученному clusterList.
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
	totalRecords := len(clusterList)
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
	pagedList := make([]*vsphere.Cluster, 0)
	if offset < end {
		pagedList = clusterList[offset:end]
	}

	this.Data["ClusterFullList"] = pagedList
	this.Data["ClusterCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = 0
	if totalRecords > 0 {
		this.Data["PageFrom"] = offset + 1
	}
	this.Data["PageTo"] = end
}

// clusterSprKey строит ключ вида "Vcenter/Datacenter/Name" для поиска записи
// справочника кластеров (models.ClusterSprRecords) - тот же формат ключа,
// что используют grpcclient.LoadClusterSpr и cmdb.SaveClusterCMDBData; Name
// здесь - именно наименование кластера, а не ClusterId.
func clusterSprKey(vcenter, datacenter, name string) string {
	return fmt.Sprintf("%s/%s/%s", vcenter, datacenter, name)
}

// clusterTypeWorkLabel переводит код models.ClusterSprRecords.TypeWork в
// текст - соответствует расшифровке, использованной в других формах
// (view2.tpl, updatebase.tpl) для того же поля у серверов.
func clusterTypeWorkLabel(v int) string {
	switch v {
	case 1:
		return "Продуктовая среда"
	case 2:
		return "Внутренний тест"
	case 3:
		return "Внешний тест"
	default:
		return "не указано"
	}
}

// clusterStatLabel переводит код models.ClusterSprRecords.Stat в текст -
// расшифровка значений взята из комментария к полю в models/models_add.go.
func clusterStatLabel(v int) string {
	switch v {
	case 1:
		return "Активен"
	case 2:
		return "Выведен из работы"
	default:
		return "не указано"
	}
}

// clusterListXLSXHeader - заголовок листа XLSX-экспорта /sprsrv/listclusterview.
// Порядок и состав столбцов повторяют то, что показывает html-таблица этой
// страницы (views/sprsrv/clusterlist.tpl), плюс колонки из справочника
// кластеров (тип среды/статус/примечание).
var clusterListXLSXHeader = []string{
	"Id", "vCenter", "Датацентр", "Кластер",
	"Хосты",
	"Процессоры", "Ядра", "Потоки", "CPU макс, %",
	"Память, ГБ", "Память макс, %",
	"Хранилище, ГБ", "Занято, %",
	"Виртуальные машины",
	"Тип среды", "Статус", "Примечание",
}

// clusterListXLSXRow формирует одну строку XLSX-экспорта из vsphere.Cluster,
// дополняя её данными из справочника кластеров (models.ClusterSprRecords),
// найденного по ключу vCenter/Датацентр/Имя кластера.
func clusterListXLSXRow(cl *vsphere.Cluster, sprMap map[string]*models.ClusterSprRecords) []string {
	if cl == nil {
		return make([]string, len(clusterListXLSXHeader))
	}

	typeWorkText := clusterTypeWorkLabel(0)
	statText := clusterStatLabel(0)
	annotation := ""
	if spr, ok := sprMap[clusterSprKey(cl.Vcenter, cl.Datacenter, cl.Name)]; ok && spr != nil {
		typeWorkText = clusterTypeWorkLabel(spr.TypeWork)
		statText = clusterStatLabel(spr.Stat)
		annotation = spr.Annotation
	}

	return []string{
		strconv.FormatInt(cl.ID, 10),
		cl.Vcenter,
		cl.Datacenter,
		cl.Name,
		strconv.FormatInt(cl.HostCount, 10),
		strconv.FormatInt(int64(cl.AllNumCPUPkgs), 10),
		strconv.FormatInt(int64(cl.AllNumCPUCores), 10),
		strconv.FormatInt(int64(cl.AllNumCPUThreads), 10),
		fmt.Sprintf("%.2f", cl.CPUUsageMaximum),
		fmt.Sprintf("%.2f", cl.AllMemorySizeGB),
		fmt.Sprintf("%.2f", cl.MemUsageMaximum),
		fmt.Sprintf("%.2f", cl.AllCapacityGb),
		fmt.Sprintf("%.2f", cl.AllPercentUsed),
		strconv.FormatInt(cl.VMCount, 10),
		typeWorkText,
		statText,
		annotation,
	}
}

// ExportClusterListXLSX отдаёт найденный по фильтру список кластеров (весь,
// а не только текущую страницу) файлом XLSX на скачивание - формирование
// листа аналогично экспорту /sprsrv/vm (ExportVMListXLSX, controllers/sprsrvvm.go).
func ExportClusterListXLSX(this *MainController, flash *beego.FlashData, clusterList []*vsphere.Cluster, sprMap map[string]*models.ClusterSprRecords) {
	if len(clusterList) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("cluster%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Кластеры"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &clusterListXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, cl := range clusterList {
		row := clusterListXLSXRow(cl, sprMap)
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" %v", err)
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

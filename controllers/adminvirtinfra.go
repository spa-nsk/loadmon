package controllers

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"sort"
	"strings"

	models "loadmon/models"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/client/orm"
)

// ---- JSON-модель для canvas-страницы "Виртуальная инфраструктура" ----
//
// Иерархия vCenter -> Datacenter -> Cluster -> Host, аналогично
// buildClusterMapPage (controllers/infrastructure_canvas.go), но с
// добавлением списка хостов и их параметров (для подсказки при наведении) -
// то, чего в старой странице "Кластеры vSphere" не было (там только счётчики).

type VirtInfraHost struct {
	Name       string  `json:"name"`
	SerialNum  string  `json:"serialNum,omitempty"`
	PowerOn    bool    `json:"powerOn"`
	CPUCount   float64 `json:"cpuCount"`
	CPULogical float64 `json:"cpuLogical"`
	MemSizeGb  float64 `json:"memSizeGb"`
	VMCount    int     `json:"vmCount"`
}

type VirtInfraCluster struct {
	Name      string          `json:"name"`
	TypeWork  int             `json:"typeWork"`
	VMCount   int             `json:"vmCount"`
	HostCount int             `json:"hostCount"`
	Hosts     []VirtInfraHost `json:"hosts"`
}

type VirtInfraDatacenter struct {
	Name     string             `json:"name"`
	Clusters []VirtInfraCluster `json:"clusters"`
}

type VirtInfraVcenter struct {
	Name        string                `json:"name"`
	Datacenters []VirtInfraDatacenter `json:"datacenters"`
}

type VirtInfraStats struct {
	VcenterCount    int `json:"vcenterCount"`
	DatacenterCount int `json:"datacenterCount"`
	ClusterCount    int `json:"clusterCount"`
	HostCount       int `json:"hostCount"`
	VMCount         int `json:"vmCount"`
}

type VirtInfraData struct {
	Vcenters []VirtInfraVcenter `json:"vcenters"`
	Stats    VirtInfraStats     `json:"stats"`
}

func buildVirtInfraData() (VirtInfraData, error) {
	o := orm.NewOrm()
	var data VirtInfraData

	var sprRows []models.ServerSprRecords
	if _, err := models.ServerSprQS(o).
		Filter("Stat", 1).Filter("Active", 1).All(&sprRows); err != nil {
		return data, err
	}
	var clusterRows []models.ClusterSprRecords
	if _, err := o.QueryTable(new(models.ClusterSprRecords)).Filter("Stat", 1).All(&clusterRows); err != nil {
		return data, err
	}

	type key struct{ vc, dc, cl string }
	norm := func(s string) string {
		s = strings.TrimSpace(s)
		if s == "" {
			return "—"
		}
		return s
	}

	hostsByKey := map[key][]VirtInfraHost{}
	vmCountByKey := map[key]int{}
	for _, s := range sprRows {
		if strings.TrimSpace(s.Cluster) == "" {
			continue
		}
		k := key{vc: norm(s.Vcenter), dc: norm(s.Datacenter), cl: norm(s.Cluster)}
		switch s.Type {
		case 1:
			vmCountByKey[k]++
		case 2:
			hostsByKey[k] = append(hostsByKey[k], VirtInfraHost{
				Name: s.Name, SerialNum: s.SerialNumber, PowerOn: s.PowerState == 1,
				CPUCount: s.CPUCount, CPULogical: s.CPULogical, MemSizeGb: s.MemSize,
			})
		}
	}
	// VMCount на хост - недоступен без ссылки VM->host в этой модели
	// (VMSprId/HWSprId не образуют такую связь напрямую), поэтому вложенные
	// ВМ на уровне кластера агрегируются одним числом (см. Cluster.VMCount),
	// а не на каждый хост - для наведения на хост это соответствует
	// имеющимся данным.

	typeWorkByKey := map[key]int{}
	vcMap := map[string]map[string][]VirtInfraCluster{}
	seen := map[key]bool{}
	for _, cl := range clusterRows {
		k := key{vc: norm(cl.Vcenter), dc: norm(cl.Datacenter), cl: norm(cl.Name)}
		if strings.TrimSpace(cl.Name) == "" {
			continue
		}
		seen[k] = true
		typeWorkByKey[k] = cl.TypeWork
		if _, ok := vcMap[k.vc]; !ok {
			vcMap[k.vc] = map[string][]VirtInfraCluster{}
		}
		hosts := hostsByKey[k]
		sort.Slice(hosts, func(i, j int) bool { return hosts[i].Name < hosts[j].Name })
		vcMap[k.vc][k.dc] = append(vcMap[k.vc][k.dc], VirtInfraCluster{
			Name: k.cl, TypeWork: cl.TypeWork, VMCount: vmCountByKey[k],
			HostCount: len(hosts), Hosts: hosts,
		})
	}
	// Кластеры, встречающиеся в sprRows, но отсутствующие в справочнике.
	for k, hosts := range hostsByKey {
		if seen[k] {
			continue
		}
		if _, ok := vcMap[k.vc]; !ok {
			vcMap[k.vc] = map[string][]VirtInfraCluster{}
		}
		sortedHosts := append([]VirtInfraHost(nil), hosts...)
		sort.Slice(sortedHosts, func(i, j int) bool { return sortedHosts[i].Name < sortedHosts[j].Name })
		vcMap[k.vc][k.dc] = append(vcMap[k.vc][k.dc], VirtInfraCluster{
			Name: k.cl, TypeWork: typeWorkByKey[k], VMCount: vmCountByKey[k],
			HostCount: len(sortedHosts), Hosts: sortedHosts,
		})
	}

	vcNames := make([]string, 0, len(vcMap))
	for name := range vcMap {
		vcNames = append(vcNames, name)
	}
	sort.Strings(vcNames)

	totalDC := 0
	for _, vcName := range vcNames {
		dcMap := vcMap[vcName]
		dcNames := make([]string, 0, len(dcMap))
		for dcName := range dcMap {
			dcNames = append(dcNames, dcName)
		}
		sort.Strings(dcNames)
		vc := VirtInfraVcenter{Name: vcName}
		for _, dcName := range dcNames {
			clusters := dcMap[dcName]
			sort.Slice(clusters, func(i, j int) bool { return clusters[i].Name < clusters[j].Name })
			vc.Datacenters = append(vc.Datacenters, VirtInfraDatacenter{Name: dcName, Clusters: clusters})
			totalDC++
			data.Stats.ClusterCount += len(clusters)
			for _, c := range clusters {
				data.Stats.HostCount += c.HostCount
				data.Stats.VMCount += c.VMCount
			}
		}
		data.Vcenters = append(data.Vcenters, vc)
	}
	data.Stats.VcenterCount = len(data.Vcenters)
	data.Stats.DatacenterCount = totalDC

	return data, nil
}

// AdminVirtInfra отображает canvas-страницу "Виртуальная инфраструктура":
// vCenter -> Datacenter -> Cluster -> Host, с подсказками по параметрам при
// наведении. Реализована обычным JS+canvas.
//
// Маршрут: /admin/virtinfra
func (this *MainController) AdminVirtInfra() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/virtinfra")
	flash := beego.NewFlash()
	RegisterRequest(this, LogTypeAdminAction)

	data, err := buildVirtInfraData()
	if err != nil {
		log.Printf("AdminVirtInfra: build data: %v", err)
		flash.Error("Не удалось загрузить данные виртуальной инфраструктуры: %v", err)
		this.Data["VirtInfraJSON"] = base64.StdEncoding.EncodeToString([]byte("{}"))
	} else {
		raw, err := json.Marshal(data)
		if err != nil {
			log.Printf("AdminVirtInfra: marshal: %v", err)
			flash.Error("Ошибка сериализации данных виртуальной инфраструктуры")
			this.Data["VirtInfraJSON"] = base64.StdEncoding.EncodeToString([]byte("{}"))
		} else {
			this.Data["VirtInfraJSON"] = base64.StdEncoding.EncodeToString(raw)
		}
	}

	flash.Store(&this.Controller)
	this.TplName = "admin/virtinfra.tpl"
}

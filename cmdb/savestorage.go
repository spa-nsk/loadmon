package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"
	"strings"

	//"loadmon/itsystem"
	models "loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func LoadStorageSpr() (map[string]*models.StorageSystems, map[string]*models.StorageSystems) {
	o := orm.NewOrm()
	//o.Using("default")
	var storSys []*models.StorageSystems
	qs := o.QueryTable("storage_systems")
	qs = qs.OrderBy("Id")
	num, err := qs.All(&storSys)
	if err != nil || num == 0 {
		return nil, nil
	}
	mapStorSys := make(map[string]*models.StorageSystems)
	mapStorSysWWN := make(map[string]*models.StorageSystems)
	for i, shd := range storSys {
		if shd.Power != 1 {
			continue
		} //пропускаем выключенные СХД
		mapStorSys[shd.ShortName] = storSys[i] //запоминаем в карте по краткому имени СХД

		/*
			wwnMask := strings.Split(storSys[i].WWNMask, "\n")
			for _, mask := range wwnMask {
				wwn := GetWWNMaskKey(mask)
				mapStorSysWWN[wwn] = storSys[i]
			}
		*/
		for mask := range strings.SplitSeq(storSys[i].WWNMask, "\n") {
			wwn := GetWWNMaskKey(mask)
			mapStorSysWWN[wwn] = storSys[i]
		}

	}
	return mapStorSys, mapStorSysWWN
}

// SaveStorageVolumes сохраняет в справочнике StorageVolumes сведения о хранилищах
// (датасторах) указанного кластера vSphere, создавая либо обновляя записи по UID тома.
//
// Перед скан ом ЭТОГО кластера сбрасывает Status=0 у ранее сохранённых
// vSphere-датасторов именно этого кластера (resetClusterStorageVolumesStatus)
// - функция вызывается только ПОСЛЕ успешного сохранения записи кластера
// (см. cmdb/savemetric_orchestration.go:saveClusterData, вызов идёт сразу за
// успешным SaveClusterRecord), поэтому это корректный, локально
// подтверждённый "mark and sweep": сбрасываем статус только того, что
// прямо сейчас будет заново просканировано, а не всех датасторов БД разом
// (как раньше делала общая cmdb.ClearStatusVolumeStorage - удалена, см.
// kontext/05-conventions-and-gotchas.md, раздел про СХД). Датастор, реально
// не найденный в текущем списке ds, останется со Status=0 - датастор,
// найденный - будет переустановлен в 1 через ImportStorage ниже.
func SaveStorageVolumes(cl *vsphere.Cluster) {
	ds := grpcclient.GetClusterStorages(cl.Vcenter, cl.Datacenter, cl.Name)

	o := orm.NewOrm()

	resetClusterStorageVolumesStatus(o, cl)

	for _, st := range ds {

		trn, err := o.Begin()
		//defer trn.RollbackUnlessCommit()
		if err != nil {
			fmt.Println("Ошибка начала транзакции ", err)
			return
		}

		volume := models.StorageVolumes{}
		volume.UID = strings.ToUpper(st.UID)
		foundVolume := false
		err = trn.Read(&volume, "UID")
		if err == nil {
			foundVolume = true
		}
		if foundVolume && volume.SourceUpdateType == models.StorageSourceUpdateManual {
			// запись заведена вручную - приоритет источников (2026-09-03):
			// автообновление (в т.ч. приоритетное vSphere) её не трогает, см.
			// kontext/05-conventions-and-gotchas.md, раздел про СХД
			trn.Rollback()
			continue
		}
		ImportStorageCL(cl, &volume)
		ImportStorage(st, &volume)
		if foundVolume {
			_, err = trn.Update(&volume)
			if err != nil {
				fmt.Println("Ошибка обновления записи StorageVolumes UID=", st.UID, err)
				trn.Rollback()
				continue
			}
		} else {
			_, err = trn.Insert(&volume)
			if err != nil {
				fmt.Println("Ошибка вставки записи StorageVolumes UID=", st.UID, err)
				trn.Rollback()
				continue
			}
		}
		err = trn.Commit()
		if err != nil {
			fmt.Println("Ошибка завершения транзакции", err)
			trn.Rollback()
		}
	}
}

// resetClusterStorageVolumesStatus сбрасывает Status=0 у vSphere-датасторов
// (SourceUpdateType == StorageSourceUpdateVMware) конкретного кластера,
// сейчас находящихся в Status=1 - подготовка к их пересбору в
// SaveStorageVolumes. Тома других источников (StorageSourceUpdateStorageAPI,
// StorageSourceUpdateManual) этим кластером не затрагиваются - см. подробный
// комментарий у SaveStorageVolumes выше. Один SQL UPDATE через QuerySeter,
// без построчного Read+Update.
func resetClusterStorageVolumesStatus(o orm.Ormer, cl *vsphere.Cluster) {
	_, err := o.QueryTable("storage_volumes").
		Filter("Vcenter", cl.Vcenter).
		Filter("Datacenter", cl.Datacenter).
		Filter("Cluster", cl.Name).
		Filter("SourceUpdateType", models.StorageSourceUpdateVMware).
		Filter("Status", 1).
		Update(orm.Params{"Status": 0})
	if err != nil {
		fmt.Println("Ошибка сброса статуса датасторов кластера перед пересканом:", cl.Vcenter, cl.Datacenter, cl.Name, err)
	}
}

// ImportStorageCL заполняет в записи тома StorageVolumes принадлежность к кластеру
// vSphere и список подключённых к нему хостов.
func ImportStorageCL(cl *vsphere.Cluster, volume *models.StorageVolumes) {
	volume.Vcenter = cl.Vcenter
	volume.Datacenter = cl.Datacenter
	volume.Cluster = cl.Name

	//добавить получение списка хостов для кластера сейчас пустой список
	hosts := grpcclient.GetClusterHost(cl.Vcenter, cl.Datacenter, cl.Name)
	for i, host := range hosts {
		hn, _ := SplitNameFQDN(host.Name)
		if i == 0 {
			volume.Server = hn
		} else {
			volume.Server = fmt.Sprintf("%s;%s", volume.Server, hn)
		}
	}
}

// ImportStorage переносит данные датастора vSphere (Datastore) в запись справочника
// томов хранения StorageVolumes.
func ImportStorage(from *vsphere.Datastore, to *models.StorageVolumes) {
	to.Status = 1
	to.UID = strings.ToUpper(from.UID)
	to.VirtualName = from.StorageName
	to.DiskType = from.Type
	to.CapacityGb = from.CapacityGb
	to.FreeSpaceGb = from.FreeSpaceGb
	to.PercentUsed = from.PercentUsed
	to.Time_Mfy = time.Now().Local()
	to.SourceUpdateType = models.StorageSourceUpdateVMware
}

// SaveClusterRecord создаёт запись метрик кластера ClusterRecord за текущий период и
// сохраняет связанные с ней метрики производительности ClusterPerf. Возвращает Id
// созданной записи ClusterRecord.

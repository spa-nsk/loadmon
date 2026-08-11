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
func SaveStorageVolumes(cl *vsphere.Cluster) {
	ds := grpcclient.GetClusterStorages(cl.Vcenter, cl.Datacenter, cl.Name)

	o := orm.NewOrm()

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
}

// SaveClusterRecord создаёт запись метрик кластера ClusterRecord за текущий период и
// сохраняет связанные с ней метрики производительности ClusterPerf. Возвращает Id
// созданной записи ClusterRecord.

package cmdb

import (
	"fmt"
	"loadmon/lenovostorage"
	models "loadmon/models"
	datamap "loadmon/utilities"
	"loadmon/vspheremetrics"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// UpdateLenovoVolume подключается к СХД Lenovo, получает список томов, групп хостов и
// их привязку к кластерам, после чего обновляет либо создаёт соответствующие записи
// StorageVolumes в базе данных CMDB.
//
// Возвращает список Id обновлённых/вставленных за этот вызов записей
// StorageVolumes и ошибку уровня подключения/авторизации - см. комментарий у
// UpdateHuaweyVolume (тот же принцип, используется тем же вызывающим кодом
// cmdb.UpdateVolumeStorage).
func UpdateLenovoVolume(shd *models.StorageSystems) ([]int64, error) {
	key := "monitor"
	c, err := lenovostorage.NewSession(shd.IpAddress, true, key, datamap.GetKeyData(key))
	if err != nil {
		fmt.Println("Ошибка авторизации", err)
		return nil, err
	}
	volumes, err := c.GetVolumes()
	if err != nil {
		fmt.Println("Error GetVolumes")
		return nil, err
	}

	cluster := make(map[string][]string, 0)
	hostGroups, err := c.GetHostGroup()

	if err != nil {
		fmt.Println("Error GetHostGroup")
	}

	for _, hgrp := range hostGroups {
		cluster[hgrp.ClusterRef] = make([]string, 0)
		fmt.Println(hgrp.Name)
	}

	hosts, err := c.GetHosts()

	if err != nil {
		fmt.Println("Error GetHosts")
	}

	for _, host := range hosts {
		cl, ok := cluster[host.ClusterRef]
		if ok {
			cluster[host.ClusterRef] = append(cl, host.Name)
		} else {
			s := make([]string, 1)
			s[0] = host.Name
			cluster[host.ClusterRef] = s
		}
	}

	o := orm.NewOrm()

	touchedIds := make([]int64, 0, len(volumes))

	for _, volume := range volumes {
		uid := vspheremetrics.VmfsStrLunToUID(volume.WorldWideName)
		capacity, err := strconv.ParseUint(volume.Capacity, 10, 64)
		if err != nil {
			fmt.Println("Error convert ", volume.Capacity, "to uint64", err)
		}
		capacity = capacity >> 30

		server := ""
		for _, listMap := range volume.ListOfMappings {
			cl, ok := cluster[listMap.MapRef]
			if ok {

				for _, c := range cl {
					if server == "" {
						server = c
					}
					server = fmt.Sprintf("%s;%s", server, c)
				}
			}
		}
		foundLUN := false

		voln := models.StorageVolumes{}

		if !foundLUN {
			voln.UID = strings.ToUpper(uid)
			err := o.Read(&voln, "UID")
			if err == nil {
				foundLUN = true
			}
		}

		if !foundLUN {
			voln.VirtualName = strings.TrimSpace(volume.Name) //убираем пробелы до и после
			err := o.Read(&voln, "VirtualName")
			if err == nil {
				foundLUN = true
			}
		}

		// приоритет источников (2026-09-03): vSphere-запись СХД не
		// перезаписывает целиком, только дополняет; ручную запись не трогает
		// вовсе - см. storageAPIWriteModeFor в cmdb/volumestorage.go
		mode := storageAPIWriteModeFor(voln, foundLUN)
		if mode == storageAPIWriteSkip {
			continue
		}
		if mode == storageAPIWriteSupplement {
			voln.StorageSystemsId = shd.Id
			voln.ShortName = shd.ShortName
			voln.StorageName = strings.TrimSpace(volume.Name)
			voln.Time_Mfy = time.Now().Local()
			if _, updErr := o.Update(&voln, "StorageSystemsId", "ShortName", "StorageName", "Time_Mfy"); updErr != nil {
				fmt.Println("Ошибка дополнения данных тома (приоритет vSphere):", volume.Name, updErr)
			} else {
				touchedIds = append(touchedIds, voln.Id)
			}
			continue
		}

		voln.StorageSystemsId = shd.Id
		voln.Status = 2
		voln.UID = uid
		voln.ShortName = shd.ShortName //запоминаем краткое наименование СХД
		voln.StorageName = strings.TrimSpace(volume.Name)
		voln.CapacityGb = float64(capacity)
		voln.Time_Mfy = time.Now().Local()
		voln.SourceUpdateType = models.StorageSourceUpdateStorageAPI

		if foundLUN {
			_, err = o.Update(&voln)
			if err != nil {
				fmt.Println("Ошибка обновления данных тома:", volume.Name, uid, err)
				continue
			}
		} else {
			if voln.UID == "" {
				fmt.Println("Ошибка при вставке VNX UID пустой", voln.StorageName, voln.UID)
			}
			_, err = o.Insert(&voln)
			if err != nil {
				fmt.Println("Ошибка вставки данных тома:", volume.Name, uid, err)
				continue
			}
		}
		touchedIds = append(touchedIds, voln.Id)
	}
	return touchedIds, nil
}

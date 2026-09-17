package cmdb

import (
	"context"
	"fmt"
	models "loadmon/models"
	datamap "loadmon/utilities"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/lovi-cloud/go-dorado-sdk/dorado"
)

// DoradoLunInfo содержит сведения о томе (LUN) СХД Huawei Dorado, полученные через API,
// используется как промежуточная структура при синхронизации с таблицей StorageVolumes.
type DoradoLunInfo struct {
	Id                int64   //
	UID               string  //
	Name              string  //
	EnableCompression bool    //
	EnableSmartDedup  bool    //
	CapacityGb        float64 //
	AllocCapacityGb   float64 //
	Servers           string  //
}

// UpdateHuaweyVolume подключается к СХД Huawei Dorado, получает список томов и обновляет
// либо создаёт соответствующие записи StorageVolumes в базе данных CMDB.
//
// Возвращает список Id обновлённых/вставленных за этот вызов записей
// StorageVolumes (используется вызывающим кодом cmdb.UpdateVolumeStorage для
// отложенной деактивации томов, реально не найденных при ЭТОМ успешном
// опросе - см. комментарий там же) и ошибку ТОЛЬКО уровня подключения/
// получения списка (сбой обновления/вставки отдельного тома не прерывает
// обработку остальных и не считается ошибкой всего опроса - как и раньше).
func UpdateHuaweyVolume(shd *models.StorageSystems) ([]int64, error) {

	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)

	if err != nil {
		fmt.Println("Ошибка подключения к СХД", shd.ShortName, err)
		return nil, err
	}

	volumes, err := GetDoradoLun(client, shd)

	if err != nil {
		fmt.Println("Ошибка получения списка томов СХД", shd.ShortName, shd.IpAddress, err)
		return nil, err
	}

	o := orm.NewOrm()

	touchedIds := make([]int64, 0, len(volumes))

	for _, volume := range volumes {
		foundLUN := false

		voln := models.StorageVolumes{}

		if !foundLUN {
			voln.UID = strings.ToUpper(volume.UID)
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
		voln.UID = strings.ToUpper(volume.UID)
		voln.ShortName = shd.ShortName //запоминаем краткое наименование СХД
		voln.StorageName = strings.TrimSpace(volume.Name)
		voln.CapacityGb = float64(volume.CapacityGb)
		voln.Time_Mfy = time.Now().Local()
		voln.SourceUpdateType = models.StorageSourceUpdateStorageAPI

		if foundLUN {
			_, err = o.Update(&voln)
			if err != nil {
				fmt.Println("Ошибка обновления данных тома:", volume.Name, volume.UID, err)
				continue
			}
		} else {
			if voln.UID == "" {
				fmt.Println("Ошибка при вставке VNX UID пустой", voln.StorageName, voln.UID)
			}
			_, err = o.Insert(&voln)
			if err != nil {
				fmt.Println("Ошибка вставки данных тома:", volume.Name, volume.UID, err)
				continue
			}
		}
		touchedIds = append(touchedIds, voln.Id)
	}
	return touchedIds, nil
}

// GetDoradoLun запрашивает у СХД Huawei Dorado список LUN-ов и связанных с ними хостов
// и возвращает карту DoradoLunInfo по имени тома.
//
// shd передаётся ТОЛЬКО ради диагностических сообщений (ShortName/IpAddress
// конкретной СХД) - ИЗМЕНЕНО 2026-09-17 по замечанию пользователя, что в
// логе строки "Ошибка получения привязки томов и хостов LUN is not found
// <id> <hostname>" не позволяли понять, к какой именно из нескольких
// одновременно опрашиваемых Huawei СХД относится ошибка и что вообще с чем
// пытались связать (сообщение раньше не называло ни СХД, ни саму операцию
// человеческим языком). Сам объект shd в логике функции не используется.
func GetDoradoLun(client *dorado.Client, shd *models.StorageSystems) (map[string]*DoradoLunInfo, error) {

	ctx := context.Background()

	mapLun := make(map[string]*DoradoLunInfo, 0)

	luns, err := client.LocalDevice.GetLUNs(ctx, nil)
	if err != nil {
		return mapLun, err
	}

	for _, l := range luns {
		alloccapacity, err := strconv.ParseInt(l.ALLOCCAPACITY, 10, 64)
		if err != nil {
			fmt.Println("Ошибка преобразования ParseInt", l.ALLOCCAPACITY, err)
			continue
		}

		sectorsize, err := strconv.ParseInt(l.SECTORSIZE, 10, 64)
		if err != nil {
			fmt.Println("Ошибка преобразования ParseInt", l.SECTORSIZE, err)
			continue
		}

		alloccapacity = (alloccapacity * sectorsize) >> 30
		capacity := (int64(l.CAPACITY) * sectorsize) >> 30

		dLun := DoradoLunInfo{
			Id:              int64(l.ID),
			UID:             strings.ToUpper(l.WWN), //заглавными
			Name:            l.NAME,
			CapacityGb:      float64(capacity),
			AllocCapacityGb: float64(alloccapacity)}
		if l.ENABLECOMPRESSION == "true" {
			dLun.EnableCompression = true
		}
		if l.ENABLESMARTDEDUP == "true" {
			dLun.EnableSmartDedup = true
		}
		mapLun[l.NAME] = &dLun
	}

	hosts, err := client.LocalDevice.GetHosts(ctx, nil)
	if err != nil {
		fmt.Println("Ошибка получения списка хостов", err)
		return mapLun, err
	}
	for _, h := range hosts {
		lunsa, err := client.LocalDevice.GetHostAssociatedLUNs(ctx, h.ID)
		if err != nil {
			// Расширенное сообщение (2026-09-17, по замечанию пользователя
			// "непонятно кто и с кем связывается"): раньше строка не
			// называла ни СХД, ни саму операцию словами - теперь явно:
			// какая СХД (ShortName+IP), какой хост Huawei (Id+Name) и что
			// именно запрашивалось - список LUN, замапленных на этот хост
			// (dorado.LocalDevice.GetHostAssociatedLUNs). err == dorado.
			// ErrLunNotFound - ОЖИДАЕМЫЙ, не ошибочный случай: у хоста
			// просто нет ни одного назначенного тома на момент опроса
			// (например, ESXi-хост ещё не подключен к датастору) - не
			// признак сбоя СХД или кода.
			if err == dorado.ErrLunNotFound {
				fmt.Printf("СХД Huawei %s (%s): у хоста Id=%s Name=%s нет ни одного привязанного LUN (это нормально, если хосту ещё не назначены тома)\n",
					shd.ShortName, shd.IpAddress, h.ID, h.NAME)
			} else {
				fmt.Printf("СХД Huawei %s (%s): ошибка запроса списка LUN, привязанных к хосту Id=%s Name=%s: %v\n",
					shd.ShortName, shd.IpAddress, h.ID, h.NAME, err)
			}
			continue
		}
		for _, l := range lunsa {
			lun, ok := mapLun[l.NAME]
			if ok {
				if lun.Servers == "" {
					lun.Servers = h.NAME
				} else {
					lun.Servers = fmt.Sprintf("%s;%s", lun.Servers, h.NAME)
				}
			}
		}
	}
	return mapLun, nil
}

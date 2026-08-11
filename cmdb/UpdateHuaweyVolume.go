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
func UpdateHuaweyVolume(shd *models.StorageSystems) error {

	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)

	if err != nil {
		fmt.Println("Ошибка подключения к СХД", shd.ShortName, err)
		return err
	}

	volumes, err := GetDoradoLun(client)

	if err != nil {
		fmt.Println("Ошибка получения списка томов СХД", shd.ShortName, shd.IpAddress, err)
		return err
	}

	o := orm.NewOrm()

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

		voln.StorageSystemsId = shd.Id
		voln.Status = 2
		voln.UID = strings.ToUpper(volume.UID)
		voln.ShortName = shd.ShortName //запоминаем краткое наименование СХД
		voln.StorageName = strings.TrimSpace(volume.Name)
		voln.CapacityGb = float64(volume.CapacityGb)
		voln.Time_Mfy = time.Now().Local()

		if foundLUN {
			_, err = o.Update(&voln)
			if err != nil {
				fmt.Println("Ошибка обновления данных тома:", volume.Name, volume.UID, err)
			}
		} else {
			if voln.UID == "" {
				fmt.Println("Ошибка при вставке VNX UID пустой", voln.StorageName, voln.UID)
			}
			_, err = o.Insert(&voln)
			if err != nil {
				fmt.Println("Ошибка вставки данных тома:", volume.Name, volume.UID, err)
			}
		}
	}
	return nil
}

// GetDoradoLun запрашивает у СХД Huawei Dorado список LUN-ов и связанных с ними хостов
// и возвращает карту DoradoLunInfo по имени тома.
func GetDoradoLun(client *dorado.Client) (map[string]*DoradoLunInfo, error) {

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
			fmt.Println("Ошибка получения привязки томов и хостов", err, h.ID, h.NAME)
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

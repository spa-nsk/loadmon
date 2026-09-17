package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	models "loadmon/models"
	_ "loadmon/routers"
	"loadmon/storages"
	"os"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// deactivateAbsentStorageVolumes деактивирует (Status=0) записи StorageVolumes
// с данной StorageSystemsId и SourceUpdateType==StorageSourceUpdateStorageAPI,
// которых НЕТ в touchedIds - то есть тома, реально не найденные при ТОЛЬКО
// ЧТО завершившемся УСПЕШНОМ опросе конкретной СХД (см. вызовы в
// UpdateVolumeStorage ниже - вызывается только когда pollOK==true для этой
// СХД). Тома с SourceUpdateType, отличным от StorageSourceUpdateStorageAPI
// (например, StorageSourceUpdateManual, если такой когда-нибудь появится),
// сознательно не трогаются - тот же принцип разделения по источнику, что и
// у ServerSprRecords/ServerHardware.SourceUpdateType.
func deactivateAbsentStorageVolumes(o orm.Ormer, storageSystemsId int64, touchedIds []int64) {
	touched := make(map[int64]bool, len(touchedIds))
	for _, id := range touchedIds {
		touched[id] = true
	}

	var volumes []*models.StorageVolumes
	_, err := o.QueryTable("storage_volumes").
		Filter("StorageSystemsId", storageSystemsId).
		Filter("SourceUpdateType", models.StorageSourceUpdateStorageAPI).
		All(&volumes)
	if err != nil {
		fmt.Println("Ошибка чтения томов для деактивации, StorageSystemsId=", storageSystemsId, err)
		return
	}

	deactivated := 0
	for _, vol := range volumes {
		if vol == nil || touched[vol.Id] || vol.Status == 0 {
			continue
		}
		vol.Status = 0
		if _, err := o.Update(vol, "Status"); err != nil {
			fmt.Println("Ошибка деактивации тома Id=", vol.Id, err)
			continue
		}
		deactivated++
	}
	if deactivated > 0 {
		fmt.Println("Деактивировано (Status=0) томов, не найденных при опросе СХД, StorageSystemsId=", storageSystemsId, ":", deactivated)
	}
}

// storageAPIWriteMode определяет, как опрос API конкретной СХД должен
// поступить с уже НАЙДЕННОЙ (по StorageName/UID/VirtualName) записью
// StorageVolumes - правило приоритета источников (2026-09-03, см.
// kontext/05-conventions-and-gotchas.md, раздел про СХД):
//   - vSphere-скан (SourceUpdateType==StorageSourceUpdateVMware) имеет
//     приоритет - опрос СХД такую запись целиком не перезаписывает
//     (Status/CapacityGb/FreeSpaceGb/PercentUsed остаются от VMware), а
//     только дополняет идентификационные поля (StorageSystemsId/
//     ShortName/StorageName) - см. storageAPIWriteSupplement.
//   - Запись, заведённая вручную (StorageSourceUpdateManual), опросом СХД
//     вообще не трогается - см. storageAPIWriteSkip.
//   - Новая запись либо запись, ранее принадлежавшая самой СХД
//     (StorageSourceUpdateStorageAPI/None), пишется как раньше целиком -
//     storageAPIWriteFull.
type storageAPIWriteMode int

const (
	storageAPIWriteFull       storageAPIWriteMode = iota // пишем полностью (новая запись либо запись самой СХД)
	storageAPIWriteSupplement                             // запись принадлежит vSphere - только дополняем
	storageAPIWriteSkip                                   // запись заведена вручную - не трогаем
)

// storageAPIWriteModeFor вычисляет режим записи для найденной (или не
// найденной) записи StorageVolumes - см. storageAPIWriteMode.
func storageAPIWriteModeFor(existing models.StorageVolumes, foundLUN bool) storageAPIWriteMode {
	if !foundLUN {
		return storageAPIWriteFull
	}
	switch existing.SourceUpdateType {
	case models.StorageSourceUpdateManual:
		return storageAPIWriteSkip
	case models.StorageSourceUpdateVMware:
		return storageAPIWriteSupplement
	default:
		return storageAPIWriteFull
	}
}

// recordStorageSystemScanResult сохраняет в самой записи StorageSystems
// результат последней попытки её опроса (LastScanTime/LastScanOK/
// LastScanError) - см. комментарий у этих полей в models/models_add.go.
// Ошибка отдельно логируется, но не прерывает основной цикл
// UpdateVolumeStorage - запись результата опроса вспомогательна.
func recordStorageSystemScanResult(o orm.Ormer, shd *models.StorageSystems, pollErr error) {
	shd.LastScanTime = time.Now().Local()
	shd.LastScanOK = pollErr == nil
	if pollErr != nil {
		shd.LastScanError = pollErr.Error()
	} else {
		shd.LastScanError = ""
	}
	if _, err := o.Update(shd, "LastScanTime", "LastScanOK", "LastScanError"); err != nil {
		fmt.Println("Ошибка сохранения результата опроса СХД Id=", shd.Id, err)
	}
}

// UpdateVolumeStorageOnly - если задано (не пустая строка), ограничивает
// цикл UpdateVolumeStorage ОДНОЙ СХД по ShortName (сравнение без учёта
// регистра, точное совпадение). Добавлено 2026-09-17 по запросу пользователя
// loadmon "сделай отдельное приложение ... которое будет делать
// синхронизацию данных СХД и больше ничего, на нём и отладим код" -
// используется ТОЛЬКО отдельной отладочной утилитой tools/storagesync (см.
// её main.go), чтобы гонять полный цикл синхронизации для одной проблемной
// СХД без ожидания опроса остальных. Пустая строка (значение по умолчанию,
// не меняется) - без фильтра, обычное поведение всех СХД, ничего не меняет
// в production-поведении metrics_grpc.
var UpdateVolumeStorageOnly string

// UpdateVolumeStorage() функция обновления информации о томах СХД
func UpdateVolumeStorage() {
	file, ferr := os.Create("storage_update.log")
	if ferr == nil {
		defer file.Close()
	}
	fmtdate := "02.01.2006 15:04:05.000"
	currentTime := time.Now().Local()
	fmt.Println(currentTime.Format(fmtdate), "Обновление данных томов СХД")
	o := orm.NewOrm()
	//o.Using("default")
	var storSys []*models.StorageSystems
	qs := o.QueryTable("storage_systems")
	qs = qs.OrderBy("Id")
	num, err := qs.All(&storSys)

	mapStorSys := make(map[string]*models.StorageSystems)
	mapStorSysWWN := make(map[string]*models.StorageSystems)

	if num > 0 && err == nil {
		for i, shd := range storSys {
			if shd.Power != 1 {
				continue
			}
			//пропускаем выключенные СХД
			if UpdateVolumeStorageOnly != "" && !strings.EqualFold(shd.ShortName, UpdateVolumeStorageOnly) {
				continue // отладочный фильтр tools/storagesync - см. комментарий у переменной
			}
			keyStor := strings.ToUpper(shd.ShortName)
			mapStorSys[keyStor] = storSys[i] //запоминаем в карте по краткому имени СХД
			if shd.TagName != "" {
				for tagName := range strings.SplitSeq(shd.TagName, ";") {
					mapStorSys[strings.ToUpper(tagName)] = storSys[i]
				}
			}
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

			fmt.Println(time.Now().Local().Format(fmtdate), "Запрос к СХД:", shd.InventoryNum, shd.Model, shd.ShortName, shd.IpAddress, shd.ApiType)

			// pollErr - ошибка уровня "не удалось получить актуальный список
			// томов этой СХД целиком" (соединение/авторизация/список LUN) -
			// в отличие от ошибки обновления/вставки ОТДЕЛЬНОГО тома (та не
			// прерывает обработку остальных и не считается сбоем всего
			// опроса, как и раньше). Именно pollErr решает, будет ли вызвана
			// deactivateAbsentStorageVolumes ниже - если опрос СХД не
			// удался, тома этой СХД НЕ трогаются вовсе (остаются с тем
			// статусом, что был), а не бланкетно обнуляются, как раньше в
			// ClearStatusVolumeStorage (удалена, см.
			// kontext/05-conventions-and-gotchas.md).
			var pollErr error
			touchedIds := make([]int64, 0)

			switch shd.ApiType {
			case 1: //VNX CLI
				luns, lunsErr := storages.GetLunsCmd(shd.IpAddress)
				pollErr = lunsErr
				if lunsErr != nil {
					fmt.Println("Ошибка GetLunsCmd :", lunsErr)
				} else {
					fmt.Println("Получены данные ", len(luns), "LUN")
				}

				hbas, hbaErr := storages.GetHBAInfoCmd(shd.IpAddress)

				if hbaErr != nil {
					fmt.Println("Ошибка GetHBAInfoCmd :", hbaErr)
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, hbaErr)
					}
				} else {
					fmt.Println("Получены данные ", len(hbas), "HBA")
				}

				for _, lun := range luns {
					foundLUN := false
					fmt.Println("\t", lun.UID, lun.Name, lun.PoolName, lun.RaidType, lun.UserCapacityGBs)
					voln := models.StorageVolumes{}
					if !foundLUN {
						voln.StorageName = strings.TrimSpace(lun.Name) //убираем пробелы до и после
						err := o.Read(&voln, "StorageName")
						if err == nil {
							foundLUN = true
						}
					}
					if !foundLUN {
						voln.UID = strings.ToUpper(lun.UID)
						err := o.Read(&voln, "UID")
						if err == nil {
							foundLUN = true
						}
					}

					if !foundLUN {
						voln.VirtualName = strings.TrimSpace(lun.Name) //убираем пробелы до и после
						err := o.Read(&voln, "VirtualName")
						if err == nil {
							foundLUN = true
						}
					}

					mode := storageAPIWriteModeFor(voln, foundLUN)
					if mode == storageAPIWriteSkip {
						continue // запись заведена вручную - опрос СХД её не трогает
					}
					if mode == storageAPIWriteSupplement {
						// запись принадлежит vSphere-скану - приоритет у него,
						// СХД только дополняет идентификацию
						voln.StorageSystemsId = shd.Id
						voln.ShortName = shd.ShortName
						voln.StorageName = strings.TrimSpace(lun.Name)
						voln.Time_Mfy = time.Now().Local()
						if _, updErr := o.Update(&voln, "StorageSystemsId", "ShortName", "StorageName", "Time_Mfy"); updErr != nil {
							fmt.Println("Ошибка дополнения данных тома (приоритет vSphere):", lun.Name, updErr)
						} else {
							touchedIds = append(touchedIds, voln.Id)
						}
						continue
					}

					voln.StorageSystemsId = shd.Id
					voln.Status = 2
					voln.UID = strings.ToUpper(lun.UID)
					voln.ShortName = shd.ShortName //запоминаем краткое наименование СХД
					voln.StorageName = strings.TrimSpace(lun.Name)
					voln.CapacityGb = lun.UserCapacityGBs
					voln.Time_Mfy = time.Now().Local()
					voln.SourceUpdateType = models.StorageSourceUpdateStorageAPI

					var updErr error
					if foundLUN {
						_, updErr = o.Update(&voln)
						if updErr != nil {
							fmt.Println("Ошибка обновления данных тома:", lun.Name, strings.ToUpper(lun.UID), updErr)
						}
					} else {
						if voln.UID == "" {
							fmt.Println("Ошибка при вставке VNX UID пустой", voln.StorageName, strings.ToUpper(voln.UID))
						}
						_, updErr = o.Insert(&voln)
						if updErr != nil {
							fmt.Println("Ошибка вставки данных тома:", lun.Name, strings.ToUpper(lun.UID), updErr)
						}

					}
					if updErr == nil {
						touchedIds = append(touchedIds, voln.Id)
					}
				}
				for _, hba := range hbas {
					fmt.Println("\t", hba.HBAUID, hba.ServerName, hba.ServerIPAdress)
					for _, port := range hba.PortList {
						fmt.Println("\t\t", port.SPName, port.SPName, port.HBADeviceName, port.StorageGroupName)
					}
				}
			case 2: //Unity Rest
				mapHosts := make(map[string]string, 0)                  //карта по именам lun в качестве ключа, значение имена хостов
				hosts, hostsErr := storages.GetHostsRest(shd.IpAddress) //запрашиваем список хостов

				if hostsErr == nil {
					for _, host := range hosts {
						for _, lunId := range host.HostLUNs {
							keyLun := strings.ToUpper(lunId.LUN.Name)
							s, ok := mapHosts[keyLun]
							if ok {
								hn, _ := SplitNameFQDN(host.Name)
								s = fmt.Sprintf("%s;%s", s, hn) //добавляем имена хостов имеющих доступ к lun
							} else {
								hn, _ := SplitNameFQDN(host.Name)
								s = hn
							}
							mapHosts[keyLun] = strings.TrimSpace(s)
						}
					}
				} else {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, hostsErr)
					}
				}
				luns, lunsErr := storages.GetLunsRest(shd.IpAddress)
				pollErr = lunsErr
				if lunsErr != nil {
					fmt.Println("Ошибка GetLunsRest :", lunsErr)
				} else {
					fmt.Println("Получены данные ", len(luns), "LUN")
				}

				for _, lun := range luns {
					keyLun := strings.ToUpper(strings.TrimSpace(lun.Name))
					listHostName, foundHostNames := mapHosts[keyLun]
					foundLUN := false
					fmt.Println("\t", lun.Wwn, lun.Name, lun.SizeTotal, "[", listHostName, "]")

					voln := models.StorageVolumes{}
					if !foundLUN {
						voln.StorageName = strings.TrimSpace(lun.Name)
						err := o.Read(&voln, "StorageName")
						if err == nil {
							foundLUN = true
						}
					}

					if !foundLUN {
						voln.UID = strings.ToUpper(lun.Wwn)
						err := o.Read(&voln, "UID")
						if err == nil {
							foundLUN = true
						}
					}

					if !foundLUN {
						voln.VirtualName = strings.TrimSpace(lun.Name) //убираем пробелы до и после
						err := o.Read(&voln, "VirtualName")
						if err == nil {
							foundLUN = true
						}
					}

					mode := storageAPIWriteModeFor(voln, foundLUN)
					if mode == storageAPIWriteSkip {
						continue // запись заведена вручную - опрос СХД её не трогает
					}
					if mode == storageAPIWriteSupplement {
						// запись принадлежит vSphere-скану - приоритет у него,
						// СХД только дополняет идентификацию и список хостов
						voln.StorageSystemsId = shd.Id
						voln.ShortName = shd.ShortName
						voln.StorageName = strings.TrimSpace(lun.Name)
						voln.Annotation = lun.Description
						if foundHostNames {
							voln.Server = listHostName
						}
						voln.Time_Mfy = time.Now().Local()
						if _, updErr := o.Update(&voln, "StorageSystemsId", "ShortName", "StorageName", "Annotation", "Server", "Time_Mfy"); updErr != nil {
							fmt.Println("Ошибка дополнения данных тома (приоритет vSphere):", lun.Name, updErr)
						} else {
							touchedIds = append(touchedIds, voln.Id)
						}
						continue
					}

					voln.StorageSystemsId = shd.Id
					voln.Status = 2
					voln.UID = strings.ToUpper(lun.Wwn)
					voln.ShortName = shd.ShortName
					voln.StorageName = strings.TrimSpace(lun.Name)
					voln.CapacityGb = float64(lun.SizeTotal >> 30)
					voln.Annotation = lun.Description
					if foundHostNames {
						voln.Server = listHostName
					}
					voln.Time_Mfy = time.Now().Local()
					voln.SourceUpdateType = models.StorageSourceUpdateStorageAPI

					var updErr error
					if foundLUN {
						if voln.UID == "" {
							fmt.Println("Ошибка при обновлении Unity UID пустой", voln.StorageName, voln.UID)
						}
						_, updErr = o.Update(&voln)
						if updErr != nil {
							fmt.Println("Ошибка обновления данных тома:", lun.Name, strings.ToUpper(lun.Wwn), updErr)
						}
					} else {
						if voln.UID == "" {
							fmt.Println("Ошибка при вставке Unity UID пустой", voln.StorageName, voln.UID)
						}
						_, updErr = o.Insert(&voln)
						if updErr != nil {
							fmt.Println("Ошибка вставки данных тома:", lun.Name, strings.ToUpper(lun.Wwn), updErr)
						}
					}
					if updErr == nil {
						touchedIds = append(touchedIds, voln.Id)
					}
				}
			case 3: //Dorado rest
				touchedIds, pollErr = UpdateHuaweyVolume(shd)
				if pollErr != nil {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, pollErr)
					}
				}
			case 4: //OceanStor
				touchedIds, pollErr = UpdateHuaweyVolume(shd)
				if pollErr != nil {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, pollErr)
					}
				}
			case 5: //Lenovo Rest
				touchedIds, pollErr = UpdateLenovoVolume(shd)
				if pollErr != nil {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, pollErr)
					}
				}
			default:
				fmt.Println("СХД:", shd.InventoryNum, shd.IpAddress, shd.ApiType, "не обработано")
				pollErr = fmt.Errorf("неизвестный ApiType=%d", shd.ApiType)
			}

			recordStorageSystemScanResult(o, shd, pollErr)
			logStorageAvailability(o, shd, pollErr) // история опроса - см. cmdb/storageeventlog.go
			if pollErr == nil {
				deactivateAbsentStorageVolumes(o, shd.Id, touchedIds)
				if hwErr := SyncStorageHardwareForSystem(o, shd); hwErr != nil {
					fmt.Println("Ошибка синхронизации hardware СХД Id=", shd.Id, hwErr)
				}
			}
		}
	}

	var storVol []*models.StorageVolumes
	qs = o.QueryTable("storage_volumes")
	qs = qs.OrderBy("Id")
	num, err = qs.All(&storVol)
	if num > 0 && err == nil {
		for _, vol := range storVol {
			volumes := models.StorageVolumes{Id: vol.Id}
			err := o.Read(&volumes, "Id")
			if err != nil {
				continue
			}
			foundSHD := false
			foundDiskType := false

			shortName, diskType := GetShortNameStoreSys(volumes.StorageName, mapStorSys) //выделить краткое наименование СХД из имени lun в StorageName
			if shortName != "" {
				foundSHD = true
			}
			if diskType != "" {
				foundDiskType = true
			}

			shortNameV, diskTypeV := GetShortNameStoreSys(volumes.VirtualName, mapStorSys) //выделить краткое наименование СХД из имени lun в VirtualName
			if !foundSHD && shortNameV != "" {
				shortName = shortNameV
				foundSHD = true
			}
			if !foundDiskType && diskTypeV != "" {
				diskType = diskTypeV
				foundDiskType = true
			}

			if foundSHD {
				volumes.ShortName = shortName
			}
			if foundDiskType {
				volumes.DiskType = diskType
			}
			if foundSHD {
				key := strings.ToUpper(shortName)
				shd, ok := mapStorSys[key]
				if ok {
					volumes.StorageSystemsId = shd.Id
					volumes.ShortName = shd.ShortName
				} else {
					wwn := GetWWNMaskKey(volumes.UID)
					shd, ok = mapStorSysWWN[wwn]
					if ok {
						volumes.StorageSystemsId = shd.Id
						volumes.ShortName = shd.ShortName
					}
				}
			}
			_, err = o.Update(&volumes)
			if err != nil {
				fmt.Println("Ошибка обновления :", volumes.ShortName, " ", err)
			}

			// том активен (найден в текущем скане какого-либо источника), но
			// ни по имени, ни по WWN не удалось привязать его ни к одной
			// известной СХД - логируем (с дедупликацией по ключу тома, см.
			// cmdb/storageeventlog.go:logUnknownLun), чтобы это было видно, а
			// не терялось молча
			if volumes.StorageSystemsId == 0 && volumes.Status != 0 {
				logUnknownLun(o, &volumes)
			}
		}
	}
	fmt.Println(time.Now().Local().Format(fmtdate), "Данные томов СХД обновлены")
}

// GetWWNMaskKey формирует укороченный ключ маски WWN (удаляет разделители ":" и
// обрезает до 16 символов) для сопоставления идентификаторов томов между системами.
func GetWWNMaskKey(str string) string {
	s := strings.Split(str, ":")
	s2 := ""
	for _, substr := range s {
		s2 += substr
	}
	n := 16
	v := []rune(s2)
	if n >= len(v) {
		return str
	}
	return string(v[:n])
}

// GetShortNameStoreSys по имени тома LUN пытается определить краткое имя СХД и, при
// наличии, имя пула/подсистемы, сверяясь со справочником известных систем хранения.
func GetShortNameStoreSys(lun string, mapStorSys map[string]*models.StorageSystems) (string, string) {
	s := strings.Split(lun, "_")
	for i, sp := range s {
		keyS := strings.ToUpper(sp)
		_, ok := mapStorSys[keyS]
		if ok {
			if i == 0 && len(s) > 2 {
				return s[0], s[1]
			} else {
				return sp, ""
			}
		}
	}
	return "", ""
}

// переписать
func UpdateVirtualStorage() {

	fmtdate := "02.01.2006 15:04:05.000"
	currentTime := time.Now().Local()
	fmt.Println(currentTime.Format(fmtdate), "Обновление данных по виртуальным хранилищам")
	//Infra := vsphere.GetGlobalVMInfrastructure() //получаем указатель на объект VM инфраструктуры
	//if Infra == nil {
	//    fmt.Println("Обновление данных по виртуальным хранилищам прервано: Инфраструктура не инициализирована")
	//    return
	//}

	clusterSpr, err := grpcclient.LoadClusterSpr("all", "all", "all")
	if err != nil || clusterSpr == nil {
		return
	}

	clList := grpcclient.LoadClusterList("all", "all")
	//count := len(clList)

	o := orm.NewOrm()
	//o.Using("default")

	for _, clShort := range clList {
		cl := grpcclient.GetClusterMedium(clShort.Vcenter, clShort.Datacenter, clShort.Name)

		vml, err := grpcclient.GetVMListKey(cl.Vcenter, cl.Datacenter, cl.Name, "all")
		if err != nil {
			fmt.Println("Ошибка получения ключей виртуальных машин кластера: ", cl.Name, err)
			return
		}

		for _, vmk := range vml.KeyRecs {
			vm, err := grpcclient.GetVM(vmk.Key)
			if err != nil {
				fmt.Println("Ошибка получения данных виртуальной машины ", vmk.Key, err)
				continue
			}

			if len(vm.UsageDatastore) > 0 { // проверяем есть ли данные по использованию хранилищ виртуальной машиной
				//вставить запись данных записей хранилища (дополнение к записям кластеров)
				for _, st := range vm.UsageDatastore {
					var err error
					ds, ok := grpcclient.GetDSData(st.Datastore)
					//
					//					ds, ok := Infra.DatastoreMapList.Get(st.Datastore) // ищем хранилище в карте хранилищ
					if ds != nil && ok {
						volume := models.StorageVolumes{}
						if ds.UID != "" { //если хранилище на СХД
							volume.UID = ds.UID
							err = o.Read(&volume, "UID") //Ищем запись в базе по UID (Предварительно все записи с СХД перенесены в базу и обновлены)
						} else {
							volume.VirtualName = ds.StorageName //если хранилище не на СХД ищем по имени хранилища в виртуальной структуре (могут различаться с именами L
							err = o.Read(&volume, "VirtualName")
						}

						volume.VirtualName = ds.StorageName
						//volume.StorageName = ds.StorageName

						volume.Vcenter = vm.Vcenter
						volume.Datacenter = vm.Datacenter
						volume.Cluster = vm.Cluster

						if err == nil { //есть такая запись в списке томов
							volume.UID = ds.UID
							volume.Status = 1
							volume.CapacityGb = ds.CapacityGb
							volume.FreeSpaceGb = ds.FreeSpaceGb
							volume.PercentUsed = ds.PercentUsed
							volume.Id_User = int64(0)
							volume.Time_Mfy = time.Now().Local()
							_, err = o.Update(&volume)
							if err != nil {
								fmt.Println("Ошибка обновления записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
							}
						} else { //записи нет вставляем новую
							volume.Status = 1
							volume.UID = ds.UID
							volume.CapacityGb = ds.CapacityGb
							volume.FreeSpaceGb = ds.FreeSpaceGb
							volume.PercentUsed = ds.PercentUsed
							volume.Id_User = int64(0)
							volume.Time_Mfy = time.Now().Local()
							_, err = o.Insert(&volume)
							if err != nil {
								fmt.Println("Ошибка вставки записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
							}
						}
					} else {
						// Хранилище не найдено вставить обработку записи RDM дисков
						fmt.Println("RDM диск", st.Vcenter, st.Datacenter, st.Cluster, "\""+st.StorageName+"\"", st.Datastore) // добавить создание записи о хранилище
						volume := models.StorageVolumes{}
						if st.UID != "" { //если хранилище на СХД
							volume.UID = st.UID
							err = o.Read(&volume, "UID") //Ищем запись в базе по UID (Предварительно все записи с СХД перенесены в базу и обновлены)
						} else {
							volume.VirtualName = st.StorageName //если хранилище не на СХД ищем по имени хранилища в виртуальной структуре (могут различаться с именами L
							err = o.Read(&volume, "VirtualName")
						}
						if volume.VirtualName == "" {
							volume.VirtualName = st.StorageName
							//volume.StorageName = st.StorageName
						} else {
							if !strings.Contains(volume.VirtualName, st.StorageName) {
								volume.VirtualName += "," + st.StorageName
							}
						}
						volume.Vcenter = vm.Vcenter
						volume.Datacenter = vm.Datacenter
						volume.Cluster = vm.Cluster

						if volume.Server == "" {
							volume.Server = vm.Name
						} else {
							if !strings.Contains(volume.Server, vm.Name) { //добавляем сервер к списку имен разделяющих между собой хранилище
								volume.Server += ";" + vm.Name
							}
						}

						if err == nil { //есть такая запись в списке томов
							volume.UID = st.UID
							volume.Status = 1
							volume.CapacityGb = st.CommittedGb
							volume.Id_User = int64(0)
							volume.Time_Mfy = time.Now().Local()
							_, err = o.Update(&volume)
							if err != nil {
								fmt.Println("Ошибка обновления записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
							}
						} else { //записи нет вставляем новую
							volume.Status = 1
							volume.UID = st.UID
							volume.CapacityGb = st.CommittedGb
							volume.Id_User = int64(0)
							volume.Time_Mfy = time.Now().Local()
							_, err = o.Insert(&volume)
							if err != nil {
								fmt.Println("Ошибка вставки записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
							}
						}

					}
				}
			}
		}
	}
	fmt.Println(time.Now().Local().Format(fmtdate), "Данные по виртуальным хранилищам обновлены")
}

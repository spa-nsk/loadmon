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

// ClearStatusVolumeStorage сброс статуса томов
func ClearStatusVolumeStorage() {
	var volumes []*models.StorageVolumes
	fmt.Println("Обнуление статуса томов")
	o := orm.NewOrm()
	qs := o.QueryTable("storage_volumes")
	qs = qs.OrderBy("Id")
	num, err := qs.All(&volumes)
	if num > 0 && err == nil {
		for _, vol := range volumes {
			voln := models.StorageVolumes{Id: vol.Id}
			err := o.Read(&voln, "Id")
			if err == nil {
				voln.Status = 0
				o.Update(&voln)
			}
		}
	}
	fmt.Println("Статусы томов обновлены")
}

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

			switch shd.ApiType {
			case 1: //VNX CLI
				luns, err := storages.GetLunsCmd(shd.IpAddress)
				if err != nil {
					fmt.Println("Ошибка GetLunsCmd :", err)
				} else {
					fmt.Println("Получены данные ", len(luns), "LUN")
				}

				hbas, err := storages.GetHBAInfoCmd(shd.IpAddress)

				if err != nil {
					fmt.Println("Ошибка GetHBAInfoCmd :", err)
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, err)
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

					voln.StorageSystemsId = shd.Id
					voln.Status = 2
					voln.UID = strings.ToUpper(lun.UID)
					voln.ShortName = shd.ShortName //запоминаем краткое наименование СХД
					voln.StorageName = strings.TrimSpace(lun.Name)
					voln.CapacityGb = lun.UserCapacityGBs
					voln.Time_Mfy = time.Now().Local()

					if foundLUN {
						_, err = o.Update(&voln)
						if err != nil {
							fmt.Println("Ошибка обновления данных тома:", lun.Name, strings.ToUpper(lun.UID), err)
						}
					} else {
						if voln.UID == "" {
							fmt.Println("Ошибка при вставке VNX UID пустой", voln.StorageName, strings.ToUpper(voln.UID))
						}
						_, err = o.Insert(&voln)
						if err != nil {
							fmt.Println("Ошибка вставки данных тома:", lun.Name, strings.ToUpper(lun.UID), err)
						}

					}
				}
				for _, hba := range hbas {
					fmt.Println("\t", hba.HBAUID, hba.ServerName, hba.ServerIPAdress)
					for _, port := range hba.PortList {
						fmt.Println("\t\t", port.SPName, port.SPName, port.HBADeviceName, port.StorageGroupName)
					}
				}
			case 2: //Unity Rest
				mapHosts := make(map[string]string, 0)          //карта по именам lun в качестве ключа, значение имена хостов
				hosts, err := storages.GetHostsRest(shd.IpAddress) //запрашиваем список хостов

				if err == nil {
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
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, err)
					}
				}
				luns, err := storages.GetLunsRest(shd.IpAddress)
				if err != nil {
					fmt.Println("Ошибка GetLunsRest :", err)
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

					if foundLUN {
						if voln.UID == "" {
							fmt.Println("Ошибка при обновлении Unity UID пустой", voln.StorageName, voln.UID)
						}
						_, err = o.Update(&voln)
						if err != nil {
							fmt.Println("Ошибка обновления данных тома:", lun.Name, strings.ToUpper(lun.Wwn), err)
						}
					} else {
						if voln.UID == "" {
							fmt.Println("Ошибка при вставке Unity UID пустой", voln.StorageName, voln.UID)
						}
						_, err = o.Insert(&voln)
						if err != nil {
							fmt.Println("Ошибка вставки данных тома:", lun.Name, strings.ToUpper(lun.Wwn), err)
						}
					}
				}
			case 3: //Dorado rest
				err = UpdateHuaweyVolume(shd)
				if err != nil {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, err)
					}
				}
			case 4: //OceanStor
				err = UpdateHuaweyVolume(shd)
				if err != nil {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, err)
					}
				}
			case 5: //Lenovo Rest
				err = UpdateLenovoVolume(shd)
				if err != nil {
					if ferr == nil {
						fmt.Fprintf(file, "%s %s %s\n", shd.ShortName, shd.IpAddress, err)
					}
				}
			default:
				fmt.Println("СХД:", shd.InventoryNum, shd.IpAddress, shd.ApiType, "не обработано")
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
		}
	}
	fmt.Println(time.Now().Local().Format(fmtdate), "Данные томов СХД обновлены")
}

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

package cmdb

import (
	"fmt"
	"loadmon/models"
	"loadmon/oneview"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// проверить и тестировать
// CheckAndCorrectSrv выбирает из набора дублирующихся записей ServerSprRecords с одинаковым
// именем сервера ту, что наиболее полно заполнена и соответствует переданному имени и
// количеству процессоров, связывает её с конфигурацией оборудования, а остальные дубли
// помечает как архивные. Возвращает Id выбранной записи.
func CheckAndCorrectSrv(records []*models.ServerSprRecords, serverName string, cpuCount int) int64 {
	//strings.EqualFold(string, string)
	count := len(records)
	if count == 0 {
		return 0
	}
	if count == 1 {
		return records[0].Id
	}

	foundIndex := make([]int, len(records))
	var excludeIndex []int

	o := orm.NewOrm()

	for i := range records {
		if records[i].Stat == 0 { //пропускаем архивные записи
			continue
		}

		if !strings.EqualFold(records[i].NameFQDN, serverName) || records[i].CPUCount != float64(cpuCount) { //нет соответствия имени или количества процессоров
			excludeIndex = append(excludeIndex, i)
			continue
		}
		h := models.ServerHardware{}

		foundHW := false

		if records[i].HWSprId > 0 { //если есть ссылка на ServerHardware то пытаемся прочитать
			h.Id = records[i].HWSprId
			err := o.Read(&h, "Id")
			if err != nil { //нет такой записи, возможно сервер уже выведен из работы или старый формат записи
				fmt.Println("Ошибка поиска информации по конфигурации сервера:", records[i].Name, records[i].NameFQDN, "ServerSprRecords.HWSprId=", records[i].HWSprId)
				h.Id = 0
			} else {
				foundHW = true
			}
		}
		if !foundHW && records[i].SerialNumber != "" { //пытаемся найти по серийному номеру последнему или из истории первоначальному
			h.SerialNum = records[i].SerialNumber //
			err := o.Read(&h, "SerialNum")        //ищем конфигурации по серийному номеру
			if err != nil {
				fmt.Println("Ошибка поиска информации по конфигурации сервера:", records[i].Name, records[i].NameFQDN, "SerialNumber=", records[i].SerialNumber)
			} else {
				foundHW = true
			}
		}
		if !foundHW {
			alterSN := GetSNFromServerSprRecords(records[i])
			if alterSN != "" {
				h.SerialNum = alterSN          //
				err := o.Read(&h, "SerialNum") //ищем конфигурации по альтернативному серийному номеру (из поля Details)
				if err != nil {
					fmt.Println("Ошибка поиска информации по конфигурации сервера:", records[i].Name, records[i].NameFQDN, "альтернативный SerialNumber=", alterSN)
					h.Id = 0
					h.SerialNum = ""
				} else {
					foundHW = true
				}
			} else {
				foundHW = true
			}
		}

		if foundHW {
			s := models.ServerSprRecords{Id: records[i].Id}
			if err := o.Read(&s, "Id"); err != nil {
				continue
			}

			s.HWSprId = h.Id
			h.ServerId = records[i].Id

			if num, err := o.Update(&h); err != nil || num < 1 {
				fmt.Println("Ошибка обновление записи по конфигурации сервера:", records[i].Name, records[i].NameFQDN, "HWSprId=", s.HWSprId)
			}

			hwAlter, _, _ := GetHostData(h.SerialNum)
			if hwAlter != nil {
				nameFQDN := oneview.GetHWserverNameFQDN(hwAlter)
				if !strings.EqualFold(nameFQDN, serverName) {
					excludeIndex = append(excludeIndex, i)
					continue //пропускаем запись
				}
			}
		} else { //????
			excludeIndex = append(excludeIndex, i)
			continue //пропускаем запись
		}

		if records[i].SZHistory != "" { //история служебных записок
			foundIndex[i]++
		}
		if records[i].ITSystemKey != "" { //выбрана ИТ система
			foundIndex[i]++
		}
		if records[i].Details != "" { //детали
			foundIndex[i]++
		}
	}
	max := 0
	idx := 0
	for j := range foundIndex {
		if max < foundIndex[j] {
			max = foundIndex[j]
			idx = j
		}
	}

	for i := range records {
		if i == idx {
			continue
		}
		exclude := false                  //флаг исключения
		for _, ex := range excludeIndex { //проверяем текущий индекс в списке исключений
			if i == ex {
				exclude = true
			}
		}
		if exclude {
			continue
		}

		srvspr := models.ServerSprRecords{Id: records[i].Id}
		if err := o.Read(&srvspr, "Id"); err == nil {
			srvspr.Stat = 0 //архивная запись
			srvspr.PowerState = 2
			srvspr.Active = 2
			num, err := o.Update(&srvspr)
			_ = num
			_ = err
		}
	}
	return records[idx].Id
}

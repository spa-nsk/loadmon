package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	"loadmon/models"
	"loadmon/oneview"

	"github.com/beego/beego/v2/client/orm"
)

func CheckServerSprRecordsLink() {
	records := make([]models.ServerSprRecords, 0)
	o := orm.NewOrm()
	qs := o.QueryTable("server_spr_records")
	qs = qs.OrderBy("name") //сортировка по именам серверов
	num, err := qs.All(&records)
	if err != nil || num < 1 {
		fmt.Println("Ошибка получения списка серверов: ", err)
		return
	}
	for _, rec := range records {
		switch rec.Type {
		case 0:
			//checkHwServerLink(rec)
		case 1:
			checkVirtualServerLink(rec)
		case 2:
			//checkHypervisorServerLink(rec)
		}
	}
}

func checkHwServerLink(rec models.ServerSprRecords) {
	records := make([]*models.ServerSprRecords, 0)
	ov := oneview.GetGlobalOVInfrastructure()
	o := orm.NewOrm()
	qs := o.QueryTable("server_spr_records")
	qs = qs.OrderBy("name").Filter("Name", rec.Name).Filter("Type__ne", 1) //сортировка по именам серверов и фильтр по имени и Type !=1
	num, err := qs.All(&records)
	if err != nil || num < 1 {
		fmt.Println("Запрос серверов по фильтру Name=", rec.Name, "Ошибка: ", err)
		return
	}
	for _, srv := range records {
		if srv.Type == 1 { //если с этим именем есть виртуальная машина
			continue
		}

		s := models.ServerSprRecords{Id: srv.Id}
		err := o.Read(&s, "Id")
		if err != nil {
			continue
		}
		var (
			hwAlter, hw  *oneview.ServerHardware
			h            models.ServerHardware
			serialNumber string
			name         string
			nameFQDN     string
		)
		if s.SerialNumber == "" { //если нет данных о серийном номере
			serialNumber = GetSNFromServerSprRecords(&s)
			if serialNumber != "" {
				hwAlter, _, _ = GetHostData(serialNumber)
				if hwAlter != nil {
					ImportServerSprFromHW(&s, hwAlter)
					name = oneview.GetHWserverName(hwAlter)
					nameFQDN = oneview.GetHWserverNameFQDN(hwAlter)
					if hwAlter.Base.PowerState == "On" { //проверяем включен ли сервер
						s.PowerState = 1
						s.Active = 1
						s.Stat = 1

					} else {
						s.PowerState = 2
					}
					s.Name = name
					s.NameFQDN = nameFQDN
					s.SerialNumber = string(hwAlter.Base.SerialNumber)
				} else { //не найдет сервер в HP One View по серийному номеру
					s.Stat = 0
					s.Active = 2
					s.PowerState = 2
				}
			}
		} else {
			hw, _, _ = GetHostData(s.SerialNumber)
			if hw != nil {

				ImportServerSprFromHW(&s, hw)

				name = oneview.GetHWserverName(hw)
				nameFQDN = oneview.GetHWserverNameFQDN(hw)

				if hw.Base.PowerState == "On" { //проверяем включен ли сервер
					s.PowerState = 1
					s.Active = 1
					s.Stat = 1
				} else {
					s.PowerState = 2
				}
				s.Name = name
				s.NameFQDN = nameFQDN
				s.SerialNumber = hw.Base.SerialNumber.String()
			} else { //не найдет сервер в HP One View по серийному номеру
				s.Stat = 0
				s.Active = 2
				s.PowerState = 2
			}
		}
		foundHWSpr := false

		if !foundHWSpr && s.SerialNumber != "" { //пытаемся найти по серийному номеру последнему или из истории первоначальному
			h.SerialNum = s.SerialNumber
			err = o.Read(&h, "SerialNum") //ищем конфигурации по серийному номеру
			if err != nil {
				fmt.Println("Ошибка поиска информации по конфигурации сервера:", srv.Name, srv.NameFQDN, "SerialNumber=", srv.SerialNumber)
			} else {
				foundHWSpr = true
			}
		}

		if !foundHWSpr && srv.HWSprId > 0 { //если есть ссылка на ServerHardware то пытаемся прочитать
			h.Id = srv.HWSprId
			err = o.Read(&h, "Id")
			if err != nil { //нет такой записи, возможно сервер уже выведен из работы
				fmt.Println("Ошибка поиска информации по конфигурации сервера:", srv.Name, srv.NameFQDN, "HWSprId=", srv.HWSprId)
				s.HWSprId = 0 //очищаем информацию
			} else {
				foundHWSpr = true
			}
		} else {
			h.Id = 0
		}

		if !foundHWSpr && srv.Id > 0 {
			h.ServerId = srv.Id //пытаемся найти по другой связи ServerHardware
			err = o.Read(&h, "ServerId")
			if err != nil { //данных по конфигурации сервера нет
				fmt.Println("Ошибка поиска информации по конфигурации сервера:", srv.Name, srv.NameFQDN, "ServerId=", srv.Id)
			} else {
				foundHWSpr = true
			}
		}

		if foundHWSpr {
			if hw != nil {
				ImportHWSprRecord(&h, hw, ov)
			} else if hwAlter != nil {
				ImportHWSprRecord(&h, hwAlter, ov)
			}
			s.HWSprId = h.Id
			h.ServerId = srv.Id
			s.SerialNumber = h.SerialNum
			h.Active = s.Active
			num, err = o.Update(&h)
			if err != nil || num < 1 {
				fmt.Println("Ошибка обновления записи по конфигурации сервера:", srv.Name, srv.NameFQDN, "HWSprId=", srv.HWSprId)
			}
		} else {
			if serialNumber != "" {
				s.SerialNumber = serialNumber //запоминаем
			}
			if hw != nil {
				s.HWSprId = InsertHWSpr(hw, &s, ov)
			} else {
				s.HWSprId = 0
			}
		}

		if CheckHypervisorOS(&s) { //если это гипервизор
			host := grpcclient.GetHostBySN(s.SerialNumber) //получаем данные из vmware
			if host != nil {                               //хост найден
				s.Vcenter = host.Vcenter
				s.Datacenter = host.Datacenter
				s.Cluster = host.Cluster
				s.Type = 2
				s.Stat = 1
				s.Active = 1
				s.NameFQDN = host.Name
				if host.PowerState {
					s.PowerState = 1
				}
			}
		}

		num, err := o.Update(&s)
		if err != nil || num < 1 {
			fmt.Println("Ошибка обновления информации сервера: ", s.Name, s.NameFQDN, s.NameInfo)
		}
	}
}

func checkVirtualServerLink(rec models.ServerSprRecords) {
	vmrs := make([]models.VMSprRecord, 0)
	vmrec := models.VMSprRecord{Id: rec.VMSprId}
	o := orm.NewOrm()
	if rec.VMSprId == 0 {
		srv := models.ServerSprRecords{Id: rec.Id}
		err := o.Read(&srv, "Id")
		srv.Stat = 0
		if err == nil {
			o.Update(&srv)
		}
		return
	}
	err := o.Read(&vmrec, "Id")
	if err != nil {
		srv := models.ServerSprRecords{Id: rec.Id}
		err := o.Read(&srv, "Id")
		srv.Stat = 0
		srv.VMSprId = 0
		if err == nil {
			o.Update(&srv)
		}
		return
	}

	qs := o.QueryTable("vm_spr_record")
	qs = qs.OrderBy("Vcenter", "Vm_Uuid")
	qs = qs.Filter("Vcenter", vmrec.Vcenter).Filter("Vm_Uuid", vmrec.Vm_Uuid)
	num, err := qs.All(&vmrs)
	if err != nil || num < 1 { //возможно избыточно
		srv := models.ServerSprRecords{Id: rec.Id}
		err := o.Read(&srv, "Id")
		srv.Stat = 0
		if err == nil {
			o.Update(&srv)
		}
		return
	}
	if num == 1 {
		return
	}
	foundVM := false
	for _, vmr := range vmrs {
		if vmr.Id != vmrec.Id {
			vmd := models.VMSprRecord{Id: vmr.Id}
			err := o.Read(&vmd, "Id")
			if err == nil {
				o.Delete(&vmd, "Id")
			}
		} else {
			foundVM = true
		}
	}
	_ = foundVM
}

func checkHypervisorServerLink(rec models.ServerSprRecords) {

}

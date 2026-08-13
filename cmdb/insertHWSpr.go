package cmdb

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// InsertHWSpr создаёт новую запись конфигурации оборудования (ServerHardware) на основании
// данных HP OneView, переносит сведения об оперативной памяти и дисковых компонентах
// в справочник ServerHardwareComponents. Возвращает Id созданной записи ServerHardware.
func InsertHWSpr(srv *oneview.Server, srvspr *models.ServerSprRecords, ov *oneview.Infrastructure) int64 {
	if srv == nil || srvspr == nil {
		return 0
	}
	hw := models.ServerHardware{ServerId: srvspr.Id}

	o := orm.NewOrm()

	ImportServerSprFromHW(srvspr, srv) //перенести данные из oneview2.Server в models.ServerSprRecords
	ImportHWSprRecord(&hw, srv, ov)
	if srvspr.Active != 0 {
		hw.Active = srvspr.Active
	}
	hw.Time_Mfy = time.Now().Local()
	HWSprId, err := o.Insert(&hw)
	if err != nil {
		fmt.Println("Ошибка вставки записи конфигурации сервера:", srvspr.Name, srvspr.NameFQDN, err)
		return 0
	}
	srvspr.HWSprId = HWSprId

	hw.SerialNum = srv.Identity.SerialNumber
	var m models.ServerHardwareComponents
	if err := o.Read(&hw, "SerialNum"); err == nil { //ищем запись в справочнике конфигураций по серийному номеру сервера
		//запись в справочнике конфигураций найдена
		hw.ServerId = srvspr.Id //записываем ссылку на справочник серверов

		m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 1}
		if nDel, err := o.Delete(&m, "ServerId", "Type"); err != nil { //удаляем записи о оперативной памяти из базы (для последующего обновления данных )
			fmt.Println("Ошибка удаления записей о оперативной памяти сервера: ", srvspr.Name, srvspr.NameFQDN, err)
		} else {
			_ = nDel
		}
	}

	m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 1} //проверяем остались ли данные по памяти
	if err := o.Read(&m, "ServerId", "Type"); err != nil {        //нет записей по памяти
		for idx, mem := range srv.Memory { //проходим в цикле по всем модулям памяти
			ram := models.ServerHardwareComponents{ //формируем данные для записи по памяти
				ServerId:        hw.Id,
				Type:            1,
				Vendor:          mem.Manufacturer,
				Model:           fmt.Sprintf("%.0fGb", float64(mem.CapacityMiB)/1024.0) + " " + mem.BaseModuleType + " " + mem.MemoryDeviceType + " " + strconv.Itoa(mem.OperatingSpeedMHz),
				CapacityGb:      float64(mem.CapacityMiB) / 1024.0,
				SerialNum:       hw.SerialNum + "-" + strconv.Itoa(idx),
				Slot:            mem.Name,
				ContractDoc:     hw.ContractDoc,
				DateContractDoc: hw.DateContractDoc,
				Invoice:         hw.Invoice,
				DateInvoice:     hw.DateInvoice,
				ServiceContract: hw.ServiceContract,
				DateService:     hw.DateService,
				Time_Mfy:        hw.Time_Mfy,
				Id_User:         hw.Id_User}
			//вставить запись компонент

			if _, err := o.Insert(&ram); err != nil { //вставка записи о оперативной памяти
				fmt.Println("Ошибка записи данных о памяти", err)
			}
		}
	}

	m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 2}
	if nDel, err := o.Delete(&m, "ServerId", "Type"); err != nil { //удаляем записи о дисках из базы (для последующего обновления данных )
		fmt.Println("Ошибка удаления записей о дисках сервера: ", srvspr.Name, srvspr, hw.SerialNum, err)
		_ = nDel
	}

	m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 5}
	if nDel, err := o.Delete(&m, "ServerId", "Type"); err != nil { //удаляем записи о контроллерах дисков из базы (для последующего обновления данных )
		fmt.Println("Ошибка удаления записей о дисках сервера: ", srvspr.Name, srvspr, hw.SerialNum, err)
		_ = nDel
	}

	m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 2}
	if err := o.Read(&m, "ServerId", "Type"); err != nil { //проверяем есть ли записи по этому серверу
		for _, storage := range srv.Controllers { //записей нет, записываем данные по физическим дискам
			raid := models.ServerHardwareComponents{
				ServerId:        hw.Id,
				Type:            5,
				Vendor:          "HPE",
				Model:           storage.Model,
				CapacityGb:      float64(storage.CacheMemorySizeMiB) / 1024.0,
				SerialNum:       strings.TrimSpace(storage.SerialNumber),
				Slot:            storage.Location, //формат строки расположения "ControllerPort:Box:Bay"
				ContractDoc:     hw.ContractDoc,
				DateContractDoc: hw.DateContractDoc,
				Invoice:         hw.Invoice,
				DateInvoice:     hw.DateInvoice,
				ServiceContract: hw.ServiceContract,
				DateService:     hw.DateService,
				Time_Mfy:        hw.Time_Mfy,
				Id_User:         hw.Id_User}
			// Примечание про oneview2: у StorageController нет отдельного
			// серийного номера кэш-модуля (в отличие от старого
			// CacheModuleSerialNumber), поэтому этот довесок к SerialNum
			// больше не добавляется.
			//вставить запись компонент
			if _, err := o.Insert(&raid); err != nil {
				fmt.Println("Ошибка записи данных о диске", err)
			}

			for _, disk := range storage.Drives {
				dsk := models.ServerHardwareComponents{
					ServerId:        hw.Id,
					Type:            2,
					Vendor:          "HPE",
					Model:           fmt.Sprintf("%.0fGb", float64(disk.CapacityMiB)/1024.0) + " " + disk.MediaType + " " + strings.TrimSpace(disk.Model) + " (" + strings.TrimSpace(disk.SerialNumber) + ")",
					CapacityGb:      float64(disk.CapacityMiB) / 1000.0,
					SerialNum:       strings.TrimSpace(disk.SerialNumber),
					Slot:            disk.Location, //формат строки расположения "ControllerPort:Box:Bay"
					ContractDoc:     hw.ContractDoc,
					DateContractDoc: hw.DateContractDoc,
					Invoice:         hw.Invoice,
					DateInvoice:     hw.DateInvoice,
					ServiceContract: hw.ServiceContract,
					DateService:     hw.DateService,
					Time_Mfy:        hw.Time_Mfy,
					Id_User:         hw.Id_User}
				switch disk.MediaType {
				case "HDD":
					dsk.SubType = 2
				case "SSD":
					dsk.SubType = 1
				}
				//вставить запись компонент
				if _, err := o.Insert(&dsk); err != nil {
					fmt.Println("Ошибка записи данных о диске", err)
				}
			}
		}
	}
	//обновление записи в справочнике
	// Примечание про oneview2: у Server/ServerIdentity нет поля Description
	// (в отличие от старого Base.Description) - используем Product сервера.
	hw.SourceUpdate = "HP OneView " + srv.Product
	if _, err := o.Update(&hw); err != nil {
		fmt.Println("Ошибка обновления данных конфигурации серверов", err)
	}
	return HWSprId
}

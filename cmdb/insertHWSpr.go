package cmdb

import (
	"fmt"
	"loadmon/models"
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
	ImportHWSprRecord(o, &hw, srv, ov)
	if srvspr.Active != 0 {
		hw.Active = srvspr.Active
	}
	hw.Time_Mfy = time.Now().Local()
	hw.SourceUpdateType = models.SourceUpdateOneView
	HWSprId, err := o.Insert(&hw)
	if err != nil {
		fmt.Println("Ошибка вставки записи конфигурации сервера:", srvspr.Name, srvspr.NameFQDN, err)
		return 0
	}
	srvspr.HWSprId = HWSprId

	hw.SerialNum = srv.Identity.SerialNumber
	if err := o.Read(&hw, "SerialNum"); err == nil { //ищем запись в справочнике конфигураций по серийному номеру сервера
		//запись в справочнике конфигураций найдена
		hw.ServerId = srvspr.Id //записываем ссылку на справочник серверов
	}

	// Полная пересборка всех компонентов сервера (память/диски/RAID/сеть/
	// FC/PCI-устройства/прошивка/процессоры) - вынесена в
	// cmdb/syncserverhardwarecomponents.go (2026-09-04), см. комментарий
	// там; раньше здесь была локальная копия только для памяти/дисков/RAID.
	SyncServerHardwareComponents(o, &hw, srv)
	//обновление записи в справочнике
	// Примечание про oneview2: у Server/ServerIdentity нет поля Description
	// (в отличие от старого Base.Description) - используем Product сервера.
	hw.SourceUpdate = "HP OneView " + srv.Product
	hw.SourceUpdateType = models.SourceUpdateOneView
	if _, err := o.Update(&hw); err != nil {
		fmt.Println("Ошибка обновления данных конфигурации серверов", err)
	}
	return HWSprId
}

package models

import "time"

// TapeLibrary - ленточная библиотека (CMDB). Данные заводятся импортом из
// Итиллиум (itilliumimport/importtapelibraries.go, конфигурация
// update/tape_library.yaml). Имена полей совпадают с одноимёнными полями
// ServerHardware.
//
// Размещение - один шкаф на запись: площадка и шкаф берутся из «Местоположение»
// (RackCabinetId -> RackCabinet.DatacenterSiteId), начальный юнит - из «Unit N»
// того же поля, размер - из колонки units (SizeUnit).
type TapeLibrary struct {
	Id int64 `form:"Id" json:"Id"`
	// ItiliumStatus - статус по данным Итилиум (единственный статус записи);
	// коды и имена - models.ItiliumStatusList.
	ItiliumStatus   int       `orm:"default(0)" form:"ItiliumStatus" csv:"Статус Итилиум" json:"ItiliumStatus"`
	TimeActiveState time.Time `orm:"null" form:"TimeActiveState" json:"TimeActiveState"` // дата изменения статуса в Итилиум
	// ServerId - ссылка на ServerSprRecords.Id: сервер, к которому подключена
	// библиотека (по «Имя ПК»); 0 - не найден в справочнике серверов.
	ServerId int64 `orm:"null" form:"ServerId" json:"ServerId"`
	// PcName - «Имя ПК» из Итилиум как есть (FQDN сервера-подключения).
	PcName            string    `form:"PcName" json:"PcName"`
	Name              string    `form:"Name" json:"Name"`                              // описание (название)
	Model             string    `form:"Model" json:"Model"`                            // марка/модель
	SerialNum         string    `form:"SerialNum" json:"SerialNum"`                    // серийный номер (ключ поиска при импорте)
	IPAddress         string    `form:"IPAddress" json:"IPAddress"`                    // адрес управления библиотекой
	RackCabinetId     int64     `orm:"null" form:"RackCabinetId" json:"RackCabinetId"` // шкаф (RackCabinet.Id)
	StartUnitPos      int       `form:"StartUnitPos" json:"StartUnitPos"`              // начальный юнит в шкафу
	SizeUnit          int       `form:"SizeUnit" json:"SizeUnit"`                      // размер библиотеки в юнитах
	Owner             string    `form:"Owner" json:"Owner"`                            // автор/ответственный
	Counterparty      string    `orm:"null" form:"Counterparty" json:"Counterparty"`   // поставщик
	ContractDoc       string    `form:"ContractDoc" json:"ContractDoc"`                // документы закупки
	DateContractDoc   time.Time `orm:"null" form:"DateContractDoc" json:"DateContractDoc"`
	InventoryNum      string    `form:"InventoryNum" json:"InventoryNum"`
	InventoryNumBuh   string    `form:"InventoryNumBuh" json:"InventoryNumBuh"`
	TechnicalNumberIT string    `form:"TechnicalNumberIT" json:"TechnicalNumberIT"`
	Anotation         string    `form:"Anotation" json:"Anotation"`
	Id_User           int64     `form:"Id_User" json:"Id_User"`
	Time_Mfy          time.Time `form:"Time_Mfy" json:"Time_Mfy"`
	TimeCreate        time.Time `orm:"null" form:"TimeCreate" json:"TimeCreate"`
	CreatorUser       int64     `orm:"null" form:"CreatorUser" json:"CreatorUser"`
	SourceUpdateType  int       `form:"SourceUpdateType" json:"SourceUpdateType"` // как ServerHardware.SourceUpdateType
}

// TableName возвращает имя таблицы БД для модели TapeLibrary (используется beego ORM).
func (a *TapeLibrary) TableName() string {
	return "tape_library"
}

// TableIndex возвращает список индексов для регистрации таблицы модели TapeLibrary в beego ORM.
func (u *TapeLibrary) TableIndex() [][]string {
	return [][]string{
		[]string{"SerialNum"},
		[]string{"TechnicalNumberIT"},
		[]string{"RackCabinetId"},
		[]string{"ServerId"},
	}
}

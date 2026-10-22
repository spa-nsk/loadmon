package models

import "time"

// NetworkEquipmentType - справочник типов сетевого оборудования (коммутатор,
// маршрутизатор, точка доступа Wi-Fi, межсетевой экран и т.д.). Заполняется
// автоматически импортом из Итиллиум по колонке "Тип оборудования"
// (itilliumimport/importnetworkequipment.go) и правится вручную через
// /sprsrv/networkequipmenttypelist. Записи SwitchSprRecords (справочник
// сетевого оборудования) ссылаются на него через TypeId.
type NetworkEquipmentType struct {
	Id         int64     `form:"Id" orm:"auto" json:"Id"`
	Name       string    `form:"Name" json:"Name"`                        // наименование типа (уникально)
	Annotation string    `orm:"null" form:"Annotation" json:"Annotation"` // описание
	Id_User    int64     `orm:"null" form:"Id_User" json:"Id_User"`
	Time_Mfy   time.Time `orm:"null" form:"Time_Mfy" json:"Time_Mfy"`
}

// TableName возвращает имя таблицы БД для модели NetworkEquipmentType (используется beego ORM).
func (a *NetworkEquipmentType) TableName() string {
	return "network_equipment_type"
}

// TableIndex возвращает список индексов для регистрации таблицы модели NetworkEquipmentType в beego ORM.
func (u *NetworkEquipmentType) TableIndex() [][]string {
	return [][]string{
		[]string{"Name"},
	}
}

package itilliumimport

import (
	"loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// resolveRackCabinet ищет или заводит запись models.RackCabinet по ПАРЕ
// (name, datacenterSiteId), а не только по name.
//
// До этого исправления каждый из четырёх импортёров (СХД, rack-серверы,
// блейд-корзины, блейд-сервера) искал шкаф методом trn.Read(&rack, "Name") -
// строго по имени, без учёта площадки. Так как имена шкафов ("3.1-A-08" и
// т.п.) переиспользуются на разных площадках/ЦОД (см. анализ примера
// СХД_темпл2.xlsx - шкафы "3.1-A-08" и т.п. могут встречаться в разных
// зданиях), первый найденный шкаф с таким Name считался "тем самым", и его
// DatacenterSiteId просто перезаписывался на площадку из последней
// импортированной строки - это молча "переносило" уже размещённое в нём
// оборудование на другую площадку.
//
// Уникальность теперь обеспечена на уровне схемы - составной уникальный
// индекс unique(cabinet_site) на RackCabinet.Name+RackCabinet.DatacenterSiteId
// (см. models_add.go и config.dedupeDuplicateRackCabinets), а эта функция -
// единая точка резолва, которую используют все импортёры вместо
// продублированного trn.Read(&rack, "Name").
//
// Возвращает Id найденного или созданного шкафа. annotation используется
// только при создании новой записи (на существующей не перетирается, чтобы
// не затирать ручные правки - тот же принцип "приоритета ручных правок",
// что и у остальных импортов в проекте).
func resolveRackCabinet(trn orm.TxOrmer, name string, datacenterSiteId int64, annotation string) (int64, error) {
	rack := models.RackCabinet{
		Name:             name,
		DatacenterSiteId: datacenterSiteId,
	}
	err := trn.QueryTable(new(models.RackCabinet)).
		Filter("Name", name).
		Filter("DatacenterSiteId", datacenterSiteId).
		One(&rack)
	if err == nil {
		// Шкаф с такой парой (Name, DatacenterSiteId) уже существует -
		// используем его Id как есть, ничего не перезаписываем.
		return rack.Id, nil
	}

	// Не найден - заводим новый шкаф именно на этой площадке. Раньше в этой
	// ситуации существующий шкаф с таким же Name на ДРУГОЙ площадке был бы
	// найден по имени и "перепривязан"; теперь вместо этого честно
	// создаётся отдельная запись под конкретную площадку.
	rack = models.RackCabinet{
		Name:             name,
		DatacenterSiteId: datacenterSiteId,
		Annotation:       annotation,
		Id_User:          9999,
		Time_Mfy:         time.Now().Local(),
	}
	if _, err := trn.Insert(&rack); err != nil {
		return 0, err
	}
	return rack.Id, nil
}

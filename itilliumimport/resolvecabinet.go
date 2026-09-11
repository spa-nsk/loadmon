package itilliumimport

import (
	"loadmon/models"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// splitColocationCabinetName разбирает "сырой" код шкафа вида "ЦОД_имя /
// код_шкафа" (например "ЦОД М1 / R15-1") на имя ЦОД (dcName) и собственно
// код шкафа (cabinetCode) - формат colocation-площадок, где перед кодом
// шкафа явно указано имя стороннего дата-центра, отличное от площадки самой
// строки выгрузки ("Местоположение"). Если "/" не найдено - dcName == "",
// cabinetCode == raw (без изменений).
//
// 2026-09-11, по запросу пользователя (RackCabinet.Name в справочнике
// оказывался вида "ЦОД М1 / R15-1" целиком, ПРИВЯЗАННЫЙ к площадке из
// "Местоположение" строки, а не к реальному "ЦОД М1" - видно на /sprsrv/
// rackcabinetlist и в каскадном фильтре /sprsrv/view2, где под одной
// площадкой оказывались чужие шкафы): при исправлении аналогичного бага для
// импорта СХД (см. storageplacement.go:parseStoragePlacementSegments,
// CHANGELOG про многошкафный разбор) colocation-разбор был реализован
// ТОЛЬКО там - импорт rack-серверов/блейд-корзин/блейд-серверов
// (importrackservers.go/importblades.go/importbladeservers.go) продолжал
// использовать старый простой composite rack_unit_split (convert.go/
// convertblade.go), который берёт код шкафа ЦЕЛИКОМ, включая "ЦОД_имя /"
// префикс, и резолвит его как обычный шкаф НА ПЛОЩАДКЕ СТРОКИ (ownSiteId) -
// именно так colocation-размещение оборудования (СХД, серверы, блейд-корзины
// физически стоящие в СТОРОННЕМ дата-центре) оказывалось привязано к
// площадке из "Местоположение", а не к реальному ЦОД, названному перед "/".
// Эта функция вынесена сюда (а не продублирована), чтобы использоваться
// ВСЕМИ импортёрами одинаково - см. importblades.go/importbladeservers.go/
// importrackservers.go.
func splitColocationCabinetName(raw string) (dcName, cabinetCode string) {
	raw = strings.TrimSpace(raw)
	idx := strings.Index(raw, "/")
	if idx < 0 {
		return "", raw
	}
	return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+1:])
}

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

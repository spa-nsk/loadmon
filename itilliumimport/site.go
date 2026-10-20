package itilliumimport

import (
	"strings"
	"time"

	"loadmon/locality"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// siteResult - площадка/ЦОД, определённая по полю "Местоположение".
type siteResult struct {
	Id   int64
	Name string
	Loc  *locality.Location // разобранное "Местоположение" (шкаф/позиция - запасной источник размещения)
}

// buildSiteName формирует "чистое" наименование площадки (без описания, оно
// хранится в AltName): "<населённый пункт>, <улица, дом>, этаж <N>, <комната>".
func buildSiteName(loc *locality.Location, cityName string) string {
	parts := make([]string, 0, 4)
	if cityName != "" {
		parts = append(parts, cityName)
	}
	if a := loc.Address(); a != "" {
		parts = append(parts, a)
	}
	if loc.Floor != "" {
		parts = append(parts, "этаж "+loc.Floor)
	}
	if loc.Room != "" {
		parts = append(parts, loc.Room)
	}
	return strings.Join(parts, ", ")
}

// siteFields возвращает значения полей площадки по разобранному "Местоположению"
// (rs может быть nil - тогда код населённого пункта не определяется).
func siteFields(loc *locality.Location, rs *locality.Resolver) (name, altName, address, code string) {
	cityName := loc.Locality
	if rs != nil && loc.Locality != "" {
		if m := rs.Find(loc.Locality); m != nil {
			code = m.Loc.Oktmo
			cityName = m.Loc.Name
		}
	}
	return buildSiteName(loc, cityName), loc.Description, loc.Address(), code
}

// resolveSite ищет или заводит models.DatacenterSite по сырому значению поля
// "Местоположение" (rawLocation): населённый пункт определяется по справочнику
// models.Locality (код ОКТМО -> LocalityCode), описание -> AltName, улица и дом
// -> Address, а также Floor и Room. Ключ поиска - пара (Name, AltName), затем
// структурный ключ (LocalityCode, Address, Floor, Room, AltName); уже
// существующая запись не перезаписывается (приоритет ручных правок), лишь
// проставляется пустой LocalityCode. Пустое rawLocation -> площадка 0 (не задана).
func resolveSite(trn orm.TxOrmer, rs *locality.Resolver, rawLocation string, userId int64) (siteResult, error) {
	rawLocation = strings.TrimSpace(rawLocation)
	if rawLocation == "" {
		return siteResult{Loc: &locality.Location{}}, nil
	}
	loc := locality.ParseLocation(rawLocation, rs.IsLocality)

	var name, altName, address, code string
	if loc.Service {
		// служебная отметка ("! Покинуло ФГ ...") - площадка с именем-отметкой
		name = rawLocation
	} else {
		name, altName, address, code = siteFields(loc, rs)
	}

	var found []models.DatacenterSite
	if _, err := trn.QueryTable(new(models.DatacenterSite)).Filter("Name", name).All(&found); err != nil {
		return siteResult{}, err
	}
	for _, s := range found {
		if strings.TrimSpace(s.AltName) == altName {
			if code != "" && s.LocalityCode == "" {
				s.LocalityCode = code
				if _, err := trn.Update(&s, "LocalityCode"); err != nil {
					return siteResult{}, err
				}
			}
			return siteResult{Id: s.Id, Name: s.Name, Loc: loc}, nil
		}
	}

	// Структурный ключ: та же площадка могла быть переименована вручную (Name
	// отличается) - ищем по (код населённого пункта, адрес, этаж, комната,
	// альтернативное наименование), чтобы повторный импорт не плодил дубли.
	if code != "" {
		var byKey []models.DatacenterSite
		if _, err := trn.QueryTable(new(models.DatacenterSite)).Filter("LocalityCode", code).All(&byKey); err != nil {
			return siteResult{}, err
		}
		for _, s := range byKey {
			if strings.TrimSpace(s.Address) == address && strings.TrimSpace(s.Floor) == loc.Floor &&
				strings.TrimSpace(s.Room) == loc.Room && strings.TrimSpace(s.AltName) == altName {
				return siteResult{Id: s.Id, Name: s.Name, Loc: loc}, nil
			}
		}
	}

	site := models.DatacenterSite{
		Name:         name,
		AltName:      altName,
		Address:      address,
		Floor:        loc.Floor,
		Room:         loc.Room,
		LocalityCode: code,
		Annotation:   "Местоположение (Итиллиум): " + rawLocation,
		Id_User:      userId,
		Time_Mfy:     time.Now().Local(),
	}
	if _, err := trn.Insert(&site); err != nil {
		return siteResult{}, err
	}
	return siteResult{Id: site.Id, Name: name, Loc: loc}, nil
}

// cabinetFromPlacement определяет шкаф: rawPlacement - значение "Размещение в
// шкафу/стойке" без диапазона юнитов ("альт.наименование / шкаф"); если оно
// пустое - шкаф из "Местоположения" (loc.Cabinet). Возвращает код шкафа
// (чистое наименование), альтернативное наименование шкафа и позицию из
// "Местоположения" (если шкаф взят оттуда).
func cabinetFromPlacement(rawPlacement string, loc *locality.Location) (cabinet, altName string, pos *locality.Position) {
	altName, cabinet = splitColocationCabinetName(rawPlacement)
	if cabinet == "" && loc != nil && loc.Cabinet != "" {
		return loc.Cabinet, "", loc.Pos
	}
	return cabinet, altName, nil
}

package itilliumimport

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"loadmon/locality"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// legacySiteAnnotationRe - аннотация площадки, заведённой прежним импортом:
// "ЦОД по адресу: <адрес> этаж:<этаж> комната: <комната>"; Name такой площадки
// был "<адрес> <комната>".
var legacySiteAnnotationRe = regexp.MustCompile(`(?s)^ЦОД по адресу:\s*(.*?)\s+этаж:(.*?)\s+комната:\s*(.*)$`)

// MigrateReport - итог MigrateSites.
type MigrateReport struct {
	SitesTotal      int
	SitesUpdated    int
	SitesSkipped    int // нестандартные/вручную изменённые - не трогались
	SitesNoLocality int // населённый пункт не найден в справочнике
	CabinetsUpdated int
	Duplicates      int // пары площадок с одинаковыми (Name, AltName) после обновления - вручную объединить
	Lines           []string
}

func (r *MigrateReport) logf(format string, args ...any) {
	r.Lines = append(r.Lines, fmt.Sprintf(format, args...))
}

// MigrateSites приводит данные, загруженные прежним импортом из Итиллиум, к
// новой схеме (см. resolveSite): по аннотации площадки восстанавливает
// "Местоположение" и разбирает его - населённый пункт -> LocalityCode, описание
// -> AltName, улица+дом -> Address, этаж/комната -> Floor/Room, Name -> чистое
// наименование; у шкафов вида "альт / код" имя делится на AltName и Name.
// Изменённые вручную записи (Name не равен прежнему шаблону) не трогаются,
// непустые значения полей не затираются (кроме Address/Floor, совпадающих с
// заполненными прежним скриптом). При dryRun БД не меняется.
func MigrateSites(dryRun bool) (*MigrateReport, error) {
	rep := &MigrateReport{}
	rs := locality.NewResolver()
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		return nil, err
	}
	finish := func(err error) (*MigrateReport, error) {
		if err != nil || dryRun {
			tx.Rollback()
			return rep, err
		}
		return rep, tx.Commit()
	}

	var sites []models.DatacenterSite
	if _, err := tx.QueryTable(new(models.DatacenterSite)).Limit(-1).All(&sites); err != nil {
		return finish(err)
	}
	rep.SitesTotal = len(sites)

	for _, s := range sites {
		m := legacySiteAnnotationRe.FindStringSubmatch(s.Annotation)
		if m == nil {
			rep.SitesSkipped++
			continue
		}
		addr, floor, room := strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
		if strings.TrimSpace(s.Name) != strings.TrimSpace(addr+" "+room) {
			rep.SitesSkipped++
			rep.logf("пропущена площадка %d %q: наименование изменено вручную", s.Id, s.Name)
			continue
		}

		var loc *locality.Location
		switch {
		case floor != "" && room != "":
			loc = locality.ParseLocation(addr+" | "+floor+" | "+room, rs.IsLocality)
		case floor != "":
			loc = locality.ParseLocation(addr+" | "+floor, rs.IsLocality)
		default:
			loc = locality.ParseLocation(addr, rs.IsLocality)
			if room != "" {
				loc.Room = room
			}
		}
		if loc.Locality != "" && rs.Find(loc.Locality) == nil {
			rep.SitesNoLocality++
			rep.logf("площадка %d %q: населённый пункт %q не найден в справочнике", s.Id, s.Name, loc.Locality)
		}

		var name, altName, address, code string
		if loc.Service {
			name = s.Name
		} else {
			name, altName, address, code = siteFields(loc, rs)
		}

		var cols []string
		set := func(col string, cur *string, val string, force bool) {
			if *cur == val || (!force && strings.TrimSpace(*cur) != "") {
				return
			}
			*cur = val
			cols = append(cols, col)
		}
		set("Name", &s.Name, name, true)
		set("AltName", &s.AltName, altName, false)
		set("LocalityCode", &s.LocalityCode, code, false)
		// Address/Floor, проставленные прежним SQL-скриптом (копия адреса/этажа
		// из аннотации), перезаписываются; ручные значения - нет
		set("Address", &s.Address, address, strings.TrimSpace(s.Address) == addr)
		set("Floor", &s.Floor, loc.Floor, strings.TrimSpace(s.Floor) == floor)
		set("Room", &s.Room, loc.Room, false)

		if len(cols) == 0 {
			continue
		}
		s.Time_Mfy = time.Now().Local()
		cols = append(cols, "Time_Mfy")
		if _, err := tx.Update(&s, cols...); err != nil {
			return finish(fmt.Errorf("площадка %d: %w", s.Id, err))
		}
		rep.SitesUpdated++
	}

	// шкафы: "альт / код" -> AltName + чистый Name
	var racks []models.RackCabinet
	if _, err := tx.QueryTable(new(models.RackCabinet)).Filter("Name__contains", "/").Limit(-1).All(&racks); err != nil {
		return finish(err)
	}
	for _, rc := range racks {
		alt, code := splitColocationCabinetName(rc.Name)
		if code == "" {
			continue
		}
		n, err := tx.QueryTable(new(models.RackCabinet)).
			Filter("Name", code).Filter("DatacenterSiteId", rc.DatacenterSiteId).Exclude("Id", rc.Id).Count()
		if err != nil {
			return finish(err)
		}
		if n > 0 {
			rep.logf("шкаф %d %q: шкаф %q на той же площадке уже есть - объедините вручную", rc.Id, rc.Name, code)
			continue
		}
		rc.Name = code
		if strings.TrimSpace(rc.AltName) == "" {
			rc.AltName = alt
		}
		if _, err := tx.Update(&rc, "Name", "AltName"); err != nil {
			return finish(fmt.Errorf("шкаф %d: %w", rc.Id, err))
		}
		rep.CabinetsUpdated++
	}

	// одинаковые (Name, AltName) после обновления
	var all []models.DatacenterSite
	if _, err := tx.QueryTable(new(models.DatacenterSite)).Limit(-1).All(&all); err != nil {
		return finish(err)
	}
	seen := map[string]int64{}
	for _, s := range all {
		k := s.Name + "\x00" + strings.TrimSpace(s.AltName)
		if first, ok := seen[k]; ok {
			rep.Duplicates++
			rep.logf("дубль площадки: %d и %d (%q) - объедините вручную", first, s.Id, s.Name)
			continue
		}
		seen[k] = s.Id
	}
	return finish(nil)
}

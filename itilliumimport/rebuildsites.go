package itilliumimport

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"loadmon/locality"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// Пересоздание справочника площадок/шкафов: резервная копия -> очистка ->
// (свежий импорт четырёх выгрузок Итиллиум) -> возврат ручных атрибутов.
// См. tools/siterebuild.

type backupSite struct {
	Key          string `json:"key"`
	OldId        int64  `json:"oldId"`
	IsDatacenter bool   `json:"isDatacenter"`
	CodeJira     string `json:"codeJira,omitempty"`
	Width        int64  `json:"width,omitempty"`
	Height       int64  `json:"height,omitempty"`
}

type backupCabinet struct {
	SiteKey      string `json:"siteKey"`
	Name         string `json:"name"` // чистое наименование шкафа
	OldId        int64  `json:"oldId"`
	Model        string `json:"model,omitempty"`
	SerialNumber string `json:"serialNumber,omitempty"`
	Width        int64  `json:"width,omitempty"`
	Depth        int64  `json:"depth,omitempty"`
	Height       int64  `json:"height,omitempty"`
	UHeight      int64  `json:"uHeight,omitempty"`
	Annotation   string `json:"annotation,omitempty"`
}

type backupSwitch struct {
	Id          int64  `json:"id"`
	SiteKey     string `json:"siteKey,omitempty"`
	RackSiteKey string `json:"rackSiteKey,omitempty"`
	RackName    string `json:"rackName,omitempty"`
}

// SitesBackup - содержимое файла резервной копии: полные копии строк
// справочников (для аварийного восстановления вручную) и ключи для возврата
// ручных атрибутов после повторного импорта.
type SitesBackup struct {
	Created  time.Time               `json:"created"`
	Sites    []models.DatacenterSite `json:"sites"`
	Cabinets []models.RackCabinet    `json:"cabinets"`

	ManualSites    []backupSite    `json:"manualSites"`
	ManualCabinets []backupCabinet `json:"manualCabinets"`
	Switches       []backupSwitch  `json:"switches"`
}

// siteKeyOf - ключ площадки, не зависящий от Id и старого/нового формата
// наименования: "код|адрес|этаж|комната|описание". Для записей, заведённых
// прежним импортом, ключ восстанавливается из аннотации (как в MigrateSites);
// для прочих - по полному наименованию.
func siteKeyOf(s models.DatacenterSite, rs *locality.Resolver) string {
	mk := func(code, address, floor, room, alt string) string {
		return strings.Join([]string{code, address, floor, room, alt}, "|")
	}
	if s.LocalityCode != "" {
		return mk(s.LocalityCode, strings.TrimSpace(s.Address), strings.TrimSpace(s.Floor), strings.TrimSpace(s.Room), strings.TrimSpace(s.AltName))
	}
	if m := legacySiteAnnotationRe.FindStringSubmatch(s.Annotation); m != nil {
		addr, floor, room := strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
		if strings.TrimSpace(s.Name) == strings.TrimSpace(addr+" "+room) {
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
			if !loc.Service {
				_, alt, address, code := siteFields(loc, rs)
				if code != "" {
					return mk(code, address, loc.Floor, loc.Room, alt)
				}
			}
		}
	}
	return "name:" + strings.TrimSpace(s.Name)
}

// BackupSites сохраняет в файл path копию справочников площадок и шкафов,
// ручные атрибуты и ручные привязки коммутаторов.
func BackupSites(path string) (*SitesBackup, error) {
	o := orm.NewOrm()
	rs := locality.NewResolver()
	b := &SitesBackup{Created: time.Now().Local()}

	if _, err := o.QueryTable(new(models.DatacenterSite)).Limit(-1).All(&b.Sites); err != nil {
		return nil, err
	}
	if _, err := o.QueryTable(new(models.RackCabinet)).Limit(-1).All(&b.Cabinets); err != nil {
		return nil, err
	}
	siteKey := map[int64]string{}
	for _, s := range b.Sites {
		k := siteKeyOf(s, rs)
		siteKey[s.Id] = k
		b.ManualSites = append(b.ManualSites, backupSite{Key: k, OldId: s.Id, IsDatacenter: s.IsDatacenter,
			CodeJira: s.CodeJira, Width: s.Width, Height: s.Height})
	}
	type rackRef struct{ siteKey, name string }
	rackOf := map[int64]rackRef{}
	for _, c := range b.Cabinets {
		_, code := splitColocationCabinetName(c.Name)
		if code == "" {
			code = strings.TrimSpace(c.Name)
		}
		sk := siteKey[c.DatacenterSiteId]
		rackOf[c.Id] = rackRef{sk, code}
		b.ManualCabinets = append(b.ManualCabinets, backupCabinet{SiteKey: sk, Name: code, OldId: c.Id,
			Model: c.Model, SerialNumber: c.SerialNumber, Width: c.Width, Depth: c.Depth, Height: c.Height,
			UHeight: c.UHeight, Annotation: c.Annotation})
	}
	var sws []models.SwitchSprRecords
	if _, err := o.QueryTable(new(models.SwitchSprRecords)).Limit(-1).All(&sws); err != nil {
		return nil, err
	}
	for _, sw := range sws {
		if sw.DatacenterSiteId == 0 && sw.RackCabinetId == 0 {
			continue
		}
		bs := backupSwitch{Id: sw.Id, SiteKey: siteKey[sw.DatacenterSiteId]}
		if r, ok := rackOf[sw.RackCabinetId]; ok {
			bs.RackSiteKey, bs.RackName = r.siteKey, r.name
		}
		b.Switches = append(b.Switches, bs)
	}

	data, err := json.MarshalIndent(b, "", " ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return b, nil
}

// ClearReport - сколько записей затронуто ClearSites.
type ClearReport struct {
	ServersUnlinked, BladesUnlinked, SwitchesUnlinked int64
	StoragePlacements                                 int64
	Cabinets, Sites                                   int64
	RackPower, SitePower                              int64
}

// ClearSites обнуляет привязки оборудования к шкафам/площадкам (серверы,
// блейд-корзины, коммутаторы), удаляет размещения СХД, шкафы, площадки и
// историю мощности шкафов/площадок (rack_power_record, site_power_record).
// Всё в одной транзакции; при dryRun только считает записи.
func ClearSites(dryRun bool) (*ClearReport, error) {
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		return nil, err
	}
	rep := &ClearReport{}
	fail := func(err error) (*ClearReport, error) {
		tx.Rollback()
		return rep, err
	}
	step := func(dst *int64, model any, filterField string, update bool) error {
		qs := tx.QueryTable(model).Filter(filterField+"__gt", 0)
		if dryRun {
			n, err := qs.Count()
			*dst = n
			return err
		}
		var n int64
		var err error
		if update {
			n, err = qs.Update(orm.Params{filterField: 0})
		} else {
			n, err = qs.Delete()
		}
		*dst = n
		return err
	}
	steps := []struct {
		dst    *int64
		model  any
		field  string
		update bool
		what   string
	}{
		{&rep.ServersUnlinked, new(models.ServerHardware), "RackCabinetId", true, "серверы"},
		{&rep.BladesUnlinked, new(models.BladeSystems), "RackCabinetId", true, "блейд-корзины"},
		{&rep.SwitchesUnlinked, new(models.SwitchSprRecords), "RackCabinetId", true, "коммутаторы (шкаф)"},
		{nil, new(models.SwitchSprRecords), "DatacenterSiteId", true, "коммутаторы (площадка)"},
		{&rep.StoragePlacements, new(models.StorageSystemPlacement), "Id", false, "размещения СХД"},
		{&rep.RackPower, new(models.RackPowerRecord), "Id", false, "история мощности шкафов"},
		{&rep.SitePower, new(models.SitePowerRecord), "Id", false, "история мощности площадок"},
		{&rep.Cabinets, new(models.RackCabinet), "Id", false, "шкафы"},
		{&rep.Sites, new(models.DatacenterSite), "Id", false, "площадки"},
	}
	for _, st := range steps {
		dst := st.dst
		if dst == nil {
			dst = new(int64)
		}
		if err := step(dst, st.model, st.field, st.update); err != nil {
			return fail(fmt.Errorf("%s: %w", st.what, err))
		}
	}
	if dryRun {
		tx.Rollback()
		return rep, nil
	}
	return rep, tx.Commit()
}

// RestoreReport - итог RestoreManual.
type RestoreReport struct {
	SitesRestored, CabinetsRestored, SwitchesRelinked int
	SitesNotFound, CabinetsNotFound                   int
	Lines                                             []string
}

// RestoreManual возвращает после повторного импорта ручные атрибуты
// площадок (признак ЦОД, код Jira, размеры), шкафов (модель, серийный номер,
// размеры, ручное описание) и привязки коммутаторов - по ключам из файла
// резервной копии, который создал BackupSites. Уже заполненные в новых
// записях значения не затираются. При dryRun БД не меняется.
func RestoreManual(path string, dryRun bool) (*RestoreReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b SitesBackup
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	rep := &RestoreReport{}
	rs := locality.NewResolver()
	o := orm.NewOrm()
	tx, err := o.Begin()
	if err != nil {
		return nil, err
	}
	finish := func(err error) (*RestoreReport, error) {
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
	newSiteByKey := map[string]*models.DatacenterSite{}
	for i := range sites {
		newSiteByKey[siteKeyOf(sites[i], rs)] = &sites[i]
	}

	for _, bs := range b.ManualSites {
		s := newSiteByKey[bs.Key]
		if s == nil {
			rep.SitesNotFound++
			if bs.IsDatacenter || bs.CodeJira != "" {
				rep.Lines = append(rep.Lines, fmt.Sprintf("площадка %q (код Jira %q, ЦОД=%v) не найдена после импорта", bs.Key, bs.CodeJira, bs.IsDatacenter))
			}
			continue
		}
		var cols []string
		if bs.IsDatacenter && !s.IsDatacenter {
			s.IsDatacenter = true
			cols = append(cols, "IsDatacenter")
		}
		if bs.CodeJira != "" && s.CodeJira == "" {
			s.CodeJira = bs.CodeJira
			cols = append(cols, "CodeJira")
		}
		if bs.Width != 0 && s.Width == 0 {
			s.Width = bs.Width
			cols = append(cols, "Width")
		}
		if bs.Height != 0 && s.Height == 0 {
			s.Height = bs.Height
			cols = append(cols, "Height")
		}
		if len(cols) == 0 {
			continue
		}
		if _, err := tx.Update(s, cols...); err != nil {
			return finish(err)
		}
		rep.SitesRestored++
	}

	var racks []models.RackCabinet
	if _, err := tx.QueryTable(new(models.RackCabinet)).Limit(-1).All(&racks); err != nil {
		return finish(err)
	}
	siteKeyByNewId := map[int64]string{}
	for k, s := range newSiteByKey {
		siteKeyByNewId[s.Id] = k
	}
	rackByKey := map[string]*models.RackCabinet{}
	for i := range racks {
		rackByKey[siteKeyByNewId[racks[i].DatacenterSiteId]+"\x00"+racks[i].Name] = &racks[i]
	}
	for _, bc := range b.ManualCabinets {
		rc := rackByKey[bc.SiteKey+"\x00"+bc.Name]
		if rc == nil {
			rep.CabinetsNotFound++
			continue
		}
		var cols []string
		setStr := func(col string, cur *string, val string) {
			if val != "" && strings.TrimSpace(*cur) == "" {
				*cur = val
				cols = append(cols, col)
			}
		}
		setInt := func(col string, cur *int64, val int64) {
			if val != 0 && *cur == 0 {
				*cur = val
				cols = append(cols, col)
			}
		}
		setStr("Model", &rc.Model, bc.Model)
		setStr("SerialNumber", &rc.SerialNumber, bc.SerialNumber)
		setInt("Width", &rc.Width, bc.Width)
		setInt("Depth", &rc.Depth, bc.Depth)
		setInt("Height", &rc.Height, bc.Height)
		setInt("UHeight", &rc.UHeight, bc.UHeight)
		// описание, созданное импортом ("Шкаф по ..."), не переносим - только ручное
		if !strings.HasPrefix(bc.Annotation, "Шкаф по") && !strings.HasPrefix(bc.Annotation, "Шкаф на colocation") {
			if bc.Annotation != "" && strings.HasPrefix(rc.Annotation, "Шкаф по") {
				rc.Annotation = bc.Annotation
				cols = append(cols, "Annotation")
			}
		}
		if len(cols) == 0 {
			continue
		}
		if _, err := tx.Update(rc, cols...); err != nil {
			return finish(err)
		}
		rep.CabinetsRestored++
	}

	for _, sw := range b.Switches {
		cur := models.SwitchSprRecords{Id: sw.Id}
		if err := tx.Read(&cur); err != nil {
			continue
		}
		var cols []string
		if s := newSiteByKey[sw.SiteKey]; s != nil && sw.SiteKey != "" {
			cur.DatacenterSiteId = s.Id
			cols = append(cols, "DatacenterSiteId")
		}
		if rc := rackByKey[sw.RackSiteKey+"\x00"+sw.RackName]; rc != nil && sw.RackName != "" {
			cur.RackCabinetId = rc.Id
			cols = append(cols, "RackCabinetId")
		}
		if len(cols) == 0 {
			continue
		}
		if _, err := tx.Update(&cur, cols...); err != nil {
			return finish(err)
		}
		rep.SwitchesRelinked++
	}
	return finish(nil)
}

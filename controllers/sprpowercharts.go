package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"
	"time"

	"loadmon/models"
	"loadmon/rackpower"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// Графики потребляемой мощности (серверы, блейд-корзины, СХД, шкафы).
//
//   - /sprsrv/powerrecent/:kind/:id - последние 3 суток по поминутным
//     данным (server_power_metric / storage_power_metric);
//   - /sprsrv/powerview/:kind/:id   - период по суточной агрегации
//     (server_power_record / storage_power_record: мин/сред/макс) плюс
//     детальные поминутные данные за последние 3 суток, как у метрик СХД
//     (sprstoragesystemmetricsview.go).
//
// kind: server (Id - ServerSprRecords.Id), storage (StorageSystems.Id),
// blade (BladeSystems.Id), rack (RackCabinet.Id). Корзина и шкаф - расчётные
// графики: сумма рядов входящих устройств (у корзины OneView нет временного
// ряда, берётся сумма блейд-серверов; у шкафа - rack-серверы, блейды корзин
// и СХД с долей по размещению). Блейд-серверы данных мощности не отдают -
// только корзина в целом (OneView: среднее и пик за 24 ч), поэтому у корзины
// только график за период по суточным снимкам (blade_power_record) без
// детальных данных, а в шкафу корзина входит ступенчатой линией по этим
// снимкам. Суточные мин/макс таких сумм - суммы
// суточных мин/макс устройств (оценка: экстремумы устройств разнесены во времени).

const powerRecentWindow = 3 * 24 * time.Hour

type powerMember struct {
	storage bool // СХД (StorageSystems.Id)
	chassis bool // блейд-корзина (BladeSystems.Id): только суточные данные OneView, без поминутных
	id      int64
	share   float64
}

// powerServerIds переводит ServerHardware в ServerSprRecords.Id: ServerId,
// иначе через ServerSprRecords.HWSprId.
func powerServerIds(o orm.Ormer, hws []models.ServerHardware) []int64 {
	var ids []int64
	seen := map[int64]bool{}
	var noSpr []int64
	for _, h := range hws {
		if h.ServerId > 0 {
			if !seen[h.ServerId] {
				seen[h.ServerId] = true
				ids = append(ids, h.ServerId)
			}
		} else {
			noSpr = append(noSpr, h.Id)
		}
	}
	if len(noSpr) > 0 {
		var sprs []models.ServerSprRecords
		if _, err := models.ServerSprQS(o).Filter("Stat", 1).Filter("HWSprId__in", noSpr).All(&sprs, "Id", "HWSprId"); err == nil {
			for _, s := range sprs {
				if !seen[s.Id] {
					seen[s.Id] = true
					ids = append(ids, s.Id)
				}
			}
		}
	}
	return ids
}

// powerResolve определяет заголовок и состав устройств графика.
func powerResolve(o orm.Ormer, kind string, id int64) (title, note string, members []powerMember, err error) {
	serverMembers := func(ids []int64) {
		for _, sid := range ids {
			members = append(members, powerMember{id: sid, share: 1})
		}
	}
	switch kind {
	case "server":
		var s models.ServerSprRecords
		if e := o.QueryTable("server_spr_records").Filter("Id", id).One(&s); e != nil {
			return "", "", nil, fmt.Errorf("сервер Id=%d не найден: %v", id, e)
		}
		title = "Сервер " + firstNonEmpty(s.Name, s.NameFQDN)
		members = []powerMember{{id: id, share: 1}}
	case "storage":
		var s models.StorageSystems
		if e := o.QueryTable("storage_systems").Filter("Id", id).One(&s); e != nil {
			return "", "", nil, fmt.Errorf("СХД Id=%d не найдена: %v", id, e)
		}
		title = "СХД " + firstNonEmpty(s.ShortName, s.SerialNum)
		if s.IpAddress != "" {
			title += " / " + s.IpAddress
		}
		members = []powerMember{{storage: true, id: id, share: 1}}
	case "blade":
		var b models.BladeSystems
		if e := o.QueryTable(new(models.BladeSystems)).Filter("Id", id).One(&b); e != nil {
			return "", "", nil, fmt.Errorf("корзина Id=%d не найдена: %v", id, e)
		}
		title = "Блейд-корзина " + firstNonEmpty(b.Name, b.SerialNum)
		members = []powerMember{{chassis: true, id: id, share: 1}}
		note = "Данные о мощности есть только по корзине в целом (OneView: среднее и пик за 24 часа, снимок раз в сутки). По блейд-серверам мощность не отдаётся, детальных данных нет."
	case "rack":
		var c models.RackCabinet
		if e := o.QueryTable(new(models.RackCabinet)).Filter("Id", id).One(&c); e != nil {
			return "", "", nil, fmt.Errorf("шкаф Id=%d не найден: %v", id, e)
		}
		title = "Шкаф " + c.Name
		var hws []models.ServerHardware
		if _, e := o.QueryTable(new(models.ServerHardware)).Filter("RackCabinetId", id).Exclude("Active", 2).All(&hws); e != nil {
			return "", "", nil, e
		}
		var chassis []models.BladeSystems
		_, _ = o.QueryTable(new(models.BladeSystems)).Filter("RackCabinetId", id).Exclude("Active", 2).All(&chassis, "Id")
		// блейд-серверы мощность не отдают - только корзины целиком (ниже)
		rackHws := hws[:0]
		for _, h := range hws {
			if h.FormFactor == 2 && h.BladeSystemId > 0 {
				continue
			}
			rackHws = append(rackHws, h)
		}
		serverMembers(powerServerIds(o, rackHws))
		nServers := len(members)
		for _, ch := range chassis {
			members = append(members, powerMember{chassis: true, id: ch.Id, share: 1})
		}

		var placements []models.StorageSystemPlacement
		_, _ = o.QueryTable(new(models.StorageSystemPlacement)).Filter("RackCabinetId", id).All(&placements)
		nStorage := 0
		if len(placements) > 0 {
			sysIds := make([]int64, 0, len(placements))
			for _, p := range placements {
				sysIds = append(sysIds, p.StorageSystemsId)
			}
			share := rackpower.PlacementShare(o, sysIds)
			for _, p := range placements {
				members = append(members, powerMember{storage: true, id: p.StorageSystemsId, share: share(p)})
				nStorage++
			}
		}
		note = fmt.Sprintf("Расчётный график: сумма мощности серверов шкафа (%d), долей СХД (%d) и блейд-корзин целиком (%d; по ним только среднее за 24 ч из OneView, снимок раз в сутки - в детальном графике они идут ступенями по суткам). Коммутаторы и прочее оборудование не учтены. Минимум/максимум за сутки - суммы суточных значений устройств (оценка; у корзин минимум равен среднему, максимум - пику за 24 ч).", nServers, nStorage, len(chassis))
	default:
		return "", "", nil, fmt.Errorf("неизвестный тип объекта: %s", kind)
	}
	return title, note, members, nil
}

func powerPoints(m map[int64]float64) []vmPerfSeriesPoint {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]vmPerfSeriesPoint, 0, len(keys))
	for _, k := range keys {
		out = append(out, vmPerfSeriesPoint{T: k, V: m[k]})
	}
	return out
}

func powerSplit(members []powerMember) (srv, sto, chs map[int64]float64) {
	srv, sto, chs = map[int64]float64{}, map[int64]float64{}, map[int64]float64{}
	for _, m := range members {
		switch {
		case m.storage:
			sto[m.id] += m.share
		case m.chassis:
			chs[m.id] += m.share
		default:
			srv[m.id] += m.share
		}
	}
	return
}

func powerDayKey(t time.Time) string { return t.In(time.Local).Format("2006-01-02") }

// powerChassisAvgByDay - среднее за 24 ч по корзинам (сумма) по дням:
// blade_power_record; для дней без снимка (сегодня) - текущее значение
// BladeSystems.PowerAvg24hW.
func powerChassisAvgByDay(o orm.Ormer, chs map[int64]float64, from, to time.Time) (byDay map[string]float64, current float64, err error) {
	byDay = map[string]float64{}
	var recs []models.BladePowerRecord
	if _, err = o.QueryTable(new(models.BladePowerRecord)).Filter("BladeSystemId__in", powerKeys(chs)).
		Filter("Date__gte", CorrectTimeFilter(from)).Filter("Date__lt", to).All(&recs, "BladeSystemId", "Date", "AvgW"); err != nil {
		return
	}
	for _, r := range recs {
		byDay[powerDayKey(r.Date)] += r.AvgW * chs[r.BladeSystemId]
	}
	var cur []models.BladeSystems
	if _, err = o.QueryTable(new(models.BladeSystems)).Filter("Id__in", powerKeys(chs)).All(&cur, "Id", "PowerAvg24hW"); err != nil {
		return
	}
	for _, b := range cur {
		current += b.PowerAvg24hW * chs[b.Id]
	}
	return
}

func powerKeys(m map[int64]float64) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// powerRaw - поминутный ряд: сумма мощности устройств (с учётом долей).
func powerRaw(o orm.Ormer, members []powerMember, from, to time.Time) ([]vmPerfSeriesPoint, error) {
	srv, sto, chs := powerSplit(members)
	sum := map[int64]float64{}
	if len(srv) > 0 {
		var rows []models.ServerPowerMetric
		if _, err := o.QueryTable(new(models.ServerPowerMetric)).Filter("ServerId__in", powerKeys(srv)).
			Filter("CollectedAt__gte", from).Filter("CollectedAt__lt", to).All(&rows, "ServerId", "CollectedAt", "Watts"); err != nil {
			return nil, err
		}
		for _, r := range rows {
			sum[r.CollectedAt.UnixMilli()] += r.Watts * srv[r.ServerId]
		}
	}
	if len(sto) > 0 {
		var rows []models.StoragePowerMetric
		if _, err := o.QueryTable(new(models.StoragePowerMetric)).Filter("StorageSystemsId__in", powerKeys(sto)).
			Filter("CollectedAt__gte", from).Filter("CollectedAt__lt", to).All(&rows, "StorageSystemsId", "CollectedAt", "Watts"); err != nil {
			return nil, err
		}
		for _, r := range rows {
			sum[r.CollectedAt.UnixMilli()] += r.Watts * sto[r.StorageSystemsId]
		}
	}
	if len(chs) > 0 {
		// корзины: ступени по суткам (снимки OneView), сегодня - текущее значение
		if len(sum) == 0 {
			end := to
			if now := time.Now(); end.After(now) {
				end = now
			}
			for t := from.Truncate(10 * time.Minute); t.Before(end); t = t.Add(10 * time.Minute) {
				sum[t.UnixMilli()] = 0
			}
		}
		byDay, current, err := powerChassisAvgByDay(o, chs, from, to)
		if err != nil {
			return nil, err
		}
		today := powerDayKey(time.Now())
		for k := range sum {
			day := powerDayKey(time.UnixMilli(k))
			if v, ok := byDay[day]; ok && day != today {
				sum[k] += v
			} else {
				sum[k] += current
			}
		}
	}
	return powerPoints(sum), nil
}

// powerDaily - суточные ряды мин/сред/макс (суммы по устройствам).
func powerDaily(o orm.Ormer, members []powerMember, from, to time.Time) (min, avg, max []vmPerfSeriesPoint, err error) {
	srv, sto, chs := powerSplit(members)
	mn, av, mx := map[int64]float64{}, map[int64]float64{}, map[int64]float64{}
	if len(srv) > 0 {
		var rows []models.ServerPowerRecord
		if _, err = o.QueryTable(new(models.ServerPowerRecord)).Filter("ServerId__in", powerKeys(srv)).
			Filter("Date__gte", from).Filter("Date__lt", to).All(&rows, "ServerId", "Date", "WattsMin", "WattsAvg", "WattsMax"); err != nil {
			return
		}
		for _, r := range rows {
			k, w := r.Date.UnixMilli(), srv[r.ServerId]
			mn[k] += r.WattsMin * w
			av[k] += r.WattsAvg * w
			mx[k] += r.WattsMax * w
		}
	}
	if len(sto) > 0 {
		var rows []models.StoragePowerRecord
		if _, err = o.QueryTable(new(models.StoragePowerRecord)).Filter("StorageSystemsId__in", powerKeys(sto)).
			Filter("Date__gte", from).Filter("Date__lt", to).All(&rows, "StorageSystemsId", "Date", "WattsMin", "WattsAvg", "WattsMax"); err != nil {
			return
		}
		for _, r := range rows {
			k, w := r.Date.UnixMilli(), sto[r.StorageSystemsId]
			mn[k] += r.WattsMin * w
			av[k] += r.WattsAvg * w
			mx[k] += r.WattsMax * w
		}
	}
	if len(chs) > 0 {
		var rows []models.BladePowerRecord
		if _, err = o.QueryTable(new(models.BladePowerRecord)).Filter("BladeSystemId__in", powerKeys(chs)).
			Filter("Date__gte", from).Filter("Date__lt", to).All(&rows, "BladeSystemId", "Date", "AvgW", "PeakW"); err != nil {
			return
		}
		for _, r := range rows {
			k, w := r.Date.UnixMilli(), chs[r.BladeSystemId]
			mn[k] += r.AvgW * w // минимума у корзины нет - берётся среднее
			av[k] += r.AvgW * w
			mx[k] += r.PeakW * w
		}
	}
	return powerPoints(mn), powerPoints(av), powerPoints(mx), nil
}

// SprPowerRecent - график мощности за последние 3 суток по поминутным данным.
//
// Маршрут: /sprsrv/powerrecent/:kind/:id
func (this *MainController) SprPowerRecent() { this.powerPage("recent") }

// SprPowerView - график мощности за период по суточной агрегации с
// детальными данными за последние 3 суток.
//
// Маршрут: /sprsrv/powerview/:kind/:id
func (this *MainController) SprPowerView() { this.powerPage("period") }

func (this *MainController) powerPage(mode string) {
	kind := strings.ToLower(this.Ctx.Input.Param(":kind"))
	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewLoadTable)
	if kind == "blade" && mode == "recent" {
		// детальных данных по корзине нет - только суточные
		this.Redirect("/sprsrv/powerview/blade/"+idStr, 302)
		return
	}
	this.activeContent("sprsrv/powerchart")

	this.Data["Mode"] = mode
	this.Data["Kind"] = kind
	this.Data["Id"] = idStr
	this.Data["Title"] = ""
	this.Data["Note"] = ""
	this.Data["RecordCount"] = 0
	this.Data["ChartsJSON"] = template.JS("[]")

	flash := beego.NewFlash()
	defer func() {
		if len(flash.Data) > 0 {
			flash.Store(&this.Controller)
		}
	}()

	now := time.Now().Local()
	rawFrom, rawTo := now.Add(-powerRecentWindow), now.Add(time.Minute)
	from := CorrectTimeFilter(now).AddDate(0, 0, -6)
	to := CorrectTimeFilter(now)
	if mode == "period" {
		if s := this.GetString("from"); s != "" {
			if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
				from = t
			}
		}
		if s := this.GetString("to"); s != "" {
			if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
				to = t
			}
		}
		if to.Before(from) {
			from, to = to, from
		}
		rawFrom, rawTo = from, to.AddDate(0, 0, 1)
	}
	this.Data["FromDate"] = from.Format("2006-01-02")
	this.Data["ToDate"] = to.Format("2006-01-02")
	this.Data["FromLabel"] = rawFrom.Format("02.01.2006 15:04")
	this.Data["ToLabel"] = now.Format("02.01.2006 15:04")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор: %s", idStr)
		return
	}
	o := orm.NewOrm()
	title, note, members, err := powerResolve(o, kind, id)
	if err != nil {
		flash.Error("%v", err)
		return
	}
	this.Data["Title"] = title
	this.Data["Note"] = note
	if len(members) == 0 {
		flash.Notice("Нет устройств с данными о мощности")
		return
	}

	chart := &vmPerfPeriodChart{Title: "Потребляемая мощность", Unit: "Вт"}
	count := 0
	if kind != "blade" { // у корзины детальных данных нет
		raw, err := powerRaw(o, members, rawFrom, rawTo)
		if err != nil {
			flash.Error("Ошибка загрузки детальных данных: %v", err)
			return
		}
		chart.Raw = raw
		count = len(raw)
	}
	if mode == "period" {
		mn, av, mx, err := powerDaily(o, members, from, to.AddDate(0, 0, 1))
		if err != nil {
			flash.Error("Ошибка загрузки данных за период: %v", err)
			return
		}
		chart.Min, chart.Avg, chart.Max = mn, av, mx
		if kind == "blade" {
			chart.Min = nil // у корзины только среднее и пик за 24 ч
		}
		count = len(av)
	}
	this.Data["RecordCount"] = count

	b, jerr := json.Marshal([]*vmPerfPeriodChart{chart})
	if jerr != nil {
		flash.Error("Ошибка подготовки данных графиков: %v", jerr)
		return
	}
	this.Data["ChartsJSON"] = template.JS(b)
}

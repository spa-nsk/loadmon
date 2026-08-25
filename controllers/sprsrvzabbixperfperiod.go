package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SprSrvZabbixPerfPeriod отображает динамические графики метрик производительности
// аппаратного сервера за произвольный период по данным, собранным zabbix.Get10MinutesHistory
// (таблицы zabbix_host/zabbix_key/zabbix_item, см. zabbix/zabbixupdate.go).
//
// В отличие от Get10MinutesHistory, которая при каждом опросе сохраняет "сырые" точки
// истории Zabbix без какой-либо агрегации (по одной записи models.ZabbixItem на каждую
// точку history.get, сгруппированные по models.ZabbixKey/KeyParam - Key определяет
// метрику, например "vm.memory.util" или "vfs.dev.read", а KeyParam - конкретный
// экземпляр, например имя диска/сетевого интерфейса), этот метод читает уже накопленные
// за период точки и агрегирует их по временным интервалам (бакетам), вычисляя
// минимум/среднее/максимум в каждом бакете - иначе за период в несколько дней/недель
// график из "сырых" точек (замер примерно раз в 10 минут) был бы нечитаемым.
//
// Оформление страницы и структура графика (ApexCharts, экспорт в PDF) полностью
// повторяют /sprsrv/vmperfday (controllers/sprsrvvmperfday.go, views/sprsrv/vmperfday.tpl) -
// используются те же типы vmPerfChartData/vmPerfSeriesPoint, только каждая метрика
// вместо одного ряда "сырых" точек даёт три ряда - "мин"/"сред"/"макс" (при нескольких
// экземплярах метрики на хосте, например несколько дисков - три ряда на каждый экземпляр).
//
// Маршрут: /sprsrv/zabbixperfperiod/:Id, где :Id - models.ZabbixHost.Id (ссылка
// $record.S.ZabbixHostId в views/sprsrv/view2.tpl). Период задаётся GET-параметрами
// from/to (по умолчанию - 1 сутки, аналогично /sprsrv/clusterperioddb, но короче -
// именно этот график по умолчанию открывают для быстрой проверки текущего
// состояния сервера, а не для анализа за неделю).
func (this *MainController) SprSrvZabbixPerfPeriod() {

	idStr := this.Ctx.Input.Param(":Id")
	hostId, _ := strconv.ParseInt(idStr, 10, 64)

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewLoadTable) // график загрузки аппаратного сервера за период (по данным Zabbix)

	this.activeContent("sprsrv/zabbixperfperiod")

	to := CorrectTimeFilter(time.Now())
	from := to // период по умолчанию - 1 сутки (dateTo ниже включает весь день "to")
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
	dateFrom := from
	dateTo := to.AddDate(0, 0, 1) // включительно до конца дня "to"

	this.Data["FromDate"] = from.Format("2006-01-02")
	this.Data["ToDate"] = to.Format("2006-01-02")
	this.Data["HostId"] = hostId

	// значения по умолчанию - шаблон читает их до того, как запрос мог их
	// переопределить (аналогично SprSrvVMPerfDay)
	this.Data["HostName"] = ""
	this.Data["HostNameFQDN"] = ""
	this.Data["ServerId"] = int64(0)
	this.Data["ChartsJSON"] = template.JS("[]")
	this.Data["PointCount"] = 0

	flash := beego.NewFlash()
	o := orm.NewOrm()

	if hostId <= 0 {
		flash.Error("Не указан хост Zabbix (ZabbixHostId)")
		flash.Store(&this.Controller)
		return
	}

	var host models.ZabbixHost
	if err := o.QueryTable("zabbix_host").Filter("Id", hostId).One(&host); err != nil {
		flash.Error("Хост Zabbix с Id=%d не найден: %v", hostId, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["HostName"] = host.Name
	this.Data["HostNameFQDN"] = host.NameFQDN
	this.Data["ServerId"] = host.ServerId

	var items []models.ZabbixItem
	if _, err := o.QueryTable("zabbix_item").
		Filter("HostId", hostId).
		Filter("Clock__gte", dateFrom).
		Filter("Clock__lt", dateTo).
		OrderBy("Clock").
		All(&items); err != nil {
		log.Println("SprSrvZabbixPerfPeriod: ошибка выборки zabbix_item:", err)
		flash.Error("Ошибка чтения данных Zabbix за период (zabbix_item): %v", err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["PointCount"] = len(items)
	if len(items) == 0 {
		return
	}

	// справочник ключей метрик (Key/Name/Units/ValueType), встретившихся в выборке
	keyIdSet := make(map[int64]struct{})
	for i := range items {
		keyIdSet[items[i].KeyId] = struct{}{}
	}
	keyIds := make([]int64, 0, len(keyIdSet))
	for id := range keyIdSet {
		keyIds = append(keyIds, id)
	}
	var keyList []models.ZabbixKey
	_, _ = o.QueryTable("zabbix_key").Filter("Id__in", keyIds).All(&keyList)
	keyById := make(map[int64]*models.ZabbixKey, len(keyList))
	for i := range keyList {
		keyById[keyList[i].Id] = &keyList[i]
	}

	// размер бакета подбирается так, чтобы на графике было не больше ~200 точек
	// на ряд независимо от длины периода - иначе за длинный период (недели/месяцы)
	// график из точных 10-минутных замеров стал бы нечитаемым и тяжёлым для браузера
	periodSeconds := dateTo.Sub(dateFrom).Seconds()
	const maxBuckets = 200
	bucketSeconds := periodSeconds / maxBuckets
	if bucketSeconds < 600 { // не мельче исходной гранулярности опроса Zabbix (10 минут)
		bucketSeconds = 600
	}
	bucketDur := time.Duration(bucketSeconds) * time.Second

	type bucketAgg struct {
		sum        float64
		min, max   float64
		count      int
		bucketTime time.Time
	}
	// aggByMetric[(KeyId, KeyParam)][bucketIndex] -> агрегат
	type metricKey struct {
		keyId    int64
		keyParam string
	}
	aggByMetric := make(map[metricKey]map[int64]*bucketAgg)
	metricOrder := make([]metricKey, 0)

	for i := range items {
		it := &items[i]
		key := keyById[it.KeyId]
		if key == nil {
			continue
		}
		// только числовые типы значений (0 - float, 3 - unsigned int); текст/лог/бинарные
		// метрики (1/2/4/5) на графике не отображаются
		if key.ValueType != 0 && key.ValueType != 3 {
			continue
		}
		v, err := strconv.ParseFloat(it.Value, 64)
		if err != nil {
			continue
		}
		// перевод единиц измерения к более читаемым: скорости в байтах/с
		// (Zabbix Units "Bps", сетевые интерфейсы и диски) - в мегабайты/с,
		// объёмы в байтах (Zabbix Units "B", размер оперативной памяти) - в
		// гигабайты; остальные единицы (%, мс, безразмерные) не трогаем -
		// см. displayUnit ниже, который определяет и множитель, и итоговую
		// подпись графика по исходной единице из models.ZabbixKey.Units.
		v = convertZabbixValue(v, key.Units)

		mk := metricKey{keyId: it.KeyId, keyParam: it.KeyParam}
		buckets, ok := aggByMetric[mk]
		if !ok {
			buckets = make(map[int64]*bucketAgg)
			aggByMetric[mk] = buckets
			metricOrder = append(metricOrder, mk)
		}

		idx := int64(it.Clock.Sub(dateFrom) / bucketDur)
		b, ok := buckets[idx]
		if !ok {
			b = &bucketAgg{min: v, max: v, bucketTime: dateFrom.Add(time.Duration(idx) * bucketDur)}
			buckets[idx] = b
		}
		if v < b.min {
			b.min = v
		}
		if v > b.max {
			b.max = v
		}
		b.sum += v
		b.count++
	}

	sort.Slice(metricOrder, func(i, j int) bool {
		ki, kj := keyById[metricOrder[i].keyId], keyById[metricOrder[j].keyId]
		if ki == nil || kj == nil {
			return false
		}
		if ki.Key != kj.Key {
			return ki.Key < kj.Key
		}
		return metricOrder[i].keyParam < metricOrder[j].keyParam
	})

	// группировка рядов по графикам - раньше один график вмещал сразу все
	// экземпляры метрики (все диски/интерфейсы вместе), из-за чего при
	// нескольких экземплярах график превращался в нечитаемое месиво из
	// десятка линий. Теперь график строится на каждую пару (ZabbixKey.Key,
	// KeyParam) отдельно - т.е. каждый диск/интерфейс получает свой график с
	// тремя рядами "мин"/"сред"/"макс", а не делит один график с остальными -
	// так плотность информации на каждом графике падает до читаемого уровня.
	chartByMetric := make(map[metricKey]*vmPerfChartData)
	chartOrder := make([]metricKey, 0)

	for _, mk := range metricOrder {
		key := keyById[mk.keyId]
		chart, ok := chartByMetric[mk]
		if !ok {
			title := key.Name
			if title == "" {
				title = key.Key
			}
			if mk.keyParam != "" {
				title = title + " — " + mk.keyParam
			}
			chart = &vmPerfChartData{Title: title, Unit: displayUnit(key.Units), Series: make(map[string][]vmPerfSeriesPoint)}
			chartByMetric[mk] = chart
			chartOrder = append(chartOrder, mk)
		}

		label := "Значение"
		if mk.keyParam != "" {
			label = mk.keyParam
		}
		minName := label + ", мин"
		avgName := label + ", сред"
		maxName := label + ", макс"

		buckets := aggByMetric[mk]
		idxs := make([]int64, 0, len(buckets))
		for idx := range buckets {
			idxs = append(idxs, idx)
		}
		sort.Slice(idxs, func(i, j int) bool { return idxs[i] < idxs[j] })

		minSeries := make([]vmPerfSeriesPoint, 0, len(idxs))
		avgSeries := make([]vmPerfSeriesPoint, 0, len(idxs))
		maxSeries := make([]vmPerfSeriesPoint, 0, len(idxs))
		for _, idx := range idxs {
			b := buckets[idx]
			t := b.bucketTime.UnixMilli()
			minSeries = append(minSeries, vmPerfSeriesPoint{T: t, V: b.min})
			avgSeries = append(avgSeries, vmPerfSeriesPoint{T: t, V: b.sum / float64(b.count)})
			maxSeries = append(maxSeries, vmPerfSeriesPoint{T: t, V: b.max})
		}
		chart.Series[minName] = minSeries
		chart.Series[avgName] = avgSeries
		chart.Series[maxName] = maxSeries
	}

	charts := make([]vmPerfChartData, 0, len(chartOrder))
	for _, mk := range chartOrder {
		charts = append(charts, *chartByMetric[mk])
	}

	b, jerr := json.Marshal(charts)
	if jerr != nil {
		flash.Error("Ошибка подготовки данных графиков: %v", jerr)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		return
	}
	this.Data["ChartsJSON"] = template.JS(b)
}

// normalizeZabbixUnits приводит единицу измерения Zabbix (models.ZabbixKey.Units)
// к нижнему регистру и без пробелов по краям для сравнения в convertZabbixValue/
// displayUnit - сама Zabbix API и этот код (см. zabbix/zabbixupdate.go:596-609,
// где для двух дисковых метрик units жёстко прописан как "Bytes/sec", а для
// остальных берётся "как есть" из ответа Zabbix - it.Units) не гарантируют
// единый регистр/написание единицы (встречаются и "Bps", и "bps", и
// "Bytes/sec") - без нормализации switch по точному совпадению строки просто
// не срабатывал, из-за чего конвертация ранее не применялась вовсе.
func normalizeZabbixUnits(units string) string {
	return strings.ToLower(strings.TrimSpace(units))
}

// convertZabbixValue переводит значение метрики Zabbix из исходной единицы в
// более удобную для восприятия: байты/с (сетевые интерфейсы net.if.in/out/
// total, дисковые vfs.dev.read/write - "Bps"/"bps"/"Bytes/sec"/"bytes/sec" в
// разном написании, встречающемся в данных) переводятся в мегабайты/с
// (/1024/1024), байты (объём памяти vm.memory.size[total]/[used] - "B"/"bytes")
// переводятся в гигабайты (/1024/1024/1024). Остальные единицы (%, мс,
// безразмерные величины вроде system.cpu.load) не трогаем - см. displayUnit,
// который отдаёт итоговую подпись, соответствующую этому же переводу.
func convertZabbixValue(v float64, units string) float64 {
	switch normalizeZabbixUnits(units) {
	case "bps", "b/s", "bytes/sec", "bytes/s":
		return v / (1024 * 1024)
	case "b", "bytes":
		return v / (1024 * 1024 * 1024)
	default:
		return v
	}
}

// displayUnit отдаёт подпись единицы измерения графика после перевода
// convertZabbixValue - должна вызываться с тем же исходным units.
func displayUnit(units string) string {
	switch normalizeZabbixUnits(units) {
	case "bps", "b/s", "bytes/sec", "bytes/s":
		return "МБ/с"
	case "b", "bytes":
		return "ГБ"
	default:
		return units
	}
}

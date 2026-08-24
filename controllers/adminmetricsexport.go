package controllers

import (
	"fmt"
	"loadmon/itsystem"
	"loadmon/models"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// detailedPeriodLimit - граница между "подробным" и "по дням" режимом
// единообразной выгрузки метрик (см. AdminMetricsUnifiedExport ниже).
// Совпадает со сроком хранения детальных метрик ВМ (vmPerfRetention,
// cmdb/savemetric_orchestration.go) не случайно: для периодов длиннее
// этого срока подробных данных по ВМ в базе уже физически нет (они
// вычищены ClearOldMetric), поэтому и Zabbix-часть выгрузки за такие
// периоды агрегируется по дням - чтобы формат обеих частей выгрузки был
// согласован по смыслу гранулярности, даже если для Zabbix "сырые" точки
// технически ещё могли бы быть доступны дольше.
const detailedPeriodLimit = 3 * 24 * time.Hour

// AdminMetricsUnifiedExport - единообразная выгрузка метрик утилизации ресурсов
// (CPU/память/диск/сеть) серверов за произвольный период, объединяющая два
// разнородных источника данных, уже используемых в проекте по отдельности:
//
//  1. Виртуальные машины (models.ServerSprRecords.Type == 1):
//     - "Подробный" режим (период ≤ detailedPeriodLimit = 3 суток) - из
//     models.VMPerfRecord (таблица vm_perf, детальные "сырые" замеры
//     vSphere по каждой ВМ). Запись в vm_perf включена по требованию
//     (cmdb/savevmrecord.go) только для серверов - рабочие станции
//     Windows (vm.WorkStation) пропускаются при записи и, соответственно,
//     не имеют детальных данных в выгрузке (только суточные, см. ниже).
//     Хранится vm_perf ограниченный срок - vmPerfRetention = 3 суток
//     (cmdb/savemetric_orchestration.go, ClearOldMetric).
//     - "По дням" режим (период > 3 суток) - из models.VMRecord (таблица
//     vm_record, одна агрегированная запись в сутки на ВМ, см.
//     cmdb.SaveVMRecord, вызывается раз в сутки в 23:59:59 по
//     metrics_grpc.go). Связь по ServerSprRecords.VMSprId ==
//     VMRecord.VMSprId (оба поля ссылаются на общий идентификатор в
//     vm_spr_record.Id); связь vm_perf с vm_record - VMPerfRecord.VMId ==
//     VMRecord.Id (детальные точки конкретного дня "подвешены" к суточной
//     записи этого же дня).
//  2. Аппаратные серверы, зарегистрированные в Zabbix
//     (models.ServerSprRecords.Type == 0 && ZabbixHostId > 0) - в обоих
//     режимах источник один - models.ZabbixItem (таблица zabbix_item, см.
//     zabbix.Get10MinutesHistory): каждые 10 минут пересобирается
//     10-минутное окно "сырой" истории (history.get) - то есть в отличие
//     от ВМ, поминутные данные по Zabbix-серверам физически доступны в БД
//     без отдельного срока хранения детализации. Классификация ключей на
//     CPU/память/диск/сеть - см. classifyZabbixKey ниже; перевод единиц
//     измерения - через уже существующие convertZabbixValue/displayUnit
//     (controllers/sprsrvzabbixperfperiod.go). В "по дням" режиме те же
//     "сырые" точки агрегируются (среднее) по календарным суткам -
//     отдельной таблицы суточных Zabbix-метрик в проекте нет
//     (models.ServerPerfZabbix существует, но нигде не заполняется).
//
// Оба источника после выбора режима сводятся к единой сетке через
// resampleToMinuteGrid (подробный режим - последнее известное значение
// "продлевается" на равномерную минутную сетку, step/forward-fill) либо
// aggregateToDailyGrid (режим "по дням" - среднее за календарные сутки).
// Крупные "ступеньки" в подробном режиме для ВМ с редкими вставками в
// vm_perf (например, если в течение суток сохранился только один снимок
// vm.Perf) - ожидаемое отражение реальной гранулярности исходных данных, а
// не ошибка.
//
// Формат выгрузки - "длинная" (long/tidy) таблица: одна строка на
// (объект, метрика, точка времени), а не широкая таблица со столбцом на
// каждую метрику - так проще единообразно объединить ВМ и Zabbix-серверы, у
// которых набор доступных метрик отличается (например, не у каждого
// Zabbix-хоста есть метрика по каждому диску/интерфейсу), и объекты с
// разной гранулярностью в одном и том же файле выгрузки (см. столбец
// Granularity/"Гранулярность").
//
// Маршрут: /admin/metricsexport, доступ только для models.AuthUser.Priv == 3
// (администратор), по аналогии с /admin/userlog.
type unifiedMetricRow struct {
	ObjectType  string // "Виртуальная машина" / "Аппаратный сервер (Zabbix)"
	ObjectName  string
	ITSystem    string
	Metric      string // "CPU, %" / "Память, %" / "Диск, %" / "Сеть, МБ/с" и т.п.
	Instance    string // уточнение экземпляра метрики (имя диска/интерфейса) - пусто для ВМ (сводные метрики) и для CPU/памяти сервера
	Granularity string // "1 минута" (подробный режим) / "1 день" (режим "по дням")
	Unit        string
	Timestamp   time.Time
	Value       float64
}

// unifiedMetricPoint - одна "сырая" точка временного ряда одной метрики
// одного объекта, до ресемплинга на минутную сетку.
type unifiedMetricPoint struct {
	Timestamp time.Time
	Value     float64
}

// resampleToMinuteGrid строит равномерную сетку меток времени с шагом
// stepMinutes (обычно 1 минута) от from до to включительно и для каждой
// точки сетки берёт последнее известное значение (step/forward-fill) из
// отсортированных по времени "сырых" точек points. Если к моменту первой
// точки сетки ещё нет ни одного замера (period начинается раньше первого
// реального значения), эти начальные точки сетки пропускаются (не
// заполняются нулём/интерполяцией назад, чтобы не фабриковать данные).
// Точки sparse-источника (например, ВМ с интервалом опроса 20 минут) дают в
// выгрузке характерные "ступеньки" - это ожидаемое и корректное отражение
// реальной гранулярности исходных данных, см. комментарий выше у
// AdminMetricsUnifiedExport.
func resampleToMinuteGrid(points []unifiedMetricPoint, from, to time.Time, stepMinutes int) []unifiedMetricPoint {
	if len(points) == 0 || stepMinutes <= 0 {
		return nil
	}
	sorted := make([]unifiedMetricPoint, len(points))
	copy(sorted, points)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Timestamp.Before(sorted[j].Timestamp) })

	step := time.Duration(stepMinutes) * time.Minute
	result := make([]unifiedMetricPoint, 0, int(to.Sub(from)/step)+1)

	idx := 0
	haveValue := false
	var lastValue float64
	for t := from; !t.After(to); t = t.Add(step) {
		for idx < len(sorted) && !sorted[idx].Timestamp.After(t) {
			lastValue = sorted[idx].Value
			haveValue = true
			idx++
		}
		if !haveValue {
			continue // до первого реального замера ещё не заполняем сетку
		}
		result = append(result, unifiedMetricPoint{Timestamp: t, Value: lastValue})
	}
	return result
}

// aggregateToDailyGrid группирует "сырые" точки points по календарным
// суткам (по времени точки, локальная таймзона) и для каждых суток отдаёт
// одну точку - среднее значение за сутки, с меткой времени в 00:00 этих
// суток. Используется для режима выгрузки "по дням" (период больше
// detailedPeriodLimit) - в отличие от resampleToMinuteGrid, здесь не
// применяется forward-fill: сутки, за которые вообще не было ни одной
// "сырой" точки, в результате отсутствуют (не заполняются предыдущим
// значением), чтобы не создавать видимость данных там, где их нет. Для
// models.VMRecord (уже готовые суточные записи) агрегация вырождается в
// тождественное преобразование "один вход - одна точка на выходе", что и
// требуется - функция используется единообразно для обоих источников.
func aggregateToDailyGrid(points []unifiedMetricPoint) []unifiedMetricPoint {
	if len(points) == 0 {
		return nil
	}
	type bucket struct {
		day time.Time
		sum float64
		n   int
	}
	buckets := make(map[string]*bucket, len(points))
	order := make([]string, 0, len(points))
	for _, p := range points {
		day := time.Date(p.Timestamp.Year(), p.Timestamp.Month(), p.Timestamp.Day(), 0, 0, 0, 0, p.Timestamp.Location())
		key := day.Format("2006-01-02")
		b, ok := buckets[key]
		if !ok {
			b = &bucket{day: day}
			buckets[key] = b
			order = append(order, key)
		}
		b.sum += p.Value
		b.n++
	}
	sort.Strings(order)
	result := make([]unifiedMetricPoint, 0, len(order))
	for _, key := range order {
		b := buckets[key]
		result = append(result, unifiedMetricPoint{Timestamp: b.day, Value: b.sum / float64(b.n)})
	}
	return result
}

// classifyZabbixKey определяет, к какой единообразной метрике относится
// "сырой" ключ Zabbix (models.ZabbixKey.Key) - по тем же паттернам ключей,
// которые реально используются сборщиком при заполнении zabbix_item (см.
// zabbix/zabbixupdate.go: GetItemsFilterByHost/GetItemidByHost). Возвращает
// пустую строку, если ключ не относится ни к одной из четырёх базовых
// метрик утилизации (CPU/память/диск/сеть) - такие ключи в единообразную
// выгрузку не включаются.
func classifyZabbixKey(key string) string {
	k := strings.ToLower(key)
	switch {
	case strings.HasPrefix(k, "system.cpu."):
		return "CPU"
	case strings.HasPrefix(k, "vm.memory."):
		return "Память"
	case strings.HasPrefix(k, "vfs.dev.") || strings.HasPrefix(k, "vfs.fs.") || strings.Contains(k, "logicaldisk"):
		return "Диск"
	case strings.HasPrefix(k, "net.if."):
		return "Сеть"
	default:
		return ""
	}
}

// AdminMetricsUnifiedExport отображает форму выбора периода/фильтра и (при
// export=xlsx) формирует единообразную выгрузку метрик утилизации по ВМ и
// зарегистрированным в Zabbix аппаратным серверам. Период ≤
// detailedPeriodLimit (3 суток) выгружается подробно (минутная сетка),
// более длинный период - по дням (среднее за календарные сутки) - см.
// комментарий к типу unifiedMetricRow и detailedPeriodLimit выше.
//
// Доступ - только models.AuthUser.Priv == 3, приём идентичен ViewUserLog
// (controllers/userlog.go): при недостаточных правах - редирект на
// справочник серверов без ошибки.
func (this *MainController) AdminMetricsUnifiedExport() {

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/admin_metricsexport", 302)
		return
	}
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/metricsexport")

	to := CorrectTimeFilter(time.Now())
	from := to.AddDate(0, 0, -1) // по умолчанию - последние сутки, единообразная минутная выгрузка за более длинный период быстро становится очень большой
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
	dateTo := to.AddDate(0, 0, 1) // верхняя граница периода включительна по дате
	detailed := dateTo.Sub(from) <= detailedPeriodLimit

	this.Data["FromDate"] = from.Format("2006-01-02")
	this.Data["ToDate"] = to.Format("2006-01-02")
	this.Data["Detailed"] = detailed
	this.Data["DetailedPeriodLimitDays"] = int(detailedPeriodLimit.Hours() / 24)

	nameFilter := strings.TrimSpace(this.GetString("name"))
	itSystemFilter := strings.TrimSpace(this.GetString("itsystem"))
	includeVM := this.GetString("includevm") != "off"     // чекбоксы по умолчанию включены (см. шаблон - "off" передаётся только явно снятым чекбоксом)
	includeZabbix := this.GetString("includezabbix") != "off"
	this.Data["CurName"] = nameFilter
	this.Data["CurITSystem"] = itSystemFilter
	this.Data["IncludeVM"] = includeVM
	this.Data["IncludeZabbix"] = includeZabbix

	// Справочник ИТ-систем для поля с поиском по коду/названию (тот же
	// клиентский виджет, что и на /sprsrv/view2 и /sprsrv/updatebase - см.
	// initITSystemPicker() в шаблоне, kontext/04-web-ui.md).
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()

	o := orm.NewOrm()

	// ⚠️ Ранее здесь стоял qs.Filter("i_t_system_key", itSystemFilter) -
	// beego ORM Filter() принимает имя ПОЛЯ структуры Go (как объявлено в
	// models.ServerSprRecords - "ITSystemKey"), а не имя колонки БД в
	// snake_case. Несуществующее имя поля приводило к ошибке ORM при
	// каждом запросе с непустым фильтром по ИТ-системе (qs.All() возвращал
	// err != nil) - именно из-за этого выгрузка не работала.
	qs := o.QueryTable("server_spr_records").Filter("Active", 1)
	if nameFilter != "" {
		qs = qs.Filter("Name__icontains", nameFilter)
	}
	if itSystemFilter != "" {
		qs = qs.Filter("ITSystemKey", itSystemFilter)
	}
	var candidates []*models.ServerSprRecords
	_, err := qs.All(&candidates)
	if err != nil {
		flash := beego.NewFlash()
		flash.Error("Ошибка чтения справочника серверов: " + err.Error())
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	rows := make([]unifiedMetricRow, 0, 1024)
	for _, srv := range candidates {
		if srv == nil {
			continue
		}
		switch {
		case srv.Type == 1 && includeVM:
			rows = append(rows, buildVMMetricRows(o, srv, from, dateTo, detailed)...)
		case srv.Type == 0 && srv.ZabbixHostId > 0 && includeZabbix:
			rows = append(rows, buildZabbixMetricRows(o, srv, from, dateTo, detailed)...)
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ObjectName != rows[j].ObjectName {
			return rows[i].ObjectName < rows[j].ObjectName
		}
		if rows[i].Metric != rows[j].Metric {
			return rows[i].Metric < rows[j].Metric
		}
		if rows[i].Instance != rows[j].Instance {
			return rows[i].Instance < rows[j].Instance
		}
		return rows[i].Timestamp.Before(rows[j].Timestamp)
	})

	if this.GetString("export") == "xlsx" {
		// Экспорт - по всей отфильтрованной выборке, не по текущей странице
		// просмотра (см. kontext/06-list-page-pattern.md, п.5).
		RegisterRequest(this, LogTypeExport) // выгрузка единообразных метрик утилизации (администрирование)
		flash := beego.NewFlash()
		exportUnifiedMetricsXLSX(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр формы/таблицы выгрузки метрик (администрирование)

	// Постраничный просмотр на экране (kontext/06-list-page-pattern.md, п.4) -
	// пагинация только для отображения, сама выборка rows строится по всему
	// отфильтрованному набору (нужна целиком и для подсчёта количества
	// страниц, и для XLSX-выгрузки выше).
	pageSize, _ := strconv.Atoi(this.GetString("pagesize"))
	switch pageSize {
	case 25, 50, 100, 200:
	default:
		pageSize = 50
	}
	page, _ := strconv.Atoi(this.GetString("page"))
	if page < 1 {
		page = 1
	}
	totalRecords := len(rows)
	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalRecords {
		end = totalRecords
	}
	pageRows := make([]unifiedMetricRow, 0)
	if offset < totalRecords {
		pageRows = rows[offset:end]
	}

	this.Data["Rows"] = pageRows
	this.Data["RowCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	if totalRecords == 0 {
		this.Data["PageFrom"] = 0
		this.Data["PageTo"] = 0
	} else {
		this.Data["PageFrom"] = offset + 1
		this.Data["PageTo"] = end
	}
}

// buildVMMetricRows строит точки CPU/память/диск для одной ВМ.
//
//   - detailed == true (период ≤ detailedPeriodLimit): источник -
//     models.VMPerfRecord (таблица vm_perf, детальные "сырые" замеры,
//     включённые в cmdb/savevmrecord.go только для серверов - у рабочих
//     станций Windows vm_perf для этого периода не будет вовсе). Связь не
//     напрямую по VMSprId - VMPerfRecord.VMId ссылается на Id суточной
//     записи models.VMRecord (vm_record), поэтому сначала находим все
//     vm_record этой ВМ за период, затем vm_perf, "подвешенные" на их Id.
//     Результат ресемплируется на минутную сетку (resampleToMinuteGrid).
//   - detailed == false (период длиннее): источник - models.VMRecord
//     напрямую (уже одна агрегированная запись в сутки), точки сводятся
//     через aggregateToDailyGrid (для vm_record это фактически
//     тождественное преобразование - агрегировать за сутки уже почти
//     нечего, но так оба режима проходят единый код построения строк).
func buildVMMetricRows(o orm.Ormer, srv *models.ServerSprRecords, from, dateTo time.Time, detailed bool) []unifiedMetricRow {
	if srv.VMSprId <= 0 {
		return nil
	}
	var recs []*models.VMRecord
	_, err := o.QueryTable("vm_record").
		Filter("VMSprId", srv.VMSprId).
		Filter("Date__gte", from).
		Filter("Date__lt", dateTo).
		OrderBy("Date").
		All(&recs)
	if err != nil || len(recs) == 0 {
		return nil
	}

	granularity := "1 день"
	var cpuPoints, memPoints, diskPoints []unifiedMetricPoint

	if detailed {
		granularity = "1 минута"
		recIds := make([]int64, 0, len(recs))
		for _, r := range recs {
			if r != nil {
				recIds = append(recIds, r.Id)
			}
		}
		var perfRecs []*models.VMPerfRecord
		_, err := o.QueryTable("vm_perf").Filter("VMId__in", recIds).OrderBy("Start").All(&perfRecs)
		if err != nil || len(perfRecs) == 0 {
			return nil // за этот период нет детальных данных (например, ВМ - рабочая станция Windows, для которой vm_perf не пишется)
		}
		cpuPoints = make([]unifiedMetricPoint, 0, len(perfRecs))
		memPoints = make([]unifiedMetricPoint, 0, len(perfRecs))
		diskPoints = make([]unifiedMetricPoint, 0, len(perfRecs))
		for _, p := range perfRecs {
			if p == nil {
				continue
			}
			cpuPoints = append(cpuPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Cpu_usage})
			memPoints = append(memPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Mem_usage})
			diskPoints = append(diskPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Disk_usage})
		}
		cpuPoints = resampleToMinuteGrid(cpuPoints, from, dateTo.Add(-time.Minute), 1)
		memPoints = resampleToMinuteGrid(memPoints, from, dateTo.Add(-time.Minute), 1)
		diskPoints = resampleToMinuteGrid(diskPoints, from, dateTo.Add(-time.Minute), 1)
	} else {
		cpuPoints = make([]unifiedMetricPoint, 0, len(recs))
		memPoints = make([]unifiedMetricPoint, 0, len(recs))
		diskPoints = make([]unifiedMetricPoint, 0, len(recs))
		for _, r := range recs {
			if r == nil {
				continue
			}
			cpuPoints = append(cpuPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Cpu_usage_average})
			memPoints = append(memPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Mem_usage_average})
			diskPoints = append(diskPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Disk_usage_average})
		}
		cpuPoints = aggregateToDailyGrid(cpuPoints)
		memPoints = aggregateToDailyGrid(memPoints)
		diskPoints = aggregateToDailyGrid(diskPoints)
	}

	rows := make([]unifiedMetricRow, 0, (len(cpuPoints)+len(memPoints)+len(diskPoints)))
	rows = append(rows, toUnifiedRows("Виртуальная машина", srv.Name, srv.ITSystemKey, "CPU, %", "", granularity, "%", cpuPoints)...)
	rows = append(rows, toUnifiedRows("Виртуальная машина", srv.Name, srv.ITSystemKey, "Память, %", "", granularity, "%", memPoints)...)
	rows = append(rows, toUnifiedRows("Виртуальная машина", srv.Name, srv.ITSystemKey, "Диск, %", "", granularity, "%", diskPoints)...)
	return rows
}

// buildZabbixMetricRows строит "сырые" точки CPU/память/диск/сеть по данным
// models.ZabbixItem для сервера, зарегистрированного в Zabbix
// (srv.ZabbixHostId - ссылка на models.ZabbixHost.Id), классифицирует ключи
// через classifyZabbixKey, переводит единицы измерения через уже
// существующие convertZabbixValue/displayUnit (см.
// controllers/sprsrvzabbixperfperiod.go) и сводит каждый экземпляр метрики
// (KeyParam - например, имя диска или сетевого интерфейса) отдельно либо на
// минутную сетку (detailed == true, resampleToMinuteGrid), либо по суткам
// (detailed == false, aggregateToDailyGrid - отдельной суточной таблицы
// Zabbix-метрик в проекте нет, агрегируем "сырые" точки на лету).
func buildZabbixMetricRows(o orm.Ormer, srv *models.ServerSprRecords, from, dateTo time.Time, detailed bool) []unifiedMetricRow {
	var items []*models.ZabbixItem
	_, err := o.QueryTable("zabbix_item").
		Filter("HostId", srv.ZabbixHostId).
		Filter("Clock__gte", from).
		Filter("Clock__lt", dateTo).
		OrderBy("Clock").
		All(&items)
	if err != nil || len(items) == 0 {
		return nil
	}

	keyIds := make(map[int64]struct{})
	for _, it := range items {
		if it != nil {
			keyIds[it.KeyId] = struct{}{}
		}
	}
	keys := make(map[int64]*models.ZabbixKey, len(keyIds))
	for id := range keyIds {
		var k models.ZabbixKey
		k.Id = id
		if err := o.Read(&k); err == nil {
			keys[id] = &k
		}
	}

	// группируем точки по (метрика, экземпляр=KeyParam)
	type groupKey struct {
		metric   string
		instance string
	}
	groups := make(map[groupKey][]unifiedMetricPoint)
	units := make(map[groupKey]string)
	for _, it := range items {
		if it == nil {
			continue
		}
		key, ok := keys[it.KeyId]
		if !ok {
			continue
		}
		metric := classifyZabbixKey(key.Key)
		if metric == "" {
			continue // ключ не относится к CPU/памяти/диску/сети - в единообразную выгрузку не включаем
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(it.Value), 64)
		if err != nil {
			continue
		}
		v = convertZabbixValue(v, key.Units)
		gk := groupKey{metric: metric, instance: it.KeyParam}
		groups[gk] = append(groups[gk], unifiedMetricPoint{Timestamp: it.Clock, Value: v})
		units[gk] = displayUnit(key.Units)
	}

	granularity := "1 день"
	if detailed {
		granularity = "1 минута"
	}

	rows := make([]unifiedMetricRow, 0, len(items))
	for gk, points := range groups {
		metricLabel := gk.metric
		if units[gk] != "" && units[gk] != "%" {
			metricLabel = fmt.Sprintf("%s, %s", gk.metric, units[gk])
		} else {
			metricLabel = fmt.Sprintf("%s, %%", gk.metric)
		}
		var resolved []unifiedMetricPoint
		if detailed {
			resolved = resampleToMinuteGrid(points, from, dateTo.Add(-time.Minute), 1)
		} else {
			resolved = aggregateToDailyGrid(points)
		}
		rows = append(rows, toUnifiedRows("Аппаратный сервер (Zabbix)", srv.Name, srv.ITSystemKey, metricLabel, gk.instance, granularity, units[gk], resolved)...)
	}
	return rows
}

// toUnifiedRows разворачивает срез точек ресемплированного временного ряда
// в строки единообразной выгрузки (unifiedMetricRow) для одного объекта и
// одной метрики/экземпляра.
func toUnifiedRows(objectType, objectName, itSystem, metric, instance, granularity, unit string, points []unifiedMetricPoint) []unifiedMetricRow {
	rows := make([]unifiedMetricRow, 0, len(points))
	for _, p := range points {
		rows = append(rows, unifiedMetricRow{
			ObjectType:  objectType,
			ObjectName:  objectName,
			ITSystem:    itSystem,
			Metric:      metric,
			Instance:    instance,
			Granularity: granularity,
			Unit:        unit,
			Timestamp:   p.Timestamp,
			Value:       p.Value,
		})
	}
	return rows
}

// unifiedMetricsXLSXHeader - заголовок листа единообразной выгрузки метрик.
var unifiedMetricsXLSXHeader = []string{
	"Тип объекта", "Имя объекта", "ИТ система", "Метрика", "Экземпляр (диск/интерфейс)", "Гранулярность", "Ед. изм.", "Время", "Значение",
}

// exportUnifiedMetricsXLSX отдаёт единообразную выгрузку метрик утилизации
// файлом XLSX на скачивание - формирование листа аналогично
// ExportVMListXLSX (controllers/sprsrvvm.go) и экспорту userlog.
func exportUnifiedMetricsXLSX(this *MainController, flash *beego.FlashData, rows []unifiedMetricRow) {
	if len(rows) == 0 {
		flash.Notice("Нет данных для выгрузки за выбранный период и фильтр.")
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("metrics_export_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	sheet := "Метрики"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/metricsexport", 302)
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &unifiedMetricsXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	for j, r := range rows {
		row := []string{
			r.ObjectType,
			r.ObjectName,
			r.ITSystem,
			r.Metric,
			r.Instance,
			r.Granularity,
			r.Unit,
			r.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.4f", r.Value),
		}
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" ", err)
			flash.Store(&this.Controller)
			file.Close()
			this.Redirect("/admin/metricsexport", 302)
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/metricsexport", 302)
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	this.Ctx.Output.Header("Content-Disposition", "attachment; filename=\""+shortName+"\"")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")
	this.Ctx.Output.Header("Content-Length", fmt.Sprint(fileInfo.Size()))

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}

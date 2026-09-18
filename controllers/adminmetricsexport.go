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
//     (models.ServerSprRecords.Type == 0 && ZabbixHostId > 0) - источник
//     переведён с "сырых" данных Zabbix (models.ZabbixItem/ZabbixKey) на
//     уже готовые метрики пакета zabbixmetrics (см. kontext/03-data-model.md):
//     - "Подробный" режим (период ≤ detailedPeriodLimit) - models.ServerMetric
//     (таблица server_metric, поминутные снимки CPU/RAM/диск/сеть по
//     HostId == models.ZabbixHost.Id, заполняется пакетом zabbixmetrics
//     independently от старого сборщика zabbix.Get10MinutesHistory).
//     - "По дням" режим - models.ServerMetricRecord (таблица
//     server_metric_record, суточная агрегация ServerMetric, строится
//     cmdb.SaveServerMetricRecord - см. cmdb/saveservermetricrecord.go),
//     поля *Average.
//     В обоих случаях метрики уже сведены к одному числу на хост
//     (zabbixmetrics сам агрегирует по хосту при сборе), поэтому в отличие
//     от старой ZabbixItem-реализации здесь не нужны ни классификация
//     "сырых" ключей (system.cpu.load/vfs.dev.read[...] и т.п.), ни свод
//     нескольких дисков/интерфейсов по максимуму (mergeInstanceSeries) - на
//     входе уже одна точка на момент времени на каждую из шести колонок.
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
// Формат выгрузки - "широкая" таблица: одна строка на (объект, точка
// времени) с отдельным столбцом на каждую метрику (Время, ИТ система, Имя
// сервера, CPU %, RAM %, Диск чтение МБ/с, Диск запись МБ/с, Сеть приём
// МБ/с, Сеть передача МБ/с) - удобно смотреть и сортировать/фильтровать как
// обычную таблицу, а не разбираться в длинном формате "объект/метрика/
// значение". Для объектов с несколькими экземплярами метрики (у
// Zabbix-сервера может быть несколько дисков и несколько сетевых
// интерфейсов) диск и сеть сводятся в единственное число на колонку по
// МАКСИМУМУ среди экземпляров на каждый момент времени (какой диск/интерфейс
// нагружен сильнее всего в этот момент) - см. mergeInstanceSeries ниже.
// Для CPU/памяти экземпляр обычно один (не считая ВМ, где эти метрики и так
// уже сводные по всей ВМ), поэтому там используется среднее (на случай
// несовпадения экземпляров, вырождается в то же единственное значение).
//
// Если у объекта в конкретный момент времени нет данных по какой-то
// метрике (например, у ВМ в режиме "по дням" нет раздельных значений приёма/
// передачи сети - vm_record хранит только объединённый Net_usage_average),
// соответствующая ячейка остаётся пустой, а не заполняется нулём или
// оценочным числом - см. комментарий в buildVMWideRows.
//
// Маршрут: /admin/metricsexport, доступ только для models.AuthUser.Priv == 3
// (администратор), по аналогии с /admin/userlog.
type wideMetricRow struct {
	Timestamp    time.Time
	ITSystem     string
	ObjectName   string
	CPUPercent   string // "%" - пусто, если данных нет
	MemPercent   string // "%"
	DiskReadMBs  string // МБ/с, максимум по дискам на этот момент времени
	DiskWriteMBs string // МБ/с, максимум по дискам на этот момент времени
	NetRecvMBs   string // МБ/с, максимум по интерфейсам на этот момент времени
	NetTransMBs  string // МБ/с, максимум по интерфейсам на этот момент времени
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

// combineIntoWideRows сводит по одному временному ряду на каждую из шести
// метрик одного объекта в строки широкой таблицы (wideMetricRow) - по одной
// строке на каждый момент времени, встретившийся хотя бы в одном из шести
// рядов (объединение меток времени, а не пересечение: ряды могут не
// совпадать по набору точек - например, у ВМ в режиме "по дням" сетевые ряды
// пусты вовсе, см. buildVMWideRows). Ячейки метрик, отсутствующих в конкретный
// момент времени у конкретного объекта, остаются пустой строкой.
func combineIntoWideRows(objectName, itSystem string, cpu, mem, diskRead, diskWrite, netRecv, netTrans []unifiedMetricPoint) []wideMetricRow {
	type cell struct {
		cpu, mem, dr, dw, nr, nt *float64
	}
	cells := make(map[int64]*cell)
	order := make([]int64, 0)

	getCell := func(t time.Time) *cell {
		key := t.UnixNano()
		c, ok := cells[key]
		if !ok {
			c = &cell{}
			cells[key] = c
			order = append(order, key)
		}
		return c
	}
	for _, p := range cpu {
		v := p.Value
		getCell(p.Timestamp).cpu = &v
	}
	for _, p := range mem {
		v := p.Value
		getCell(p.Timestamp).mem = &v
	}
	for _, p := range diskRead {
		v := p.Value
		getCell(p.Timestamp).dr = &v
	}
	for _, p := range diskWrite {
		v := p.Value
		getCell(p.Timestamp).dw = &v
	}
	for _, p := range netRecv {
		v := p.Value
		getCell(p.Timestamp).nr = &v
	}
	for _, p := range netTrans {
		v := p.Value
		getCell(p.Timestamp).nt = &v
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	fmtVal := func(v *float64) string {
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%.2f", *v)
	}

	rows := make([]wideMetricRow, 0, len(order))
	for _, key := range order {
		c := cells[key]
		rows = append(rows, wideMetricRow{
			Timestamp:    time.Unix(0, key),
			ITSystem:     itSystem,
			ObjectName:   objectName,
			CPUPercent:   fmtVal(c.cpu),
			MemPercent:   fmtVal(c.mem),
			DiskReadMBs:  fmtVal(c.dr),
			DiskWriteMBs: fmtVal(c.dw),
			NetRecvMBs:   fmtVal(c.nr),
			NetTransMBs:  fmtVal(c.nt),
		})
	}
	return rows
}

// AdminMetricsUnifiedExport отображает форму выбора периода/фильтра и (при
// export=xlsx) формирует единообразную выгрузку метрик утилизации по ВМ и
// зарегистрированным в Zabbix аппаратным серверам. Период ≤
// detailedPeriodLimit (3 суток) выгружается подробно (минутная сетка),
// более длинный период - по дням (среднее за календарные сутки) - см.
// комментарий к типу wideMetricRow и detailedPeriodLimit выше.
//
// Доступ - только models.AuthUser.Priv == 3, приём идентичен ViewUserLog
// (controllers/userlog.go): при недостаточных правах - редирект на
// справочник серверов без ошибки.
func (this *MainController) AdminMetricsUnifiedExport() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
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
	// initITSystemPicker() в шаблоне, kontext/04-web-ui.md). Это данные для
	// самой формы фильтра (выпадающий список), а не строящаяся таблица -
	// грузятся всегда, независимо от applied ниже.
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()

	// Загрузка и построение таблицы (запрос справочника серверов + сбор
	// widerows по каждому источнику) выполняется ТОЛЬКО после явного
	// применения фильтра пользователем ("Показать"/"Сформировать выгрузку"),
	// а не при первом открытии страницы или при каждом отображении формы -
	// см. шаблон admin/metricsexport.tpl:metricsExportSubmit(), которая
	// всегда добавляет скрытое поле applied=1 при отправке формы. Прямое
	// открытие /admin/metricsexport без параметров (или после "Сбросить
	// фильтр") не бьёт по БД вовсе.
	exportRequested := this.GetString("export") == "xlsx"
	applied := this.GetString("applied") == "1" || exportRequested
	this.Data["FilterApplied"] = applied

	rows := make([]wideMetricRow, 0, 1024)
	if applied {
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
			flash.Error("Ошибка чтения справочника серверов: %v", err)
			flash.Store(&this.Controller)
			this.Redirect("/admin/metricsexport", 302)
			return
		}

		for _, srv := range candidates {
			if srv == nil {
				continue
			}
			switch {
			case srv.Type == 1 && includeVM:
				rows = append(rows, buildVMWideRows(o, srv, from, dateTo, detailed)...)
			case srv.Type == 0 && srv.ZabbixHostId > 0 && includeZabbix:
				rows = append(rows, buildServerMetricWideRows(o, srv, from, dateTo, detailed)...)
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].Timestamp.Equal(rows[j].Timestamp) {
			return rows[i].Timestamp.Before(rows[j].Timestamp)
		}
		return rows[i].ObjectName < rows[j].ObjectName
	})

	if exportRequested {
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
	pageRows := make([]wideMetricRow, 0)
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

// buildVMWideRows строит широкие строки (CPU/RAM/диск чтение-запись/сеть
// приём-передача) для одной ВМ.
//
//   - detailed == true (период ≤ detailedPeriodLimit): источник -
//     models.VMPerfRecord (таблица vm_perf, детальные "сырые" замеры,
//     включённые в cmdb/savevmrecord.go только для серверов - у рабочих
//     станций Windows vm_perf для этого периода не будет вовсе). Связь не
//     напрямую по VMSprId - VMPerfRecord.VMId ссылается на Id суточной
//     записи models.VMRecord (vm_record), поэтому сначала находим все
//     vm_record этой ВМ за период, затем vm_perf, "подвешенные" на их Id.
//     Disk_read/Disk_write/Net_received/Net_transmitted в vm_perf уже
//     сводные по всей ВМ (не по отдельным виртуальным дискам/сетевым
//     картам) и хранятся в KBps (vSphere performance counters) - переводим
//     в МБ/с делением на 1024; "максимум по дискам/интерфейсам" здесь не
//     нужен - vSphere сама отдаёт для ВМ уже объединённое значение.
//   - detailed == false (период длиннее): источник - models.VMRecord
//     напрямую. Disk_read_average/Disk_write_average также в KBps - та же
//     конверсия. Раздельных Net_received/Net_transmitted в суточном
//     агрегате НЕТ (только объединённый Net_usage_average) - колонки "Сеть
//     приём"/"Сеть передача" в этом режиме для ВМ остаются пустыми, чтобы не
//     фабриковать разбивку, которой в источнике нет (см. комментарий к
//     wideMetricRow).
func buildVMWideRows(o orm.Ormer, srv *models.ServerSprRecords, from, dateTo time.Time, detailed bool) []wideMetricRow {
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

	const kbpsToMBps = 1024.0 // KBps (vSphere) -> МБ/с

	var cpuPoints, memPoints, diskReadPoints, diskWritePoints, netRecvPoints, netTransPoints []unifiedMetricPoint

	if detailed {
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
		for _, p := range perfRecs {
			if p == nil {
				continue
			}
			cpuPoints = append(cpuPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Cpu_usage})
			memPoints = append(memPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Mem_usage})
			diskReadPoints = append(diskReadPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Disk_read / kbpsToMBps})
			diskWritePoints = append(diskWritePoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Disk_write / kbpsToMBps})
			netRecvPoints = append(netRecvPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Net_received / kbpsToMBps})
			netTransPoints = append(netTransPoints, unifiedMetricPoint{Timestamp: p.Start, Value: p.Net_transmitted / kbpsToMBps})
		}
		cpuPoints = resampleToMinuteGrid(cpuPoints, from, dateTo.Add(-time.Minute), 1)
		memPoints = resampleToMinuteGrid(memPoints, from, dateTo.Add(-time.Minute), 1)
		diskReadPoints = resampleToMinuteGrid(diskReadPoints, from, dateTo.Add(-time.Minute), 1)
		diskWritePoints = resampleToMinuteGrid(diskWritePoints, from, dateTo.Add(-time.Minute), 1)
		netRecvPoints = resampleToMinuteGrid(netRecvPoints, from, dateTo.Add(-time.Minute), 1)
		netTransPoints = resampleToMinuteGrid(netTransPoints, from, dateTo.Add(-time.Minute), 1)
	} else {
		for _, r := range recs {
			if r == nil {
				continue
			}
			cpuPoints = append(cpuPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Cpu_usage_average})
			memPoints = append(memPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Mem_usage_average})
			diskReadPoints = append(diskReadPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Disk_read_average / kbpsToMBps})
			diskWritePoints = append(diskWritePoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.Disk_write_average / kbpsToMBps})
			// Net_received/Net_transmitted раздельно в vm_record не хранятся -
			// оставляем netRecvPoints/netTransPoints пустыми (см. комментарий
			// к функции выше).
		}
		cpuPoints = aggregateToDailyGrid(cpuPoints)
		memPoints = aggregateToDailyGrid(memPoints)
		diskReadPoints = aggregateToDailyGrid(diskReadPoints)
		diskWritePoints = aggregateToDailyGrid(diskWritePoints)
	}

	return combineIntoWideRows(srv.Name, srv.ITSystemKey, cpuPoints, memPoints, diskReadPoints, diskWritePoints, netRecvPoints, netTransPoints)
}

// buildServerMetricWideRows строит широкие строки (CPU/RAM/диск чтение-
// запись/сеть приём-передача) для сервера, зарегистрированного в Zabbix
// (srv.ZabbixHostId - ссылка на models.ZabbixHost.Id), из уже готовых
// метрик пакета zabbixmetrics - НЕ из "сырых" models.ZabbixItem/ZabbixKey
// (см. комментарий к AdminMetricsUnifiedExport выше):
//
//   - detailed == true (период ≤ detailedPeriodLimit): источник -
//     models.ServerMetric (таблица server_metric, поминутные снимки,
//     HostId == srv.ZabbixHostId). Метрики уже сведены к одному числу на
//     хост (агрегация по дискам/интерфейсам сделана на стороне сборщика
//     zabbixmetrics, см. kontext/03-data-model.md), поэтому в отличие от
//     старой ZabbixItem-реализации здесь не нужен свод по максимуму
//     нескольких экземпляров - сразу один ряд на метрику.
//   - detailed == false (период длиннее): источник - models.ServerMetricRecord
//     (таблица server_metric_record, суточная агрегация, строится
//     cmdb.SaveServerMetricRecord) - поля *Average, одна точка на сутки.
func buildServerMetricWideRows(o orm.Ormer, srv *models.ServerSprRecords, from, dateTo time.Time, detailed bool) []wideMetricRow {
	if srv.ZabbixHostId <= 0 {
		return nil
	}

	var cpuPoints, memPoints, diskReadPoints, diskWritePoints, netRecvPoints, netTransPoints []unifiedMetricPoint

	if detailed {
		var samples []*models.ServerMetric
		_, err := o.QueryTable(new(models.ServerMetric)).
			Filter("HostId", srv.ZabbixHostId).
			Filter("CollectedAt__gte", from).
			Filter("CollectedAt__lt", dateTo).
			OrderBy("CollectedAt").
			All(&samples)
		if err != nil || len(samples) == 0 {
			return nil // за этот период нет поминутных снимков (например, zabbixmetrics ещё не собирал данные по этому хосту)
		}
		for _, s := range samples {
			if s == nil {
				continue
			}
			cpuPoints = append(cpuPoints, unifiedMetricPoint{Timestamp: s.CollectedAt, Value: s.CPUUtilPercent})
			memPoints = append(memPoints, unifiedMetricPoint{Timestamp: s.CollectedAt, Value: s.RAMUtilPercent})
			diskReadPoints = append(diskReadPoints, unifiedMetricPoint{Timestamp: s.CollectedAt, Value: s.DiskReadMBps})
			diskWritePoints = append(diskWritePoints, unifiedMetricPoint{Timestamp: s.CollectedAt, Value: s.DiskWriteMBps})
			netRecvPoints = append(netRecvPoints, unifiedMetricPoint{Timestamp: s.CollectedAt, Value: s.NetRecvMBps})
			netTransPoints = append(netTransPoints, unifiedMetricPoint{Timestamp: s.CollectedAt, Value: s.NetSentMBps})
		}
		cpuPoints = resampleToMinuteGrid(cpuPoints, from, dateTo.Add(-time.Minute), 1)
		memPoints = resampleToMinuteGrid(memPoints, from, dateTo.Add(-time.Minute), 1)
		diskReadPoints = resampleToMinuteGrid(diskReadPoints, from, dateTo.Add(-time.Minute), 1)
		diskWritePoints = resampleToMinuteGrid(diskWritePoints, from, dateTo.Add(-time.Minute), 1)
		netRecvPoints = resampleToMinuteGrid(netRecvPoints, from, dateTo.Add(-time.Minute), 1)
		netTransPoints = resampleToMinuteGrid(netTransPoints, from, dateTo.Add(-time.Minute), 1)
	} else {
		var recs []*models.ServerMetricRecord
		_, err := o.QueryTable(new(models.ServerMetricRecord)).
			Filter("HostId", srv.ZabbixHostId).
			Filter("Date__gte", from).
			Filter("Date__lt", dateTo).
			OrderBy("Date").
			All(&recs)
		if err != nil || len(recs) == 0 {
			return nil
		}
		for _, r := range recs {
			if r == nil {
				continue
			}
			cpuPoints = append(cpuPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.CPUUtilAverage})
			memPoints = append(memPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.RAMUtilAverage})
			diskReadPoints = append(diskReadPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.DiskReadMBpsAverage})
			diskWritePoints = append(diskWritePoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.DiskWriteMBpsAverage})
			netRecvPoints = append(netRecvPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.NetRecvMBpsAverage})
			netTransPoints = append(netTransPoints, unifiedMetricPoint{Timestamp: r.Date, Value: r.NetSentMBpsAverage})
		}
		// server_metric_record уже содержит одну строку на сутки - повторная
		// агрегация (aggregateToDailyGrid) не нужна, в отличие от VM-ветки
		// buildVMWideRows, где источник детальных vm_perf-точек может быть
		// подневным только после явной агрегации.
	}

	return combineIntoWideRows(srv.Name, srv.ITSystemKey, cpuPoints, memPoints, diskReadPoints, diskWritePoints, netRecvPoints, netTransPoints)
}

// unifiedMetricsXLSXHeader - заголовок листа широкой выгрузки метрик.
var unifiedMetricsXLSXHeader = []string{
	"Время", "ИТ система", "Имя сервера", "Утилизация CPU %", "Утилизация RAM %",
	"Диск чтение МБ/сек", "Диск запись МБ/сек", "Сеть приём МБ/сек", "Сеть передача МБ/сек",
}

// exportUnifiedMetricsXLSX отдаёт широкую выгрузку метрик утилизации файлом
// XLSX на скачивание - формирование листа аналогично ExportVMListXLSX
// (controllers/sprsrvvm.go) и экспорту userlog.
func exportUnifiedMetricsXLSX(this *MainController, flash *beego.FlashData, rows []wideMetricRow) {
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
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	sheet := "Метрики"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/metricsexport", 302)
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &unifiedMetricsXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	for j, r := range rows {
		row := []string{
			r.Timestamp.Format("2006-01-02 15:04:05"),
			r.ITSystem,
			r.ObjectName,
			r.CPUPercent,
			r.MemPercent,
			r.DiskReadMBs,
			r.DiskWriteMBs,
			r.NetRecvMBs,
			r.NetTransMBs,
		}
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			this.Redirect("/admin/metricsexport", 302)
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/metricsexport", 302)
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/metricsexport", 302)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" %v", err)
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

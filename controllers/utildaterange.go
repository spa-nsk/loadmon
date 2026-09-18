package controllers

import "time"

// utilDateRangeFilter - разбор GET-параметров "from"/"to" (диапазон дат для
// отчётов утилизации /sprsrv/utildatacenter и /sprsrv/utilitsystem) - по
// прямому запросу пользователя (2026-09-18, "за какой период формируются
// оба отчета? Добавь диапазон дат для формирования за период в оба
// отчета."). Формат и конвенция ТЕ ЖЕ, что уже используются на страницах
// "*perfperiod*" (controllers/sprsrvclusterperfperiod.go,
// controllers/sprsrvzabbixperfperiod.go, controllers/sprsrvvmperfperiod.go) -
// текстовый формат "2006-01-02", часовой пояс time.Local, обычная HTML5
// <input type="date"> без JS-датапикера, обычная GET-форма с полной
// перезагрузкой страницы (эти отчёты и так уже построены как "один
// серверный рендер -> JSON внутрь <script>", отдельный AJAX-эндпоинт под
// смену периода не нужен - см. комментарий в CHANGELOG про исследование
// перед добавлением).
//
// СЕМАНТИКА (ИЗМЕНЕНО 2026-09-18, второе уточнение пользователя "нужно
// усреднение за период" - после первого добавления диапазона дат отчёты
// ещё брали только последнюю запись в пределах диапазона, что оказалось не
// тем, что требовалось): оба отчёта усредняют метрики (CPU%/RAM% и, где
// применимо, ёмкость ресурса) КАЖДОЙ сущности (хоста/VM/аппаратного
// сервера) ПО ВСЕМ её записям в выбранном диапазоне дат - т.е. это
// настоящее усреднение за период, а не "последнее состояние в окне". Сама
// эта функция (utilDateRangeFilter) отвечает только за разбор границ
// диапазона - собственно усреднение реализовано в каждом контроллере
// отдельно (см. hostAvg в controllers/sprutildatacenter.go, vmAccum/
// metricAccum в controllers/sprutilitsystem.go), т.к. формула усреднения
// зависит от конкретных полей каждой сущности.
//
// По умолчанию (без параметров from/to в GET) - тот же диапазон, что и
// раньше: [сегодня-utilLookbackDays суток; сегодня], обе границы
// включительно.
//
// Возвращает from/to (обе границы, включительно, начало суток по
// time.Local) и toExclusive = to + 1 сутки, готовое для использования как
// Filter("Date__lt", toExclusive) (стандартный в этом проекте способ
// сделать верхнюю границу диапазона дат включительной, см. те же
// perfperiod-контроллеры). Дополнительно кладёт отформатированные строки
// в this.Data["UtilFromDate"]/["UtilToDate"] - именно эти ключи ожидает
// форма фильтра в utildatacenter.tpl/utilitsystem.tpl.
func utilDateRangeFilter(this *MainController) (from, to, toExclusive time.Time) {
	to = CorrectTimeFilter(time.Now())
	from = to.AddDate(0, 0, -utilLookbackDays)

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
	if to.Before(from) { // перепутанные местами даты - не ошибка, просто меняем местами (та же конвенция, что на perfperiod)
		from, to = to, from
	}

	this.Data["UtilFromDate"] = from.Format("2006-01-02")
	this.Data["UtilToDate"] = to.Format("2006-01-02")

	toExclusive = to.AddDate(0, 0, 1)
	return from, to, toExclusive
}

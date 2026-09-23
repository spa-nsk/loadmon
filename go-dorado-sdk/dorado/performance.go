package dorado

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Controller - контроллер СХД Huawei Dorado/OceanStor (объект типа TYPE=207
// в терминологии DeviceManager REST API). Добавлено 2026-09-23 по запросу
// пользователя loadmon "переделай storagemetrics исходя из последних
// данных" - после того как пользователь предоставил официальный REST
// Interface Reference (huawei-api.md, положен в C:\work\loadmon рядом с
// кодом, см. kontext/10-huawei-dorado-rest-api.md), подтвердивший, что
// живая CPU-утилизация контроллера доступна прямо в ответе этого объекта
// (поле CPUUSAGE) - отдельный performance-запрос для неё не нужен.
//
// Заведены только поля, реально нужные проекту сейчас ("3.7.2.1 Batch
// Querying Controller Information" в huawei-api.md содержит заметно больше
// атрибутов - SOFTVER/ELABEL/PCBVER/MEMORYUSAGE и т.п., они не используются
// и не заведены здесь, чтобы не тащить непроверенные на практике поля).
type Controller struct {
	ID            string `json:"ID"`
	TYPE          int    `json:"TYPE"` // 207 = controller, подтверждено huawei-api.md
	HEALTHSTATUS  string `json:"HEALTHSTATUS"`
	RUNNINGSTATUS string `json:"RUNNINGSTATUS"`
	CPUUSAGE      string `json:"CPUUSAGE"` // string(uint32) - "CPU utilization", %
}

// CPUUsagePercent парсит Controller.CPUUSAGE в float64 (% использования
// CPU контроллера). Пустая/неразбираемая строка -> ok=false, вызывающий код
// не должен путать это с реальным нулём.
func (c Controller) CPUUsagePercent() (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(c.CPUUSAGE), 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// GetControllers возвращает список контроллеров СХД - GET /controller, без
// параметров ("Batch Querying Controller Information" в huawei-api.md).
// Controller.ID из этого ответа (например "0A"/"0B") нужен для построения
// CMO_STATISTIC_UUID у GetCurStatisticData ниже.
func (d *Device) GetControllers(ctx context.Context) ([]Controller, error) {
	spath := "/controller"
	req, err := d.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, fmt.Errorf(ErrCreateRequest+": %w", err)
	}

	var controllers []Controller
	if err = d.requestWithRetry(req, &controllers, DefaultHTTPRetryCount); err != nil {
		return nil, fmt.Errorf(ErrRequestWithRetry+": %w", err)
	}
	return controllers, nil
}

// CurStatisticDataEntry - один элемент ответа GET
// .../performace_statistic/cur_statistic_data (см.
// kontext/10-huawei-dorado-rest-api.md, раздел "Источник: huawei-api.md" ->
// "3.3.1.5"). Путь содержит опечатку "performace" (без "n") - это
// буквальный, подтверждённый официальной документацией Huawei путь REST
// API, а не опечатка в этом файле или в проекте.
type CurStatisticDataEntry struct {
	UUID        string `json:"CMO_STATISTIC_UUID"`
	DataIDList  string `json:"CMO_STATISTIC_DATA_ID_LIST"`
	DataList    string `json:"CMO_STATISTIC_DATA_LIST"` // значения через запятую, В ТОМ ЖЕ порядке, что и DataIDList - сопоставление по позиции, не по ключу (так задокументировано)
	TimestampMs string `json:"CMO_STATISTIC_TIMESTAMP"`
}

// Values разбирает DataIDList/DataList в map[код индикатора]значение -
// удобнее, чем работать с двумя параллельными CSV-строками напрямую.
// Индикатор, значение которого не удалось разобрать как число, просто
// отсутствует в результирующей map (не паникует и не прерывает разбор
// остальных).
func (e CurStatisticDataEntry) Values() map[string]float64 {
	ids := strings.Split(e.DataIDList, ",")
	vals := strings.Split(e.DataList, ",")
	out := make(map[string]float64, len(ids))
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || i >= len(vals) {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(vals[i]), 64)
		if err != nil {
			continue
		}
		out[id] = v
	}
	return out
}

// GetCurStatisticData запрашивает ТЕКУЩИЕ значения указанных индикаторов
// производительности для ОДНОГО управляемого объекта. uuid - строка формата
// "<тип объекта>:<ID>" (например "207:0A" для контроллера с ID "0A" - тип
// 207 подтверждён huawei-api.md); dataIDList - коды индикаторов через
// запятую (например "23,25,26,28,370" - таблица кодов см.
// kontext/10-huawei-dorado-rest-api.md). Добавлено 2026-09-23, см.
// комментарий у Controller выше - тот же запрос, что документирован в
// разделе "3.3.1.5 Интерфейс для Querying Current Performance Statistics"
// официального REST Interface Reference.
func (d *Device) GetCurStatisticData(ctx context.Context, uuid, dataIDList string) ([]CurStatisticDataEntry, error) {
	spath := "/performace_statistic/cur_statistic_data"
	req, err := d.newRequest(ctx, "GET", spath, nil)
	if err != nil {
		return nil, fmt.Errorf(ErrCreateRequest+": %w", err)
	}

	q := req.URL.Query()
	q.Add("CMO_STATISTIC_UUID", uuid)
	q.Add("CMO_STATISTIC_DATA_ID_LIST", dataIDList)
	req.URL.RawQuery = q.Encode()

	var entries []CurStatisticDataEntry
	if err = d.requestWithRetry(req, &entries, DefaultHTTPRetryCount); err != nil {
		return nil, fmt.Errorf(ErrRequestWithRetry+": %w", err)
	}
	return entries, nil
}

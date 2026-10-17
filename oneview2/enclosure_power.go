package oneview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Источники мощности корзины (PowerWatts.Source), в порядке убывания приоритета:
//
//  1. PowerSourceUtilization - измерение: GET {enclosure}/utilization (с refresh)
//     на ЛОКАЛЬНОМ appliance;
//  2. PowerSourceObject - поля powerAvg24Hrs/powerPeak24Hrs/powerCapacity24Hrs
//     объекта шасси (Global Dashboard), но только если powerLastUpdate не
//     старше ObjectPowerMaxAge;
//  3. PowerSourceEstimate - оценка по сумме блейдов (считается в cmdb).
//
// Номинал (RatedMaxW): сначала environmentalConfiguration.calibratedMaxPower
// (RatedSourceCalibrated), при его отсутствии - powerCapacity24Hrs объекта
// шасси (RatedSourceCapacity); см. MergeEnclosurePower. Проверка на живых
// appliance (kontext/14-hpe-oneview-rest-api.md): поля объекта в Global
// Dashboard по c7000 не обновлялись с ноября-декабря 2025 и отличались от
// реальных измерений в разы - поэтому они принимаются только свежими.
const (
	PowerSourceUtilization = "utilization"
	PowerSourceObject      = "enclosure"
	PowerSourceEstimate    = "estimate"

	RatedSourceCalibrated = "calibrated"
	RatedSourceCapacity   = "capacity24h"
)

// Настройки (пакетные переменные, чтобы их можно было менять из конфигурации).
var (
	// ObjectPowerMaxAge - максимальный возраст powerLastUpdate объекта шасси,
	// при котором его поля мощности ещё считаются достоверными.
	ObjectPowerMaxAge = 48 * time.Hour
	// EnclosureUtilizationWindow - окно усреднения /utilization шасси.
	EnclosureUtilizationWindow = 24 * time.Hour
	// EnclosureRefreshTimeout - сколько ждать задачу refresh=true.
	EnclosureRefreshTimeout = 90 * time.Second
	// UtilizationClockSkew - допуск на рассинхрон часов клиента и appliance
	// (newestSampleTime может оказаться чуть "в будущем").
	UtilizationClockSkew = 10 * time.Minute
)

const utilizationDateLayout = "20060102T150405.000Z"

// UtilizationMetric - одна метрика ответа /utilization.
type UtilizationMetric struct {
	MetricName     string            `json:"metricName"`
	MetricSamples  []json.RawMessage `json:"metricSamples"`
	MetricCapacity json.RawMessage   `json:"metricCapacity,omitempty"`
}

// UtilizationData - ответ GET .../utilization (UtilizationData в справочнике HPE).
type UtilizationData struct {
	URI              string              `json:"uri,omitempty"`
	Resolution       FlexInt             `json:"resolution,omitempty"`
	SliceStartTime   string              `json:"sliceStartTime,omitempty"`
	SliceEndTime     string              `json:"sliceEndTime,omitempty"`
	NewestSampleTime string              `json:"newestSampleTime,omitempty"`
	OldestSampleTime string              `json:"oldestSampleTime,omitempty"`
	IsFresh          bool                `json:"isFresh,omitempty"`
	RefreshTaskURI   *string             `json:"refreshTaskUri,omitempty"`
	MetricList       []UtilizationMetric `json:"metricList,omitempty"`
}

// UtilizationOptions - параметры запроса /utilization.
type UtilizationOptions struct {
	Fields  []string  // AveragePower, PeakPower, ...; пусто - все метрики
	View    string    // native | hour | day; пусто - без параметра
	Start   time.Time // нулевое - без filter=startDate
	End     time.Time // нулевое - без filter=endDate
	Refresh bool      // refresh=true: поставить задачу сбора свежих данных с устройства
}

// MetricStats - сводка по реальным сэмплам метрики.
type MetricStats struct {
	N    int     // число не-null сэмплов не позже newestSampleTime
	Mean float64 // среднее по ним
	Max  float64
	Last float64 // значение на newestSampleTime (самый свежий реальный сэмпл)
	Tail int     // сэмплов позже newestSampleTime (appliance добивает хвост нулями/повтором)
}

// NewestSample возвращает newestSampleTime.
func (u *UtilizationData) NewestSample() (time.Time, bool) {
	if u == nil || u.NewestSampleTime == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, u.NewestSampleTime)
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// Stats считает MetricStats по метрике name. Сэмплы позже newestSampleTime
// (хвост, который appliance заполняет сам нулями или повтором последнего
// значения) в расчёт НЕ входят; если newestSampleTime не задан - берутся все.
func (u *UtilizationData) Stats(name string) MetricStats {
	var st MetricStats
	if u == nil {
		return st
	}
	newest, hasNewest := u.NewestSample()
	for _, m := range u.MetricList {
		if !strings.EqualFold(m.MetricName, name) {
			continue
		}
		var sum float64
		haveLast := false
		for _, raw := range m.MetricSamples { // от новых к старым
			ms, v, ok := parseSample(raw)
			if hasNewest && ms > newest.UnixMilli() {
				st.Tail++
				continue
			}
			if !ok {
				continue
			}
			st.N++
			sum += v
			if v > st.Max {
				st.Max = v
			}
			if !haveLast {
				st.Last, haveLast = v, true
			}
		}
		if st.N > 0 {
			st.Mean = sum / float64(st.N)
		}
		return st
	}
	return st
}

// parseSample разбирает сэмпл "[время_мс, значение]"; ok=false для null/не числа.
func parseSample(raw json.RawMessage) (ms int64, v float64, ok bool) {
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) != nil || len(arr) < 2 {
		return 0, 0, false
	}
	var t float64
	if json.Unmarshal(arr[0], &t) != nil {
		return 0, 0, false
	}
	ms = int64(t)
	var fp FlexPowerW
	if string(bytesTrim(arr[1])) == "null" {
		return ms, 0, false
	}
	if json.Unmarshal(arr[1], &fp) != nil {
		return ms, 0, false
	}
	return ms, fp.Float64(), true
}

func utilizationPath(uri string, o UtilizationOptions) string {
	q := url.Values{}
	for _, f := range o.Fields {
		if f = strings.TrimSpace(f); f != "" {
			q.Add("fields", f)
		}
	}
	if o.View != "" {
		q.Set("view", o.View)
	}
	if !o.Start.IsZero() {
		q.Add("filter", "startDate="+o.Start.UTC().Format(utilizationDateLayout))
	}
	if !o.End.IsZero() {
		q.Add("filter", "endDate="+o.End.UTC().Format(utilizationDateLayout))
	}
	if o.Refresh {
		q.Set("refresh", "true")
	}
	p := strings.TrimRight(uri, "/") + "/utilization"
	if len(q) > 0 {
		p += "?" + q.Encode()
	}
	return p
}

// GetUtilization - GET {uri}/utilization (server-hardware, enclosures, ...).
func (c *Client) GetUtilization(ctx context.Context, uri string, o UtilizationOptions) (*UtilizationData, error) {
	resp, err := c.Do(ctx, http.MethodGet, utilizationPath(uri, o), nil)
	if err != nil {
		return nil, err
	}
	var u UtilizationData
	if err := decodeJSON(resp.Body, &u); err != nil {
		return nil, fmt.Errorf("oneview: decode utilization %s: %w", uri, err)
	}
	return &u, nil
}

// GetUtilizationRefreshed читает /utilization со свежими данными: запрос с
// refresh=true ставит на appliance задачу получения данных с устройства
// (refreshTaskUri); после её завершения данные читаются повторно. Если
// refreshTaskUri пуст (данные уже свежие или обновлять нечего), возвращается
// ответ первого запроса. Ошибка/таймаут ожидания задачи не фатальны - данные
// всё равно перечитываются.
func (c *Client) GetUtilizationRefreshed(ctx context.Context, uri string, o UtilizationOptions, wait time.Duration) (*UtilizationData, error) {
	ro := o
	ro.Refresh = true
	u, err := c.GetUtilization(ctx, uri, ro)
	if err != nil {
		return nil, err
	}
	if u.RefreshTaskURI == nil || *u.RefreshTaskURI == "" {
		return u, nil
	}
	if wait <= 0 {
		wait = EnclosureRefreshTimeout
	}
	wctx, cancel := context.WithTimeout(ctx, wait)
	_, _ = c.WaitTask(wctx, *u.RefreshTaskURI)
	cancel()
	o.Refresh = false
	return c.GetUtilization(ctx, uri, o)
}

// EnvironmentalConfiguration - паспортные данные устройства (не измерения).
type EnvironmentalConfiguration struct {
	CalibratedMaxPower    FlexPowerW `json:"calibratedMaxPower,omitempty"`
	IdleMaxPower          FlexPowerW `json:"idleMaxPower,omitempty"`
	LicenseRequirement    string     `json:"licenseRequirement,omitempty"` // None | iLOAdvanced | OneView
	PowerHistorySupported FlexBool   `json:"powerHistorySupported,omitempty"`
}

// GetEnvironmentalConfiguration - GET {uri}/environmentalConfiguration.
func (c *Client) GetEnvironmentalConfiguration(ctx context.Context, uri string) (*EnvironmentalConfiguration, error) {
	resp, err := c.Do(ctx, http.MethodGet, strings.TrimRight(uri, "/")+"/environmentalConfiguration", nil)
	if err != nil {
		return nil, err
	}
	var e EnvironmentalConfiguration
	if err := decodeJSON(resp.Body, &e); err != nil {
		return nil, fmt.Errorf("oneview: decode environmentalConfiguration %s: %w", uri, err)
	}
	return &e, nil
}

// EnclosurePower возвращает лучшую мощность корзины, доступную через ЭТУ
// точку подключения:
//   - локальный appliance: измерение /utilization с refresh (Source=utilization);
//     номинал - calibratedMaxPower, при его отсутствии powerCapacity24Hrs
//     свежего объекта; если измерения нет (404, нет лицензии/данных, история
//     старше окна) - поля объекта, если они свежие;
//   - Global Dashboard: только поля объекта, если powerLastUpdate свежий.
//
// Пустой PowerWatts (Source == "") означает "данных нет" - вызывающий код
// должен перейти к следующему источнику (MergeEnclosurePower, затем оценка по блейдам).
func (c *Client) EnclosurePower(ctx context.Context, enc *Enclosure) PowerWatts {
	if enc == nil {
		return PowerWatts{}
	}
	obj := EnclosurePowerWatts(enc)
	if c.IsGlobalDashboard() || strings.TrimSpace(enc.URI) == "" {
		return obj
	}

	now := time.Now().UTC()
	u, err := c.GetUtilizationRefreshed(ctx, enc.URI, UtilizationOptions{
		Fields: []string{"AveragePower", "PeakPower"},
		View:   "hour",
		Start:  now.Add(-EnclosureUtilizationWindow),
		End:    now,
	}, EnclosureRefreshTimeout)
	if err != nil {
		return obj
	}
	if newest, ok := u.NewestSample(); !ok || now.Sub(newest) > EnclosureUtilizationWindow+UtilizationClockSkew {
		return obj // истории нет или она старше окна
	}
	avg, peak := u.Stats("AveragePower"), u.Stats("PeakPower")
	if avg.N == 0 || (avg.Mean <= 0 && peak.Max <= 0) {
		return obj
	}
	p := PowerWatts{Avg24hW: round1(avg.Mean), Peak24hW: peak.Max, Source: PowerSourceUtilization}
	if p.Peak24hW < avg.Max {
		p.Peak24hW = avg.Max
	}
	if env, err := c.GetEnvironmentalConfiguration(ctx, enc.URI); err == nil && env.CalibratedMaxPower.Float64() > 0 {
		p.RatedMaxW = env.CalibratedMaxPower.Float64()
		p.RatedSource = RatedSourceCalibrated
	} else if obj.RatedMaxW > 0 {
		p.RatedMaxW = obj.RatedMaxW
		p.RatedSource = RatedSourceCapacity
	}
	return p
}

// MergeEnclosurePower выбирает мощность корзины из нескольких источников
// (например, локальный appliance и Global Dashboard) по приоритету:
// измерение /utilization > свежие поля объекта > прочее измеренное.
// Номинал: calibratedMaxPower > powerCapacity24Hrs > прочее. Пустой результат
// (Source == "" и нулевые значения) - сигнал использовать оценку по блейдам.
func MergeEnclosurePower(candidates ...PowerWatts) PowerWatts {
	var best PowerWatts
	bestRank := 0
	for _, c := range candidates {
		if !c.HasMeasured() {
			continue
		}
		if r := measuredRank(c.Source); r > bestRank {
			best, bestRank = c, r
		}
	}
	// номинал - независимо от выбранного источника измерения
	ratedBest, ratedRank := 0.0, 0
	ratedSrc := ""
	consider := func(c PowerWatts) {
		if c.RatedMaxW <= 0 {
			return
		}
		if r := ratedRankOf(c.RatedSource, c.Source); r > ratedRank {
			ratedBest, ratedRank, ratedSrc = c.RatedMaxW, r, c.RatedSource
		}
	}
	consider(best) // при равенстве рангов предпочтительнее номинал выбранного источника
	for _, c := range candidates {
		consider(c)
	}
	best.RatedMaxW, best.RatedSource = ratedBest, ratedSrc
	return best
}

func measuredRank(source string) int {
	switch source {
	case PowerSourceUtilization:
		return 3
	case PowerSourceObject:
		return 2
	default:
		return 1
	}
}

func ratedRankOf(ratedSource, source string) int {
	switch ratedSource {
	case RatedSourceCalibrated:
		return 3
	case RatedSourceCapacity:
		return 2
	}
	if source == PowerSourceObject {
		return 2
	}
	return 1
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

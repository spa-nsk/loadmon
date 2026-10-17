package oneview

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const tsLayout = "2006-01-02T15:04:05.000Z"

func encWithUpdate(t *testing.T, avg, peak, capacity, lastUpdate string) *Enclosure {
	t.Helper()
	raw := fmt.Sprintf(`{"name":"E","powerAvg24Hrs":%s,"powerPeak24Hrs":%s,"powerCapacity24Hrs":%s,"powerLastUpdate":%q}`,
		avg, peak, capacity, lastUpdate)
	var enc Enclosure
	if err := json.Unmarshal([]byte(raw), &enc); err != nil {
		t.Fatal(err)
	}
	return &enc
}

func TestEnclosurePowerWatts_freshness(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, upd string
		want      bool
	}{
		{"свежее (1 ч)", "2026-10-17T11:00:00.000Z", true},
		{"граница 47 ч", "2026-10-15T13:00:00.000Z", true},
		{"устарело (c7000 из GD, 11 мес.)", "2025-12-01T01:55:00.000Z", false},
		{"эпоха 1970 (GD: никогда)", "1970-01-01T00:00:00.000Z", false},
		{"пусто", "", false},
		{"не дата", "вчера", false},
		{"в будущем сверх допуска", "2026-10-18T12:00:00.000Z", false},
	}
	for _, tc := range cases {
		enc := encWithUpdate(t, `"3270"`, `4019`, `"6768,"`, tc.upd)
		p := EnclosurePowerWattsAt(enc, now, 48*time.Hour)
		got := p.Source == PowerSourceObject
		if got != tc.want {
			t.Errorf("%s: принято=%v ожидалось %v (%+v)", tc.name, got, tc.want, p)
		}
		if tc.want && (p.Avg24hW != 3270 || p.Peak24hW != 4019 || p.RatedMaxW != 6768 || p.RatedSource != RatedSourceCapacity) {
			t.Errorf("%s: %+v", tc.name, p)
		}
		if !tc.want && (p.HasMeasured() || p.RatedMaxW != 0) {
			t.Errorf("%s: устаревшие значения не должны проходить: %+v", tc.name, p)
		}
	}
}

func TestMergeEnclosurePower_priority(t *testing.T) {
	util := PowerWatts{Avg24hW: 2700, Peak24hW: 2900, RatedMaxW: 5212, RatedSource: RatedSourceCalibrated, Source: PowerSourceUtilization}
	gd := PowerWatts{Avg24hW: 542, Peak24hW: 544, RatedMaxW: 1130, RatedSource: RatedSourceCapacity, Source: PowerSourceObject}

	// измерение локального appliance > поля GD, порядок кандидатов не важен
	for _, in := range [][]PowerWatts{{util, gd}, {gd, util}} {
		m := MergeEnclosurePower(in...)
		if m.Source != PowerSourceUtilization || m.Avg24hW != 2700 || m.RatedMaxW != 5212 || m.RatedSource != RatedSourceCalibrated {
			t.Fatalf("%+v", m)
		}
	}

	// только GD (Synergy): берутся свежие поля объекта и их ёмкость как номинал
	m := MergeEnclosurePower(PowerWatts{}, gd)
	if m.Source != PowerSourceObject || m.Avg24hW != 542 || m.RatedMaxW != 1130 {
		t.Fatalf("%+v", m)
	}

	// измерение без calibratedMaxPower: номинал - ёмкость объекта GD
	utilNoRated := PowerWatts{Avg24hW: 2700, Peak24hW: 2900, Source: PowerSourceUtilization}
	m = MergeEnclosurePower(utilNoRated, gd)
	if m.Source != PowerSourceUtilization || m.Avg24hW != 2700 || m.RatedMaxW != 1130 || m.RatedSource != RatedSourceCapacity {
		t.Fatalf("%+v", m)
	}

	// ничего нет -> пусто (вызывающий код использует оценку по блейдам)
	m = MergeEnclosurePower(PowerWatts{}, PowerWatts{})
	if m.HasMeasured() || m.Source != "" || m.RatedMaxW != 0 {
		t.Fatalf("%+v", m)
	}
}

func TestUtilizationStats_dropsTailAfterNewest(t *testing.T) {
	// сэмплы от новых к старым; два последних слота (хвост) после newestSampleTime
	// заполнены нулём - в расчёт входить не должны
	newest := time.Date(2026, 10, 17, 5, 45, 0, 0, time.UTC)
	ms := func(d time.Duration) int64 { return newest.Add(d).UnixMilli() }
	u := &UtilizationData{
		NewestSampleTime: newest.Format(tsLayout),
		MetricList: []UtilizationMetric{{
			MetricName: "AveragePower",
			MetricSamples: []json.RawMessage{
				json.RawMessage(fmt.Sprintf("[%d, 0]", ms(2*time.Hour))),
				json.RawMessage(fmt.Sprintf("[%d, 0]", ms(time.Hour))),
				json.RawMessage(fmt.Sprintf("[%d, 479]", ms(0))),
				json.RawMessage(fmt.Sprintf(`[%d, "477"]`, ms(-time.Hour))),
				json.RawMessage(fmt.Sprintf("[%d, null]", ms(-2*time.Hour))),
				json.RawMessage(fmt.Sprintf("[%d, 475]", ms(-3*time.Hour))),
			},
		}},
	}
	st := u.Stats("AveragePower")
	if st.N != 3 || st.Tail != 2 || st.Last != 479 || st.Max != 479 || int(st.Mean+0.5) != 477 {
		t.Fatalf("%+v", st)
	}
}

// fakeEnclosureServer - локальный appliance: /utilization шасси сначала отдаёт
// устаревшую историю и refreshTaskUri, после выполнения задачи - свежие данные.
func fakeEnclosureServer(t *testing.T, refreshed *atomic.Bool, calibrated string) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/version":
			io.WriteString(w, `{"minimumVersion":120,"currentVersion":3800}`)
		case r.URL.Path == "/rest/login-sessions":
			io.WriteString(w, `{"sessionID":"t"}`)
		case r.URL.Path == "/rest/tasks/t1":
			refreshed.Store(true)
			io.WriteString(w, `{"uri":"/rest/tasks/t1","taskState":"Completed","percentComplete":100}`)
		case r.URL.Path == "/rest/enclosures/E1/utilization":
			if r.URL.Query().Get("refresh") == "true" && !refreshed.Load() {
				io.WriteString(w, `{"newestSampleTime":"2026-01-01T00:00:00.000Z","refreshTaskUri":"/rest/tasks/t1","metricList":[]}`)
				return
			}
			newest := time.Now().UTC().Truncate(5 * time.Minute)
			ts := func(d time.Duration) int64 { return newest.Add(d).UnixMilli() }
			fmt.Fprintf(w, `{"newestSampleTime":%q,"resolution":3600000,"isFresh":true,"metricList":[`+
				`{"metricName":"AveragePower","metricSamples":[[%d,0],[%d,2700],[%d,2600],[%d,2800]]},`+
				`{"metricName":"PeakPower","metricSamples":[[%d,0],[%d,2900],[%d,2950],[%d,3000]]}]}`,
				newest.Format(tsLayout),
				ts(time.Hour), ts(0), ts(-time.Hour), ts(-2*time.Hour),
				ts(time.Hour), ts(0), ts(-time.Hour), ts(-2*time.Hour))
		case r.URL.Path == "/rest/enclosures/E1/environmentalConfiguration":
			if calibrated == "" {
				http.NotFound(w, r)
				return
			}
			io.WriteString(w, `{"calibratedMaxPower":`+calibrated+`,"licenseRequirement":"None","powerHistorySupported":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func loginTo(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Config{Endpoint: srv.URL, Username: "a", Password: "b", InsecureTLS: true, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Login(context.Background()); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestClientEnclosurePower_utilizationWithRefresh(t *testing.T) {
	var refreshed atomic.Bool
	srv := fakeEnclosureServer(t, &refreshed, "5212")
	c := loginTo(t, srv)
	enc := &Enclosure{Resource: Resource{Name: "C7000-12", URI: "/rest/enclosures/E1"}}

	p := c.EnclosurePower(context.Background(), enc)
	if !refreshed.Load() {
		t.Fatal("refresh-задача не выполнялась")
	}
	// хвост (слот после newestSampleTime с нулём) не учитывается: среднее 2700, пик 3000
	if p.Source != PowerSourceUtilization || p.Avg24hW != 2700 || p.Peak24hW != 3000 {
		t.Fatalf("%+v", p)
	}
	if p.RatedMaxW != 5212 || p.RatedSource != RatedSourceCalibrated {
		t.Fatalf("номинал: %+v", p)
	}
}

func TestClientEnclosurePower_ratedFallsBackToObjectCapacity(t *testing.T) {
	var refreshed atomic.Bool
	srv := fakeEnclosureServer(t, &refreshed, "") // calibratedMaxPower недоступен
	c := loginTo(t, srv)
	fresh := time.Now().UTC().Add(-time.Hour).Format(tsLayout)
	enc := encWithUpdate(t, `100`, `200`, `6000`, fresh)
	enc.URI = "/rest/enclosures/E1"

	p := c.EnclosurePower(context.Background(), enc)
	if p.Source != PowerSourceUtilization || p.Avg24hW != 2700 {
		t.Fatalf("%+v", p)
	}
	if p.RatedMaxW != 6000 || p.RatedSource != RatedSourceCapacity {
		t.Fatalf("номинал должен браться из powerCapacity24Hrs объекта: %+v", p)
	}
}

func TestClientEnclosurePower_noMeasurementFallsBackToFreshObjectOnly(t *testing.T) {
	// на appliance нет /utilization (404): остаются только свежие поля объекта
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/version":
			io.WriteString(w, `{"minimumVersion":120,"currentVersion":3800}`)
		case "/rest/login-sessions":
			io.WriteString(w, `{"sessionID":"t"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := loginTo(t, srv)

	stale := encWithUpdate(t, `4577`, `4756`, `7134`, "2025-12-01T01:55:00.000Z")
	stale.URI = "/rest/enclosures/E1"
	if p := c.EnclosurePower(context.Background(), stale); p.HasMeasured() || p.RatedMaxW != 0 || p.Source != "" {
		t.Fatalf("устаревшие поля должны отбрасываться: %+v", p)
	}
	fresh := encWithUpdate(t, `3270`, `4019`, `6768`, time.Now().UTC().Format(tsLayout))
	fresh.URI = "/rest/enclosures/E1"
	if p := c.EnclosurePower(context.Background(), fresh); p.Source != PowerSourceObject || p.Avg24hW != 3270 {
		t.Fatalf("свежие поля должны приниматься: %+v", p)
	}
}

func TestClientEnclosurePower_globalDashboardUsesObjectOnly(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/version":
			io.WriteString(w, `{"minimumVersion":2,"currentVersion":300}`)
		case r.URL.Path == "/rest/login-sessions":
			io.WriteString(w, `{"sessionID":"t"}`)
		case strings.Contains(r.URL.Path, "/utilization"):
			calls.Add(1)
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := loginTo(t, srv)
	if !c.IsGlobalDashboard() {
		t.Fatal("ожидался Global Dashboard")
	}
	fresh := encWithUpdate(t, `3270`, `4019`, `6768`, time.Now().UTC().Format(tsLayout))
	fresh.URI = "/rest/enclosures/uuid"
	p := c.EnclosurePower(context.Background(), fresh)
	if p.Source != PowerSourceObject || p.Avg24hW != 3270 {
		t.Fatalf("%+v", p)
	}
	if calls.Load() != 0 {
		t.Fatalf("у GD /utilization не должен запрашиваться (запросов: %d)", calls.Load())
	}
}

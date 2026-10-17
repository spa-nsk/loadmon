package oneviewpowertest

// Полная проверка шасси (блейд-корзин) на каждом appliance, включая Global
// Dashboard: список ВСЕХ шасси, поля мощности в самом объекте шасси
// (powerAvg24Hrs/powerPeak24Hrs/powerCapacity24Hrs/powerLastUpdate - их читает
// oneview2.EnclosurePowerWatts), /utilization шасси (с refresh),
// environmentalConfiguration и привязка блейдов. В конце - сводка и
// СРАВНЕНИЕ источников для одной и той же корзины (по серийному номеру):
// поля объекта GD против полей локального appliance против /utilization.
// Параметры - в блоке "ПАРАМЕТРЫ ТЕСТА" файла oneview_power_test.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	oneview "github.com/spa-nsk/oneview2"
)

// encFields - метрики /utilization шасси.
const encFields = "AveragePower,PeakPower,RatedCapacity,DeratedCapacity,PowerCap,AmbientTemperature"

// encResult - итог по одному шасси на одном appliance.
type encResult struct {
	Endpoint, API, Name, SN, URI, Type, Model, State string

	// поля мощности в самом объекте шасси
	ObjAvg, ObjPeak, ObjCap float64
	ObjLastUpdate           string

	Blades, LicensedBlades int

	UtilHTTP    int
	UtilClass   string // OK / ZEROS / NO_SAMPLES / STALE / NOT_FOUND / HTTP_ERR / ERR
	Newest      string
	Tail        int
	AvgLast     float64 // значение на newestSampleTime
	AvgMean     float64
	AvgMax      float64
	PeakMax     float64
	Rated       float64 // RatedCapacity (последнее ненулевое)
	Derated     float64
	RefreshNote string

	EnvHTTP       int
	EnvReq        string
	EnvCalibrated float64
	EnvPSU        int
}

// allEnclosures накапливает результаты по всем appliance (для сводки).
var allEnclosures []encResult

// probeEnclosuresFull обходит ВСЕ шасси appliance.
func probeEnclosuresFull(ctx context.Context, c *oneview.Client, endpoint, api string, servers []oneview.ServerHardware) {
	var members []map[string]json.RawMessage
	start, total := 0, -1
	for {
		path := fmt.Sprintf("/rest/enclosures?start=%d&count=%d", start, pageSize)
		status, body, err := doGET(ctx, c, path)
		say("шасси: GET %s -> HTTP %d %s", path, status, errText(err))
		if status != 200 {
			if status == 404 {
				say("шасси: ресурс /rest/enclosures на этом appliance отсутствует")
			}
			return
		}
		var col struct {
			Total   int                          `json:"total"`
			Members []map[string]json.RawMessage `json:"members"`
		}
		if json.Unmarshal(body, &col) != nil {
			return
		}
		total = col.Total
		members = append(members, col.Members...)
		start += len(col.Members)
		if len(col.Members) == 0 || start >= total {
			break
		}
	}
	say("шасси: всего %d, к проверке %d (enclosureMax=%d)", total, len(members), enclosureMax)

	for i, m := range members {
		if enclosureMax > 0 && i >= enclosureMax {
			break
		}
		r := encResult{Endpoint: endpoint, API: api}
		r.Name = rawString(m["name"])
		r.URI = rawString(m["uri"])
		r.SN = rawString(m["serialNumber"])
		if r.SN == "" {
			r.SN = lastSegment(r.URI)
		}
		r.Type = rawString(m["enclosureType"])
		r.Model = rawString(m["enclosureModel"])
		r.State = rawString(m["state"])
		r.ObjAvg = flexWatts(m["powerAvg24Hrs"])
		r.ObjPeak = flexWatts(m["powerPeak24Hrs"])
		r.ObjCap = flexWatts(m["powerCapacity24Hrs"])
		r.ObjLastUpdate = rawString(m["powerLastUpdate"])

		// блейды этого шасси: locationUri сервера оканчивается серийником
		// шасси или id из URI шасси (в GD URI - uuid, locationUri - серийник).
		idURI := lastSegment(r.URI)
		for _, hw := range servers {
			loc := lastSegment(hw.LocationURI)
			if loc == "" || (loc != r.SN && loc != idURI) {
				continue
			}
			r.Blades++
			if l := strings.TrimSpace(hw.MpLicenseType); l != "" && !strings.EqualFold(l, "Not Licensed") {
				r.LicensedBlades++
			}
		}

		say("")
		say("шасси [%d/%d] %q SN=%s type=%q model=%q state=%s блейдов=%d (с лицензией iLO: %d)",
			i+1, len(members), r.Name, r.SN, r.Type, r.Model, r.State, r.Blades, r.LicensedBlades)
		say("    в объекте: powerAvg24Hrs=%s powerPeak24Hrs=%s powerCapacity24Hrs=%s powerLastUpdate=%q",
			unquote(m["powerAvg24Hrs"]), unquote(m["powerPeak24Hrs"]), unquote(m["powerCapacity24Hrs"]), r.ObjLastUpdate)

		if r.URI != "" {
			encUtilization(ctx, c, &r)
			encEnvironmental(ctx, c, &r)
		}
		allEnclosures = append(allEnclosures, r)
		if pauseBetweenServers > 0 {
			time.Sleep(pauseBetweenServers)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// encUtilization: /utilization шасси; при enclosureRefresh - с refresh=true,
// ожиданием задачи и повторным чтением.
func encUtilization(ctx context.Context, c *oneview.Client, r *encResult) {
	var status int
	var body []byte
	if enclosureRefresh {
		rpath := utilizationURL(r.URI, encFields) + "&refresh=true"
		st, b, err := doGET(ctx, c, rpath)
		if st == 200 {
			var u struct {
				RefreshTaskURI *string `json:"refreshTaskUri"`
			}
			if json.Unmarshal(b, &u) == nil && u.RefreshTaskURI != nil && *u.RefreshTaskURI != "" {
				wctx, cancel := context.WithTimeout(ctx, refreshWait)
				task, werr := c.WaitTask(wctx, *u.RefreshTaskURI)
				cancel()
				if werr != nil {
					r.RefreshNote = "refresh: ожидание задачи: " + shorten(werr.Error(), 80)
				} else {
					r.RefreshNote = "refresh: задача " + task.TaskState
				}
			} else {
				r.RefreshNote = "refresh: задача не поставлена (null)"
			}
		} else {
			r.RefreshNote = fmt.Sprintf("refresh: HTTP %d %s", st, errText(err))
		}
		say("    %s", r.RefreshNote)
	}
	path := utilizationURL(r.URI, encFields)
	status, body, err := doGET(ctx, c, path)
	r.UtilHTTP = status
	say("    GET %s -> HTTP %d %s", path, status, errText(err))
	switch {
	case status == 0:
		r.UtilClass = "ERR"
		return
	case status == 404:
		r.UtilClass = "NOT_FOUND"
		return
	case status >= 400:
		r.UtilClass = "HTTP_ERR"
		return
	}
	raw(body)
	var u struct {
		NewestSampleTime string `json:"newestSampleTime"`
		MetricList       []struct {
			MetricName    string            `json:"metricName"`
			MetricSamples []json.RawMessage `json:"metricSamples"`
		} `json:"metricList"`
	}
	if json.Unmarshal(body, &u) != nil {
		r.UtilClass = "ERR"
		return
	}
	r.Newest = u.NewestSampleTime
	var newestMs int64 = -1
	if ts, err := time.Parse("2006-01-02T15:04:05.000Z", u.NewestSampleTime); err == nil {
		newestMs = ts.UnixMilli()
	}
	haveAny, nonZero := false, false
	for _, m := range u.MetricList {
		var sum, max, lastReal, lastNZ float64
		n, tail := 0, 0
		haveReal, haveNZ := false, false
		var head []string
		for _, s := range m.MetricSamples {
			v, ok := sampleValue(s)
			if len(head) < printHeadSamples {
				head = append(head, sampleLabel(s, v, ok))
			}
			if ms, okT := sampleMillis(s); okT && newestMs >= 0 && ms > newestMs {
				tail++
				continue
			}
			if !ok {
				continue
			}
			n++
			sum += v
			if v > max {
				max = v
			}
			if !haveReal {
				lastReal, haveReal = v, true
			}
			if !haveNZ && v != 0 {
				lastNZ, haveNZ = v, true
			}
		}
		mean := 0.0
		if n > 0 {
			mean = round1(sum / float64(n))
			haveAny = true
			if max > 0 {
				nonZero = true
			}
		}
		if tail > r.Tail {
			r.Tail = tail
		}
		switch m.MetricName {
		case "AveragePower":
			r.AvgLast, r.AvgMean, r.AvgMax = lastReal, mean, max
		case "PeakPower":
			r.PeakMax = max
		case "RatedCapacity":
			r.Rated = lastNZ
		case "DeratedCapacity":
			r.Derated = lastNZ
		}
		say("    %-18s сэмплов=%-4d не-null=%-4d lastReal=%-8.1f mean=%-8.1f max=%-8.1f хвост=%d  свежие: %s",
			m.MetricName, len(m.MetricSamples), n, lastReal, mean, max, tail, strings.Join(head, " "))
	}
	switch {
	case len(u.MetricList) == 0 || !haveAny:
		r.UtilClass = "NO_SAMPLES"
	case !nonZero:
		r.UtilClass = "ZEROS"
	default:
		r.UtilClass = "OK"
	}
	if (r.UtilClass == "NO_SAMPLES" || r.UtilClass == "ZEROS") && newestMs >= 0 &&
		time.Since(time.UnixMilli(newestMs)) > time.Duration(windowHours)*time.Hour {
		r.UtilClass = "STALE"
	}
	say("    -> класс=%s newest=%s (%s)", r.UtilClass, u.NewestSampleTime, sampleAge(u.NewestSampleTime))
}

func encEnvironmental(ctx context.Context, c *oneview.Client, r *encResult) {
	path := r.URI + "/environmentalConfiguration"
	status, body, err := doGET(ctx, c, path)
	r.EnvHTTP = status
	say("    GET %s -> HTTP %d %s", path, status, errText(err))
	if status != 200 {
		return
	}
	raw(body)
	var e map[string]json.RawMessage
	if json.Unmarshal(body, &e) != nil {
		return
	}
	r.EnvReq = unquote(e["licenseRequirement"])
	r.EnvCalibrated, _ = rawFloat(e["calibratedMaxPower"])
	var psu []json.RawMessage
	_ = json.Unmarshal(e["psuList"], &psu)
	r.EnvPSU = len(psu)
	say("    env: licenseRequirement=%q powerHistorySupported=%s calibratedMaxPower=%.0f Вт idleMaxPower=%s psu=%d",
		r.EnvReq, unquote(e["powerHistorySupported"]), r.EnvCalibrated, unquote(e["idleMaxPower"]), r.EnvPSU)
}

// ---------------------------------------------------------------------------
// сводка и сравнение источников
// ---------------------------------------------------------------------------

func printEnclosureSummary() {
	if len(allEnclosures) == 0 {
		return
	}
	say("")
	say("================ ШАССИ: СВОДКА ================")

	// по appliance
	type agg struct {
		n, objAvg, util, notFound, blades, licensed int
	}
	byEP := map[string]*agg{}
	var order []string
	for _, r := range allEnclosures {
		k := fmt.Sprintf("%s (API %s)", r.Endpoint, r.API)
		a := byEP[k]
		if a == nil {
			a = &agg{}
			byEP[k] = a
			order = append(order, k)
		}
		a.n++
		a.blades += r.Blades
		a.licensed += r.LicensedBlades
		if r.ObjAvg > 0 {
			a.objAvg++
		}
		if r.UtilClass == "OK" {
			a.util++
		}
		if r.UtilClass == "NOT_FOUND" {
			a.notFound++
		}
	}
	sort.Strings(order)
	say("%-52s %6s %12s %12s %10s %8s %9s", "appliance", "шасси", "obj avg>0", "util OK", "util 404", "блейдов", "с лиц.iLO")
	for _, k := range order {
		a := byEP[k]
		say("%-52s %6d %12d %12d %10d %8d %9d", k, a.n, a.objAvg, a.util, a.notFound, a.blades, a.licensed)
	}

	// построчно
	say("")
	say("%-24s %-16s %-6s | объект avg/peak/cap | util class last/mean/max | peak | rated | env req | calib | блейдов(лиц.)",
		"шасси", "SN", "API")
	for _, r := range allEnclosures {
		say("%-24s %-16s %-6s | %5.0f/%5.0f/%5.0f | %-10s %6.0f/%6.0f/%6.0f | %5.0f | %5.0f | %-9s | %5.0f | %d(%d)",
			cut(r.Name, 24), cut(r.SN, 16), r.API, r.ObjAvg, r.ObjPeak, r.ObjCap,
			r.UtilClass, r.AvgLast, r.AvgMean, r.AvgMax, r.PeakMax, r.Rated, r.EnvReq, r.EnvCalibrated, r.Blades, r.LicensedBlades)
	}

	// сравнение источников для одной корзины (по серийному номеру)
	bySN := map[string][]encResult{}
	for _, r := range allEnclosures {
		if r.SN != "" {
			bySN[strings.ToUpper(r.SN)] = append(bySN[strings.ToUpper(r.SN)], r)
		}
	}
	var sns []string
	for sn, l := range bySN {
		if len(l) > 1 {
			sns = append(sns, sn)
		}
	}
	sort.Strings(sns)
	say("")
	say("---- сравнение источников для корзин, видимых с нескольких endpoint (GD и локальный appliance) ----")
	if len(sns) == 0 {
		say("совпадающих по серийному номеру корзин между endpoint не найдено")
	}
	for _, sn := range sns {
		say("SN %s:", sn)
		for _, r := range bySN[sn] {
			say("    %-48s API %-5s объект avg/peak/cap=%5.0f/%5.0f/%5.0f upd=%q | util %-9s last=%5.0f mean=%5.0f peak=%5.0f",
				cut(r.Endpoint, 48), r.API, r.ObjAvg, r.ObjPeak, r.ObjCap, r.ObjLastUpdate, r.UtilClass, r.AvgLast, r.AvgMean, r.PeakMax)
		}
	}

	// корзины, которые видны ТОЛЬКО через GD (локального источника нет)
	say("")
	say("---- корзины, у которых нет локального источника (только Global Dashboard) ----")
	only := 0
	for _, l := range bySN {
		onlyGD := true
		for _, r := range l {
			if r.API != "300" {
				onlyGD = false
			}
		}
		if onlyGD {
			only++
			r := l[0]
			say("    %-24s SN=%-16s type=%q объект avg/peak/cap=%.0f/%.0f/%.0f блейдов=%d", cut(r.Name, 24), r.SN, r.Type, r.ObjAvg, r.ObjPeak, r.ObjCap, r.Blades)
		}
	}
	if only == 0 {
		say("    нет")
	}
}

func writeEnclosureCSV(path string) error {
	var b strings.Builder
	b.WriteString("endpoint;api;name;sn;type;model;state;objAvg;objPeak;objCap;objLastUpdate;blades;licensedBlades;utilHTTP;utilClass;newest;tail;avgLast;avgMean;avgMax;peakMax;rated;derated;envHTTP;envReq;calibrated;psu;refresh\n")
	for _, r := range allEnclosures {
		fmt.Fprintf(&b, "%s;%s;%s;%s;%s;%s;%s;%.0f;%.0f;%.0f;%s;%d;%d;%d;%s;%s;%d;%.1f;%.1f;%.1f;%.1f;%.0f;%.0f;%d;%s;%.0f;%d;%s\n",
			r.Endpoint, r.API, r.Name, r.SN, r.Type, r.Model, r.State, r.ObjAvg, r.ObjPeak, r.ObjCap, r.ObjLastUpdate,
			r.Blades, r.LicensedBlades, r.UtilHTTP, r.UtilClass, r.Newest, r.Tail, r.AvgLast, r.AvgMean, r.AvgMax, r.PeakMax,
			r.Rated, r.Derated, r.EnvHTTP, r.EnvReq, r.EnvCalibrated, r.EnvPSU, strings.ReplaceAll(r.RefreshNote, ";", ","))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// rawString - строковое поле JSON ("" для null/отсутствия).
func rawString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}

// lastSegment - последний сегмент пути URI ("/rest/enclosures/ABC" -> "ABC").
func lastSegment(uri string) string {
	uri = strings.TrimRight(strings.TrimSpace(uri), "/")
	if i := strings.LastIndex(uri, "/"); i >= 0 {
		return uri[i+1:]
	}
	return uri
}

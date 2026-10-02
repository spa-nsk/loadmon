package scommetrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Обёртки над операциями SCOM REST API, используемыми сборщиком. Формы
// запросов/ответов - по официальному Operations Manager REST API Reference
// (Microsoft Learn, раздел Data); где пример и описание схемы в
// документации расходятся, разбор принимает оба варианта (отмечено ниже).

// tableResponse - TableDataResponse: {"tableColumns": [...], "rows": [{...}]}.
type tableResponse struct {
	Rows []map[string]interface{} `json:"rows"`
}

// Computer - объект SCOM выбранного класса (строка scomObjectsByClass).
type Computer struct {
	ID          string // GUID объекта
	DisplayName string
	FullName    string // "<класс>:<ключ>", для Windows Computer ключ - FQDN
}

// FQDN - имя компьютера: ключ из FullName (после ":"), иначе DisplayName.
func (c Computer) FQDN() string {
	if i := strings.LastIndex(c.FullName, ":"); i >= 0 && i+1 < len(c.FullName) {
		return strings.TrimSpace(c.FullName[i+1:])
	}
	return strings.TrimSpace(c.DisplayName)
}

// ObjectsByClass - POST /data/scomObjectsByClass.
// В документации тело описано как {fullClassName}, а в примере передан
// JSON-массив из одной строки - отправляется вариант из примера, при
// ошибке - объект с полем fullClassName.
func (c *Client) ObjectsByClass(ctx context.Context, className string) ([]Computer, error) {
	var resp tableResponse
	err := c.call(ctx, http.MethodPost, "/data/scomObjectsByClass", []string{className}, &resp)
	if err != nil {
		var resp2 tableResponse
		if err2 := c.call(ctx, http.MethodPost, "/data/scomObjectsByClass", map[string]string{"fullClassName": className}, &resp2); err2 != nil {
			return nil, err
		}
		resp = resp2
	}
	out := make([]Computer, 0, len(resp.Rows))
	for _, r := range resp.Rows {
		comp := Computer{ID: str(r["id"]), DisplayName: str(r["displayname"]), FullName: str(r["fullname"])}
		if comp.ID == "" {
			continue
		}
		out = append(out, comp)
	}
	return out, nil
}

// ObjectProperty - элемент monitoringObjectProperties ({name, value}).
// Value - interface{}: в документации это строка (может быть null), но
// числовое значение в JSON не должно ломать разбор всего ответа.
type ObjectProperty struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// RelatedObject - элемент relatedObjects ответа objectInformation.
type RelatedObject struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// ObjectInformation - ответ GET /data/objectInformation/{objectId}.
type ObjectInformation struct {
	DisplayName    string           `json:"displayName"`
	Properties     []ObjectProperty `json:"monitoringObjectProperties"`
	RelatedObjects []RelatedObject  `json:"relatedObjects"`
}

// PropertyMap - свойства объекта в виде "нормализованное имя" -> значение
// (см. normName: регистр, пробелы и знаки препинания не учитываются).
func (oi ObjectInformation) PropertyMap() map[string]string {
	m := make(map[string]string, len(oi.Properties))
	for _, p := range oi.Properties {
		v := str(p.Value)
		if v == "" {
			continue
		}
		m[normName(p.Name)] = v
	}
	return m
}

// ObjectInfo - GET /data/objectInformation/{objectId}: свойства объекта +
// связанные объекты (для компьютера среди них - объект ОС).
func (c *Client) ObjectInfo(ctx context.Context, objectID string) (ObjectInformation, error) {
	var oi ObjectInformation
	err := c.call(ctx, http.MethodGet, "/data/objectInformation/"+url.PathEscape(objectID), nil, &oi)
	return oi, err
}

// Counter - один счётчик производительности объекта. Raw - элемент ответа
// как пришёл от сервера: именно он передаётся обратно в
// performanceCounters запроса /data/performance, чтобы не придумывать
// схему элемента, которой нет в документации.
type Counter struct {
	ObjectName   string
	CounterName  string
	InstanceName string
	Raw          map[string]interface{}
}

// PerformanceCounters - GET /data/performanceCounters/{entityId}.
// Пример ответа в документации - {"rows":[{objectname, countername,
// instancename}]}, описание схемы - {"performanceCounterDatas":[{objectId,
// objectName, counterName, instanceName, ...}]}; принимаются оба.
func (c *Client) PerformanceCounters(ctx context.Context, entityID string) ([]Counter, error) {
	var resp struct {
		Rows  []map[string]interface{} `json:"rows"`
		Datas []map[string]interface{} `json:"performanceCounterDatas"`
	}
	if err := c.call(ctx, http.MethodGet, "/data/performanceCounters/"+url.PathEscape(entityID), nil, &resp); err != nil {
		return nil, err
	}
	items := resp.Datas
	if len(items) == 0 {
		items = resp.Rows
	}
	out := make([]Counter, 0, len(items))
	for _, it := range items {
		ci := lowerKeys(it)
		out = append(out, Counter{
			ObjectName:   str(ci["objectname"]),
			CounterName:  str(ci["countername"]),
			InstanceName: str(ci["instancename"]),
			Raw:          it,
		})
	}
	return out, nil
}

// PerfValue - строка legends.rows ответа /data/performance.
type PerfValue struct {
	ObjectName   string
	CounterName  string
	InstanceName string
	Last         float64
	Average      float64
	HasData      bool // false - SCOM вернул "пустые" max/min (нет замеров в окне)
}

// Performance - POST /data/performance {id, duration, performanceCounters}.
func (c *Client) Performance(ctx context.Context, requestID string, duration int, counters []Counter) ([]PerfValue, error) {
	raws := make([]map[string]interface{}, 0, len(counters))
	for _, ct := range counters {
		raws = append(raws, ct.Raw)
	}
	payload := map[string]interface{}{
		"id":                  requestID,
		"duration":            duration,
		"performanceCounters": raws,
	}
	var resp struct {
		Legends struct {
			Rows []map[string]interface{} `json:"rows"`
		} `json:"legends"`
	}
	if err := c.call(ctx, http.MethodPost, "/data/performance", payload, &resp); err != nil {
		return nil, err
	}
	out := make([]PerfValue, 0, len(resp.Legends.Rows))
	for _, r := range resp.Legends.Rows {
		ri := lowerKeys(r)
		last, okLast := num(ri["lastvalue"])
		avg, _ := num(ri["averagevalue"])
		maxV, okMax := num(ri["maximumvalue"])
		minV, okMin := num(ri["minimumvalue"])
		// В примере ответа строка без замеров имеет maximumvalue < minimumvalue
		// (-1.79e308 / +1.79e308) - такие строки считаются пустыми.
		has := okLast && !(okMax && okMin && maxV < minV)
		out = append(out, PerfValue{
			ObjectName:   str(ri["performanceobject"]),
			CounterName:  str(ri["performancecounter"]),
			InstanceName: str(ri["performanceinstance"]),
			Last:         last,
			Average:      avg,
			HasData:      has,
		})
	}
	return out, nil
}

func lowerKeys(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[strings.ToLower(k)] = v
	}
	return out
}

func str(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func num(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(strings.Replace(t, ",", ".", 1)), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// normName нормализует имя свойства/счётчика для сравнения: нижний
// регистр, только буквы и цифры ("Physical Memory (MB)" -> "physicalmemorymb").
func normName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 'а' && r <= 'я') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

package itilium

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// APIClient — методы кастомного HTTP-сервиса `/hs/api` (выгрузка КЕ и др.).
// Контракт getCU сверен с боевым URL BCS
// `…/DEV/hs/api/getCU?nextVer=…&cuClass=…&fields=…&limit=…` (не входит в
// публичную документацию 1c-itilium.ru External/Portal).
type APIClient struct {
	c *Client
}

// GetCUQuery — query-параметры GET getCU.
type GetCUQuery struct {
	NextVer int      // курсор/версия выгрузки; 0 - не передавать
	CUClass int      // класс КЕ (классификатор); 0 - не передавать
	Fields  []string // список полей; пусто - не передавать
	Limit   int      // размер страницы; 0 - не передавать
	Extra   url.Values
}

func (q GetCUQuery) values() url.Values {
	v := url.Values{}
	if q.Extra != nil {
		for k, vals := range q.Extra {
			for _, val := range vals {
				v.Add(k, val)
			}
		}
	}
	if q.NextVer != 0 {
		v.Set("nextVer", strconv.Itoa(q.NextVer))
	}
	if q.CUClass != 0 {
		v.Set("cuClass", strconv.Itoa(q.CUClass))
	}
	if len(q.Fields) > 0 {
		v.Set("fields", strings.Join(q.Fields, ","))
	}
	if q.Limit != 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	return v
}

// DefaultServerCUFields — набор полей из примера BCS для серверных КЕ.
var DefaultServerCUFields = []string{
	"code", "status", "modelServ", "hostname", "ip", "linksUnit", "sn", "user", "where", "statusHistory",
}

// CU — конфигурационная единица в ответе getCU (поля по запросу fields;
// неизвестные ключи сохраняются в Extra).
type CU struct {
	Code          FlexString      `json:"code"`
	Status        json.RawMessage `json:"status"`
	ModelServ     json.RawMessage `json:"modelServ"`
	Hostname      FlexString      `json:"hostname"`
	IP            FlexString      `json:"ip"`
	LinksUnit     json.RawMessage `json:"linksUnit"`
	SN            FlexString      `json:"sn"`
	User          json.RawMessage `json:"user"`
	Where         json.RawMessage `json:"where"`
	StatusHistory json.RawMessage `json:"statusHistory"`

	// Extra — остальные поля объекта, не попавшие в типизированные.
	Extra map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON сохраняет известные поля и складывает прочие в Extra.
func (cu *CU) UnmarshalJSON(data []byte) error {
	type alias CU
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*cu = CU(a)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	known := map[string]struct{}{
		"code": {}, "status": {}, "modelServ": {}, "hostname": {}, "ip": {},
		"linksUnit": {}, "sn": {}, "user": {}, "where": {}, "statusHistory": {},
	}
	cu.Extra = make(map[string]json.RawMessage)
	for k, v := range raw {
		if _, ok := known[k]; !ok {
			cu.Extra[k] = v
		}
	}
	return nil
}

// GetCUResponse — типичный ответ getCU. Схема на BCS может отличаться
// (массив / объект с data/items/nextVer) — парсер принимает оба варианта;
// сырое тело доступно в Raw.
type GetCUResponse struct {
	NextVer FlexString `json:"nextVer"`
	Data    []CU       `json:"data"`
	Items   []CU       `json:"items"`
	Result  []CU       `json:"result"`
	CUs     []CU       `json:"cus"`
	Raw     json.RawMessage
}

// Units возвращает первый непустой список КЕ из известных ключей ответа.
func (r *GetCUResponse) Units() []CU {
	switch {
	case len(r.Data) > 0:
		return r.Data
	case len(r.Items) > 0:
		return r.Items
	case len(r.Result) > 0:
		return r.Result
	case len(r.CUs) > 0:
		return r.CUs
	}
	return []CU{}
}

// GetCU — GET getCU с query-параметрами. Возвращает разобранный ответ и
// сырое тело (удобно сохранить при первой сверке схемы).
func (a *APIClient) GetCU(ctx context.Context, q GetCUQuery) (*GetCUResponse, []byte, error) {
	full := a.c.resolveQuery(a.c.apiPre, "getCU", q.values())
	raw, status, err := a.c.DoRaw(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, nil, err
	}
	if status < 200 || status >= 300 {
		return nil, raw, &APIError{Method: http.MethodGet, URL: full, Status: status, Body: string(raw)}
	}
	out := &GetCUResponse{Raw: append(json.RawMessage(nil), raw...)}
	if len(raw) == 0 {
		return out, raw, nil
	}
	// Ответ-массив: [ {...}, ... ]
	if raw[0] == '[' {
		var list []CU
		if err := json.Unmarshal(raw, &list); err != nil {
			return out, raw, err
		}
		out.Data = list
		return out, raw, nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return out, raw, err
	}
	out.Raw = append(json.RawMessage(nil), raw...)
	return out, raw, nil
}

// GetCUURL возвращает полный URL запроса (для логов/примеров).
func (a *APIClient) GetCUURL(q GetCUQuery) string {
	return a.c.resolveQuery(a.c.apiPre, "getCU", q.values())
}

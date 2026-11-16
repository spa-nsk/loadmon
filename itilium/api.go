package itilium

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// APIClient — методы кастомного HTTP-сервиса `/hs/api` (выгрузка КЕ и др.).
// Контракт getCU: BCS `…/ITILIUM/hs/api/getCU?cuClass=…&cuStatus=…&fields=…`
// (и опционально nextVer/limit). Схема строки КЕ — объекты label/value.
type APIClient struct {
	c *Client
}

const (
	// CUStatusAll — все статусы КЕ (типовой отбор в BCS).
	CUStatusAll = "all"
)

// GetCUQuery — query-параметры GET getCU.
type GetCUQuery struct {
	NextVer  string // курсор страницы; пусто - не передавать
	CUClass  int    // класс КЕ; 0 - не передавать
	CUStatus string // например CUStatusAll; пусто - не передавать
	Fields   []string
	Limit    int
	Extra    url.Values
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
	if q.NextVer != "" {
		v.Set("nextVer", q.NextVer)
	}
	if q.CUClass != 0 {
		v.Set("cuClass", strconv.Itoa(q.CUClass))
	}
	if q.CUStatus != "" {
		v.Set("cuStatus", q.CUStatus)
	}
	if len(q.Fields) > 0 {
		v.Set("fields", strings.Join(q.Fields, ","))
	}
	if q.Limit != 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	return v
}

// DefaultServerCUFields — набор полей для серверных КЕ (cuClass=145).
var DefaultServerCUFields = []string{
	"code", "status", "modelServ", "hostname", "ip", "linksUnit", "sn", "user", "where", "statusHistory",
}

// LabelValue — поле КЕ вида {"label":"…","value":"…"}.
// UnmarshalJSON также принимает голую JSON-строку (тогда Value=строка).
type LabelValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (lv *LabelValue) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*lv = LabelValue{}
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*lv = LabelValue{Value: s}
		return nil
	}
	type alias LabelValue
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*lv = LabelValue(a)
	return nil
}

// String возвращает Value, если не пуст, иначе Label.
func (lv LabelValue) String() string {
	if lv.Value != "" {
		return lv.Value
	}
	return lv.Label
}

// CURow — строка КЕ в ответе getCU (элемент rows).
// Базовые поля — как в структуре ItilliumCu; дополнительные (hostname, ip, …)
// появляются при указании fields= и тоже могут быть label/value или строкой.
type CURow struct {
	RowKey      string     `json:"rowKey"`
	UID         string     `json:"uid"`
	Deleted     bool       `json:"deleted"`
	Status      LabelValue `json:"status"`
	Code        LabelValue `json:"code"`
	SortField   string     `json:"sortField"`
	Name        LabelValue `json:"name"`
	Description LabelValue `json:"description"`
	Author      LabelValue `json:"author"`
	CuClass     string     `json:"cuClass"`
	LinksUnit   string     `json:"linksUnit,omitempty"`

	// Поля из fields= (серверы и др.).
	ModelServ     LabelValue `json:"modelServ,omitempty"`
	Hostname      LabelValue `json:"hostname,omitempty"`
	IP            LabelValue `json:"ip,omitempty"`
	SN            LabelValue `json:"sn,omitempty"`
	User          LabelValue `json:"user,omitempty"`
	Where         LabelValue `json:"where,omitempty"`
	StatusHistory LabelValue `json:"statusHistory,omitempty"`
	StatusName    LabelValue `json:"statusName,omitempty"`
}

// GetCUResponse — ответ GET getCU.
// Обязательное поле — rows. Остальные (пагинация/ошибка) присутствуют
// не на всех публикациях; при отсутствии остаются нулевыми.
type GetCUResponse struct {
	IsError    bool    `json:"isError"`
	Rows       []CURow `json:"rows"`
	TotalCount int     `json:"totalCount"`
	Last       string  `json:"last"`
	HasMore    bool    `json:"hasMore"`
}

// NextPageQuery копирует q и подставляет NextVer = Last.
// ok=false, если HasMore=false или Last пуст.
func (r *GetCUResponse) NextPageQuery(q GetCUQuery) (GetCUQuery, bool) {
	if r == nil || !r.HasMore || r.Last == "" {
		return q, false
	}
	q.NextVer = r.Last
	return q, true
}

// GetCU — GET getCU. При IsError=true возвращает ошибку (HTTP может быть 200).
func (a *APIClient) GetCU(ctx context.Context, q GetCUQuery) (*GetCUResponse, error) {
	full := a.c.resolveQuery(a.c.apiPre, "getCU", q.values())
	var out GetCUResponse
	if err := a.c.doJSON(ctx, http.MethodGet, full, nil, &out); err != nil {
		return nil, err
	}
	if out.Rows == nil {
		out.Rows = []CURow{}
	}
	if out.IsError {
		return &out, fmt.Errorf("itilium: getCU isError=true (rows=%d last=%q)", len(out.Rows), out.Last)
	}
	return &out, nil
}

// GetCURaw — как GetCU, плюс сырое тело ответа.
func (a *APIClient) GetCURaw(ctx context.Context, q GetCUQuery) (*GetCUResponse, []byte, error) {
	full := a.c.resolveQuery(a.c.apiPre, "getCU", q.values())
	raw, status, err := a.c.DoRaw(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, nil, err
	}
	if status < 200 || status >= 300 {
		return nil, raw, &APIError{Method: http.MethodGet, URL: full, Status: status, Body: string(raw)}
	}
	var out GetCUResponse
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, raw, fmt.Errorf("itilium: decode getCU: %w", err)
		}
	}
	if out.Rows == nil {
		out.Rows = []CURow{}
	}
	if out.IsError {
		return &out, raw, fmt.Errorf("itilium: getCU isError=true (rows=%d last=%q)", len(out.Rows), out.Last)
	}
	return &out, raw, nil
}

// GetCUURL возвращает полный URL запроса (для логов/примеров).
func (a *APIClient) GetCUURL(q GetCUQuery) string {
	return a.c.resolveQuery(a.c.apiPre, "getCU", q.values())
}

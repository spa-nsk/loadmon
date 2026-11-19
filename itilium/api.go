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

// DefaultServerCUFields — fields= для cuClass=145 (каталог «Список полей КЕ»
// + supplemental). Предпочтительно DefaultFieldsForClass(CUClassServers);
// эта переменная — снимок для обратной совместимости флагов tools.
var DefaultServerCUFields = DefaultFieldsForClass(CUClassServers)

// LabelValue — поле КЕ вида {"label":"…","value":"…"}.
// UnmarshalJSON также принимает голую JSON-строку (тогда Value=строка).
// Value — any: в ответах 1С бывает строка, число или bool.
type LabelValue struct {
	Label string `json:"label"`
	Value any    `json:"value"`
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

// ValueString — строковое представление Value ("" если Value nil/пусто).
func (lv LabelValue) ValueString() string {
	switch v := lv.Value.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// String возвращает Value, раньше для пустого возвращалась label
func (lv LabelValue) String() string {
	return lv.ValueString()
}

// LinksUnitField — поле linksUnit: JSON-строка, один LabelValue или массив
// LabelValue. Нормализуется в срез элементов.
type LinksUnitField []LabelValue

func (f *LinksUnitField) UnmarshalJSON(data []byte) error {
	data = bytesTrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*f = nil
		return nil
	}
	switch data[0] {
	case '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		if s == "" {
			*f = nil
			return nil
		}
		*f = LinksUnitField{{Value: s}}
		return nil
	case '[':
		var list []LabelValue
		if err := json.Unmarshal(data, &list); err != nil {
			return err
		}
		*f = LinksUnitField(list)
		return nil
	case '{':
		var one LabelValue
		if err := json.Unmarshal(data, &one); err != nil {
			return err
		}
		*f = LinksUnitField{one}
		return nil
	default:
		return fmt.Errorf("itilium: linksUnit: неожиданный JSON %s", truncate(string(data), 64))
	}
}

// String склеивает значения элементов через "; " (для логов/краткого вывода).
func (f LinksUnitField) String() string {
	if len(f) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f))
	for _, lv := range f {
		if s := lv.String(); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "; ")
}

// CURow — строка КЕ в ответе getCU (элемент rows).
// Базовые поля — как в структуре ItilliumCu; дополнительные (hostname, ip, …)
// появляются при указании fields= и тоже могут быть label/value или строкой.
type CURow struct {
	RowKey      string         `json:"rowKey"`
	UID         string         `json:"uid"`
	Deleted     bool           `json:"deleted"`
	Status      LabelValue     `json:"status"`
	Code        LabelValue     `json:"code"`
	SortField   string         `json:"sortField"`
	Name        LabelValue     `json:"name"`
	Description LabelValue     `json:"description"`
	Author      LabelValue     `json:"author"`
	CuClass     string         `json:"cuClass"`
	LinksUnit   LinksUnitField `json:"linksUnit,omitempty"` // строка, LabelValue или []LabelValue

	// --- CU 145 (Серверы): коды из «Список полей КЕ.xls» -----------------
	Legal      LabelValue `json:"legal,omitempty"`     // Юр.лицо
	OS         LabelValue `json:"os,omitempty"`        // Операционная система
	Domain     LabelValue `json:"domain,omitempty"`    // Домен
	OneView    LabelValue `json:"oneview,omitempty"`   // OneView
	Role       LabelValue `json:"role,omitempty"`      // Роль
	DataDocs   LabelValue `json:"dataDocs,omitempty"`  // Дата документа закупки
	RAM1       LabelValue `json:"ram_1,omitempty"`     // RAM (1)
	CRAM1      LabelValue `json:"c_ram_1,omitempty"`   // RAM (1) (кол-во)
	RAM2       LabelValue `json:"ram_2,omitempty"`     // RAM (2)
	CRAM2      LabelValue `json:"c_ram_2,omitempty"`   // RAM (2) (кол-во)
	ILOIP      LabelValue `json:"iloip,omitempty"`     // ILO IP
	IPMI       LabelValue `json:"ipmi,omitempty"`      // IPMI
	Complect   LabelValue `json:"complect,omitempty"`  // Комплектация
	ModelServ  LabelValue `json:"modelServ,omitempty"` // Марка/модель(Норматив)
	Product    LabelValue `json:"product,omitempty"`   // Продукт
	IP         LabelValue `json:"ip,omitempty"`
	Installed  LabelValue `json:"installed,omitempty"`  // Дата установки
	Where      LabelValue `json:"where,omitempty"`      // Местоположение
	RepairDate LabelValue `json:"repairDate,omitempty"` // Дата ремонта
	Comments   LabelValue `json:"comments,omitempty"`   // Примечание
	Hostname   LabelValue `json:"hostname,omitempty"`
	User       LabelValue `json:"user,omitempty"` // Пользователь / Excel «Владелец»
	Docs       LabelValue `json:"docs,omitempty"` // Документ закупки
	Provider   LabelValue `json:"provider,omitempty"`
	SN         LabelValue `json:"sn,omitempty"`
	Net        LabelValue `json:"net,omitempty"` // NET / сетевой адаптер
	RAID       LabelValue `json:"raid,omitempty"`
	RAID2      LabelValue `json:"raid_2,omitempty"`
	CCPU       LabelValue `json:"c_cpu,omitempty"` // Процессор (кол-во)
	CPU        LabelValue `json:"cpu,omitempty"`
	HDD        LabelValue `json:"hdd,omitempty"`
	HDD2       LabelValue `json:"hdd_2,omitempty"`
	HDD3       LabelValue `json:"hdd_3,omitempty"`
	HDD4       LabelValue `json:"hdd_4,omitempty"`
	HDD5       LabelValue `json:"hdd_5,omitempty"`
	HDD6       LabelValue `json:"hdd_6,omitempty"`
	HDD7       LabelValue `json:"hdd_7,omitempty"`
	HDD8       LabelValue `json:"hdd_8,omitempty"`
	Video      LabelValue `json:"video,omitempty"`
	MB         LabelValue `json:"mb,omitempty"` // материнская плата
	BuhLegal   LabelValue `json:"buhLegal,omitempty"`
	BuhStatus  LabelValue `json:"buhStatus,omitempty"`
	BuhCode    LabelValue `json:"buhCode,omitempty"` // Бухгалтерский инв.№
	BuhComment LabelValue `json:"buhComment,omitempty"`
	BuhInv     LabelValue `json:"buhInv,omitempty"`
	EOS        LabelValue `json:"eos,omitempty"`
	EOX        LabelValue `json:"eox,omitempty"`
	PPribor    LabelValue `json:"pPribor,omitempty"`

	// --- Supplemental (есть в getCU / импорте, нет в xls-каталоге) --------
	ModelVendS      LabelValue `json:"modelVendS,omitempty"` // Модель от производителя
	Units           LabelValue `json:"units,omitempty"`      // Высота в юнитах
	StatusHistory   LabelValue `json:"statusHistory,omitempty"`
	StatusHistoryYY string     `json:"statusHistoryYY,omitempty"`
	StatusName      LabelValue `json:"statusName,omitempty"`
	Currency        LabelValue `json:"currency,omitempty"`
	CurrencyA       LabelValue `json:"currencyA,omitempty"`
	DateChPriA      LabelValue `json:"dateChPriA,omitempty"`
	DateChPric      LabelValue `json:"dateChPric,omitempty"`
	IsActual        LabelValue `json:"isActual,omitempty"`
	PriceA          LabelValue `json:"priceA,omitempty"`
	PriceCommA      LabelValue `json:"priceCommA,omitempty"`
	PriceComme      LabelValue `json:"priceComme,omitempty"`
	PriceGPL        LabelValue `json:"priceGPL,omitempty"`
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

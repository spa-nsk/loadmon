package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"loadmon/itilium"
	"loadmon/itilliumimport"
)

// Соответствие getCU (cuClass=145) ↔ update/rack_server.yaml.
// Запись в БД: itilliumimport.ImportRackServersFromCU (-import).

var unitRangeRe = regexp.MustCompile(`^\s*(\d+)\s*-\s*(\d+)\s*$`)

// LocationParts — разбор where / «Местоположение» по «|»
// (как composite location_split).
type LocationParts struct {
	Raw      string
	SiteText string // город + площадка
	ZoneCode string
	RoomCode string
	Tail     string // всё после 3-го сегмента (шкаф/unit в where)
}

// PlacementParts — разбор linksUnit / «Размещение в шкафу/стойке»
// вида «6.2-A-01 | 23-23» (как composite rack_unit_split).
type PlacementParts struct {
	Raw       string
	Cabinet   string
	StartUnit int
	SizeUnit  int
	OK        bool
}

// RackServerView — плоский вид строки getCU в терминах Excel/импорта.
type RackServerView struct {
	// Идентификаторы
	TechNumberIT string // code → Тех. номер IT / Инв.номер IT
	UID          string
	OldInvNum    string // name → Стар.инв.№
	Hostname     string // hostname → Имя (hostname)
	IP           string
	SerialNum    string // sn → S/N

	// Модель / описание
	NormativeModel string // modelServ → Марка/модель(Норматив)
	VendorModel    string // modelVendS → Модель от производителя
	Description    string // description → Описание
	Author         string
	User           string // user → Пользователь / Excel «Владелец»

	// Статус (кортеж как в update/rack_server.yaml enums.status)
	StatusLabel     string
	StatusCode      string
	StatusHistory   string
	StatusHistoryYY string
	Stat            int // ServerSprRecords.Stat
	PowerState      int // ServerSprRecords.PowerState
	Active          int // ServerSprRecords.Active
	HwPowerState    int
	HwActive        int
	ItiliumStatus   int // ServerHardware.ItiliumStatus
	StatusKnown     bool

	// Размещение
	Location  LocationParts
	Placement PlacementParts
	UnitsH    int // units.value — «Высота в юнитах» (норматив)

	Deleted bool
	CuClass string
}

func parseLocation(raw string) LocationParts {
	p := LocationParts{Raw: strings.TrimSpace(raw)}
	if p.Raw == "" {
		return p
	}
	parts := strings.Split(p.Raw, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) > 0 {
		p.SiteText = parts[0]
	}
	if len(parts) > 1 {
		p.ZoneCode = parts[1]
	}
	if len(parts) > 2 {
		p.RoomCode = parts[2]
	}
	if len(parts) > 3 {
		p.Tail = strings.Join(parts[3:], " | ")
	}
	return p
}

func parsePlacement(raw string) PlacementParts {
	p := PlacementParts{Raw: strings.TrimSpace(raw)}
	if p.Raw == "" {
		return p
	}
	parts := strings.SplitN(p.Raw, "|", 2)
	p.Cabinet = strings.TrimSpace(parts[0])
	if len(parts) < 2 {
		p.OK = p.Cabinet != ""
		return p
	}
	m := unitRangeRe.FindStringSubmatch(strings.TrimSpace(parts[1]))
	if m == nil {
		return p
	}
	start, _ := strconv.Atoi(m[1])
	end, _ := strconv.Atoi(m[2])
	size := end - start + 1
	if size < 1 {
		size = 1
	}
	p.StartUnit, p.SizeUnit, p.OK = start, size, true
	return p
}

func mapStatus(label string) (itilliumimport.StatusMapped, bool) {
	return itilliumimport.MapCUStatusLabel(label)
}

func unitsInt(lv itilium.LabelValue) int {
	s := strings.TrimSpace(lv.ValueString())
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// MapRackServer строит Excel-совместимый вид из CURow (cuClass=145).
func MapRackServer(r itilium.CURow) RackServerView {
	st, known := mapStatus(r.Status.Label)
	return RackServerView{
		TechNumberIT:    r.Code.String(),
		UID:             firstNonEmpty(r.UID, r.RowKey, r.Code.String()),
		OldInvNum:       r.Name.String(),
		Hostname:        r.Hostname.String(),
		IP:              r.IP.String(),
		SerialNum:       r.SN.String(),
		NormativeModel:  r.ModelServ.String(),
		VendorModel:     r.ModelVendS.String(),
		Description:     r.Description.String(),
		Author:          r.Author.String(),
		User:            r.User.String(),
		StatusLabel:     r.Status.Label,
		StatusCode:      r.Status.ValueString(),
		StatusHistory:   r.StatusHistory.String(),
		StatusHistoryYY: strings.TrimSpace(r.StatusHistoryYY),
		Stat:            st.Stat,
		PowerState:      st.Power,
		Active:          st.Active,
		HwPowerState:    st.HwPower,
		HwActive:        st.HwActive,
		ItiliumStatus:   st.ItiliumStatus,
		StatusKnown:     known,
		Location:        parseLocation(r.Where.String()),
		Placement:       parsePlacement(r.LinksUnit.String()),
		UnitsH:          unitsInt(r.Units),
		Deleted:         r.Deleted,
		CuClass:         r.CuClass,
	}
}

// EffectiveSizeUnit: приоритет диапазону из linksUnit, иначе units.
func (v RackServerView) EffectiveSizeUnit() int {
	if v.Placement.OK && v.Placement.SizeUnit > 0 {
		return v.Placement.SizeUnit
	}
	if v.UnitsH > 0 {
		return v.UnitsH
	}
	return 0
}

func printRackAnalyze(rows []itilium.CURow) {
	var (
		n          = len(rows)
		withPlace  int
		withWhere  int
		unknownSt  int
		noSN       int
		noHost     int
		statusHist = map[string]int{}
	)
	limit := n
	if limit > 20 {
		limit = 20
	}
	fmt.Printf("\n=== analyze rack (cuClass=145): rows=%d (показ первых %d) ===\n", n, limit)
	fmt.Printf("%-16s %-18s %-22s %-18s %-12s %-10s %-10s %s\n",
		"techIT", "hostname", "sn", "status", "cabinet", "units", "ItilSt", "where/site")
	for i := 0; i < n; i++ {
		v := MapRackServer(rows[i])
		statusHist[v.StatusLabel]++
		if v.Location.Raw != "" {
			withWhere++
		}
		if v.Placement.OK {
			withPlace++
		}
		if !v.StatusKnown && v.StatusLabel != "" {
			unknownSt++
		}
		if v.SerialNum == "" {
			noSN++
		}
		if v.Hostname == "" {
			noHost++
		}
		if i >= limit {
			continue
		}
		cab := v.Placement.Cabinet
		unit := ""
		if v.Placement.OK {
			unit = fmt.Sprintf("%d+%d", v.Placement.StartUnit, v.Placement.SizeUnit)
		} else if v.UnitsH > 0 {
			unit = fmt.Sprintf("H=%d", v.UnitsH)
		}
		itil := "?"
		if v.StatusKnown {
			itil = fmt.Sprintf("%d A=%d", v.ItiliumStatus, v.Active)
		} else if v.StatusLabel != "" {
			itil = "raw"
		}
		fmt.Printf("%-16s %-18s %-22s %-18s %-12s %-10s %-10s %s\n",
			trunc(v.TechNumberIT, 16), trunc(v.Hostname, 18), trunc(v.SerialNum, 22),
			trunc(v.StatusLabel, 18), trunc(cab, 12), trunc(unit, 10), itil,
			trunc(v.Location.SiteText, 48))
	}
	fmt.Printf("--- сводка: where=%d/%d linksUnit(ok)=%d/%d noSN=%d noHost=%d unknownStatus=%d\n",
		withWhere, n, withPlace, n, noSN, noHost, unknownSt)
	fmt.Printf("--- статусы (словарь update/rack_server.yaml):\n")
	for k, c := range statusHist {
		label := k
		if label == "" {
			label = "(пусто)"
		}
		_, ok := mapStatus(k)
		mark := ""
		if k != "" && !ok {
			mark = "  ← нет в enums.status"
		}
		fmt.Printf("  %5d  %s%s\n", c, label, mark)
	}
	fmt.Printf("--- маппинг Excel↔getCU (эталон update/rack_server.yaml):\n")
	fmt.Print(rackFieldMapHelp())
}

func trunc(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func rackFieldMapHelp() string {
	return `  Excel «Тех. номер IT»          ← code (Инв.номер IT) → TechnicalNumberIT
  Excel «Стар.инв.№»             ← name → InventoryNum
  Excel «Имя (hostname)»         ← hostname → Name/NameFQDN
  Excel «ip»                     ← ip → IPAddress
  Excel «S/N»                    ← sn → SerialNum
  Excel «Местоположение»         ← where   (разбор «|» → site|zone|room)
  Excel «Размещение в шкафу/стойке» ← linksUnit (шкаф | start-end)
  Excel «Марка/модель(Норматив)» ← modelServ → extra.normative_model
  Excel «Модель от производителя»← modelVendS → ServerHardware.Model
  Excel «Статус»                 ← status.label → Stat|Power|Active|…|ItiliumStatus
  Excel «Автор»                  ← author → extra.author_name
  Excel «Владелец»               ← user (API «Пользователь») → Owner
  Excel «Описание…»              ← description → Details
  (нет в Excel) Высота в юнитах  ← units
  (нет в Excel) Дата статуса     ← statusHistory / statusHistoryYY
  (нет в Excel) Стоимость/валюта ← priceGPL, currency, isActual, …
  Excel есть, в getCU samples нет: Бух.инв.№, Продукт, ОС, Роль, CPU, IPMI,
  Память(1/2), HDD/RAID, Документы, Поставщик, Юр.Лицо, Комплектация…
`
}

package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"loadmon/itilium"
	"loadmon/locality"
)

// Проверка getCU-выгрузки rack-серверов по правилам распознавания импорта
// (update/rack_server.yaml + locality.ParseLocation/ParsePlacement +
// cabinetFromPlacement/splitColocation). Без записи в БД.

const (
	sevError    = "error"
	sevWarn     = "warn"
	sevQuestion = "question"
)

// CheckIssue — одно замечание/неоднозначность по строке.
type CheckIssue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // error | warn | question
	Message  string `json:"message"`
	Question string `json:"question,omitempty"`
}

// CheckRow — результат разбора одной КЕ для импорта.
type CheckRow struct {
	Index int `json:"index"`

	TechNumberIT string `json:"techNumberIT"`
	UID          string `json:"uid"`
	OldInvNum    string `json:"oldInvNum"`
	Hostname     string `json:"hostname"`
	IP           string `json:"ip"`
	SerialNum    string `json:"serialNum"`
	StatusLabel  string `json:"statusLabel"`
	StatusCode   string `json:"statusCode"`
	Deleted      bool   `json:"deleted"`

	WhereRaw     string `json:"whereRaw"`
	LinksUnitRaw string `json:"linksUnitRaw"`
	VendorModel  string `json:"vendorModel"`
	NormModel    string `json:"normativeModel"`
	UnitsH       int    `json:"unitsHeight"`

	// Распознавание как у импорта
	Locality         string `json:"locality,omitempty"`
	LocalityDesc     string `json:"localityDescription,omitempty"`
	Street           string `json:"street,omitempty"`
	House            string `json:"house,omitempty"`
	Floor            string `json:"floor,omitempty"`
	Room             string `json:"room,omitempty"`
	LocCabinet       string `json:"locationCabinet,omitempty"`
	LocUnit          string `json:"locationUnit,omitempty"`
	ServiceLocation  bool   `json:"serviceLocation,omitempty"`
	PlacementCabinet string `json:"placementCabinet,omitempty"`
	PlacementAlt     string `json:"placementAltName,omitempty"`
	ColocationDC     string `json:"colocationDC,omitempty"`
	StartUnit        int    `json:"startUnit,omitempty"`
	SizeUnit         int    `json:"sizeUnit,omitempty"`
	EffectiveCabinet string `json:"effectiveCabinet,omitempty"`
	OutsideRack      bool   `json:"outsideRack,omitempty"`

	Stat          int  `json:"sprStat"`
	PowerState    int  `json:"sprPowerState"`
	Active        int  `json:"sprActive"`
	HwPowerState  int  `json:"hwPowerState"`
	HwActive      int  `json:"hwActive"`
	ItiliumStatus int  `json:"itiliumStatus"`
	StatusKnown   bool `json:"statusKnown"`

	OK     bool         `json:"ok"`
	Issues []CheckIssue `json:"issues,omitempty"`
}

// ImportCheckReport — полная проверка выгрузки.
type ImportCheckReport struct {
	GeneratedAt string         `json:"generatedAt"`
	Source      string         `json:"source"`
	RulesRef    string         `json:"rulesRef"`
	TotalRows   int            `json:"totalRows"`
	OKCount     int            `json:"okCount"`
	IssueCount  int            `json:"issueCount"`
	ByCode      map[string]int `json:"issueCountsByCode"`
	Rows        []CheckRow     `json:"rows"`
}

// QuestionsReport — только проблемные строки + вопросы.
type QuestionsReport struct {
	GeneratedAt string         `json:"generatedAt"`
	Source      string         `json:"source"`
	TotalRows   int            `json:"totalRows"`
	ProblemRows int            `json:"problemRows"`
	ByCode      map[string]int `json:"issueCountsByCode"`
	Items       []QuestionItem `json:"items"`
}

// QuestionItem — строка с неоднозначностями для ручного разбора.
type QuestionItem struct {
	Index        int          `json:"index"`
	TechNumberIT string       `json:"techNumberIT"`
	Hostname     string       `json:"hostname"`
	SerialNum    string       `json:"serialNum"`
	StatusLabel  string       `json:"statusLabel"`
	WhereRaw     string       `json:"whereRaw"`
	LinksUnitRaw string       `json:"linksUnitRaw"`
	Issues       []CheckIssue `json:"issues"`
	Questions    []string     `json:"questions"`
}

func validateDump(rows []itilium.CURow, source string) (ImportCheckReport, QuestionsReport) {
	now := time.Now().Format(time.RFC3339)
	rep := ImportCheckReport{
		GeneratedAt: now,
		Source:      source,
		RulesRef:    "update/rack_server.yaml + locality.ParseLocation/ParsePlacement",
		TotalRows:   len(rows),
		ByCode:      map[string]int{},
		Rows:        make([]CheckRow, 0, len(rows)),
	}

	// Индексы дублей по всей выгрузке.
	byCode := map[string][]int{}
	bySN := map[string][]int{}
	byHost := map[string][]int{}
	byUID := map[string][]int{}

	checks := make([]CheckRow, len(rows))
	for i, row := range rows {
		cr := checkOneRow(i, row)
		checks[i] = cr
		if t := strings.TrimSpace(cr.TechNumberIT); t != "" {
			byCode[t] = append(byCode[t], i)
		}
		if t := strings.TrimSpace(cr.SerialNum); t != "" {
			bySN[strings.ToUpper(t)] = append(bySN[strings.ToUpper(t)], i)
		}
		if t := strings.ToLower(strings.TrimSpace(cr.Hostname)); t != "" {
			byHost[t] = append(byHost[t], i)
		}
		if t := strings.TrimSpace(cr.UID); t != "" {
			byUID[t] = append(byUID[t], i)
		}
	}

	addDup := func(idxs []int, code, what string) {
		if len(idxs) < 2 {
			return
		}
		others := make([]string, 0, len(idxs))
		for _, j := range idxs {
			others = append(others, checks[j].TechNumberIT)
		}
		for _, j := range idxs {
			q := fmt.Sprintf("В выгрузке несколько КЕ с одинаковым %s (%v). Какая запись каноническая для импорта?", what, others)
			issue := CheckIssue{
				Code: code, Severity: sevError,
				Message:  fmt.Sprintf("дубль %s среди индексов %v", what, idxs),
				Question: q,
			}
			checks[j].Issues = append(checks[j].Issues, issue)
			checks[j].OK = false
		}
	}
	for _, idxs := range byCode {
		addDup(idxs, "dup_tech_number", "Тех. номер IT (code)")
	}
	for _, idxs := range bySN {
		addDup(idxs, "dup_serial", "S/N")
	}
	for _, idxs := range byHost {
		addDup(idxs, "dup_hostname", "hostname")
	}
	for _, idxs := range byUID {
		addDup(idxs, "dup_uid", "uid")
	}

	qrep := QuestionsReport{
		GeneratedAt: now,
		Source:      source,
		TotalRows:   len(rows),
		ByCode:      map[string]int{},
		Items:       nil,
	}

	for _, cr := range checks {
		if len(cr.Issues) == 0 {
			cr.OK = true
			rep.OKCount++
		} else {
			cr.OK = false
			rep.IssueCount++
			qs := make([]string, 0, len(cr.Issues))
			for _, is := range cr.Issues {
				rep.ByCode[is.Code]++
				qrep.ByCode[is.Code]++
				if is.Question != "" {
					qs = append(qs, is.Question)
				}
			}
			qrep.Items = append(qrep.Items, QuestionItem{
				Index:        cr.Index,
				TechNumberIT: cr.TechNumberIT,
				Hostname:     cr.Hostname,
				SerialNum:    cr.SerialNum,
				StatusLabel:  cr.StatusLabel,
				WhereRaw:     cr.WhereRaw,
				LinksUnitRaw: cr.LinksUnitRaw,
				Issues:       cr.Issues,
				Questions:    qs,
			})
		}
		rep.Rows = append(rep.Rows, cr)
	}
	qrep.ProblemRows = len(qrep.Items)
	return rep, qrep
}

func checkOneRow(index int, row itilium.CURow) CheckRow {
	v := MapRackServer(row)
	cr := CheckRow{
		Index:         index,
		TechNumberIT:  v.TechNumberIT,
		UID:           v.UID,
		OldInvNum:     v.OldInvNum,
		Hostname:      v.Hostname,
		IP:            v.IP,
		SerialNum:     v.SerialNum,
		StatusLabel:   v.StatusLabel,
		StatusCode:    v.StatusCode,
		Deleted:       v.Deleted,
		WhereRaw:      v.Location.Raw,
		LinksUnitRaw:  v.Placement.Raw,
		VendorModel:   v.VendorModel,
		NormModel:     v.NormativeModel,
		UnitsH:        v.UnitsH,
		Stat:          v.Stat,
		PowerState:    v.PowerState,
		Active:        v.Active,
		HwPowerState:  v.HwPowerState,
		HwActive:      v.HwActive,
		ItiliumStatus: v.ItiliumStatus,
		StatusKnown:   v.StatusKnown,
		OK:            true,
	}

	add := func(code, sev, msg, question string) {
		cr.Issues = append(cr.Issues, CheckIssue{Code: code, Severity: sev, Message: msg, Question: question})
		cr.OK = false
	}

	inService := v.StatusKnown && v.Active == 1 && v.ItiliumStatus == 1

	if cr.TechNumberIT == "" {
		add("missing_tech_number", sevError, "пустой code (Тех. номер IT)",
			"У КЕ нет Инв.номера IT. Чем ключюем запись при импорте?")
	}
	if cr.SerialNum == "" {
		sev := sevWarn
		q := "S/N пуст — импорт подставит Тех. номер IT как SerialNum. Это приемлемо?"
		if inService {
			sev = sevError
			q = "Сервер «в работе», но S/N пуст. Какой серийный номер использовать?"
		}
		add("missing_sn", sev, "пустой sn", q)
	}
	if cr.Hostname == "" && inService {
		add("missing_hostname", sevWarn, "пустой hostname при статусе «в работе»",
			"Hostname пуст при статусе в работе. Оставляем Name=Тех.номер / описание?")
	}
	if cr.VendorModel == "" && inService {
		add("missing_vendor_model", sevQuestion, "пустой modelVendS (Модель от производителя)",
			"Нет «Модель от производителя». Брать modelServ (норматив) в ServerHardware.Model?")
	}

	if cr.StatusLabel == "" {
		add("missing_status", sevError, "пустой status",
			"Статус пуст. Какой ItiliumStatus/Active ставить?")
	} else if !v.StatusKnown {
		add("unknown_status", sevError,
			fmt.Sprintf("статус %q нет в update/rack_server.yaml enums.status", cr.StatusLabel),
			fmt.Sprintf("Статус %q (value=%s) отсутствует в словаре импорта. Как маппить в Stat/Power/Active/ItiliumStatus?", cr.StatusLabel, cr.StatusCode))
	} else if cr.StatusLabel == "В работе (эксплуатация)" {
		add("status_spelling", sevQuestion,
			"в getCU «эксплуатация», в YAML — «эксплуатации»",
			"В API label «В работе (эксплуатация)», в update/rack_server.yaml ключ «В работе (эксплуатации)». Считать эквивалентом или править YAML/1С?")
	}

	if cr.Deleted {
		add("deleted_flag", sevWarn, "deleted=true",
			"КЕ помечена deleted=true. Пропускать при импорте или деактивировать существующую запись?")
	}

	if ip := strings.TrimSpace(cr.IP); ip != "" {
		if net.ParseIP(ip) == nil {
			add("bad_ip", sevWarn, fmt.Sprintf("ip %q не похож на IP", ip),
				fmt.Sprintf("Поле ip=%q не парсится как IP. Игнорировать или править в Итилиум?", ip))
		}
	}

	// Местоположение — locality.ParseLocation (как resolveSite).
	// Без справочника ОКТМО: isLocality всегда false → первый токен = locality (как в ParseLocation fallback).
	loc := locality.ParseLocation(cr.WhereRaw, func(string) bool { return false })
	cr.Locality = loc.Locality
	cr.LocalityDesc = loc.Description
	cr.Street = loc.Street
	cr.House = loc.House
	cr.Floor = loc.Floor
	cr.Room = loc.Room
	cr.LocCabinet = loc.Cabinet
	if loc.Pos != nil {
		cr.LocUnit = loc.Pos.String()
	}
	cr.ServiceLocation = loc.Service

	if cr.WhereRaw == "" {
		if inService {
			add("missing_where", sevError, "пустое where (Местоположение) при «в работе»",
				"Нет Местоположения у сервера в работе. Какая площадка/ЦОД?")
		} else {
			add("missing_where", sevWarn, "пустое where (Местоположение)",
				"Местоположение пусто. Оставляем без площадки (DatacenterSite=0)?")
		}
	} else if loc.Service {
		add("service_location", sevQuestion, "служебная отметка в where (! …)",
			fmt.Sprintf("where=%q — служебная отметка. Создавать площадку с этим именем или не размещать?", cr.WhereRaw))
	} else if loc.Locality == "" {
		add("where_no_locality", sevQuestion, "не удалось выделить населённый пункт",
			fmt.Sprintf("Не разобран населённый пункт из where=%q. Верна ли строка?", cr.WhereRaw))
	}

	// Размещение — locality.ParsePlacement (+ colocation «Альт / шкаф»).
	pl := locality.ParsePlacement(cr.LinksUnitRaw)
	if pl != nil {
		cr.PlacementCabinet = pl.Cabinet
		cr.PlacementAlt = pl.AltName
		if pl.Pos != nil {
			cr.StartUnit = pl.Pos.Start
			cr.SizeUnit = pl.Pos.End - pl.Pos.Start + 1
			if cr.SizeUnit < 1 {
				cr.SizeUnit = 1
			}
		}
		// colocation: AltName в ParsePlacement уже часть до последнего "/"
		if pl.AltName != "" {
			cr.ColocationDC = pl.AltName
			add("colocation_placement", sevQuestion,
				fmt.Sprintf("linksUnit с префиксом «%s / %s»", pl.AltName, pl.Cabinet),
				fmt.Sprintf("Размещение «%s». Импорт пишет шкаф %q на площадку из where, а %q считает AltName. Верно ли это для данной строки?",
					cr.LinksUnitRaw, pl.Cabinet, pl.AltName))
		}
		if cr.LinksUnitRaw != "" && pl.Cabinet == "" {
			add("placement_no_cabinet", sevError, "linksUnit без кода шкафа",
				fmt.Sprintf("linksUnit=%q не дал код шкафа. Как размещать?", cr.LinksUnitRaw))
		}
		if cr.LinksUnitRaw != "" && strings.Contains(cr.LinksUnitRaw, "|") && pl.Pos == nil {
			add("placement_bad_units", sevError,
				fmt.Sprintf("не разобран диапазон юнитов в linksUnit=%q", cr.LinksUnitRaw),
				fmt.Sprintf("Ожидается «шкаф | N-M», получено %q. Какой StartUnit/SizeUnit?", cr.LinksUnitRaw))
		}
	} else if inService {
		// нет linksUnit — шкаф из where или «вне шкафа»
		if loc.Cabinet != "" {
			add("placement_from_where", sevQuestion,
				"linksUnit пуст, шкаф взят из Местоположения",
				fmt.Sprintf("Нет «Размещение в шкафу/стойке», в where есть шкаф %q (unit %s). Использовать его?", loc.Cabinet, cr.LocUnit))
		} else if cr.WhereRaw != "" && !loc.Service {
			cr.OutsideRack = true
			add("outside_rack", sevQuestion,
				"нет linksUnit и шкафа в where — импорт заведёт «вне шкафа»",
				fmt.Sprintf("Сервер в работе, where=%q без шкафа. Размещать в служебном шкафу «вне шкафа» площадки?", cr.WhereRaw))
		}
	}

	// Эффективный шкаф как cabinetFromPlacement.
	cab := cr.PlacementCabinet
	if cab == "" && loc.Cabinet != "" {
		cab = loc.Cabinet
		if cr.StartUnit == 0 && loc.Pos != nil {
			cr.StartUnit = loc.Pos.Start
			cr.SizeUnit = loc.Pos.End - loc.Pos.Start + 1
			if cr.SizeUnit < 1 {
				cr.SizeUnit = 1
			}
		}
	}
	cr.EffectiveCabinet = cab
	if cab == "" && cr.WhereRaw != "" && !loc.Service && inService && pl == nil {
		cr.OutsideRack = true
	}

	if cr.UnitsH > 0 && cr.SizeUnit > 0 && cr.UnitsH != cr.SizeUnit {
		add("units_mismatch", sevQuestion,
			fmt.Sprintf("units=%d, из linksUnit SizeUnit=%d", cr.UnitsH, cr.SizeUnit),
			fmt.Sprintf("«Высота в юнитах»=%d, диапазон размещения даёт SizeUnit=%d. Что авторитетно для ServerHardware.SizeUnit?", cr.UnitsH, cr.SizeUnit))
	}

	return cr
}

package itilliumimport

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"loadmon/itilium"
	"loadmon/locality"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// ImportCURackStats — итог ImportRackServersFromCU.
type ImportCURackStats struct {
	Total     int
	Inserted  int
	Updated   int
	Skipped   int
	Inactive  int
	NoLoc     int // списан / в пути / «! Покинуло ФГ» — площадка не резолвилась
	UnknownSt int
	Errors    int
}

// leftFGPrefix — where, начинающееся с этой строки, не создаёт площадку.
const leftFGPrefix = "! Покинуло ФГ"

// locationUndefinedForCU — площадка/шкаф не определяются (это нормально).
func locationUndefinedForCU(where string, itiliumStatus int) bool {
	if models.ItiliumStatusHiddenFromPlans(itiliumStatus) {
		return true // списан / в пути
	}
	return strings.HasPrefix(strings.TrimSpace(where), leftFGPrefix)
}

// Маркеры в description («Описание»): если modelVendS и modelServ пусты,
// а описание содержит один из них — description считается моделью сервера.
var cuModelDescMarkers = []string{"HP", "Chieftec", "DELL", "Inflex", "DEPO", "Cisco"}

func cuModel(row itilium.CURow) string {
	if m := strings.TrimSpace(row.ModelVendS.String()); m != "" {
		return m
	}
	if m := strings.TrimSpace(row.ModelServ.String()); m != "" {
		return m
	}
	desc := strings.TrimSpace(row.Description.String())
	if desc == "" {
		return ""
	}
	upper := strings.ToUpper(desc)
	for _, marker := range cuModelDescMarkers {
		if strings.Contains(upper, strings.ToUpper(marker)) {
			return desc
		}
	}
	return ""
}

func cuTrim(lv itilium.LabelValue) string { return strings.TrimSpace(lv.String()) }

func cuMpIP(row itilium.CURow) string {
	if s := cuTrim(row.ILOIP); s != "" {
		return s
	}
	return cuTrim(row.IPMI)
}

func cuDetails(row itilium.CURow) string {
	parts := make([]string, 0, 3)
	for _, s := range []string{cuTrim(row.Description), cuTrim(row.Comments), cuTrim(row.Complect)} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "; ")
}

var cuMemoryGbRe = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(?:gb|гб)\b`)

func cuMemoryMb(row itilium.CURow) int {
	var totalGb float64
	found := false
	add := func(spec, countStr string) {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			return
		}
		m := cuMemoryGbRe.FindStringSubmatch(spec)
		if m == nil {
			return
		}
		gb, err := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
		if err != nil || gb <= 0 {
			return
		}
		count := 1.0
		if c := strings.TrimSpace(countStr); c != "" {
			if v, err := strconv.ParseFloat(strings.Replace(c, ",", ".", 1), 64); err == nil && v > 0 {
				count = v
			}
		}
		totalGb += gb * count
		found = true
	}
	add(cuTrim(row.RAM1), cuTrim(row.CRAM1))
	add(cuTrim(row.RAM2), cuTrim(row.CRAM2))
	if !found {
		return 0
	}
	return int(totalGb * 1024)
}

func cuParseDateRU(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	// «23.10.2025 0:00:00» или «23.10.2025»
	if i := strings.IndexAny(s, " T"); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	t, err := time.ParseInLocation("02.01.2006", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// cuStatusHistoryTime — дата из statusHistory (ДД.ММ.ГГГГ), иначе statusHistoryYY (ГГГГ-ММ-ДД).
func cuStatusHistoryTime(row itilium.CURow) (time.Time, bool) {
	if t, ok := cuParseDateRU(cuTrim(row.StatusHistory)); ok {
		return t, true
	}
	s := strings.TrimSpace(row.StatusHistoryYY)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func fillEmpty(dst *string, src string) bool {
	if strings.TrimSpace(*dst) != "" || strings.TrimSpace(src) == "" {
		return false
	}
	*dst = src
	return true
}

// ImportRackServersFromCU пишет КЕ getCU (cuClass=145) в
// ServerHardware/ServerSprRecords (+ DatacenterSite/RackCabinet) по тем же
// правилам, что ImportItiliumServer (Excel), с уточнениями:
//   - списан / в пути / «! Покинуло ФГ…» → без площадки и шкафа;
//   - модель: modelVendS, иначе modelServ, иначе description при наличии
//     маркера HP/Chieftec/DELL/Inflex/DEPO/Cisco; у существующей HW Model не
//     перезаписывается, если уже непустая.
func ImportRackServersFromCU(rows []itilium.CURow) ImportCURackStats {
	var st ImportCURackStats
	st.Total = len(rows)
	o := orm.NewOrm()
	rs := locality.NewResolver()

	for i, row := range rows {
		stOne := importOneCURack(o, rs, row, i)
		switch {
		case stOne.err:
			st.Errors++
		case stOne.skipped:
			st.Skipped++
		case stOne.inserted:
			st.Inserted++
		default:
			st.Updated++
		}
		if stOne.inactive {
			st.Inactive++
		}
		if stOne.noLoc {
			st.NoLoc++
		}
		if stOne.unknownSt {
			st.UnknownSt++
		}
	}
	log.Printf("itilliumimport: ImportRackServersFromCU: total=%d inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Total, st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}

type cuImportOne struct {
	inserted, skipped, inactive, noLoc, unknownSt, err bool
}

func importOneCURack(o orm.Ormer, rs *locality.Resolver, row itilium.CURow, idx int) cuImportOne {
	var out cuImportOne
	tech := strings.TrimSpace(row.Code.String())
	if tech == "" {
		tech = strings.TrimSpace(row.UID)
	}
	if tech == "" {
		log.Printf("itilliumimport: getCU[%d]: пропуск — нет code/uid", idx)
		out.skipped = true
		return out
	}

	statusLabel := strings.TrimSpace(row.Status.Label)
	mapped, known := MapCUStatusLabel(statusLabel)
	if !known {
		log.Printf("itilliumimport: getCU %s: неизвестный статус %q — ItiliumStatus=0, Active=0", tech, statusLabel)
		out.unknownSt = true
	}

	sn := cuTrim(row.SN)
	if sn == "" {
		sn = tech
	}
	hostname := cuTrim(row.Hostname)
	ip := cuTrim(row.IP)
	inv := cuTrim(row.Name)
	where := cuTrim(row.Where)
	links := strings.TrimSpace(row.LinksUnit.String())
	model := cuModel(row)
	owner := cuTrim(row.User)
	details := cuDetails(row)
	osName := cuTrim(row.OS)
	role := cuTrim(row.Role)
	cpu := cuTrim(row.CPU)
	mpIP := cuMpIP(row)
	partNum := cuTrim(row.Product)
	contractDoc := cuTrim(row.Docs)
	provider := cuTrim(row.Provider)
	buhInv := cuTrim(row.BuhCode)
	if buhInv == "" {
		buhInv = cuTrim(row.BuhInv)
	}
	memMb := cuMemoryMb(row)
	docDate, hasDocDate := cuParseDateRU(cuTrim(row.DataDocs))
	descOnly := cuTrim(row.Description)
	statusHist, hasStatusHist := cuStatusHistoryTime(row)

	skipLoc := locationUndefinedForCU(where, mapped.ItiliumStatus)
	if skipLoc {
		out.noLoc = true
	}

	trn, err := o.Begin()
	if err != nil {
		log.Printf("itilliumimport: getCU %s: Begin: %v", tech, err)
		out.err = true
		return out
	}

	var (
		hw  models.ServerHardware
		spr models.ServerSprRecords
	)
	hw.TechnicalNumberIT = tech
	hw.InventoryNum = inv
	hw.InventoryNumBuh = buhInv
	hw.SerialNum = sn
	hw.Model = model
	hw.PartNum = partNum
	hw.CPUModel = cpu
	hw.MpIPAddress = mpIP
	hw.ContractDoc = contractDoc
	hw.Counterparty = provider
	hw.MemoryMb = memMb
	hw.Description = descOnly
	if hasDocDate {
		hw.DateContractDoc = docDate
	}
	if hasStatusHist {
		hw.StatusHistory = statusHist
	}
	hw.FormFactor = 1
	hw.ItiliumStatus = mapped.ItiliumStatus
	hw.PowerState = mapped.HwPower
	hw.Active = mapped.HwActive
	hw.SourceUpdateType = models.SourceUpdateManual
	hw.Id_User = models.UserSourceManual
	hw.Time_Mfy = time.Now().Local()

	spr.Name = hostname
	spr.NameFQDN = hostname
	spr.IPAddress = ip
	spr.SerialNumber = sn
	spr.Owner = owner
	spr.OS = osName
	spr.Role = role
	spr.Details = details
	spr.Stat = mapped.Stat
	spr.PowerState = mapped.Power
	spr.Active = mapped.Active
	spr.SourceUpdateType = models.SourceUpdateManual
	spr.Id_User = models.UserSourceManual
	spr.Time_Mfy = time.Now().Local()
	if mapped.Active == 1 {
		spr.ActivatorUser = models.UserSourceManual
	} else if known {
		spr.TimeDeActiveState = time.Now().Local()
		spr.DeActivatorUser = models.UserSourceManual
		out.inactive = true
	}

	// --- Площадка / шкаф -------------------------------------------------
	if !skipLoc && where != "" {
		site, errSite := resolveSite(trn, rs, where, models.UserSourceManual)
		if errSite != nil {
			log.Printf("itilliumimport: getCU %s: resolveSite: %v", tech, errSite)
			trn.Rollback()
			out.err = true
			return out
		}
		// Служебная «!»-отметка (не «! Покинуло ФГ») — не плодим площадку.
		if site.Loc != nil && site.Loc.Service {
			out.noLoc = true
			hw.RackCabinetId, hw.StartUnitPos, hw.SizeUnit = 0, 0, 0
		} else {
			datacenterId := site.Id
			pl := locality.ParsePlacement(links)
			rackNum := ""
			startU, sizeU := 0, 0
			if pl != nil {
				rackNum = pl.Cabinet
				if pl.Pos != nil {
					startU = pl.Pos.Start
					sizeU = pl.Pos.End - pl.Pos.Start + 1
					if sizeU < 1 {
						sizeU = 1
					}
				}
			}
			cabinetCode, cabinetAlt, locPos := cabinetFromPlacement(rackNum, site.Loc)
			if cabinetCode != "" {
				rackId, errRack := resolveRackCabinet(trn, cabinetCode, cabinetAlt, datacenterId,
					"Шкаф по данным getCU Итиллиум, площадка: "+site.Name)
				if errRack != nil {
					log.Printf("itilliumimport: getCU %s: resolveRack: %v", tech, errRack)
					trn.Rollback()
					out.err = true
					return out
				}
				hw.RackCabinetId = rackId
				if startU > 0 {
					hw.StartUnitPos, hw.SizeUnit = startU, sizeU
				} else if locPos != nil {
					hw.StartUnitPos = locPos.Start
					hw.SizeUnit = locPos.End - locPos.Start + 1
					if hw.SizeUnit < 1 {
						hw.SizeUnit = 1
					}
				}
			} else if datacenterId > 0 {
				outsideId, errOut := resolveOutsideCabinet(trn, datacenterId)
				if errOut != nil {
					log.Printf("itilliumimport: getCU %s: outside: %v", tech, errOut)
					trn.Rollback()
					out.err = true
					return out
				}
				hw.RackCabinetId = outsideId
				hw.StartUnitPos, hw.SizeUnit = 0, 0
			}
			log.Printf("itilliumimport: getCU %s: площадка=%d (%s) шкаф=%q id=%d unit=%d size=%d",
				tech, datacenterId, site.Name, cabinetCode, hw.RackCabinetId, hw.StartUnitPos, hw.SizeUnit)
		}
	} else {
		hw.RackCabinetId, hw.StartUnitPos, hw.SizeUnit = 0, 0, 0
		if skipLoc {
			log.Printf("itilliumimport: getCU %s: площадка не задаётся (статус=%d / where)", tech, mapped.ItiliumStatus)
		}
	}

	// units (высота) — только если размер из размещения не задан
	if hw.SizeUnit == 0 {
		if u := strings.TrimSpace(row.Units.ValueString()); u != "" {
			if n, err := strconv.Atoi(u); err == nil && n > 0 {
				hw.SizeUnit = n
			}
		}
	}

	// --- ServerHardware --------------------------------------------------
	var exist models.ServerHardware
	exist.SerialNum = sn
	hwFound := trn.Read(&exist, "SerialNum") == nil

	if !hwFound {
		hw.TimeCreate = time.Now().Local()
		hw.CreatorUser = models.UserSourceManual
		id, errIns := trn.Insert(&hw)
		if errIns != nil {
			log.Printf("itilliumimport: getCU %s: insert hw: %v", tech, errIns)
			trn.Rollback()
			out.err = true
			return out
		}
		hw.Id = id
		out.inserted = true
	} else {
		hw.Id = exist.Id
		exist.FormFactor = 1
		exist.RackCabinetId = hw.RackCabinetId
		exist.StartUnitPos = hw.StartUnitPos
		exist.SizeUnit = hw.SizeUnit
		exist.ItiliumStatus = hw.ItiliumStatus
		cols := []string{"RackCabinetId", "StartUnitPos", "SizeUnit", "ItiliumStatus", "FormFactor"}
		if exist.SourceUpdateType == models.SourceUpdateManual {
			exist.PowerState = hw.PowerState
			exist.Active = hw.Active
			cols = append(cols, "PowerState", "Active")
		}
		if strings.TrimSpace(exist.TechnicalNumberIT) == "" && tech != "" {
			exist.TechnicalNumberIT = tech
			cols = append(cols, "TechnicalNumberIT")
		}
		if strings.TrimSpace(exist.InventoryNum) == "" && inv != "" {
			exist.InventoryNum = inv
			cols = append(cols, "InventoryNum")
		}
		// Непустые поля HW не перезаписываем (как Model).
		if fillEmpty(&exist.Model, model) {
			cols = append(cols, "Model")
		}
		if fillEmpty(&exist.PartNum, partNum) {
			cols = append(cols, "PartNum")
		}
		if fillEmpty(&exist.CPUModel, cpu) {
			cols = append(cols, "CPUModel")
		}
		if fillEmpty(&exist.MpIPAddress, mpIP) {
			cols = append(cols, "MpIPAddress")
		}
		if fillEmpty(&exist.ContractDoc, contractDoc) {
			cols = append(cols, "ContractDoc")
		}
		if fillEmpty(&exist.Counterparty, provider) {
			cols = append(cols, "Counterparty")
		}
		if fillEmpty(&exist.InventoryNumBuh, buhInv) {
			cols = append(cols, "InventoryNumBuh")
		}
		if exist.MemoryMb == 0 && memMb > 0 {
			exist.MemoryMb = memMb
			cols = append(cols, "MemoryMb")
		}
		if exist.DateContractDoc.IsZero() && hasDocDate {
			exist.DateContractDoc = docDate
			cols = append(cols, "DateContractDoc")
		}
		if descOnly != "" {
			exist.Description = descOnly
			cols = append(cols, "Description")
		}
		if hasStatusHist {
			exist.StatusHistory = statusHist
			cols = append(cols, "StatusHistory")
		}
		exist.Time_Mfy = time.Now().Local()
		exist.Id_User = models.UserSourceManual
		cols = append(cols, "Time_Mfy", "Id_User")
		if _, errUp := trn.Update(&exist, cols...); errUp != nil {
			log.Printf("itilliumimport: getCU %s: update hw: %v", tech, errUp)
			trn.Rollback()
			out.err = true
			return out
		}
		hw = exist
	}

	// --- ServerSprRecords ------------------------------------------------
	spr.HWSprId = hw.Id
	var sprsrv models.ServerSprRecords
	sprFound := hw.ServerId != 0
	if sprFound {
		sprsrv = models.ServerSprRecords{Id: hw.ServerId}
		sprFound = trn.Read(&sprsrv) == nil && !models.IsServerSprDestroyed(&sprsrv)
	}
	if !sprFound {
		if strings.TrimSpace(spr.Name) == "" {
			// как Excel: без имени справочник не создаём
			if err := trn.Commit(); err != nil {
				log.Printf("itilliumimport: getCU %s: commit (hw only): %v", tech, err)
				out.err = true
				return out
			}
			return out
		}
		spr.TimeCreate = time.Now().Local()
		spr.CreatorUser = models.UserSourceManual
		spr.TimeActiveState = time.Now().Local()
		sprId, errIns := trn.Insert(&spr)
		if errIns != nil {
			log.Printf("itilliumimport: getCU %s: insert spr: %v", tech, errIns)
			trn.Rollback()
			out.err = true
			return out
		}
		hw.ServerId = sprId
		if _, errUp := trn.Update(&hw, "ServerId"); errUp != nil {
			log.Printf("itilliumimport: getCU %s: hw.ServerId: %v", tech, errUp)
			trn.Rollback()
			out.err = true
			return out
		}
		out.inserted = true
	} else {
		sprsrv.HWSprId = hw.Id
		cols := []string{"HWSprId"}
		if sprsrv.SourceUpdateType == models.SourceUpdateManual {
			sprsrv.Stat = spr.Stat
			sprsrv.Active = spr.Active
			sprsrv.PowerState = spr.PowerState
			cols = append(cols, "Stat", "Active", "PowerState")
		}
		if fillEmpty(&sprsrv.OS, osName) {
			cols = append(cols, "OS")
		}
		if fillEmpty(&sprsrv.Role, role) {
			cols = append(cols, "Role")
		}
		if fillEmpty(&sprsrv.Owner, owner) {
			cols = append(cols, "Owner")
		}
		if fillEmpty(&sprsrv.IPAddress, ip) {
			cols = append(cols, "IPAddress")
		}
		if fillEmpty(&sprsrv.Details, details) {
			cols = append(cols, "Details")
		}
		if _, errUp := trn.Update(&sprsrv, cols...); errUp != nil {
			log.Printf("itilliumimport: getCU %s: update spr: %v", tech, errUp)
			trn.Rollback()
			out.err = true
			return out
		}
	}

	if err := trn.Commit(); err != nil {
		log.Printf("itilliumimport: getCU %s: commit: %v", tech, err)
		out.err = true
		return out
	}
	return out
}

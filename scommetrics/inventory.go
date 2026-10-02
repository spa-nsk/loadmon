package scommetrics

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"loadmon/models"
)

// Свойства объектов SCOM, из которых берутся сведения о сервере.
// /data/objectInformation отдаёт свойства с ОТОБРАЖАЕМЫМИ именами
// ("Display Name", "Object Status" - пример в документации), поэтому
// сравнение идёт по нормализованному имени (normName), и для каждого поля
// перечислено несколько вариантов. Точные имена в конкретной инсталляции
// проверяются дампом (tools/scomdump -info <id>) - при расхождении правятся
// только эти списки.
var (
	propFQDN       = []string{"dnsname", "principalname", "fullyqualifieddomainname"}
	propIP         = []string{"ipaddress", "ipaddresses"}
	propLogicalCPU = []string{"logicalprocessors", "numberoflogicalprocessors"}
	propMemoryMB   = []string{"physicalmemorymb", "physicalmemory", "totalphysicalmemorymb"}
	propOSCaption  = []string{"caption", "osversiondisplayname", "operatingsystem"}
)

func firstProp(m map[string]string, names []string) string {
	for _, n := range names {
		if v, ok := m[n]; ok && v != "" {
			return v
		}
	}
	return ""
}

// monitoredComputer - компьютер, по которому в этом цикле собираются метрики.
type monitoredComputer struct {
	hostID   int64
	name     string
	objectID string
	counters []Counter // только счётчики, нужные для server_metric (classify != fieldUnknown)
}

// sprIndex - сопоставление имени компьютера SCOM с ServerSprRecords.
type sprIndex struct {
	byFQDN  map[string][]*models.ServerSprRecords
	byShort map[string][]*models.ServerSprRecords
}

func shortName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.Index(s, "."); i > 0 {
		return s[:i]
	}
	return s
}

func loadSprIndex(o orm.Ormer) (*sprIndex, error) {
	var recs []*models.ServerSprRecords
	if _, err := models.ServerSprQS(o).Filter("Active", 1).All(&recs); err != nil {
		return nil, err
	}
	idx := &sprIndex{byFQDN: map[string][]*models.ServerSprRecords{}, byShort: map[string][]*models.ServerSprRecords{}}
	for _, r := range recs {
		if r == nil {
			continue
		}
		if f := strings.ToLower(strings.TrimSpace(r.NameFQDN)); f != "" {
			idx.byFQDN[f] = append(idx.byFQDN[f], r)
			idx.byShort[shortName(f)] = append(idx.byShort[shortName(f)], r)
		}
		if n := shortName(r.Name); n != "" {
			idx.byShort[n] = append(idx.byShort[n], r)
		}
	}
	return idx, nil
}

// match ищет ОДНУ запись: сначала по полному FQDN, затем по короткому
// имени (без частичных совпадений). Несколько кандидатов - выбирается
// единственный со Stat == 1, иначе сопоставление не делается (неоднозначно).
func (idx *sprIndex) match(fqdn string) (*models.ServerSprRecords, bool) {
	pick := func(list []*models.ServerSprRecords) (*models.ServerSprRecords, bool) {
		uniq := map[int64]*models.ServerSprRecords{}
		for _, r := range list {
			uniq[r.Id] = r
		}
		if len(uniq) == 1 {
			for _, r := range uniq {
				return r, false
			}
		}
		var cur []*models.ServerSprRecords
		for _, r := range uniq {
			if r.Stat == 1 {
				cur = append(cur, r)
			}
		}
		if len(cur) == 1 {
			return cur[0], false
		}
		return nil, len(uniq) > 1
	}
	f := strings.ToLower(strings.TrimSpace(fqdn))
	if list := idx.byFQDN[f]; len(list) > 0 {
		if r, amb := pick(list); r != nil || amb {
			return r, amb
		}
	}
	return pick(idx.byShort[shortName(f)])
}

// inventory обновляет справочник ScomHost и сведения в ServerSprRecords,
// и возвращает список компьютеров для сбора метрик: только сопоставленные
// с аппаратным сервером (Type == 0) БЕЗ ZabbixHostId - для серверов в
// Zabbix метрики берутся из Zabbix (поминутные), SCOM их не дублирует.
func inventory(ctx context.Context, c *Client, className string) ([]monitoredComputer, error) {
	comps, err := c.ObjectsByClass(ctx, className)
	if err != nil {
		return nil, err
	}
	o := orm.NewOrm()
	resolver, err := models.NewScomHostResolver()
	if err != nil {
		return nil, err
	}
	idx, err := loadSprIndex(o)
	if err != nil {
		return nil, err
	}

	var (
		monitored                      []monitoredComputer
		matched, ambiguous, sprUpdated int
	)
	for _, comp := range comps {
		if ctx.Err() != nil {
			return monitored, ctx.Err()
		}
		host, err := resolver.Ensure(c.ep.Name, comp.ID, comp.DisplayName)
		if err != nil {
			log.Printf("scommetrics[%s]: %s: %v", c.ep.Name, comp.DisplayName, err)
			continue
		}
		fillHostDetails(ctx, c, comp, host)

		spr, amb := idx.match(firstNonEmpty(host.NameFQDN, comp.FQDN()))
		if amb {
			ambiguous++
			log.Printf("scommetrics[%s]: %s - несколько подходящих записей ServerSprRecords, сопоставление пропущено", c.ep.Name, comp.FQDN())
		}
		host.ServerId = 0
		if spr != nil {
			host.ServerId = spr.Id
			matched++
		}
		host.TimeMfy = time.Now().Local()
		if err := resolver.Save(host); err != nil {
			log.Printf("scommetrics[%s]: сохранение scom_host %s: %v", c.ep.Name, comp.DisplayName, err)
		}
		if spr == nil {
			continue
		}
		if enrichSpr(o, spr, host) {
			sprUpdated++
		}
		if spr.Type != 0 || spr.ZabbixHostId > 0 {
			continue // метрики SCOM - только для аппаратных серверов без Zabbix
		}
		counters, err := c.PerformanceCounters(ctx, comp.ID)
		if err != nil {
			log.Printf("scommetrics[%s]: счётчики %s: %v", c.ep.Name, comp.FQDN(), err)
			continue
		}
		need := NeededCounters(counters)
		if len(need) == 0 {
			log.Printf("scommetrics[%s]: %s - среди %d счётчиков нет нужных для server_metric (см. scommetrics/match.go)", c.ep.Name, comp.FQDN(), len(counters))
			continue
		}
		monitored = append(monitored, monitoredComputer{hostID: host.Id, name: firstNonEmpty(spr.Name, comp.FQDN()), objectID: comp.ID, counters: need})
	}
	log.Printf("scommetrics[%s]: объектов %s: %d, сопоставлено с ServerSprRecords: %d (неоднозначно: %d), дозаполнено записей: %d, к сбору метрик: %d",
		c.ep.Name, className, len(comps), matched, ambiguous, sprUpdated, len(monitored))
	return monitored, nil
}

// fillHostDetails дозаполняет ScomHost свойствами объекта компьютера и
// связанного объекта ОС (если удалось найти). Ошибки только логируются -
// отсутствие свойств не мешает сбору метрик.
func fillHostDetails(ctx context.Context, c *Client, comp Computer, host *models.ScomHost) {
	info, err := c.ObjectInfo(ctx, comp.ID)
	if err != nil {
		log.Printf("scommetrics[%s]: свойства %s: %v", c.ep.Name, comp.DisplayName, err)
		return
	}
	props := info.PropertyMap()
	if v := firstProp(props, propFQDN); v != "" {
		host.NameFQDN = v
	} else if host.NameFQDN == "" {
		host.NameFQDN = comp.FQDN()
	}
	if v := firstProp(props, propIP); v != "" {
		host.IpAddress = normalizeIPList(v)
	}
	if v := firstProp(props, propLogicalCPU); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			host.LogicalNumCPU = n
		}
	}
	if v := firstProp(props, propMemoryMB); v != "" {
		if f, ok := num(v); ok && f > 0 {
			host.MemoryTotalMb = f
		}
	}

	// Объект ОС - связанный объект с именем "Microsoft Windows ..." (так
	// SCOM называет экземпляры Windows Operating System). Из него - название
	// ОС и, если у компьютера его не было, объём памяти.
	for _, rel := range info.RelatedObjects {
		if !strings.HasPrefix(strings.ToLower(rel.DisplayName), "microsoft windows") {
			continue
		}
		host.OperationalSystem = strings.TrimSpace(rel.DisplayName)
		osInfo, err := c.ObjectInfo(ctx, rel.ID)
		if err != nil {
			break
		}
		osProps := osInfo.PropertyMap()
		if v := firstProp(osProps, propOSCaption); v != "" {
			host.OperationalSystem = v
		}
		if host.MemoryTotalMb <= 0 {
			if v := firstProp(osProps, propMemoryMB); v != "" {
				if f, ok := num(v); ok && f > 0 {
					host.MemoryTotalMb = f
				}
			}
		}
		if host.LogicalNumCPU <= 0 {
			if v := firstProp(osProps, propLogicalCPU); v != "" {
				if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
					host.LogicalNumCPU = n
				}
			}
		}
		break
	}
}

// enrichSpr проставляет ScomHostId и дозаполняет ТОЛЬКО ПУСТЫЕ поля
// ServerSprRecords сведениями из SCOM (NameFQDN, IPAddress, OS, CPULogical,
// MemSize в ГБ) - данные других источников и ручной ввод не
// перезаписываются. SourceUpdateType = SCOM ставится, только если поле
// было "не указано" (0): признак ручной записи (1) другими модулями
// используется как защита от автообновления и не должен теряться.
func enrichSpr(o orm.Ormer, spr *models.ServerSprRecords, host *models.ScomHost) bool {
	var cols []string
	if spr.ScomHostId != host.Id {
		spr.ScomHostId = host.Id
		cols = append(cols, "ScomHostId")
	}
	filled := false
	if strings.TrimSpace(spr.NameFQDN) == "" && host.NameFQDN != "" {
		spr.NameFQDN = host.NameFQDN
		cols = append(cols, "NameFQDN")
		filled = true
	}
	if strings.TrimSpace(spr.IPAddress) == "" && host.IpAddress != "" {
		spr.IPAddress = host.IpAddress
		cols = append(cols, "IPAddress")
		filled = true
	}
	if strings.TrimSpace(spr.OS) == "" && host.OperationalSystem != "" {
		spr.OS = host.OperationalSystem
		cols = append(cols, "OS")
		filled = true
	}
	if spr.CPULogical <= 0 && host.LogicalNumCPU > 0 {
		spr.CPULogical = float64(host.LogicalNumCPU)
		cols = append(cols, "CPULogical")
		filled = true
	}
	if spr.MemSize <= 0 && host.MemoryTotalMb > 0 {
		spr.MemSize = host.MemoryTotalMb / 1024.0
		cols = append(cols, "MemSize")
		filled = true
	}
	if filled && spr.SourceUpdateType == models.SourceUpdateNone {
		spr.SourceUpdateType = models.SourceUpdateSCOM
		cols = append(cols, "SourceUpdateType")
	}
	if len(cols) == 0 {
		return false
	}
	spr.Time_Mfy = time.Now().Local()
	cols = append(cols, "Time_Mfy")
	if _, err := o.Update(spr, cols...); err != nil {
		log.Printf("scommetrics: обновление ServerSprRecords %d (%s): %v", spr.Id, spr.Name, err)
		return false
	}
	return true
}

// normalizeIPList приводит список адресов SCOM ("a, b" или "a; b") к
// формату ServerSprRecords.IPAddress ("a; b").
func normalizeIPList(s string) string {
	f := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	return strings.Join(f, "; ")
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

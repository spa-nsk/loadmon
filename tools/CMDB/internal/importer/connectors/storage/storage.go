// Package storage — общий слой коннекторов СХД (ONTAP, Dell Unity, Dell VNX, Lenovo DE, Huawei
// Dorado/OceanStor; plan.md E5.3/E8.1). Каждый коннектор переводит ответ своего API в нейтральный
// Snapshot, а Build строит из него канонические записи: единые классы КЕ, внешние идентификаторы,
// связи и компоненты — так поведение конвейера (идентификация, mark-and-sweep, конфликты)
// одинаково для всех вендоров, а различия остаются в разборе ответов.
//
// Один коннектор — один массив (кластер): результат опроса конкретной СХД фиксируется как прогон
// источника, том пропадает только по итогам успешного опроса ЭТОЙ СХД (harness/03, правила 1–2).
package storage

import (
	"fmt"
	"math"
	"net"
	"sort"
	"strings"

	"cmdb/internal/importer/canon"
)

// Snapshot — нейтральное описание одного массива.
type Snapshot struct {
	Array   Array
	Pools   []Pool
	SVMs    []SVM
	Volumes []Volume
	Qtrees  []Qtree
	Shares  []Share
	// Mappings — презентация томов инициаторам (LUN masking): WWPN, которым виден том. MappingsKnown —
	// коннектор умеет читать презентацию: тогда у каждого тома список (возможно, пустой) достоверен.
	Mappings      []Mapping
	MappingsKnown bool
	Controllers   []Component
	Enclosures    []Component
	Disks         []Disk
}

// Mapping — том и WWPN инициаторов, которым он презентован (нормализованные NormalizeWWPN).
type Mapping struct {
	VolumeID string
	WWPNs    []string
}

// Array — сам массив (кластер).
type Array struct {
	Name      string
	Serial    string // "" — у источника нет серийного номера (ONTAP: идентификация по имени/ExternalID)
	Family    string // вендор и модель: "NetApp AFF A400", "Dell Unity 480F"
	Microcode string
	MgmtIP    string
	Vendor    map[string]any
}

// Pool — пул/агрегат/группа томов.
type Pool struct {
	ID, Name, RAID, Tier string
	CapacityTB           float64
}

// SVM — виртуальный сервер хранилища (ONTAP SVM, Unity NAS-сервер).
type SVM struct {
	ID, Name, MgmtIP string
}

// Volume — том: LUN блочного доступа либо файловый том/файловая система.
type Volume struct {
	ID, Name       string
	PoolID, SVMID  string
	LunID          *int
	WWN            string // NAA/WWN, нижний регистр без разделителей; "" — не сообщён
	SizeGB         float64
	Thin           *bool
	Tier, Protocol string
}

// Qtree — квота-дерево внутри тома.
type Qtree struct {
	ID, Name, SVMID, VolumeID, Path string
	QuotaGB                         float64
}

// Share — файловый ресурс (SMB) или экспорт (NFS).
type Share struct {
	ID, Name, SVMID, VolumeID, Type, Path string // Type: smb | nfs
}

// Component — контроллер или полка.
type Component struct {
	ID, Name, Serial, Model, Firmware string
}

// Disk — накопитель.
type Disk struct {
	ID, Serial, Model, Manufacturer, Firmware string
	Location                                  string // слот/позиция
	EnclosureID                               string
	Type                                      string // disk | ssd | nvme
	CapacityGB                                float64
}

func fp(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func nilStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ipOrNil — корректный IP или nil: колонка mgmt_ip имеет тип inet, мусор от источника не должен
// ронять запись целиком.
func ipOrNil(s string) any {
	if net.ParseIP(strings.TrimSpace(s)) == nil {
		return nil
	}
	return strings.TrimSpace(s)
}

func nilNum(f float64) any {
	if f <= 0 {
		return nil
	}
	return math.Round(f*100) / 100
}

// Build строит записи снимка. Имена пулов/томов/SVM получают префикс — имя массива: у КЕ класса
// активное имя уникально (ci_name_active_uniq), а «LUN_1» есть на каждой СХД; имя источника
// сохраняется в составе имени и не переименовывается.
func Build(s Snapshot, code string) []canon.Record {
	arr := s.Array
	arrayID := "array:" + code
	prefix := arr.Name + "/"

	rawGB := 0.0
	for _, d := range s.Disks {
		rawGB += d.CapacityGB
	}

	rec := canon.Record{
		Class: "storage_array", SourceName: arr.Name, ExternalID: arrayID, SerialNumber: arr.Serial, Identifiers: map[string]string{},
		Attrs: map[string]any{
			"array_family": nilStr(arr.Family), "microcode": nilStr(arr.Microcode), "controllers": nz(len(s.Controllers)),
			"raw_tb": nilNum(rawGB / 1000), "mgmt_ip": ipOrNil(arr.MgmtIP),
		},
		Vendor: arr.Vendor,
	}
	if arr.Serial != "" {
		rec.Identifiers["serial_number"] = arr.Serial
	}

	// компоненты массива: контроллеры, полки, диски (диск вложен в полку, если известна)
	comps := []canon.Component{}
	for _, c := range s.Controllers {
		comps = append(comps, canon.Component{Key: "ctrl:" + c.ID, Type: "controller", Slot: firstNonEmpty(c.Name, c.ID), PartNumber: c.Model,
			SerialNumber: c.Serial, Firmware: c.Firmware})
	}
	encKnown := map[string]bool{}
	for _, e := range s.Enclosures {
		encKnown[e.ID] = true
		comps = append(comps, canon.Component{Key: "enc:" + e.ID, Type: "enclosure", Slot: firstNonEmpty(e.Name, e.ID), PartNumber: e.Model,
			SerialNumber: e.Serial, Firmware: e.Firmware})
	}
	for _, d := range s.Disks {
		typ := d.Type
		if typ == "" {
			typ = "disk"
		}
		c := canon.Component{Key: "disk:" + firstNonEmpty(d.Serial, d.ID), Type: typ, Slot: firstNonEmpty(d.Location, d.ID),
			Manufacturer: d.Manufacturer, PartNumber: d.Model, SerialNumber: d.Serial, Firmware: d.Firmware,
			CapacityValue: fp(math.Round(d.CapacityGB*10) / 10), CapacityUnit: "GB"}
		if d.EnclosureID != "" && encKnown[d.EnclosureID] {
			c.ParentKey = "enc:" + d.EnclosureID
		}
		comps = append(comps, c)
	}
	rec.Components = comps

	mapped := map[string][]string{}
	for _, m := range s.Mappings {
		seen := map[string]bool{}
		for _, w := range mapped[m.VolumeID] {
			seen[w] = true
		}
		for _, w := range m.WWPNs {
			if w = NormalizeWWPN(w); w != "" && !seen[w] {
				seen[w] = true
				mapped[m.VolumeID] = append(mapped[m.VolumeID], w)
			}
		}
	}
	for id := range mapped {
		sort.Strings(mapped[id])
	}
	for _, v := range s.Volumes {
		if mapped[v.ID] == nil {
			mapped[v.ID] = []string{}
		}
	}

	recs := []canon.Record{rec}
	poolKnown := map[string]bool{}
	svmKnown := map[string]bool{}
	volKnown := map[string]bool{}

	for _, p := range s.Pools {
		poolKnown[p.ID] = true
		recs[0].Relations = append(recs[0].Relations, canon.Relation{Type: "contains", TargetExternalID: "pool:" + p.ID})
		recs = append(recs, canon.Record{
			Class: "storage_pool", SourceName: prefix + p.Name, ExternalID: "pool:" + p.ID,
			Attrs: map[string]any{"raid_type": nilStr(p.RAID), "tier": nilStr(p.Tier), "capacity_tb": nilNum(p.CapacityTB)},
		})
	}
	for _, v := range s.SVMs {
		svmKnown[v.ID] = true
		recs[0].Relations = append(recs[0].Relations, canon.Relation{Type: "contains", TargetExternalID: "svm:" + v.ID})
		recs = append(recs, canon.Record{
			Class: "svm", SourceName: prefix + v.Name, ExternalID: "svm:" + v.ID,
			Attrs: map[string]any{"mgmt_ip": ipOrNil(v.MgmtIP)},
		})
	}
	for _, v := range s.Volumes {
		volKnown[v.ID] = true
		attrs := map[string]any{"wwn": nilStr(v.WWN), "size_gb": nilNum(v.SizeGB), "tier": nilStr(v.Tier), "protocol": nilStr(v.Protocol)}
		if v.LunID != nil {
			attrs["lun_id"] = *v.LunID
		}
		if v.Thin != nil {
			attrs["thin"] = *v.Thin
		}
		if s.MappingsKnown {
			attrs["mapped_wwpns"] = mapped[v.ID] // не nil: пустой список — «презентации нет»
		}
		r := canon.Record{Class: "volume", SourceName: prefix + v.Name, ExternalID: "vol:" + v.ID, Identifiers: map[string]string{}, Attrs: attrs}
		if v.WWN != "" {
			r.Identifiers["wwn"] = v.WWN
		}
		switch {
		case svmKnown[v.SVMID]:
			// том NAS принадлежит SVM; размещение на пуле (агрегате) — отдельной связью
			recs[indexOf(recs, "svm:"+v.SVMID)].Relations = append(recs[indexOf(recs, "svm:"+v.SVMID)].Relations,
				canon.Relation{Type: "contains", TargetExternalID: "vol:" + v.ID})
			if poolKnown[v.PoolID] {
				r.Relations = append(r.Relations, canon.Relation{Type: "runs_on", TargetExternalID: "pool:" + v.PoolID})
			}
		case poolKnown[v.PoolID]:
			recs[indexOf(recs, "pool:"+v.PoolID)].Relations = append(recs[indexOf(recs, "pool:"+v.PoolID)].Relations,
				canon.Relation{Type: "contains", TargetExternalID: "vol:" + v.ID})
		default:
			recs[0].Relations = append(recs[0].Relations, canon.Relation{Type: "contains", TargetExternalID: "vol:" + v.ID})
		}
		recs = append(recs, r)
	}
	for _, q := range s.Qtrees {
		r := canon.Record{Class: "qtree", SourceName: prefix + q.Name, ExternalID: "qtree:" + q.ID,
			Attrs: map[string]any{"path": nilStr(q.Path), "quota_gb": nilNum(q.QuotaGB)}}
		if volKnown[q.VolumeID] {
			recs[indexOf(recs, "vol:"+q.VolumeID)].Relations = append(recs[indexOf(recs, "vol:"+q.VolumeID)].Relations,
				canon.Relation{Type: "contains", TargetExternalID: "qtree:" + q.ID})
		}
		recs = append(recs, r)
	}
	for _, sh := range s.Shares {
		typ := strings.ToLower(sh.Type)
		if typ != "smb" && typ != "nfs" {
			typ = ""
		}
		r := canon.Record{Class: "share", SourceName: prefix + sh.Name, ExternalID: "share:" + sh.ID,
			Attrs: map[string]any{"share_type": nilStr(typ), "path": nilStr(sh.Path)}}
		switch {
		case volKnown[sh.VolumeID]:
			recs[indexOf(recs, "vol:"+sh.VolumeID)].Relations = append(recs[indexOf(recs, "vol:"+sh.VolumeID)].Relations,
				canon.Relation{Type: "contains", TargetExternalID: "share:" + sh.ID})
		case svmKnown[sh.SVMID]: // том ресурса не определён (CIFS-ресурс ONTAP не ссылается на том) — принадлежит SVM
			recs[indexOf(recs, "svm:"+sh.SVMID)].Relations = append(recs[indexOf(recs, "svm:"+sh.SVMID)].Relations,
				canon.Relation{Type: "contains", TargetExternalID: "share:" + sh.ID})
		}
		recs = append(recs, r)
	}
	return recs
}

func indexOf(recs []canon.Record, ext string) int {
	for i := range recs {
		if recs[i].ExternalID == ext {
			return i
		}
	}
	return 0
}

// Items оборачивает записи в canon.Item с порядковыми номерами.
func Items(recs []canon.Record) []canon.Item {
	items := make([]canon.Item, len(recs))
	for i := range recs {
		items[i] = canon.Item{Seq: i + 1, Record: recs[i]}
	}
	return items
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func nz(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

// NormalizeWWN приводит WWN/NAA к нижнему регистру без разделителей (":", "-", пробелы, префикс "naa.").
func NormalizeWWN(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "naa.")
	return strings.NewReplacer(":", "", "-", "", " ", "").Replace(s)
}

// NormalizeWWPN приводит WWPN к виду aa:bb:cc:dd:ee:ff:00:11 (нижний регистр). Принимает записи с
// разделителями и без, с префиксом 0x; 32 hex-цифры («WWNN:WWPN» у Unity) — берётся хвост (порт).
// Не WWPN (IQN iSCSI, мусор) — "".
func NormalizeWWPN(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "0x")
	s = strings.NewReplacer(":", "", "-", "", " ", "").Replace(s)
	if len(s) == 32 {
		s = s[16:]
	}
	if len(s) != 16 {
		return ""
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return ""
		}
	}
	var sb strings.Builder
	for i := 0; i < 16; i += 2 {
		if i > 0 {
			sb.WriteByte(':')
		}
		sb.WriteString(s[i : i+2])
	}
	return sb.String()
}

// Errorf — ошибка коннектора с префиксом вида и кода.
func Errorf(kind, code, format string, args ...any) error {
	return fmt.Errorf("%s %s: %s", kind, code, fmt.Sprintf(format, args...))
}

// SampleWWPN — детерминированный WWPN для mock-серверов и тестов: tag (hex) делает порт уникальным
// в общей БД, n различает порты одного стенда.
func SampleWWPN(tag string, n int) string {
	return fmt.Sprintf("21:00:%s:%s:%s:%s:%s:%02x", tag[0:2], tag[2:4], tag[4:6], tag[6:8], tag[8:10], n&0xff)
}

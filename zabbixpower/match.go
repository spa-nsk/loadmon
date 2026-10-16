package zabbixpower

import (
	"net"
	"sort"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"loadmon/models"
)

// matcher сопоставляет имя хоста Zabbix (ILO-S-NSK01-DBDEV20,
// ilo-esxi-prod-127.global.bcs) с ServerSprRecords.Id.
//
// Порядок:
//  1. имя совпало с ServerHardware.MpHostName (имя iLO из OneView) -
//     ServerHardware.ServerId;
//  2. имя без префикса "ilo-" (или суффикса "-ilo") совпало с первой
//     меткой Name/NameFQDN/NameInfo сервера.
//
// Сравнение без регистра, домен отбрасывается.
type matcher struct {
	byMp   map[string]int64
	byName map[string][]sprRef
}

type sprRef struct {
	id       int64
	stat     int
	typ      int
	hwLinked bool
}

func firstLabel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || net.ParseIP(s) != nil {
		return s
	}
	if i := strings.IndexByte(s, '.'); i > 0 {
		return s[:i]
	}
	return s
}

func stripIlo(s string) string {
	for _, p := range []string{"ilo-", "ilo_"} {
		if strings.HasPrefix(s, p) {
			return s[len(p):]
		}
	}
	for _, p := range []string{"-ilo", "_ilo"} {
		if strings.HasSuffix(s, p) {
			return s[:len(s)-len(p)]
		}
	}
	return s
}

func newMatcher() (*matcher, error) {
	o := orm.NewOrm()
	m := &matcher{byMp: map[string]int64{}, byName: map[string][]sprRef{}}

	var hws []*models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).Filter("ServerId__gt", 0).
		All(&hws, "Id", "ServerId", "MpHostName"); err != nil {
		return nil, err
	}
	for _, h := range hws {
		if k := firstLabel(h.MpHostName); k != "" {
			m.byMp[k] = h.ServerId
		}
	}

	var sprs []*models.ServerSprRecords
	if _, err := models.ServerSprQS(o).All(&sprs, "Id", "Name", "NameFQDN", "NameInfo", "Stat", "Type", "HWSprId"); err != nil {
		return nil, err
	}
	for _, s := range sprs {
		ref := sprRef{id: s.Id, stat: s.Stat, typ: s.Type, hwLinked: s.HWSprId > 0}
		seen := map[string]bool{}
		for _, n := range []string{s.Name, s.NameFQDN, s.NameInfo} {
			k := firstLabel(n)
			if k == "" || seen[k] {
				continue
			}
			seen[k] = true
			m.byName[k] = append(m.byName[k], ref)
		}
	}
	return m, nil
}

// match возвращает ServerSprRecords.Id по имени хоста Zabbix (name - видимое
// имя, host - техническое).
func (m *matcher) match(name, host string) (int64, bool) {
	for _, n := range []string{name, host} {
		k := firstLabel(n)
		if k == "" {
			continue
		}
		if id, ok := m.byMp[k]; ok {
			return id, true
		}
		if id, ok := m.pick(stripIlo(k)); ok {
			return id, true
		}
	}
	return 0, false
}

// pick выбирает лучшую запись: используемая (Stat=1), не ВМ (Type 0/2),
// связанная с оборудованием, меньший Id.
func (m *matcher) pick(key string) (int64, bool) {
	list := m.byName[key]
	if len(list) == 0 {
		return 0, false
	}
	rank := func(r sprRef) int {
		v := 0
		if r.stat != models.ServerSprStatUsed {
			v += 4
		}
		if r.typ == 1 {
			v += 2
		}
		if !r.hwLinked {
			v++
		}
		return v
	}
	sort.SliceStable(list, func(i, j int) bool {
		if ri, rj := rank(list[i]), rank(list[j]); ri != rj {
			return ri < rj
		}
		return list[i].id < list[j].id
	})
	return list[0].id, true
}

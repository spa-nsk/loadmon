package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ZabbixHostResolver сопоставляет хосты из Zabbix API с models.ZabbixHost.Id.
// На один цикл сбора метрик достаточно одного экземпляра: он кэширует
// справочник и при необходимости создаёт недостающие записи.
type ZabbixHostResolver struct {
	o      orm.Ormer
	byHost map[string]int64
	byName map[string]int64
}

// NewZabbixHostResolver загружает справочник zabbix_host в память.
func NewZabbixHostResolver() (*ZabbixHostResolver, error) {
	r := &ZabbixHostResolver{
		o:      orm.NewOrm(),
		byHost: make(map[string]int64),
		byName: make(map[string]int64),
	}
	var hosts []*ZabbixHost
	if _, err := r.o.QueryTable(new(ZabbixHost)).All(&hosts); err != nil {
		return nil, fmt.Errorf("models: load zabbix_host: %w", err)
	}
	for _, h := range hosts {
		if h == nil || h.Id <= 0 {
			continue
		}
		if host := strings.TrimSpace(h.Host); host != "" {
			r.byHost[host] = h.Id
		}
		if name := strings.TrimSpace(h.Name); name != "" {
			if _, ok := r.byName[name]; !ok {
				r.byName[name] = h.Id
			}
		}
	}
	return r, nil
}

// EnsureID возвращает ZabbixHost.Id для пары host/name из Zabbix API.
// Сначала ищет по Host, затем по Name; при отсутствии создаёт запись.
func (r *ZabbixHostResolver) EnsureID(host, name string) (int64, error) {
	host = strings.TrimSpace(host)
	name = strings.TrimSpace(name)
	if host == "" && name == "" {
		return 0, fmt.Errorf("models: empty zabbix host identity")
	}

	if host != "" {
		if id, ok := r.byHost[host]; ok {
			return id, nil
		}
	}
	if name != "" {
		if id, ok := r.byName[name]; ok {
			return id, nil
		}
	}

	rec := ZabbixHost{
		Host:    host,
		Name:    name,
		TimeMfy: time.Now().Local(),
	}
	if rec.Host == "" {
		rec.Host = name
	}
	if rec.Name == "" {
		rec.Name = host
	}

	id, err := r.o.Insert(&rec)
	if err != nil {
		return 0, fmt.Errorf("models: insert zabbix_host %q: %w", rec.Host, err)
	}
	rec.Id = id
	r.byHost[rec.Host] = id
	if rec.Name != "" {
		r.byName[rec.Name] = id
	}
	return id, nil
}

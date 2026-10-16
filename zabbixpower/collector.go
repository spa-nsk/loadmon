package zabbixpower

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"loadmon/models"
)

var (
	// Interval — период сбора.
	Interval = time.Minute
	// Retention — срок хранения поминутных замеров.
	Retention = 3 * 24 * time.Hour
	// StaleAfter — значения старше не пишутся (датчик не опрашивается).
	StaleAfter = 10 * time.Minute
)

// Collect снимает мощность со всех Sources за текущую минуту и пишет
// server_power_metric для хостов, сопоставленных с ServerSprRecords.
func Collect(ctx context.Context) error {
	now := time.Now().Local().Truncate(time.Minute)
	m, err := newMatcher()
	if err != nil {
		return fmt.Errorf("zabbixpower: load directories: %w", err)
	}

	byServer := map[int64]*models.ServerPowerMetric{}
	var unmatched, stale int
	for _, src := range sources() {
		list, err := GetServerPowerFrom(ctx, src)
		if err != nil {
			log.Printf("zabbixpower: %s: %v", src.Name, err)
			continue
		}
		for _, p := range list {
			if !p.Clock.IsZero() && now.Sub(p.Clock) > StaleAfter {
				stale++
				continue
			}
			id, ok := m.match(p.Name, p.Host)
			if !ok {
				unmatched++
				continue
			}
			if _, dup := byServer[id]; dup {
				continue
			}
			byServer[id] = &models.ServerPowerMetric{ServerId: id, CollectedAt: now, HostName: p.Name, Watts: p.Watts}
		}
	}
	rows := make([]*models.ServerPowerMetric, 0, len(byServer))
	for _, r := range byServer {
		rows = append(rows, r)
	}
	if err := models.ReplaceServerPower(ctx, rows); err != nil {
		return fmt.Errorf("zabbixpower: store: %w", err)
	}
	initHardwarePower(rows)
	log.Printf("zabbixpower: stored %d, unmatched %d, stale %d at %s", len(rows), unmatched, stale, now.Format("15:04"))
	return nil
}

// RunAll запускает минутный сбор и суточное обслуживание (агрегация и
// очистка) до отмены ctx.
func RunAll(ctx context.Context) {
	go maintenance(ctx)
	if Interval <= 0 {
		Interval = time.Minute
	}
	if err := Collect(ctx); err != nil {
		log.Printf("zabbixpower: collect: %v", err)
	}
	t := time.NewTicker(Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := Collect(ctx); err != nil {
				log.Printf("zabbixpower: collect: %v", err)
			}
		}
	}
}

// maintenance: при старте догоняет агрегацию за прошедшие дни хранения, затем
// каждые сутки в 00:00 агрегирует прошедший день и удаляет замеры старше Retention.
func maintenance(ctx context.Context) {
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("zabbixpower: maintenance panic: %v", r)
			}
		}()
		today := dayStart(time.Now().Local())
		for d := int(Retention / (24 * time.Hour)); d >= 1; d-- {
			AggregateDay(today.AddDate(0, 0, -d), d == 1) // в ServerHardware - итог последних закрытых суток
		}
		ClearOld(today.Add(-Retention))
	}
	run()
	for {
		now := time.Now().Local()
		next := dayStart(now).AddDate(0, 0, 1).Add(5 * time.Second) // +5с: дать записаться последней минуте
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
			run()
		}
	}
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// ClearOld удаляет минутные замеры старше before порциями.
func ClearOld(before time.Time) {
	o := orm.NewOrm()
	total := int64(0)
	for {
		res, err := o.Raw("DELETE FROM server_power_metric WHERE id IN (SELECT id FROM server_power_metric WHERE collected_at < ? LIMIT 1000)", before).Exec()
		if err != nil {
			log.Printf("zabbixpower: clear server_power_metric: %v", err)
			return
		}
		n, _ := res.RowsAffected()
		total += n
		if n == 0 {
			break
		}
	}
	log.Printf("zabbixpower: deleted %d rows older than %s", total, before.Format("2006-01-02"))
}

// AggregateDay строит (идемпотентно) суточные записи server_power_record за
// сутки, начинающиеся в day.
// При setHW суточные среднее/максимум дополнительно пишутся в
// ServerHardware.PowerAvg24hW/PowerPeak24hW.
func AggregateDay(day time.Time, setHW bool) {
	day = dayStart(day)
	end := day.AddDate(0, 0, 1)
	o := orm.NewOrm()

	var ids []orm.ParamsList
	if _, err := o.Raw("SELECT DISTINCT server_id FROM server_power_metric WHERE collected_at >= ? AND collected_at < ?", day, end).ValuesList(&ids); err != nil {
		log.Printf("zabbixpower: aggregate %s: %v", day.Format("2006-01-02"), err)
		return
	}
	count := 0
	for _, row := range ids {
		var serverId int64
		switch v := row[0].(type) {
		case int64:
			serverId = v
		case string:
			fmt.Sscan(v, &serverId)
		}
		if serverId <= 0 {
			continue
		}
		var samples []*models.ServerPowerMetric
		if _, err := o.QueryTable(new(models.ServerPowerMetric)).Filter("ServerId", serverId).
			Filter("CollectedAt__gte", day).Filter("CollectedAt__lt", end).OrderBy("CollectedAt").All(&samples); err != nil || len(samples) == 0 {
			continue
		}
		rec := aggregate(serverId, day, samples)
		existing := models.ServerPowerRecord{ServerId: serverId, Date: day}
		if err := o.Read(&existing, "ServerId", "Date"); err == nil {
			rec.Id = existing.Id
			if _, err := o.Update(rec); err != nil {
				log.Printf("zabbixpower: update record server=%d: %v", serverId, err)
			}
		} else if _, err := o.Insert(rec); err != nil {
			log.Printf("zabbixpower: insert record server=%d: %v", serverId, err)
		}
		if setHW {
			setHardwarePower(o, rec)
		}
		count++
	}
	log.Printf("zabbixpower: aggregated %d servers for %s", count, day.Format("2006-01-02"))
}

func aggregate(serverId int64, day time.Time, samples []*models.ServerPowerMetric) *models.ServerPowerRecord {
	rec := &models.ServerPowerRecord{ServerId: serverId, Date: day, PerfCount: int64(len(samples)), Time_Mfy: time.Now().Local()}
	max, min, sum := -math.MaxFloat64, math.MaxFloat64, 0.0
	for _, s := range samples {
		sum += s.Watts
		if s.Watts > max {
			max, rec.WattsMaxTime = s.Watts, s.CollectedAt
		}
		if s.Watts < min {
			min, rec.WattsMinTime = s.Watts, s.CollectedAt
		}
	}
	rec.WattsMax, rec.WattsMin = max, min
	rec.WattsAvg = sum / float64(len(samples))
	rec.EnergyKWh = sum / 60 / 1000 // каждый замер - минута
	return rec
}

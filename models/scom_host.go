package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ScomHost - справочник компьютеров Microsoft SCOM (объекты класса
// Microsoft.Windows.Computer), по образцу models.ZabbixHost. На него
// ссылаются models.ServerMetric.HostId/ServerMetricRecord.HostId при
// Source == MetricSourceSCOM и ServerSprRecords.ScomHostId.
//
// Заполняется пакетом scommetrics (см. kontext/13-scom-metrics.md):
// ObjectId/DisplayName - из POST /OperationsManager/data/scomObjectsByClass,
// остальные сведения - из GET /data/monitoringObjectProperties/{id}.
type ScomHost struct {
	Id                int64     `form:"Id" orm:"auto;pk;column(id)"`
	ManagementGroup   string    `form:"ManagementGroup" orm:"column(management_group);size(255)"`  // ScomEndpoint.Name из app2.json - из какого SCOM пришёл объект
	ObjectId          string    `form:"ObjectId" orm:"column(object_id);size(64);unique"`          // GUID объекта SCOM (поле "id" ответа scomObjectsByClass)
	ServerId          int64     `form:"ServerId" orm:"column(server_id);index"`                    // ссылка на ServerSprRecords.Id (0 - не сопоставлен)
	DisplayName       string    `form:"DisplayName" orm:"column(display_name);size(255);index"`    // отображаемое имя в SCOM (для Windows Computer - обычно FQDN)
	NameFQDN          string    `form:"NameFQDN" orm:"column(name_fqdn);size(255)"`                // DNS-имя (свойство объекта), если получено
	OperationalSystem string    `form:"OperationalSystem" orm:"column(operational_system);size(255)"` // ОС в формате SCOM
	LogicalNumCPU     int       `form:"LogicalNumCPU" orm:"column(logical_num_cpu)"`               // количество логических процессоров
	MemoryTotalMb     float64   `form:"MemoryTotalMb" orm:"column(memory_total_mb)"`               // объём RAM, МБ
	IpAddress         string    `form:"IpAddress" orm:"column(ip_address);size(1024)"`            // IP-адреса через "; "
	TimeMfy           time.Time `form:"TimeMfy" orm:"column(time_mfy);null"`                       // время последнего обновления сведений
}

func (a *ScomHost) TableName() string {
	return "scom_host"
}

// ScomHostResolver сопоставляет объекты SCOM с ScomHost.Id. Один экземпляр
// на цикл сбора: справочник кэшируется, недостающие записи создаются.
type ScomHostResolver struct {
	o          orm.Ormer
	byObjectId map[string]*ScomHost
}

// NewScomHostResolver загружает справочник scom_host в память.
func NewScomHostResolver() (*ScomHostResolver, error) {
	r := &ScomHostResolver{o: orm.NewOrm(), byObjectId: map[string]*ScomHost{}}
	var hosts []*ScomHost
	if _, err := r.o.QueryTable(new(ScomHost)).All(&hosts); err != nil {
		return nil, fmt.Errorf("models: load scom_host: %w", err)
	}
	for _, h := range hosts {
		if h == nil || h.Id <= 0 {
			continue
		}
		if id := strings.ToLower(strings.TrimSpace(h.ObjectId)); id != "" {
			r.byObjectId[id] = h
		}
	}
	return r, nil
}

// Ensure возвращает запись ScomHost для объекта SCOM (по GUID), создавая её
// при отсутствии и обновляя ManagementGroup/DisplayName при изменении.
func (r *ScomHostResolver) Ensure(managementGroup, objectId, displayName string) (*ScomHost, error) {
	key := strings.ToLower(strings.TrimSpace(objectId))
	if key == "" {
		return nil, fmt.Errorf("models: empty scom object id")
	}
	displayName = strings.TrimSpace(displayName)
	if h, ok := r.byObjectId[key]; ok {
		if h.DisplayName != displayName || h.ManagementGroup != managementGroup {
			h.DisplayName = displayName
			h.ManagementGroup = managementGroup
			if _, err := r.o.Update(h, "DisplayName", "ManagementGroup"); err != nil {
				return h, fmt.Errorf("models: update scom_host %d: %w", h.Id, err)
			}
		}
		return h, nil
	}
	h := &ScomHost{
		ManagementGroup: managementGroup,
		ObjectId:        key,
		DisplayName:     displayName,
		TimeMfy:         time.Now().Local(),
	}
	id, err := r.o.Insert(h)
	if err != nil {
		return nil, fmt.Errorf("models: insert scom_host %s: %w", displayName, err)
	}
	h.Id = id
	r.byObjectId[key] = h
	return h, nil
}

// Save сохраняет изменённые сведения записи (после обогащения из свойств объекта).
func (r *ScomHostResolver) Save(h *ScomHost) error {
	if h == nil || h.Id <= 0 {
		return nil
	}
	_, err := r.o.Update(h)
	return err
}

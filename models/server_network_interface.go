package models

import (
	"context"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ServerNetworkInterface — структурированная запись об одном сетевом
// интерфейсе сервера (физического или виртуального): IP-адрес, MAC-адрес,
// маска подсети, имя сети/портгруппы, привязка к ServerSprRecords.Id.
//
// Это Stage 1 доработки "распределённого" хранения сетевых адресов —
// см. CHANGELOG.md, запись про ServerNetworkInterface. Сервер может иметь
// несколько сетевых интерфейсов (несколько строк с одним ServerSprId),
// это справедливо и для аппаратных серверов (источник "zabbix",
// "vmware_host_vnic"), и для виртуальных машин (источник
// "vmware_vm_guest").
//
// Таблица работает СТРОГО АДДИТИВНО к уже существующим ";"-разделённым
// полям ServerSprRecords.IPAddress/MacAddress и к производному справочнику
// IPAddressSprRecords (см. cmdb/syncnetworkip.go) — ни одно из них в рамках
// этой доработки не меняется и не удаляется. Ручного редактирования данных
// этой таблицы пока не предусмотрено (нет форм редактирования) — она
// только накапливается фоновыми синхронизациями из Zabbix/vSphere.
//
// Один и тот же источник для одного ServerSprId на каждом опросе полностью
// перезаписывает свой набор строк (см. cmdb.ReplaceServerNetworkInterfaces) —
// так же, как это сделано для ServerMetric (см. models.ReplaceMetrics),
// поэтому устаревшие/исчезнувшие интерфейсы не накапливаются в таблице.
//
// Поле Gateway сознательно не включено: ни один из используемых источников
// данных (vm.Guest.Net, hi.Config.Network.Vnic, Zabbix host.Interfaces) не
// предоставляет шлюз для конкретного интерфейса, а придумывать источник
// для него сейчас нельзя (см. правило "не гадать",
// kontext/05-conventions-and-gotchas.md).
type ServerNetworkInterface struct {
	Id           int64     `form:"Id" orm:"auto;pk;column(id)"`
	ServerSprId  int64     `form:"ServerSprId" orm:"column(server_spr_id);index"`                    // ссылка на ServerSprRecords.Id
	Source       string    `form:"Source" orm:"column(source);size(32);index"`                       // "zabbix" | "vmware_vm_guest" | "vmware_host_vnic"
	InterfaceName string   `form:"InterfaceName" orm:"column(interface_name);size(128);null"`        // имя NIC/vnic, если источник его предоставляет (пока не заполняется — ни один из источников не даёт имя интерфейса отдельно от MAC/Network)
	MacAddress   string    `form:"MacAddress" orm:"column(mac_address);size(64);null"`                // MAC-адрес интерфейса (для Zabbix не заполняется — источник его не даёт)
	IPAddress    string    `form:"IPAddress" orm:"column(ip_address);size(64);index"`                 // один IP-адрес (v4 или v6) на строку, в отличие от ";"-строки в ServerSprRecords
	IPVersion    int8      `form:"IPVersion" orm:"column(ip_version)"`                                // 4 или 6
	SubnetMask   string    `form:"SubnetMask" orm:"column(subnet_mask);size(64);null"`                // только для источника "vmware_host_vnic" (vnic.Spec.Ip.SubnetMask)
	NetworkName  string    `form:"NetworkName" orm:"column(network_name);size(255);null"`             // имя портгруппы/сети (только для источника "vmware_vm_guest")
	Time_Mfy     time.Time `form:"Time_Mfy" orm:"column(time_mfy);type(datetime);auto_now"`           // время последнего обновления строки
}

func (m *ServerNetworkInterface) TableName() string {
	return "server_network_interface"
}

func (m *ServerNetworkInterface) TableEngine() string {
	return "INNODB"
}

// ReplaceServerNetworkInterfaces записывает набор строк интерфейсов для
// одной пары (ServerSprId, Source), полностью заменяя ранее сохранённые
// строки того же источника для того же сервера — по аналогии с
// ReplaceMetrics (см. models/server_metric.go). Другие источники для этого
// же ServerSprId (например, уже сохранённые ранее из другого источника)
// не затрагиваются.
//
// serverSprId <= 0 считается отсутствием привязки к справочнику серверов
// (например, если ServerSprRecords ещё не создан для найденного узла) —
// вызывающий код обязан сам отфильтровать такие случаи ДО вызова этой
// функции; сама функция в этом случае просто ничего не делает, чтобы
// не плодить строки-сироты.
// ormQuerySetInserter — минимальный интерфейс с единственными двумя
// методами, которые реально нужны этой функции. В beego v2
// (github.com/beego/beego/v2/client/orm) orm.Ormer и orm.TxOrmer - это
// ДВА РАЗНЫХ интерфейса (TxOrmer НЕ реализует Ormer - у Ormer есть лишний
// метод Begin()/BeginWithCtx() для открытия новой транзакции, которого нет
// у TxOrmer), поэтому обычный orm.Ormer здесь не подходит как тип
// параметра - пришлось бы заводить два разных вызова под открытую
// транзакцию и под "голый" orm.NewOrm(). За счёт структурной типизации Go
// этому собственному интерфейсу удовлетворяют оба реальных типа: и
// результат orm.NewOrm() (orm.Ormer), и открытая транзакция (orm.TxOrmer,
// как trn в zabbix/zabbixupdate.go).
type ormQuerySetInserter interface {
	QueryTable(ptrStructOrTableName interface{}) orm.QuerySeter
	InsertMultiWithCtx(ctx context.Context, bulk int, mds interface{}) (int64, error)
}

func ReplaceServerNetworkInterfaces(ctx context.Context, o ormQuerySetInserter, serverSprId int64, source string, rows []*ServerNetworkInterface) error {
	if serverSprId <= 0 || source == "" {
		return nil
	}
	qs := o.QueryTable(new(ServerNetworkInterface))
	if _, err := qs.Filter("ServerSprId", serverSprId).Filter("Source", source).Delete(); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	_, err := o.InsertMultiWithCtx(ctx, 50, rows)
	return err
}

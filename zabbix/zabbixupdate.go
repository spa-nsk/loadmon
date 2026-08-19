// Package zabbix реализует интеграцию с системой мониторинга Zabbix через JSON-RPC API (api_jsonrpc.php):
// получение списка хостов и элементов данных (items), выборку истории значений и сохранение метрик
// (загрузка CPU, память, дисковый ввод-вывод, сетевой трафик и т.п.) в базу данных loadmon.
package zabbix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"loadmon/cmdb"
	"loadmon/models"
	"loadmon/utilities"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/lib/pq"
)

// Client - клиент JSON-RPC API Zabbix (хранит URL, токен авторизации и http-клиент)
type Client struct {
	url   string
	token string
	http  *http.Client
	mu    sync.Mutex
	id    int64
}

// NewClient создаёт клиент Zabbix API по URL и токену авторизации
func NewClient(url, token string) *Client {
	return &Client{
		url:   url,
		token: token,
		http:  &http.Client{Timeout: 120 * time.Second},
		id:    1,
	}
}

// rpcRequest - тело запроса JSON-RPC к Zabbix API
type rpcRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	Auth    string `json:"auth,omitempty"`
	ID      int64  `json:"id"`
}

// rpcError - описание ошибки в ответе JSON-RPC Zabbix API
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// rpcResponse - обобщённое тело ответа JSON-RPC Zabbix API
type rpcResponse[T any] struct {
	Jsonrpc string    `json:"jsonrpc"`
	Result  T         `json:"result"`
	Error   *rpcError `json:"error,omitempty"`
	ID      int64     `json:"id"`
}

// Call выполняет вызов произвольного метода Zabbix JSON-RPC API и распаковывает результат в out
func (c *Client) Call(ctx context.Context, method string, params any, out any) error {
	c.mu.Lock()
	id := c.id
	c.id++
	c.mu.Unlock()

	body, err := json.Marshal(rpcRequest{
		Jsonrpc: "2.0",
		Method:  method,
		Params:  params,
		Auth:    c.token,
		ID:      id,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json-rpc")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token)) //под вопросом
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//os.WriteFile("zabix.resp", b, 0666)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}

	var raw rpcResponse[json.RawMessage]
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if raw.Error != nil {
		return fmt.Errorf("zabbix api error %d: %s (%s)", raw.Error.Code, raw.Error.Message, raw.Error.Data)
	}
	return json.Unmarshal(raw.Result, out)
}

// Interface - сетевой интерфейс хоста Zabbix (тип, адрес, порт)
type Interface struct {
	InterfaceId string `json:"interfaceid"`
	Type        string `json:"type"`
	Main        string `json:"main"`
	UseIp       string `json:"useip"`
	Ip          string `json:"ip"`
	Dns         string `json:"dns"`
	Port        string `json:"port"`
}

// Host - хост Zabbix (ответ метода host.get)
type Host struct {
	HostID       string      `json:"hostid"`
	Host         string      `json:"host"`
	Name         string      `json:"name"`
	Status       string      `json:"status"`
	Available    string      `json:"available"`
	Error        string      `json:"error"`
	Lastaccess   string      `json:"lastaccess"`
	Maintenance  string      `json:"maintenance_status"`
	Interfaces   []Interface `json:"interfaces"`
}

// ItemId - идентификатор элемента данных (item) Zabbix
type ItemId struct {
	ItemID string `json:"itemid"` //id элемента
}

// Item - элемент данных (item) Zabbix с последним значением
type Item struct {
	ItemID    string `json:"itemid"`     //id элемента
	HostID    string `json:"hostid"`     //id хоста
	Name      string `json:"name"`       //наименование элемента
	Key       string `json:"key_"`       //ключ элемента
	ValueType string `json:"value_type"` //тип значения 0 float, 3 uint
	Units     string `json:"units"`      //единицы измерения
	Lastvalue string `json:"lastvalue"`  //последнее значение
	Lastclock string `json:"lastclock"`  //время последнего значения
	Prevvalue string `json:"prevvalue"`  //предыдущее значение
}

// ItemData - элемент данных (item) Zabbix с последним значением, поля приведены к go-типам (в отличие от Item)
type ItemData struct { //структура в формате go
	ItemID    int64     `json:"itemid"`     //id элемента
	HostID    int64     `json:"hostid"`     //id хоста
	Name      string    `json:"name"`       //наименование элемента
	Key       string    `json:"key_"`       //ключ элемента
	ValueType int       `json:"value_type"` //тип значения 0 float, 3 uint
	Units     string    `json:"units"`      //единицы измерения
	Lastvalue string    `json:"lastvalue"`  //последнее значение
	Lastclock time.Time `json:"lastclock"`  //время последнего значения
	Prevvalue string    `json:"prevvalue"`  //предыдущее значение
}

// HistoryPoint - одна историческая точка значения элемента данных Zabbix (ответ метода history.get)
type HistoryPoint struct {
	ItemID string `json:"itemid"`
	Clock  string `json:"clock"`
	Value  string `json:"value"`
	Ns     string `json:"ns"`
}

/******* API *******/

// GetHosts возвращает список хостов группы "физические серверы" (метод host.get)
func (c *Client) GetHosts(ctx context.Context) ([]Host, error) {
	params := map[string]any{
		//"output":          []string{"hostid", "host", "name"},
		"output":           []string{"hostid", "host", "name", "status", "available", "error", "lastaccess", "maintenance_status", "interfaces"},
		"selectgroups":     []string{"extend"},
		"selectinterfaces": []string{"interfaceid", "type", "main", "ip", "dns", "port", "useip"},
		"filter": map[string]any{
			"groupids": 1185, //только физические серверы
		},
	}

	var hosts []Host
	if err := c.Call(ctx, "host.get", params, &hosts); err != nil {
		return nil, err
	}
	return hosts, nil
}

// GetItemsByHost возвращает все включённые элементы данных указанного хоста (метод item.get)
func (c *Client) GetItemsByHost(ctx context.Context, hostID string) ([]Item, error) {
	params := map[string]any{
		"output":  []string{"itemid", "hostid", "name", "key_", "value_type", "units", "lastvalue", "lastclock"},
		"hostids": []string{hostID},
		"filter": map[string]any{
			"status": "0", // enabled
		},
	}

	var items []Item
	if err := c.Call(ctx, "item.get", params, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// GetItemsFilterByHost возвращает элементы данных хоста, отфильтрованные по заранее заданному
// набору ключей мониторинга (CPU, память, диски, сеть и т.п.)
func (c *Client) GetItemsFilterByHost(ctx context.Context, hostID string) ([]Item, error) {
	params := map[string]any{
		"output":  []string{"itemid", "hostid", "name", "key_", "value_type", "units", "lastvalue", "lastclock"},
		"hostids": []string{hostID},
		"filter": map[string]any{
			"key_": []string{
				"system.cpu.num",
				"system.num.cpu",
				"system.hostname",
				"agent.hostname",
				"system.uname",
				"system.sw.os",
				"system.sw.arch",
				"system.cpu.frequency",
				"wmi.get[root/cimv2",
				"system.cpu.load[,avg1]",
				"vm.memory.size[total]",
				"vm.memory.size[used]",
				"vm.memory.util",
				"vfs.dev.read[",
				"vfs.dev.write[",
				"net.if.in[",
				"net.if.out[",
				"net.if.total[",
				"vfs.dev.read.await",
				"vfs.dev.write.await",
				"latency",
				"vfs.fs.size[",
			},
			"status": "0", // enabled
		},
	}
	var items []Item
	if err := c.Call(ctx, "item.get", params, &items); err != nil {
		return nil, err
	}
	return items, nil
}

/*
Отличие от vfs.dev.latency:
В большинстве случаев vfs.dev.await и vfs.dev.latency в Zabbix связаны и могут быть эквивалентны, особенно в версиях 6 и 7.
Однако:
vfs.dev.await - прямое чтение из await из /proc/diskstats.
vfs.dev.latency - может быть более семантически точным (например, включать усреднение или фильтрацию).
В новых версиях Zabbix latency может быть предпочтительным ключом, но await остается корректным и часто используется.
*/

// GetItemidByHost возвращает идентификаторы элементов данных хоста, отфильтрованных по набору
// ключей мониторинга производительности (аналогично GetItemsFilterByHost, но возвращает только id)
func (c *Client) GetItemidByHost(ctx context.Context, hostID string) ([]ItemId, error) {
	var itemids []ItemId
	params := map[string]any{
		"output":  []string{"itemid"},
		"hostids": []string{hostID},
		"filter": map[string]any{
			"key_": []string{
				"system.cpu.num",
				"system.num.cpu",
				"system.hostname",
				"agent.hostname",
				"system.uname",
				"system.sw.os",
				"system.sw.arch",
				"system.cpu.frequency",
				"wmi.get[root/cimv2",
				"system.cpu.load[,avg1]",
				"vm.memory.size[total]",
				"vm.memory.size[used]",
				"vm.memory.util",
				"vfs.dev.util[",        //показывает утилизацию блочного устройства, то есть насколько диск/устройство было занято операциями ввода-вывода.
				"vfs.dev.queue_size[",  //средний размер очереди запросов к блочному устройству
				"vfs.dev.read[",
				"vfs.dev.write[",
				"net.if.in[",
				"net.if.out[",
				"net.if.total[",
				"vfs.dev.read.await",
				"vfs.dev.write.await",
				"latency",
				"vfs.fs.size[",
				"perf_counter[\"\\LogicalDisk(",
			},
			"status": "0", // enabled
		},
	}
	if err := c.Call(ctx, "item.get", params, &itemids); err != nil {
		return nil, err
	}
	return itemids, nil
}

// HistoryGetMinutes возвращает исторические точки за последние 5 минут по всем отслеживаемым
// элементам данных хоста (метод history.get)
func (c *Client) HistoryGetMinutes(ctx context.Context, hostID string) ([]HistoryPoint, error) {
	since := time.Now().Local()
	itemIDs, err := c.GetItemidByHost(ctx, hostID)
	if err != nil {
		return nil, err
	}
	itemids := make([]string, 0)
	for _, rec := range itemIDs {
		itemids = append(itemids, rec.ItemID)
	}
	params := map[string]any{
		"output":    []string{"itemid", "clock", "value", "ns"},
		"itemids":   itemids,
		"history":   0,
		"time_from": since.Add(-300 * time.Second).Unix(),
		"time_till": since.Unix(),
		"sortfield": "clock",
		"sortorder": "DESC"}

	var points []HistoryPoint
	if err := c.Call(ctx, "history.get", params, &points); err != nil {
		return nil, err
	}
	return points, nil
}

// HistoryGetItem возвращает исторические точки конкретного элемента данных за 10 минут до момента since
func (c *Client) HistoryGetItem(ctx context.Context, hostID string, item Item, since time.Time) ([]HistoryPoint, error) {
	itemids := make([]string, 0)
	itemids = append(itemids, item.ItemID)
	params := map[string]any{
		"output":    []string{"itemid", "clock", "value", "ns"},
		"itemids":   itemids,
		"history":   item.ValueType,
		"time_from": since.Add(-600 * time.Second).Unix(),
		"time_till": since.Unix(),
		"sortfield": "clock",
		"sortorder": "DESC"}

	var points []HistoryPoint
	if err := c.Call(ctx, "history.get", params, &points); err != nil {
		return nil, err
	}
	return points, nil
}

// HistoryGet возвращает исторические точки заданных элементов данных начиная с момента since,
// с ограничением количества записей limit (метод history.get)
func (c *Client) HistoryGet(ctx context.Context, historyType int, itemIDs []string, since time.Time, limit int) ([]HistoryPoint, error) {
	params := map[string]any{
		"output":    []string{"itemid", "clock", "value"},
		"history":   historyType,
		"itemids":   itemIDs,
		"time_from": since.Unix(),
		"sortfield": "clock",
		"sortorder": "DESC",
		"limit":     limit,
	}
	var points []HistoryPoint
	if err := c.Call(ctx, "history.get", params, &points); err != nil {
		return nil, err
	}
	return points, nil
}

/********* mapping/cache ********/

// MetricItem - элемент данных Zabbix, сопоставленный с хостом, для кэша метрик мониторинга
type MetricItem struct {
	HostID    string
	HostName  string
	ItemID    string
	Key       string
	ItemName  string
	Units     string
	ValueType string
}

// Build cache: host -> matched metric items
func buildMetricCache(ctx context.Context, c *Client, hosts []Host, keyPatterns []string) (map[string][]MetricItem, map[string]MetricItem, error) {
	cache := make(map[string][]MetricItem, len(hosts))
	byItemID := make(map[string]MetricItem, 1024)

	for _, h := range hosts {
		hist, err := c.HistoryGetMinutes(ctx, h.HostID)
		_ = hist
		_ = err
		//items, err := c.GetItemsByHost(ctx, h.HostID)
		items, err := c.GetItemsFilterByHost(ctx, h.HostID)
		if err != nil {
			return nil, nil, fmt.Errorf("item.get host %s: %w", h.Host, err)
		}
		for _, it := range items {
			// numeric only (0 float, 3 uint) for history.get metrics
			if it.ValueType != "0" && it.ValueType != "3" {
				continue
			}
			k := it.Key
			ok := false
			for _, p := range keyPatterns {
				if strings.Contains(k, p) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
			mi := MetricItem{
				HostID:    h.HostID,
				HostName:  h.Name,
				ItemID:    it.ItemID,
				Key:       it.Key,
				ItemName:  it.Name,
				Units:     it.Units,
				ValueType: it.ValueType,
			}
			cache[h.HostID] = append(cache[h.HostID], mi)
			byItemID[it.ItemID] = mi
		}
	}

	return cache, byItemID, nil
}

/******** main polling ********/

func chunk[T any](in []T, n int) [][]T {
	if n <= 0 {
		return nil
	}
	var out [][]T
	for i := 0; i < len(in); i += n {
		j := i + n
		if j > len(in) {
			j = len(in)
		}
		out = append(out, in[i:j])
	}
	return out
}

func trimData(str string) string {
	s := strings.TrimRight(str, "\"")
	s = strings.TrimLeft(s, "\"")
	return strings.TrimSpace(s)
}

// GetSubKeyItem извлекает первый параметр ключа элемента данных Zabbix (содержимое до первой запятой
// внутри квадратных скобок), например "vfs.dev.read[sda,,avg1]" -> "sda"
func GetSubKeyItem(key string) string {
	start := strings.Split(key, "[")
	if len(start) > 1 {
		next := strings.Split(start[1], "]")
		if len(next) > 1 {
			sub := strings.Split(next[0], ",")
			if len(sub) > 1 {
				return trimData(sub[0])
			} else {
				return trimData(next[0])
			}
		}
	}
	return trimData(key)
}

// GetSubKey2Item извлекает полное содержимое параметров ключа элемента данных Zabbix
// (всё, что находится внутри квадратных скобок)
func GetSubKey2Item(key string) string {
	start := strings.Split(key, "[")
	if len(start) > 1 {
		next := strings.Split(start[1], "]")
		if len(next) > 1 {
			return trimData(next[0])
		}
	}
	return trimData(key)
}

// GetKeyItem возвращает базовую часть ключа элемента данных Zabbix (без параметров в квадратных скобках)
func GetKeyItem(key string) string {
	start := strings.Split(key, "[")
	if len(start) > 0 {
		return strings.TrimSpace(start[0])
	}
	return strings.TrimSpace(key)
}

// GetKeyName возвращает последнюю часть строки после двоеточия (используется для выделения
// краткого имени элемента данных)
func GetKeyName(key string) string {
	start := strings.Split(key, ":")
	n := len(start)
	if n > 1 {
		return trimData(start[n-1])
	}
	return trimData(key)
}

// GetUnixTime преобразует строку unix-времени (как возвращает Zabbix API) в форматированную строку даты/времени
func GetUnixTime(timeStr string) string {
	clUnix, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		fmt.Println("Ошибка преобразования в int64", timeStr, err)
	}
	if clUnix > 0 {
		t := time.Unix(clUnix, 0)
		return t.Format("2006-01-02 15:04:05")
	}
	return ""
}

// GetTime преобразует строку unix-времени (как возвращает Zabbix API) в time.Time
func GetTime(timeStr string) time.Time {
	clUnix, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		fmt.Println("Ошибка преобразования в int64", timeStr, err)
	}
	if clUnix > 0 {
		return time.Unix(clUnix, 0)
	}
	return time.Date(1901, time.January, 1, 0, 0, 0, 0, time.Local)
}

// GetValueType преобразует строковый код типа значения элемента данных Zabbix в int
func GetValueType(valuetype string) int {
	vtype, err := strconv.ParseInt(valuetype, 10, 32)
	if err != nil {
		return 1
	}
	return int(vtype)
}

func importHostSrv(srv *models.ServerSprRecords, h *Host) {
	srv.Name = h.Host
	nameFQDN, ip, err := cmdb.GetFQDNNameAndIP(h.Host)
	if err == nil {
		srv.NameFQDN = nameFQDN
		srv.IPAddress = ip
	}
	//srv.Active = 1
	//srv.Stasus = 1
}

func importHostDB(hostdb *models.ZabbixHost, h *Host) {
	hostdb.Host = h.Host
	hostdb.Name = h.Name
}

func isZeroValue(valueType, lastvalue string) bool {
	switch valueType {
	case "0":
		val, err := strconv.ParseFloat(lastvalue, 64) //float
		if err == nil {
			if val == 0.0 {
				return true
			}
		}
	case "3": //int
		val, err := strconv.ParseInt(lastvalue, 10, 64)
		if err == nil {
			if val == 0 {
				return true
			}
		}
	}
	return false
}

// SaveHistoryValue сохраняет исторические точки элемента данных в БД loadmon, при необходимости
// создавая новый ключ метрики (models.ZabbixKey) и добавляя записи значений (models.ZabbixItem)
func SaveHistoryValue(trn orm.TxOrmer, hostid int64, point []HistoryPoint, it Item, key string, commit *bool) error {

	//itemkeydb := models.ZabbixKey{Key: GetKeyItem(it.Key)}
	subkey1 := "LogicalDisk"
	subkey2 := "Disk Write Bytes/sec"
	subkey3 := "Disk Read Bytes/sec"
	keyparam := GetSubKeyItem(it.Key)
	keyname := GetKeyName(it.Name)
	units := it.Units
	itemkeydb := models.ZabbixKey{Key: key}
	switch {
	case strings.Contains(it.Key, subkey1) && strings.Contains(it.Key, subkey2):
		itemkeydb.Key = fmt.Sprintf("%s[%s:%s]", key, subkey1, subkey2)
		keyparam = GetSubKey2Item(it.Key)
		keyname = fmt.Sprintf("%s %s", subkey1, subkey2)
		units = "Bytes/sec"
	case strings.Contains(it.Key, subkey1) && strings.Contains(it.Key, subkey3):
		itemkeydb.Key = fmt.Sprintf("%s[%s:%s]", key, subkey1, subkey3)
		keyparam = GetSubKey2Item(it.Key)
		keyname = fmt.Sprintf("%s %s", subkey1, subkey3)
		units = "Bytes/sec"

	}

	err := trn.Read(&itemkeydb, "Key") //поиск ключа
	if err != nil {
		//новый ключ
		if !isZeroValue(it.ValueType, it.Lastvalue) {
			itemkeydb.Name = keyname
			itemkeydb.ValueType = GetValueType(it.ValueType)
			itemkeydb.Units = units
			itemkeydb.Id, err = trn.Insert(&itemkeydb)
			if err != nil {
				fmt.Println("Ошибка записи ключа item Zabbix", itemkeydb.Key)
				return err
			}
		}
	}

	for _, p := range point {
		if !isZeroValue(it.ValueType, p.Value) {
			itemdb := models.ZabbixItem{HostId: hostid, KeyId: itemkeydb.Id, KeyParam: keyparam /* KeyName: keyname,*/, Value: p.Value, Clock: GetTime(p.Clock)}
			_, err = trn.Insert(&itemdb)
			if err != nil {
				fmt.Println("Ошибка вставки записи models.ZabbixItem")
				*commit = false
				continue
			}
		}
	}
	return nil
}

// SaveItemValue сохраняет последнее значение элемента данных в БД loadmon, при необходимости
// создавая новый ключ метрики (models.ZabbixKey) и добавляя запись значения (models.ZabbixItem)
func SaveItemValue(trn orm.TxOrmer, hostid int64, it Item, key string, commit *bool) error {
	//itemkeydb := models.ZabbixKey{Key: GetKeyItem(it.Key)}
	itemkeydb := models.ZabbixKey{Key: key}
	err := trn.Read(&itemkeydb, "Key") //поиск ключа
	if err != nil {
		if !isZeroValue(it.ValueType, it.Lastvalue) {
			//новый ключ
			itemkeydb.Name = GetKeyName(it.Name)
			itemkeydb.ValueType = GetValueType(it.ValueType)
			itemkeydb.Units = it.Units
			itemkeydb.Id, err = trn.Insert(&itemkeydb)
			if err != nil {
				fmt.Println("Ошибка записи ключа item Zabbix", itemkeydb.Key)
				return err
			}
		}
	}

	itemdb := models.ZabbixItem{HostId: hostid, KeyId: itemkeydb.Id, KeyParam: GetSubKeyItem(it.Key) /* KeyName: keyname, */, Value: it.Lastvalue, Clock: GetTime(it.Lastclock)}
	_, err = trn.Insert(&itemdb)
	if err != nil {
		fmt.Println("Ошибка вставки записи models.ZabbixItem")
		*commit = false
	}
	return nil
}

// Get10MinutesHistory - основная точка входа опроса Zabbix: получает список хостов, забирает их
// элементы данных и историю значений за последние минуты и сохраняет/обновляет соответствующие
// записи в БД loadmon (ZabbixHost, ZabbixKey, ZabbixItem, привязку к ServerSprRecords).
func Get10MinutesHistory() {
	url := "https://zabbixcorp.global.bcs/api_jsonrpc.php" // https://.../api_jsonrpc.php
	token := utilities.GetKeyData("spausvc")              // API token

	ctx := context.Background()
	c := NewClient(url, token)

	hosts, err := c.GetHosts(ctx)
	if err != nil {
		fmt.Println("Ошибка получения списка хостов", err)
		return
	}

	log.Printf("Количество хостов: %d\n", len(hosts))
	since := time.Now().Local()

	o := orm.NewOrm()

	for _, h := range hosts {
		records := make([]*models.ServerSprRecords, 0)
		qs := o.QueryTable("server_spr_records")
		qs = qs.OrderBy("active", "name")
		cond := orm.NewCondition()
		shortName := strings.Split(h.Host, ".")[0]
		cond = cond.Or("name__icontains", shortName)
		cond = cond.Or("name__icontains", h.Name)
		cond = cond.Or("name_f_q_d_n__icontains", h.Host)
		cond = cond.Or("name_f_q_d_n__icontains", h.Name)
		qs = qs.SetCond(cond)

		num, err := qs.All(&records)
		_ = num

		foundSrv := false

		if err == nil && num > 0 {
			foundSrv = true
		}

		//log.Printf("\nhost:%s %s\n", h.Host, h.Name)
		log.Printf("host:%s\n", h.Host)
		ipAddress := ""
		for _, rec := range h.Interfaces {
			//fmt.Printf(" id=%s ip=%s %s", rec.InterfaceId, rec.Ip, rec.Dns)
			if rec.Ip != "" {
				ipAddress = ipAddress + rec.Ip + "; "
			}
		}
		ipAddress = strings.TrimSpace(ipAddress)

		fmt.Println("")
		items, err := c.GetItemsByHost(ctx, h.HostID)
		if err != nil {
			fmt.Println("Ошибка получения данных по хосту", err)
			//return nil, nil, fmt.Errorf("item.get host %s: %w", h.Host, err)
		}
		hostdb := models.ZabbixHost{}
		srv := models.ServerSprRecords{}

		trn, err := o.Begin()
		commit := true

		if err != nil {
			fmt.Println("Ошибка начала транзакции", err)
			break //прерываем цикл обработки, так как ошибка критическая
		}
		defer trn.RollbackUnlessCommit() //если транзакция не будет подтверждена то откатываем изменения

		//foundHost := false
		hostdb.Host = h.Host
		err = trn.Read(&hostdb, "Host")
		if err == nil {
			//foundHost = true
		} else {
			importHostDB(&hostdb, &h)
			hostdb.IpAddress = ipAddress
			hostdb.TimeMfy = time.Now().Local()
			hostdbid, err := trn.Insert(&hostdb)
			if err == nil {
				hostdb.Id = hostdbid
			} else {
				fmt.Println("Ошибка вставки записи models.ZabixHost", h.Host)
				commit = false
			}
		}

		newSrv := false
		if foundSrv {
			if num == 1 {
				srv.Id = records[0].Id
			} else {
				for _, rec := range records {
					if rec.Stat == 1 && rec.SerialNumber != "" {
						srv.Id = rec.Id //возможно так
						break
					}
				}
			}
			err = trn.Read(&srv, "Id")
		} else {
			srv.Name = h.Host
			err = trn.Read(&srv, "Name")
		}
		if err == nil {
			hostdb.ServerId = srv.Id
			srv.ZabbixHostId = hostdb.Id
			_, err = trn.Update(&srv)
			if err != nil {
				fmt.Println("Ошибка обновления записи models.ServerSprRecords", srv.NameFQDN, h.Host)
				commit = false
			}
		} else {
			/* блокируем запись в таблицу ServerSprRecords
			newSrv = true
			importHostSrv(&srv, &h) //получить данные из hosts
			srv.Time_Mfy = time.Now().Local()
			serverId, err := trn.Insert(&srv)
			if err == nil {
				hostdb.ServerId = serverId
				//foundSrv = true
			} else {
				fmt.Println("Ошибка вставки записи models.ServerSprRecords", h.Host)
				commit = false
			}
			*/
		}

		for _, it := range items {
			// numeric only (0 float, 3 uint) for history.get metrics
			//fmt.Printf("Key: %s Name: %s ValueType: %s Lastvalue: %s Host: %s Name: %s\n", it.Key, it.Name, it.ValueType, it.Lastvalue, h.Host, h.Name)

			Lastclock := ""
			_ = Lastclock

			clUnix, err := strconv.ParseInt(it.Lastclock, 10, 64)
			if err != nil {
				fmt.Println("Ошибка преобразования в int64", it.Lastclock, err)
			}
			if clUnix > 0 {
				t := time.Unix(clUnix, 0)
				Lastclock = t.Format("2006-01-02 15:04:05")
			}

			if strings.Contains(it.Key, "system.hostname") {
				//fmt.Println("Имя хоста (как зарегистрировано в системе): ", it.Lastvalue)
				if hostdb.NameFQDN == "" {
					hostdb.NameFQDN = strings.TrimSpace(it.Lastvalue)
					if newSrv { // убрать если это не новая запись
						srv.NameFQDN = hostdb.NameFQDN
					}
				}
				continue
			}

			if strings.Contains(it.Key, "agent.hostname") { //может быть не всегда
				//fmt.Println("Имя хоста: ", it.Lastvalue)
				hostdb.NameFQDN = strings.TrimSpace(it.Lastvalue)
				if newSrv { // убрать если это не новая запись
					srv.NameFQDN = hostdb.NameFQDN
				}
				continue
			}

			if strings.Contains(it.Key, "system.uname") {
				//fmt.Println("Полная информация об ОС (uname -a Linux; systeminfo - Windows): ", it.Lastvalue)
				hostdb.OperationalSystem = strings.TrimSpace(it.Lastvalue)
				continue
			}

			if strings.Contains(it.Key, "system.sw.os") {
				//fmt.Println("Операционная система: ", it.Lastvalue)
				hostdb.OperationalSystem = strings.TrimSpace(it.Lastvalue)
				continue
			}

			if strings.Contains(it.Key, "system.sw.arch") {
				//fmt.Println("Архитектура ОС: ", it.Lastvalue)
				hostdb.OSArchitecture = strings.TrimSpace(it.Lastvalue)
				continue
			}

			if strings.Contains(it.Key, "system.num.cpu") {
				//fmt.Println("Количество CPU: ", it.Lastvalue)
				i, err := strconv.ParseInt(it.Lastvalue, 10, 32)
				if err == nil {
					hostdb.LogicalNumCPU = int(i)
				}
				continue
			}

			if strings.Contains(it.Key, "system.cpu.num") {
				//fmt.Println("Количество процессоров: ", it.Lastvalue)
				i, err := strconv.ParseInt(it.Lastvalue, 10, 32)
				if err == nil {
					hostdb.LogicalNumCPU = int(i)
				}
				continue
			}

			if strings.Contains(it.Key, "system.cpu.frequency") {
				//fmt.Println("Текущая частота CPU в герцах (поддерживается на современных системах): ", it.Lastvalue)
				continue
			}

			if strings.Contains(it.Key, "wmi.get[root/cimv2") {
				//fmt.Println("Количество логических процессоров: ", it.Lastvalue)
				i, err := strconv.ParseInt(it.Lastvalue, 10, 32)
				if err == nil {
					hostdb.LogicalNumCPU = int(i)
				}
				continue
			}

			if strings.Contains(it.Key, "system.cpu.load[,avg1]") {
				//fmt.Println("Общая средняя загрузка CPU за последнюю минуту (суммарно по всем ядрам): ", it.Key, it.Lastvalue, Lastclock)

				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, "system.cpu.load[,avg1]", &commit) //
				}
				continue
			}

			if strings.Contains(it.Key, "vm.memory.size[total]") {
				SaveItemValue(trn, hostdb.Id, it, it.Key, &commit)
				//fmt.Println("Общий объём оперативной памяти (в байтах): ", it.Key, it.Lastvalue, Lastclock)
				bytes, err := strconv.ParseInt(it.Lastvalue, 10, 64)
				if err == nil {
					bytes = bytes >> 30
					hostdb.MemoryTotal = float64(bytes)
				}
				continue
			}

			if strings.Contains(it.Key, "vm.memory.size[used]") {
				SaveItemValue(trn, hostdb.Id, it, "vm.memory.size[used]", &commit)
				//fmt.Println("Использовано ОЗУ (в байтах): ", it.Key, it.Lastvalue, Lastclock)
				continue
			}

			if strings.Contains(it.Key, "vm.memory.util") {
				//fmt.Println("Процент использования оперативной памяти (0-100%): ", it.Key, it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, "vm.memory.util", &commit)
					//				for _, p := range point {
					//					fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//				}
				}
				continue
			}

			if strings.Contains(it.Key, "vfs.dev.util[") { //показывает утилизацию блочного устройства, то есть насколько диск/устройство было занято операциями ввода-вывода.
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
				}
				continue
			}

			/*
				Значение	Интерпретация
				0		очереди почти нет
				0.1-1		небольшая нагрузка
				1-5		есть заметная очередь
				5+		возможная перегрузка диска
				10+		сильная очередь, I/O может ждать
				для HDD очередь 2-5 уже может быть проблемой;
				для SATA SSD допустимо больше;
				для NVMe очередь может быть высокой и при этом нормальной.
				зависимости от других параметров
				queue_size растёт
				await растёт
				util близко к 100%
				признак того, что дисковая подсистема может не успевать обрабатывать запросы
			*/

			if strings.Contains(it.Key, "vfs.dev.queue_size") { //средняя длинна очереди блочного устройства
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
				}
				continue
			}

			if strings.Contains(it.Key, "vfs.dev.read[") {
				//fmt.Println("Чтение с устройства: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//			for _, p := range point {
					//				fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//			}
				}
				continue
			}

			if strings.Contains(it.Key, "vfs.dev.write[") {
				//fmt.Println("Запись на устройства: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "net.if.in[") && !strings.Contains(it.Key, "\",") {
				//fmt.Println("Количество байт, полученных по сетевому интерфейсу байт/сек: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "net.if.out[") && !strings.Contains(it.Key, "\",") {
				//fmt.Println("Количество байт, отправленных по интерфейсу байт/сек: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "net.if.total[") && !strings.Contains(it.Key, "\",") {
				//fmt.Println("Общий трафик (входящий + исходящий) по интерфейсу байт/сек: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "vfs.dev.read.await") {
				//fmt.Println("Среднее время ожидания операции чтения на устройстве в миллисекундах: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "vfs.dev.write.await") {
				//fmt.Println("Среднее время ожидания операции записи на устройстве в миллисекундах: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "latency") {
				//fmt.Println("Средняя задержка операции ввода-вывода на устройстве в миллисекундах: ", it.Key, it.Lastvalue, Lastclock)
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}

			if strings.Contains(it.Key, "vfs.fs.size[") && strings.Contains(it.Key, "pused") {
				SaveItemValue(trn, hostdb.Id, it, GetKeyItem(it.Key), &commit)
				//fmt.Println("Процент использования раздела: ", it.Key, GetSubKeyItem(it.Key), it.Lastvalue, Lastclock)
				continue
			}

			if strings.Contains(it.Key, "perf_counter[\\LogicalDisk(") && (strings.Contains(it.Key, "Disk Write Bytes/sec") || strings.Contains(it.Key, "Disk Read Bytes/sec")) {
				point, err := c.HistoryGetItem(ctx, h.HostID, it, since)
				if err == nil {
					SaveHistoryValue(trn, hostdb.Id, point, it, GetKeyItem(it.Key), &commit)
					//for _, p := range point {
					//	fmt.Printf("%s %s\n", GetUnixTime(p.Clock), p.Value)
					//}
				}
				continue
			}
		}
		hostdbTmp := models.ZabbixHost{Id: hostdb.Id}
		err = trn.Read(&hostdbTmp, "Id")
		if err == nil {
			hostdb.TimeMfy = time.Now().Local()

			if strings.Contains(hostdb.Host, ".") && !strings.Contains(hostdb.NameFQDN, ".") {
				hostdb.NameFQDN = hostdb.Host
			}
			if strings.Contains(hostdb.Name, ".") && !strings.Contains(hostdb.NameFQDN, ".") {
				hostdb.NameFQDN = hostdb.Name
			}
			if strings.Contains(srv.NameFQDN, ".") && !strings.Contains(hostdb.NameFQDN, ".") {
				hostdb.NameFQDN = srv.NameFQDN
			}

			hostdb.IpAddress = ipAddress
			_, err = trn.Update(&hostdb)
			if err != nil {
				fmt.Println("Ошибка обновления записи models.ZabbixHost")
				commit = false
			}
		} else {
			fmt.Println("Ошибка чтения записи models.ZabbixHost id=", hostdb.Id)
			commit = false
		}
		/* блокируем запись в ServerSprRecords
		if srv.Id > 0 {
			srvtmp := models.ServerSprRecords{Id: srv.Id}
			err = trn.Read(&srvtmp, "Id")
			if err == nil {
				srv.Time_Mfy = time.Now().Local()
				_, err = trn.Update(&srv)
				if err != nil {
					fmt.Println("Ошибка обновления записи models.ServerSprRecords")
					commit = false
				}
			} else {
				fmt.Println("Ошибка чтения записи models.ServerSprRecords", srv.Id)
				commit = false
			}
		} else {
			fmt.Println("Ошибка server не найден в базе", h.Host)
			commit = false
		}
		*/
		if commit {
			trn.Commit()
		}
	}
}

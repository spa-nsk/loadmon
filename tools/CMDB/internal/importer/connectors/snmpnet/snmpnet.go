// Package snmpnet — коннектор сетевых устройств через SNMPv3 (plan.md E8.4): коммутаторы,
// межсетевые экраны, балансировщики и т. п. без собственного REST/NETCONF API (в т. ч.
// Eltex/QTECH/B4COM) — ENTITY-MIB (шасси, модули, БП, вентиляторы), IF-MIB (интерфейсы), LLDP-MIB
// (соседи по каналу). Всегда authPriv (аутентификация + шифрование) — SNMPv3 без приватности или
// без аутентификации не поддерживается: это ослабление защиты, не частный случай конфигурации.
//
// Один коннектор — одно устройство (harness/03, правило источника СХД распространяется на любой
// опрашиваемый по протоколу источник): имя КЕ — sysName устройства, класс — device_class конфига
// (net_switch/net_firewall/net_loadbalancer/net_wifi_ap/net_proxy/net_mgmt — все используют одну
// таблицу-расширение ci_network_device).
//
// OID стандартных MIB (RFC 1213/2737/2863, IEEE 802.1AB) — публичная, стабильная, десятилетиями не
// менявшаяся часть SNMP; в отличие от остальных коннекторов сессии это НЕ область с рисками
// неверных полей. Риск здесь другой: у SNMP нет способа проверить код на живом устройстве иначе,
// чем реальным аппаратным SNMP-агентом (в отличие от REST/SOAP, где ответ можно замокать HTTP-
// сервером) — протокольный уровень (аутентификация/шифрование USM, ASN.1/BER) не тестируется
// собственным мок-сервером, а полагается на общедоступную, широко используемую библиотеку
// github.com/gosnmp/gosnmp; тестами здесь покрыта только логика сведения PDU -> canon (buildRecords),
// через тонкий интерфейс fetcher с ручной тестовой заглушкой, а не HTTP-мок, как у остальных
// коннекторов E5/E8.
package snmpnet

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
)

// Kind — значение imp.connector.kind.
const Kind = "snmp"

func init() { connector.Register(Kind, New) }

// допустимые классы КЕ устройства — все с таблицей-расширением ci_network_device (seed 005).
var deviceClasses = map[string]bool{
	"net_switch": true, "net_firewall": true, "net_loadbalancer": true,
	"net_wifi_ap": true, "net_proxy": true, "net_mgmt": true,
}

// fetcher — подмножество gosnmp.Handler, которым пользуется коннектор; *gosnmp.GoSNMP реализует
// его напрямую (структурная типизация), без обёртки. Тестовая заглушка подменяет только этот
// интерфейс — протокольный уровень (USM-аутентификация/шифрование, ASN.1) не мокается (см. комментарий пакета).
type fetcher interface {
	Connect() error
	WalkAll(rootOid string) ([]gosnmp.SnmpPDU, error)
	Close() error
}

// Client — коннектор одного устройства.
type Client struct {
	code        string
	host        string
	port        uint16
	user        string
	authProto   gosnmp.SnmpV3AuthProtocol
	authPass    string
	privProto   gosnmp.SnmpV3PrivProtocol
	privPass    string
	deviceClass string
	deviceRole  string
	timeout     time.Duration
}

// New создаёт коннектор. Настройки imp.connector.config: host, port (161), username,
// auth_protocol (SHA/SHA224/SHA256/SHA384/SHA512/MD5, по умолчанию SHA256), auth_pass_env,
// priv_protocol (AES/AES192/AES256/DES, по умолчанию AES), priv_pass_env, device_class (класс КЕ
// — net_switch/net_firewall/net_loadbalancer/net_wifi_ap/net_proxy/net_mgmt, по умолчанию
// net_switch), device_role (свободный текст, необязательно), timeout_sec (на запрос, 10).
func New(cfg connector.Config) (connector.Connector, error) {
	host := cfg.String("host")
	if host == "" {
		return nil, fmt.Errorf("snmp %s: не задан host", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("snmp %s: не задан username", cfg.Code)
	}
	authPass, err := cfg.Secret("auth_pass_env")
	if err != nil {
		return nil, err
	}
	privPass, err := cfg.Secret("priv_pass_env")
	if err != nil {
		return nil, err
	}
	authProto, err := parseAuthProtocol(cfg.String("auth_protocol"))
	if err != nil {
		return nil, fmt.Errorf("snmp %s: %w", cfg.Code, err)
	}
	privProto, err := parsePrivProtocol(cfg.String("priv_protocol"))
	if err != nil {
		return nil, fmt.Errorf("snmp %s: %w", cfg.Code, err)
	}
	class := cfg.String("device_class")
	if class == "" {
		class = "net_switch"
	}
	if !deviceClasses[class] {
		return nil, fmt.Errorf("snmp %s: device_class %q не поддерживается", cfg.Code, class)
	}
	return &Client{
		code: cfg.Code, host: host, port: uint16(cfg.Int("port", 161)), user: user,
		authProto: authProto, authPass: authPass, privProto: privProto, privPass: privPass,
		deviceClass: class, deviceRole: cfg.String("device_role"),
		timeout: time.Duration(cfg.Int("timeout_sec", 10)) * time.Second,
	}, nil
}

func parseAuthProtocol(s string) (gosnmp.SnmpV3AuthProtocol, error) {
	switch strings.ToUpper(s) {
	case "", "SHA256":
		return gosnmp.SHA256, nil
	case "SHA":
		return gosnmp.SHA, nil
	case "SHA224":
		return gosnmp.SHA224, nil
	case "SHA384":
		return gosnmp.SHA384, nil
	case "SHA512":
		return gosnmp.SHA512, nil
	case "MD5":
		return gosnmp.MD5, nil
	}
	return 0, fmt.Errorf("неизвестный auth_protocol %q", s)
}

func parsePrivProtocol(s string) (gosnmp.SnmpV3PrivProtocol, error) {
	switch strings.ToUpper(s) {
	case "", "AES":
		return gosnmp.AES, nil
	case "AES192":
		return gosnmp.AES192, nil
	case "AES256":
		return gosnmp.AES256, nil
	case "DES":
		return gosnmp.DES, nil
	}
	return 0, fmt.Errorf("неизвестный priv_protocol %q", s)
}

func (c *Client) newHandler() *gosnmp.GoSNMP {
	return &gosnmp.GoSNMP{
		Target: c.host, Port: c.port, Version: gosnmp.Version3, Timeout: c.timeout, Retries: 1,
		Context: context.Background(), SecurityModel: gosnmp.UserSecurityModel, MsgFlags: gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName: c.user, AuthenticationProtocol: c.authProto, AuthenticationPassphrase: c.authPass,
			PrivacyProtocol: c.privProto, PrivacyPassphrase: c.privPass,
		},
	}
}

// Test открывает и закрывает сессию.
func (c *Client) Test(ctx context.Context) error {
	h := c.newHandler()
	if err := h.Connect(); err != nil {
		return fmt.Errorf("snmp %s: %w", c.code, err)
	}
	defer h.Close()
	if _, err := h.WalkAll(oidSystem); err != nil {
		return fmt.Errorf("snmp %s: system: %w", c.code, err)
	}
	return nil
}

const (
	oidSystem   = "1.3.6.1.2.1.1"          // sysDescr(.1.0), sysName(.5.0)
	oidEntity   = "1.3.6.1.2.1.47.1.1.1.1" // entPhysicalTable
	oidIfTable  = "1.3.6.1.2.1.2.2.1"      // ifTable
	oidLldpRem  = "1.0.8802.1.1.2.1.4.1.1" // lldpRemEntry
	sysDescrOID = "1.3.6.1.2.1.1.1.0"
	sysNameOID  = "1.3.6.1.2.1.1.5.0"
)

// Extract строит полный снимок устройства.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	h := c.newHandler()
	if err := h.Connect(); err != nil {
		return nil, fmt.Errorf("snmp %s: %w", c.code, err)
	}
	defer h.Close()
	return c.extract(h)
}

// extract — то же самое поверх fetcher (не *gosnmp.GoSNMP напрямую), для теста заглушкой.
func (c *Client) extract(f fetcher) ([]canon.Item, error) {
	sys, err := f.WalkAll(oidSystem)
	if err != nil {
		return nil, fmt.Errorf("snmp %s: system: %w", c.code, err)
	}
	ent, err := f.WalkAll(oidEntity)
	if err != nil {
		return nil, fmt.Errorf("snmp %s: entity: %w", c.code, err)
	}
	ifs, err := f.WalkAll(oidIfTable)
	if err != nil {
		return nil, fmt.Errorf("snmp %s: if-table: %w", c.code, err)
	}
	lldp, err := f.WalkAll(oidLldpRem)
	if err != nil {
		return nil, fmt.Errorf("snmp %s: lldp: %w", c.code, err)
	}
	rec := buildRecord(sys, ent, ifs, lldp, c.code, c.host, c.deviceClass, c.deviceRole)
	return []canon.Item{{Seq: 1, Record: rec}}, nil
}

// pduStr — значение PDU как строка независимо от точного ASN.1-типа (OctetString — []byte/string;
// Integer/Counter32/Gauge32/TimeTicks — через gosnmp.ToBigInt), см. комментарий пакета: типы
// значений стандартны для этих MIB десятилетиями.
func pduStr(p gosnmp.SnmpPDU) string {
	switch v := p.Value.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case nil:
		return ""
	}
	if n := gosnmp.ToBigInt(p.Value); n != nil {
		return n.String()
	}
	return ""
}

// column разбирает OID PDU вида "<root>.<col>.<index...>" на номер колонки (первый компонент
// после root) и хвост-индекс (остаток, может быть составным — LLDP-таблица трёхиндексная).
func column(p gosnmp.SnmpPDU, root string) (col string, index string, ok bool) {
	name := strings.TrimPrefix(p.Name, ".")
	root = strings.TrimPrefix(root, ".")
	rest := strings.TrimPrefix(name, root+".")
	if rest == name { // префикс не совпал
		return "", "", false
	}
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func buildRecord(sys, ent, ifs, lldp []gosnmp.SnmpPDU, code, host, class, role string) canon.Record {
	sysName, sysDescr := "", ""
	for _, p := range sys {
		switch strings.TrimPrefix(p.Name, ".") {
		case sysNameOID:
			sysName = pduStr(p)
		case sysDescrOID:
			sysDescr = pduStr(p)
		}
	}
	name := sysName
	if name == "" {
		name = code
	}

	rec := canon.Record{
		Class: class, SourceName: name, ExternalID: "device:" + code, Identifiers: map[string]string{},
		Attrs: map[string]any{"device_role": nilIfEmpty(role), "os_version": nilIfEmpty(sysDescr), "mgmt_ip": ipOrNil(host)},
	}

	// ENTITY-MIB: шасси (class=3) -> серийный номер/модель устройства; модуль(9)/БП(6)/вентилятор(7)
	// -> физические компоненты.
	entRows := map[string]map[string]string{} // index -> колонка -> значение
	for _, p := range ent {
		colN, idx, ok := column(p, oidEntity)
		if !ok {
			continue
		}
		if entRows[idx] == nil {
			entRows[idx] = map[string]string{}
		}
		entRows[idx][colN] = pduStr(p)
	}
	// entPhysicalDescr=2, entPhysicalClass(int)=5, entPhysicalName=7, entPhysicalSerialNum=11, entPhysicalModelName=13
	classOf := func(idx string) int {
		n := 0
		fmt.Sscanf(entRows[idx]["5"], "%d", &n)
		return n
	}
	var comps []canon.Component
	for idx, row := range entRows {
		switch classOf(idx) {
		case 3: // chassis
			if s := row["11"]; s != "" {
				rec.SerialNumber = s
				rec.Identifiers["serial_number"] = s
			}
			if m := row["13"]; m != "" {
				rec.Vendor = map[string]any{"model": m}
			}
		case 6, 7, 9: // powerSupply, fan, module
			typ := map[int]string{6: "psu", 7: "fan", 9: "module"}[classOf(idx)]
			comps = append(comps, canon.Component{Key: typ + ":" + firstNonEmpty(row["11"], idx), Type: typ,
				Slot: firstNonEmpty(row["7"], idx), PartNumber: row["13"], SerialNumber: row["11"]})
		}
	}
	rec.Components = comps

	// IF-MIB: ifDescr=2, ifSpeed(bps)=5, ifPhysAddress(MAC)=6, ifOperStatus(int, 1=up)=8
	ifRows := map[string]map[string]string{}
	for _, p := range ifs {
		colN, idx, ok := column(p, oidIfTable)
		if !ok {
			continue
		}
		if ifRows[idx] == nil {
			ifRows[idx] = map[string]string{}
		}
		ifRows[idx][colN] = pduStr(p)
	}
	rec.Interfaces = []canon.Interface{}
	for _, row := range ifRows {
		mac := formatMAC(row["6"])
		speed, _ := strconv.ParseInt(row["5"], 10, 64)
		rec.Interfaces = append(rec.Interfaces, canon.Interface{
			Name: row["2"], IfType: "ethernet", MAC: mac, SpeedMbps: speed / 1e6, Enabled: row["8"] == "1",
		})
	}

	// LLDP-MIB: сосед по каналу для локального порта — информационно, в вендорских сведениях
	// устройства (не строится топология/связи — соответствие локальный порт -> сосед не то же
	// самое, что связь между КЕ в CMDB без физической презентации порта в модель сети).
	// Индекс lldpRemEntry трёхчастный: <TimeMark>.<LocalPortNum>.<RemIndex>; здесь используется
	// только LocalPortNum для группировки — lldpRemSysName=9, lldpRemPortId=7.
	lldpRows := map[string]map[string]string{}
	for _, p := range lldp {
		colN, idx, ok := column(p, oidLldpRem)
		if !ok {
			continue
		}
		parts := strings.SplitN(idx, ".", 3)
		if len(parts) < 2 {
			continue
		}
		localPort := parts[1]
		if lldpRows[localPort] == nil {
			lldpRows[localPort] = map[string]string{}
		}
		lldpRows[localPort][colN] = pduStr(p)
	}
	var neighbors []map[string]any
	for port, row := range lldpRows {
		sysN, portID := row["9"], row["7"]
		if sysN == "" && portID == "" {
			continue
		}
		neighbors = append(neighbors, map[string]any{"local_port": port, "remote_system": nilIfEmpty(sysN), "remote_port": nilIfEmpty(portID)})
	}
	if len(neighbors) > 0 {
		if rec.Vendor == nil {
			rec.Vendor = map[string]any{}
		}
		rec.Vendor["lldp_neighbors"] = neighbors
	}

	return rec
}

// formatMAC — 6 сырых байт ifPhysAddress в aa:bb:cc:dd:ee:ff; неполный/пустой адрес — "".
func formatMAC(raw string) string {
	if len(raw) != 6 {
		return ""
	}
	parts := make([]string, 6)
	for i := 0; i < 6; i++ {
		parts[i] = strconv.FormatUint(uint64(raw[i]), 16)
		if len(parts[i]) == 1 {
			parts[i] = "0" + parts[i]
		}
	}
	return strings.Join(parts, ":")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func ipOrNil(s string) any {
	// значение конфига host — обычно уже IP; DNS-имя не должно ронять запись, колонка inet
	// просто останется NULL.
	if net.ParseIP(strings.TrimSpace(s)) == nil {
		return nil
	}
	return strings.TrimSpace(s)
}

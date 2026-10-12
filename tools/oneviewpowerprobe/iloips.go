package main

// Режим -list-ilo: выгрузка IP-адресов iLO всех server-hardware по
// endpoint'ам HPE OneView (включая Global Dashboard, API 300) - для заявки на
// сетевой доступ (МЭ) с хоста loadmon до iLO (tcp/443).
//
// Что реально отдаёт endpoint, не предполагаем: для каждого endpoint печатаем,
// у скольких серверов адрес найден в mpIpAddress / mpHostInfo.mpIpAddresses /
// только mpDnsName / нет вообще. Если Global Dashboard не отдаёт mp-поля в
// списке, пробуем GET по одному серверу (-list-ilo-detail).

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/utilities"

	oneview "github.com/spa-nsk/oneview2"
)

type iloRow struct {
	Endpoint string
	Product  string
	API      int
	Server   string
	Serial   string
	Model    string
	MpModel  string
	MpLic    string
	MpState  string
	IPs      []string // "ip(Type)"; чистые IP - в Plain
	Plain    []string
	DNS      string
	Source   string // откуда взят адрес
}

func ovPassword(ovi config.OvEndpoint) string {
	pwd := ovi.Password
	if pwd == "password" {
		pwd = utilities.GetKeyData("usr-nsk-dati-02@global.bcs")
	}
	return pwd
}

// collectIloAddrs собирает все адреса iLO сервера без дублей; link-local IPv6
// (fe80::) для заявки не нужны и пропускаются.
func collectIloAddrs(hw *oneview.ServerHardware) (labeled, plain []string, source string) {
	seen := map[string]bool{}
	add := func(addr, typ, src string) {
		addr = strings.TrimSpace(addr)
		if addr == "" || seen[addr] {
			return
		}
		if ip := net.ParseIP(addr); ip != nil && ip.IsLinkLocalUnicast() {
			return
		}
		seen[addr] = true
		plain = append(plain, addr)
		if typ != "" {
			labeled = append(labeled, addr+"("+typ+")")
		} else {
			labeled = append(labeled, addr)
		}
		if source == "" {
			source = src
		} else if !strings.Contains(source, src) {
			source += "+" + src
		}
	}
	add(hw.MpIpAddress, "", "mpIpAddress")
	for _, a := range hw.MpHostInfo.MpIpAddresses {
		add(a.Address, a.Type, "mpHostInfo")
	}
	return
}

func listIloForEndpoint(ctx context.Context, ovi config.OvEndpoint, detail bool) ([]iloRow, []string) {
	var log []string
	cfg := oneview.NewEndpointConfig(ovi.Endpoint, ovi.Domain, ovi.Login, ovPassword(ovi), ovi.Version, config.ConfigData.OneViewOperationTimeoutFor(&ovi))
	c, err := oneview.New(cfg)
	if err != nil {
		return nil, []string{fmt.Sprintf("endpoint=%s ОШИБКА создания клиента: %v", ovi.Endpoint, err)}
	}
	if err := c.Login(ctx); err != nil {
		return nil, []string{fmt.Sprintf("endpoint=%s ОШИБКА входа: %v", ovi.Endpoint, err)}
	}
	defer c.Logout(ctx)

	log = append(log, fmt.Sprintf("endpoint=%s product=%s apiVersion=%d globalDashboard=%v",
		c.BaseURL(), c.Product().String(), c.APIVersion(), c.IsGlobalDashboard()))

	col, err := c.ListServerHardware(ctx, oneview.ListOptions{Count: -1})
	if err != nil {
		return nil, append(log, fmt.Sprintf("  ОШИБКА списка server-hardware: %v", err))
	}
	var rows []iloRow
	for i := range col.Members {
		hw := col.Members[i]
		labeled, plain, src := collectIloAddrs(&hw)
		// В списке mp-полей может не быть (особенно GD) - добираем GET'ом по серверу.
		if len(plain) == 0 && detail && hw.URI != "" {
			if full, e := c.GetServerHardware(ctx, hw.URI); e == nil && full != nil {
				hw = *full
				labeled, plain, src = collectIloAddrs(&hw)
				if len(plain) > 0 {
					src += "(GET)"
				}
			}
		}
		if src == "" {
			src = "нет"
		}
		rows = append(rows, iloRow{
			Endpoint: c.BaseURL(), Product: c.Product().String(), API: c.APIVersion(),
			Server: hw.Name, Serial: hw.SerialNumber, Model: hw.Model,
			MpModel: hw.MpModel, MpLic: hw.MpLicenseType, MpState: hw.MpState,
			IPs: labeled, Plain: plain, DNS: hw.MpDnsName, Source: src,
		})
	}
	var withIP, dnsOnly, none int
	for _, r := range rows {
		switch {
		case len(r.Plain) > 0:
			withIP++
		case r.DNS != "":
			dnsOnly++
		default:
			none++
		}
	}
	log = append(log, fmt.Sprintf("  server-hardware: %d; с IP iLO: %d; только mpDnsName: %d; без адреса: %d",
		len(rows), withIP, dnsOnly, none))
	return rows, log
}

// runListIlo - точка входа режима. only != "" - один endpoint, иначе все ConfigOV.
// csv=true - выводить CSV (; разделитель) вместо читаемого текста.
func runListIlo(only string, detail, csv bool) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var out []string
	var all []iloRow
	for _, ovi := range config.ConfigData.ConfigOV {
		if only != "" && !strings.EqualFold(strings.TrimSpace(ovi.Endpoint), strings.TrimSpace(only)) {
			continue
		}
		rows, lg := listIloForEndpoint(ctx, ovi, detail)
		out = append(out, lg...)
		all = append(all, rows...)
	}

	if csv {
		out = append(out, "endpoint;product;api;server;serial;model;ilo_model;ilo_license;ilo_state;ilo_ips;ilo_dns;source")
		for _, r := range all {
			out = append(out, strings.Join([]string{r.Endpoint, r.Product, fmt.Sprint(r.API), r.Server, r.Serial,
				r.Model, r.MpModel, r.MpLic, r.MpState, strings.Join(r.IPs, ","), r.DNS, r.Source}, ";"))
		}
	} else {
		for _, r := range all {
			out = append(out, fmt.Sprintf("%s | api=%d | %s | sn=%s | %s | iLO=%s %s | ip=%s | dns=%s | src=%s",
				r.Endpoint, r.API, r.Server, r.Serial, r.Model, r.MpModel, r.MpLic,
				strings.Join(r.IPs, ","), r.DNS, r.Source))
		}
	}

	// Сводка для заявки: уникальные IP и подсети /24 (IPv4).
	uniq := map[string]bool{}
	nets := map[string]int{}
	for _, r := range all {
		for _, a := range r.Plain {
			if uniq[a] {
				continue
			}
			uniq[a] = true
			if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
				nets[ip.Mask(net.CIDRMask(24, 32)).String()+"/24"]++
			} else {
				nets["ipv6/прочее"]++
			}
		}
	}
	ips := make([]string, 0, len(uniq))
	for a := range uniq {
		ips = append(ips, a)
	}
	sort.Slice(ips, func(i, j int) bool {
		a, b := net.ParseIP(ips[i]), net.ParseIP(ips[j])
		if a == nil || b == nil {
			return ips[i] < ips[j]
		}
		return strings.Compare(string(a.To16()), string(b.To16())) < 0
	})
	keys := make([]string, 0, len(nets))
	for k := range nets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out = append(out, "", fmt.Sprintf("=== Сводка для заявки: серверов %d, уникальных IP iLO %d, порт tcp/443 (HTTPS iLO/Redfish) ===", len(all), len(ips)))
	out = append(out, "Подсети /24 (адресов в подсети):")
	for _, k := range keys {
		out = append(out, fmt.Sprintf("  %s  (%d)", k, nets[k]))
	}
	out = append(out, "Уникальные IP:")
	out = append(out, strings.Join(ips, "\n"))
	return out
}

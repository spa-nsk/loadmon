// Package zabbixpower получает потребляемую мощность серверов ("Present Power Reading")
// из Zabbix (locator.global.bcs), группа "62.1 NSK - HW".
package zabbixpower

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"loadmon/utilities"
)

var (
	// Endpoint — URL JSON-RPC Zabbix.
	Endpoint = "https://locator.global.bcs/api_jsonrpc.php"
	// Token — API-токен Zabbix (Authorization: Bearer).
	Token = utilities.GetKeyData("spausvc2")
	// HostGroupID — группа "62.1 NSK - HW".
	HostGroupID = 1744
	// ItemName — имя item'а с потребляемой мощностью.
	ItemName = "Present Power Reading"
	// InsecureSkipVerify отключает проверку TLS-сертификата.
	InsecureSkipVerify = true
	// Timeout — таймаут HTTP-запроса.
	Timeout = 60 * time.Second
)

// ServerPower — значение потребляемой мощности одного сервера.
type ServerPower struct {
	HostID   string
	Host     string
	Name     string
	ItemID   string
	ItemName string
	Key      string
	Units    string
	Watts    float64
	Clock    time.Time // время последнего значения (zero, если неизвестно)
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    string `json:"data"`
	} `json:"error"`
}

type zItem struct {
	ItemID    string `json:"itemid"`
	Name      string `json:"name"`
	Key       string `json:"key_"`
	Units     string `json:"units"`
	LastValue string `json:"lastvalue"`
	LastClock string `json:"lastclock"`
	Hosts     []struct {
		HostID string `json:"hostid"`
		Host   string `json:"host"`
		Name   string `json:"name"`
	} `json:"hosts"`
}

func call(ctx context.Context, method string, params, dest interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "method": method, "params": params, "id": 1,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+Token)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: InsecureSkipVerify}
	resp, err := (&http.Client{Timeout: Timeout, Transport: tr}).Do(req)
	if err != nil {
		return fmt.Errorf("zabbix %s: %w", method, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("zabbix %s: read body: %w", method, err)
	}
	if resp.StatusCode/100 != 2 {
		s := string(raw)
		if len(s) > 300 {
			s = s[:300] + "..."
		}
		return fmt.Errorf("zabbix %s: http %d: %s", method, resp.StatusCode, s)
	}
	var rpc rpcResponse
	if err := json.Unmarshal(raw, &rpc); err != nil {
		return fmt.Errorf("zabbix %s: decode: %w", method, err)
	}
	if rpc.Error != nil {
		return fmt.Errorf("zabbix %s: rpc %d: %s (%s)", method, rpc.Error.Code, rpc.Error.Message, rpc.Error.Data)
	}
	return json.Unmarshal(rpc.Result, dest)
}

// GetServerPower возвращает последние значения "Present Power Reading"
// по всем включённым серверам группы HostGroupID.
func GetServerPower(ctx context.Context) ([]ServerPower, error) {
	var items []zItem
	err := call(ctx, "item.get", map[string]interface{}{
		"output":      []string{"itemid", "name", "key_", "units", "lastvalue", "lastclock"},
		"groupids":    []string{strconv.Itoa(HostGroupID)},
		"selectHosts": []string{"hostid", "host", "name"},
		"search":      map[string]interface{}{"name": ItemName},
		"filter":      map[string]interface{}{"status": 0},
		"monitored":   true,
	}, &items)
	if err != nil {
		return nil, err
	}

	out := make([]ServerPower, 0, len(items))
	for _, it := range items {
		if len(it.Hosts) == 0 {
			continue
		}
		w, err := strconv.ParseFloat(it.LastValue, 64)
		if err != nil {
			continue // нет данных / нечисловое значение
		}
		sp := ServerPower{
			HostID:   it.Hosts[0].HostID,
			Host:     it.Hosts[0].Host,
			Name:     it.Hosts[0].Name,
			ItemID:   it.ItemID,
			ItemName: it.Name,
			Key:      it.Key,
			Units:    it.Units,
			Watts:    w,
		}
		if c, err := strconv.ParseInt(it.LastClock, 10, 64); err == nil && c > 0 {
			sp.Clock = time.Unix(c, 0)
		}
		out = append(out, sp)
	}
	return out, nil
}

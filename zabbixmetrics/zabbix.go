package zabbixmetrics

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int64       `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      int64           `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

func (e *rpcError) Error() string {
	if e == nil {
		return ""
	}
	if e.Data != "" {
		return fmt.Sprintf("zabbix rpc %d: %s (%s)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("zabbix rpc %d: %s", e.Code, e.Message)
}

type zabbixHost struct {
	HostID string `json:"hostid"`
	Host   string `json:"host"`
	Name   string `json:"name"`
}

func (h zabbixHost) displayName() string {
	if h.Name != "" {
		return h.Name
	}
	return h.Host
}

type zabbixItem struct {
	ItemID    string `json:"itemid"`
	HostID    string `json:"hostid"`
	Key       string `json:"key_"`
	Name      string `json:"name"`
	ValueType string `json:"value_type"`
	Units     string `json:"units"`
	Status    string `json:"status"`
	State     string `json:"state"`
}

type historyPoint struct {
	ItemID string `json:"itemid"`
	Clock  string `json:"clock"`
	Value  string `json:"value"`
	Ns     string `json:"ns"`
}

type zabbixClient struct {
	endpoint string
	token    string
	http     *http.Client
	seq      atomic.Int64
}

func newZabbixClient() *zabbixClient {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: InsecureSkipVerify}
	return &zabbixClient{
		endpoint: Endpoint,
		token:    Token,
		http: &http.Client{
			Timeout:   Timeout,
			Transport: tr,
		},
	}
}

func (c *zabbixClient) call(ctx context.Context, method string, params interface{}, dest interface{}) error {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      c.seq.Add(1),
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("zabbix %s: %w", method, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("zabbix %s: read body: %w", method, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("zabbix %s: http %d: %s", method, resp.StatusCode, truncate(raw, 300))
	}

	var rpc rpcResponse
	if err := json.Unmarshal(raw, &rpc); err != nil {
		return fmt.Errorf("zabbix %s: decode: %w", method, err)
	}
	if rpc.Error != nil {
		return rpc.Error
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(rpc.Result, dest); err != nil {
		return fmt.Errorf("zabbix %s: decode result: %w", method, err)
	}
	return nil
}

func (c *zabbixClient) hostsByGroupID(ctx context.Context, groupID int) ([]zabbixHost, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("zabbix: invalid host group id %d", groupID)
	}

	var hosts []zabbixHost
	if err := c.call(ctx, "host.get", map[string]interface{}{
		"output":   []string{"hostid", "host", "name"},
		"groupids": []string{strconv.Itoa(groupID)},
		"filter":   map[string]interface{}{"status": 0},
	}, &hosts); err != nil {
		return nil, err
	}
	return hosts, nil
}

func (c *zabbixClient) itemsForHosts(ctx context.Context, hostIDs []string) ([]zabbixItem, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}
	var items []zabbixItem
	if err := c.call(ctx, "item.get", map[string]interface{}{
		"output":     []string{"itemid", "hostid", "key_", "name", "value_type", "units", "status", "state"},
		"hostids":    hostIDs,
		"monitored":  true,
		"search":     map[string]interface{}{"key_": itemSearchNeedles()},
		"searchByAny": true,
		"startSearch": true,
	}, &items); err != nil {
		return nil, err
	}
	out := items[:0]
	for _, it := range items {
		if it.Status == "0" && classifyItem(it) != fieldUnknown {
			out = append(out, it)
		}
	}
	return out, nil
}

func (c *zabbixClient) lastValues(ctx context.Context, items []zabbixItem, from, till time.Time) (map[string]float64, error) {
	byType := map[int][]string{}
	for _, it := range items {
		vt, _ := strconv.Atoi(it.ValueType)
		if vt != 0 && vt != 3 {
			continue
		}
		byType[vt] = append(byType[vt], it.ItemID)
	}
	result := make(map[string]float64, len(items))
	latest := make(map[string]int64, len(items))
	for historyType, ids := range byType {
		for _, chunk := range chunkStrings(ids, 200) {
			var points []historyPoint
			if err := c.call(ctx, "history.get", map[string]interface{}{
				"output":    "extend",
				"history":   historyType,
				"itemids":   chunk,
				"time_from": from.Unix(),
				"time_till": till.Unix(),
				"sortfield": "clock",
				"sortorder": "ASC",
			}, &points); err != nil {
				return nil, err
			}
			for _, p := range points {
				clock, _ := strconv.ParseInt(p.Clock, 10, 64)
				ns, _ := strconv.ParseInt(p.Ns, 10, 64)
				stamp := clock*1_000_000_000 + ns
				if prev, ok := latest[p.ItemID]; ok && stamp < prev {
					continue
				}
				v, err := strconv.ParseFloat(p.Value, 64)
				if err != nil {
					continue
				}
				latest[p.ItemID] = stamp
				result[p.ItemID] = v
			}
		}
	}
	return result, nil
}

func itemSearchNeedles() []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, list := range [][]string{
		ItemKeys.CPUUtil,
		ItemKeys.RAMUtil,
		ItemKeys.DiskReadMBps,
		ItemKeys.DiskWriteMBps,
		ItemKeys.DiskReadLatency,
		ItemKeys.DiskWriteLatency,
		ItemKeys.DiskReadQueue,
		ItemKeys.DiskWriteQueue,
		ItemKeys.DiskQueue,
		ItemKeys.NetIn,
		ItemKeys.NetOut,
	} {
		for _, p := range list {
			add(searchNeedle(p))
		}
	}
	for _, p := range []string{
		"system.cpu.util",
		"vm.memory.util",
		"vm.memory.utilization",
		"vm.memory.size",
		"vfs.dev",
		"net.if.in",
		"net.if.out",
		"perf_counter",
		"perf_counter_en",
	} {
		add(p)
	}
	return out
}

func searchNeedle(pattern string) string {
	if i := indexAny(pattern, `\[`); i >= 0 {
		return pattern[:i]
	}
	if i := indexAny(pattern, ` `); i >= 0 {
		return pattern[:i]
	}
	return pattern
}

func indexAny(s, chars string) int {
	for i, r := range s {
		for _, c := range chars {
			if r == c {
				return i
			}
		}
	}
	return -1
}

func chunkStrings(in []string, size int) [][]string {
	if size <= 0 {
		size = 100
	}
	var out [][]string
	for i := 0; i < len(in); i += size {
		j := i + size
		if j > len(in) {
			j = len(in)
		}
		out = append(out, in[i:j])
	}
	return out
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

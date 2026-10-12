// Package kubernetes — коннектор Kubernetes через API-сервер (plan.md E8.3): кластер, узлы,
// пространства имён. Один API-сервер = один кластер (у core/v1 API нет отдельного объекта
// «кластер» — имя задаётся конфигом cluster_name, как manager_name у vcenter). Кластер и
// API-сервер как управляющая система — существующие классы cluster/virt_manager (тот же приём,
// что у vCenter/OneView/OpenManage: своей таблицы-расширения им не нужно).
//
// Аутентификация — только bearer-токен (token_env, обычно токен ServiceAccount с правами list на
// nodes/namespaces): вход по логину/паролю в современных кластерах отключён по умолчанию,
// авторизация по клиентскому сертификату — отдельная задача, не реализована.
//
// core/v1 API (version, nodes, namespaces) — самая стабильная, документированная часть
// Kubernetes API (GA с 1.0), в отличие от vSAN-коннектора здесь схема ответов проверена не по
// памяти, а является публичным контрактом, поэтому пометки «не проверено» не требуется.
package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
)

// Kind — значение imp.connector.kind.
const Kind = "kubernetes"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного API-сервера Kubernetes.
type Client struct {
	code        string
	http        *httpx.Client
	clusterName string
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://api.example.local:6443),
// token_env (имя переменной окружения с bearer-токеном ServiceAccount), cluster_name (имя КЕ
// кластера, по умолчанию код коннектора), timeout_sec (на запрос, 60), insecure_skip_verify
// (самоподписанный CA кластера).
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("kubernetes %s: не задан base_url", cfg.Code)
	}
	tok, err := cfg.Secret("token_env")
	if err != nil {
		return nil, err
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.Header.Set("Authorization", "Bearer "+tok)
	clusterName := cfg.String("cluster_name")
	if clusterName == "" {
		clusterName = cfg.Code
	}
	return &Client{code: cfg.Code, http: h, clusterName: clusterName}, nil
}

// Test проверяет версию API-сервера и доступ (список узлов правами ServiceAccount).
func (c *Client) Test(ctx context.Context) error {
	var ver versionInfo
	if err := c.http.JSON(ctx, http.MethodGet, "/version", nil, &ver); err != nil {
		return err
	}
	if ver.GitVersion == "" {
		return fmt.Errorf("kubernetes %s: /version не вернул gitVersion", c.code)
	}
	var nodes nodeList
	if err := c.http.JSON(ctx, http.MethodGet, "/api/v1/nodes?limit=1", nil, &nodes); err != nil {
		return fmt.Errorf("kubernetes %s: проверка доступа к nodes: %w", c.code, err)
	}
	return nil
}

type versionInfo struct {
	GitVersion string `json:"gitVersion"`
	Platform   string `json:"platform"`
}

type objectMeta struct {
	Name   string            `json:"name"`
	UID    string            `json:"uid"`
	Labels map[string]string `json:"labels"`
}

type listMeta struct {
	Continue string `json:"continue"`
}

type nodeCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type nodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type nodeInfo struct {
	MachineID      string `json:"machineID"`
	SystemUUID     string `json:"systemUUID"`
	KubeletVersion string `json:"kubeletVersion"`
	OSImage        string `json:"osImage"`
	Architecture   string `json:"architecture"`
}

type node struct {
	Metadata objectMeta `json:"metadata"`
	Status   struct {
		NodeInfo   nodeInfo        `json:"nodeInfo"`
		Addresses  []nodeAddress   `json:"addresses"`
		Conditions []nodeCondition `json:"conditions"`
	} `json:"status"`
}

type nodeList struct {
	Metadata listMeta `json:"metadata"`
	Items    []node   `json:"items"`
}

type namespace struct {
	Metadata objectMeta `json:"metadata"`
	Status   struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

type namespaceList struct {
	Metadata listMeta    `json:"metadata"`
	Items    []namespace `json:"items"`
}

// nodeReady — Ready-условие из status.conditions (harness-стиль: список условий, не единичный флаг).
func nodeReady(conds []nodeCondition) bool {
	for _, c := range conds {
		if c.Type == "Ready" {
			return c.Status == "True"
		}
	}
	return false
}

// nodeRole — по конвенции kubeadm: наличие лейбла node-role.kubernetes.io/control-plane (текущее
// имя) или .../master (устаревшее, встречается на старых кластерах) — узел управляющей плоскости.
func nodeRole(labels map[string]string) string {
	if _, ok := labels["node-role.kubernetes.io/control-plane"]; ok {
		return "control-plane"
	}
	if _, ok := labels["node-role.kubernetes.io/master"]; ok {
		return "control-plane"
	}
	return "worker"
}

func internalIP(addrs []nodeAddress) string {
	for _, a := range addrs {
		if a.Type == "InternalIP" {
			return a.Address
		}
	}
	return ""
}

// Extract строит полный снимок: API-сервер (менеджер), кластер, узлы, пространства имён.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	var ver versionInfo
	if err := c.http.JSON(ctx, http.MethodGet, "/version", nil, &ver); err != nil {
		return nil, fmt.Errorf("kubernetes %s: %w", c.code, err)
	}

	var nodes []node
	next := "/api/v1/nodes?limit=200"
	for i := 0; next != "" && i < 100000; i++ {
		var page nodeList
		if err := c.http.JSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, fmt.Errorf("kubernetes %s: nodes: %w", c.code, err)
		}
		nodes = append(nodes, page.Items...)
		next = continuePath("/api/v1/nodes", 200, page.Metadata.Continue)
	}

	var namespaces []namespace
	next = "/api/v1/namespaces?limit=200"
	for i := 0; next != "" && i < 100000; i++ {
		var page namespaceList
		if err := c.http.JSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, fmt.Errorf("kubernetes %s: namespaces: %w", c.code, err)
		}
		namespaces = append(namespaces, page.Items...)
		next = continuePath("/api/v1/namespaces", 200, page.Metadata.Continue)
	}

	recs := buildRecords(ver.GitVersion, c.code, c.clusterName, nodes, namespaces)
	items := make([]canon.Item, len(recs))
	for i := range recs {
		items[i] = canon.Item{Seq: i + 1, Record: recs[i]}
	}
	return items, nil
}

func continuePath(path string, limit int, token string) string {
	if token == "" {
		return ""
	}
	return fmt.Sprintf("%s?limit=%d&continue=%s", path, limit, url.QueryEscape(token))
}

// buildRecords — сведение snapshot -> canon без обращения к API (тестируется без кластера).
func buildRecords(gitVersion, code, clusterName string, nodes []node, namespaces []namespace) []canon.Record {
	managerID := "manager:" + code
	clusterID := "cluster:" + code
	manager := canon.Record{
		Class: "virt_manager", SourceName: clusterName + "-api", ExternalID: managerID,
		Attrs:     map[string]any{"product": "Kubernetes", "version": gitVersion},
		Relations: []canon.Relation{{Type: "contains", TargetExternalID: clusterID}},
	}
	cluster := canon.Record{
		Class: "cluster", SourceName: clusterName, ExternalID: clusterID,
		Attrs: map[string]any{"cluster_kind": "k8s", "platform": "Kubernetes"},
	}

	var recs []canon.Record
	for _, n := range nodes {
		id := "node:" + n.Metadata.UID
		rec := canon.Record{
			Class: "k8s_node", SourceName: n.Metadata.Name, ExternalID: id, Identifiers: map[string]string{},
			Attrs: map[string]any{"role": nodeRole(n.Metadata.Labels), "kubelet_version": nilIfEmpty(n.Status.NodeInfo.KubeletVersion),
				"os_image": nilIfEmpty(n.Status.NodeInfo.OSImage), "arch": nilIfEmpty(n.Status.NodeInfo.Architecture),
				"ready": nodeReady(n.Status.Conditions)},
			Relations: []canon.Relation{{Type: "member_of", TargetExternalID: clusterID}},
		}
		// systemUUID узла = BIOS UUID гипервизора/ВМ, на которой он запущен (E5.5 — мост к vcenter/zvirt).
		if u := strings.ToLower(n.Status.NodeInfo.SystemUUID); u != "" {
			rec.Identifiers["bios_uuid"] = u
		}
		if ip := internalIP(n.Status.Addresses); ip != "" {
			rec.Vendor = map[string]any{"internal_ip": ip}
		}
		recs = append(recs, rec)
	}

	for _, ns := range namespaces {
		id := "ns:" + ns.Metadata.UID
		recs = append(recs, canon.Record{
			Class: "k8s_namespace", SourceName: ns.Metadata.Name, ExternalID: id,
			Attrs:     map[string]any{"status": nilIfEmpty(ns.Status.Phase)},
			Relations: []canon.Relation{{Type: "member_of", TargetExternalID: clusterID}},
		})
	}

	return append([]canon.Record{manager, cluster}, recs...)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Package kubernetesmock — mock API-сервера Kubernetes для тестов (plan.md E8.3): /version,
// /api/v1/nodes и /api/v1/namespaces с постраничной выдачей через metadata.continue (тот же
// механизм, что и у настоящего core/v1 API).
package kubernetesmock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Node — один узел кластера.
type Node struct {
	Name           string
	UID            string
	Labels         map[string]string
	MachineID      string
	SystemUUID     string
	KubeletVersion string
	OSImage        string
	Architecture   string
	InternalIP     string
	Ready          bool
}

// Namespace — одно пространство имён.
type Namespace struct {
	Name  string
	UID   string
	Phase string
}

// Server — состояние и обработчик mock API-сервера.
type Server struct {
	mu         sync.Mutex
	Token      string
	GitVersion string
	PageCap    int
	Fail       bool

	Nodes      []Node
	Namespaces []Namespace
}

// Default — стенд: 1 control-plane + 1 worker узел, 2 пространства имён (tag делает UID/имена
// уникальными в общей тестовой БД).
func Default(tag string) *Server {
	return &Server{
		Token: "sa-token-" + tag, GitVersion: "v1.29.2", PageCap: 1,
		Nodes: []Node{
			{Name: "cp-" + tag, UID: "node-cp-" + tag, Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
				MachineID: "machine-cp-" + tag, SystemUUID: "aaaaaaaa-1111-2222-3333-" + pad(tag), KubeletVersion: "v1.29.2",
				OSImage: "Ubuntu 22.04.3 LTS", Architecture: "amd64", InternalIP: "10.1.0.1", Ready: true},
			{Name: "wk-" + tag, UID: "node-wk-" + tag, Labels: map[string]string{},
				MachineID: "machine-wk-" + tag, SystemUUID: "bbbbbbbb-1111-2222-3333-" + pad(tag), KubeletVersion: "v1.29.2",
				OSImage: "Ubuntu 22.04.3 LTS", Architecture: "amd64", InternalIP: "10.1.0.2", Ready: true},
		},
		Namespaces: []Namespace{
			{Name: "default-" + tag, UID: "ns-default-" + tag, Phase: "Active"},
			{Name: "kube-system-" + tag, UID: "ns-kubesys-" + tag, Phase: "Active"},
		},
	}
}

func pad(tag string) string {
	s := "000000000000" + tag
	return s[len(s)-12:]
}

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.URL.Path == "/version" {
		writeJSON(w, http.StatusOK, map[string]any{"gitVersion": s.GitVersion, "platform": "linux/amd64"})
		return
	}

	if s.Token != "" && r.Header.Get("Authorization") != "Bearer "+s.Token {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Unauthorized"})
		return
	}

	switch {
	case r.URL.Path == "/api/v1/nodes":
		if s.Fail {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "internal error"})
			return
		}
		s.serveNodes(w, r)
	case r.URL.Path == "/api/v1/namespaces":
		s.serveNamespaces(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) serveNodes(w http.ResponseWriter, r *http.Request) {
	skip := min(continueOffset(r.URL.Query().Get("continue")), len(s.Nodes))
	end := min(skip+s.PageCap, len(s.Nodes))
	var items []any
	for _, n := range s.Nodes[skip:end] {
		items = append(items, map[string]any{
			"metadata": map[string]any{"name": n.Name, "uid": n.UID, "labels": n.Labels},
			"status": map[string]any{
				"nodeInfo": map[string]any{"machineID": n.MachineID, "systemUUID": n.SystemUUID, "kubeletVersion": n.KubeletVersion,
					"osImage": n.OSImage, "architecture": n.Architecture},
				"addresses":  []any{map[string]any{"type": "InternalIP", "address": n.InternalIP}},
				"conditions": []any{map[string]any{"type": "Ready", "status": boolStr(n.Ready)}},
			},
		})
	}
	out := map[string]any{"kind": "NodeList", "items": items}
	if end < len(s.Nodes) {
		out["metadata"] = map[string]any{"continue": strconv.Itoa(end)}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) serveNamespaces(w http.ResponseWriter, r *http.Request) {
	skip := min(continueOffset(r.URL.Query().Get("continue")), len(s.Namespaces))
	end := min(skip+s.PageCap, len(s.Namespaces))
	var items []any
	for _, ns := range s.Namespaces[skip:end] {
		items = append(items, map[string]any{
			"metadata": map[string]any{"name": ns.Name, "uid": ns.UID},
			"status":   map[string]any{"phase": ns.Phase},
		})
	}
	out := map[string]any{"kind": "NamespaceList", "items": items}
	if end < len(s.Namespaces) {
		out["metadata"] = map[string]any{"continue": strconv.Itoa(end)}
	}
	writeJSON(w, http.StatusOK, out)
}

func continueOffset(token string) int {
	if token == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(token))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func boolStr(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

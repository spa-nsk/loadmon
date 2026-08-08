package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

//go:embed ui.html
var uiHTML embed.FS

type dbPool struct {
	mu   sync.RWMutex
	dbs  map[string]*sql.DB
	path map[string]string
}

func newDBPool() *dbPool {
	return &dbPool{
		dbs:  make(map[string]*sql.DB),
		path: make(map[string]string),
	}
}

func (p *dbPool) add(name, path string) error {
	db, err := openSQLite(path)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if old, ok := p.dbs[name]; ok {
		old.Close()
	}
	p.dbs[name] = db
	p.path[name] = path
	return nil
}

func (p *dbPool) get(name string) (*sql.DB, string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	db, ok := p.dbs[name]
	if !ok {
		return nil, "", fmt.Errorf("база %q не подключена", name)
	}
	return db, p.path[name], nil
}

func (p *dbPool) list() []map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]map[string]string, 0, len(p.dbs))
	for name := range p.dbs {
		out = append(out, map[string]string{"name": name, "path": p.path[name]})
	}
	return out
}

type server struct {
	pool *dbPool
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (s *server) currentDB(r *http.Request) (string, *sql.DB, error) {
	name := r.URL.Query().Get("db")
	if name == "" {
		name = "default"
	}
	db, _, err := s.pool.get(name)
	return name, db, err
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, err := uiHTML.ReadFile("ui.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b)
}

func (s *server) handleDatabases(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.pool.list())
}

func (s *server) handleTables(w http.ResponseWriter, r *http.Request) {
	_, db, err := s.currentDB(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	tables, err := listTables(db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tables)
}

func (s *server) handleSchema(w http.ResponseWriter, r *http.Request) {
	table := strings.TrimPrefix(r.URL.Path, "/api/schema/")
	_, db, err := s.currentDB(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	schema, err := tableSchemaInfo(db, table)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, schema)
}

func (s *server) handleRows(w http.ResponseWriter, r *http.Request) {
	table := strings.TrimPrefix(r.URL.Path, "/api/rows/")
	switch r.Method {
	case http.MethodGet:
		s.getRows(w, r, table)
	case http.MethodPut:
		s.putRow(w, r, table)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) getRows(w http.ResponseWriter, r *http.Request, table string) {
	_, db, err := s.currentDB(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	result, err := tableRows(db, table, limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	schema, _ := tableSchemaInfo(db, table)
	writeJSON(w, http.StatusOK, map[string]any{
		"data":       result,
		"primaryKey": schema.PrimaryKey,
	})
}

func (s *server) putRow(w http.ResponseWriter, r *http.Request, table string) {
	_, db, err := s.currentDB(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req struct {
		PrimaryKey map[string]any `json:"primaryKey"`
		Values     map[string]any `json:"values"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	affected, err := updateRow(db, table, req.PrimaryKey, req.Values)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rowsAffected": affected})
}

func (s *server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, db, err := s.currentDB(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req struct {
		SQL string `json:"sql"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := runSQL(db, req.SQL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/databases", s.handleDatabases)
	mux.HandleFunc("/api/tables", s.handleTables)
	mux.HandleFunc("/api/schema/", s.handleSchema)
	mux.HandleFunc("/api/rows/", s.handleRows)
	mux.HandleFunc("/api/query", s.handleQuery)
	return mux
}

func startServer(addr string, pool *dbPool) error {
	s := &server{pool: pool}
	log.Printf("sqliteadmin: http://%s", addr)
	for _, db := range pool.list() {
		log.Printf("sqliteadmin: база %q -> %s", db["name"], db["path"])
	}
	return http.ListenAndServe(addr, s.routes())
}

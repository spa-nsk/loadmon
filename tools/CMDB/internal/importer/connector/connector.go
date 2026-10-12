// Package connector — контракт коннектора-источника (plan.md E4.3) и реестр видов коннекторов.
// Коннектор знает только свой источник: проверяет доступность и отдаёт полный снимок в
// канонической форме (canon.Item); идентификация, согласование и запись — общий конвейер
// internal/importer/pipeline, одинаковый для CSV и API.
package connector

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"cmdb/internal/importer/canon"
)

// Connector — один настроенный источник.
type Connector interface {
	// Test проверяет доступность и учётные данные источника, ничего не загружая.
	Test(ctx context.Context) error
	// Extract возвращает ПОЛНЫЙ снимок источника. Ошибка означает, что снимок не построен: конвейер
	// не должен считать отсутствующие в нём КЕ «пропавшими» (harness/04, урок 3), поэтому
	// частичный снимок с nil-ошибкой возвращать нельзя — только целиком или ошибка. Ошибочные
	// отдельные записи отдаются в Item.Err и снимок не прерывают.
	Extract(ctx context.Context) ([]canon.Item, error)
}

// Mode — как снимок источника применяется к данным.
type Mode string

const (
	// ModeFull — источник владеет своими КЕ: создаёт, обновляет, помечает пропавшие (по умолчанию).
	ModeFull Mode = "full"
	// ModeSupplement — источник только дополняет уже существующие КЕ (Zabbix, plan.md E5.4): не
	// создаёт КЕ, не переименовывает, заполняет лишь пустые поля и вендорные атрибуты, подтверждает
	// существование (last_seen_at); запись без найденной КЕ пропускается с причиной.
	ModeSupplement Mode = "supplement"
)

// Moded — необязательный интерфейс коннектора, чей режим отличается от ModeFull.
type Moded interface{ Mode() Mode }

// ModeOf возвращает режим коннектора.
func ModeOf(c Connector) Mode {
	if m, ok := c.(Moded); ok {
		return m.Mode()
	}
	return ModeFull
}

// Config — настройки коннектора из imp.connector.
type Config struct {
	Code     string
	Kind     string
	Settings map[string]any // imp.connector.config
}

// String возвращает строковую настройку ("" если нет).
func (c Config) String(key string) string {
	v, _ := c.Settings[key].(string)
	return v
}

// Int возвращает числовую настройку (JSON-числа приходят float64), def — если не задана.
func (c Config) Int(key string, def int) int {
	switch v := c.Settings[key].(type) {
	case float64: // из JSON
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}

// Bool возвращает логическую настройку.
func (c Config) Bool(key string) bool {
	v, _ := c.Settings[key].(bool)
	return v
}

// Secret читает секрет из переменной окружения, имя которой задано настройкой key. Секреты не
// хранятся в БД (imp.connector.config видит любой, кто читает таблицу): в конфигурации лежит
// только имя переменной, значение задаёт окружение процесса cmdb-worker/cmdb-web.
func (c Config) Secret(key string) (string, error) {
	name := c.String(key)
	if name == "" {
		return "", fmt.Errorf("connector %s: не задана настройка %q (имя переменной окружения с секретом)", c.Code, key)
	}
	v, ok := os.LookupEnv(name)
	if !ok || v == "" {
		return "", fmt.Errorf("connector %s: переменная окружения %s не задана", c.Code, name)
	}
	return v, nil
}

// Factory создаёт коннектор вида kind по конфигурации.
type Factory func(cfg Config) (Connector, error)

var (
	mu        sync.RWMutex
	factories = map[string]Factory{}
)

// Register регистрирует вид коннектора; вызывается из init() пакета коннектора.
func Register(kind string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := factories[kind]; dup {
		panic("connector: повторная регистрация вида " + kind)
	}
	factories[kind] = f
}

// New создаёт коннектор по настройке из БД.
func New(cfg Config) (Connector, error) {
	mu.RLock()
	f, ok := factories[cfg.Kind]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("connector %s: неизвестный вид %q (зарегистрированы: %v)", cfg.Code, cfg.Kind, Kinds())
	}
	return f(cfg)
}

// Kinds возвращает зарегистрированные виды (для сообщений об ошибке и UI).
func Kinds() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

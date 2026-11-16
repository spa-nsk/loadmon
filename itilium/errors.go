package itilium

import (
	"errors"
	"fmt"
)

// Ошибки конфигурации и разбора ответа (без сетевого уровня).
var (
	ErrEmptyBaseURL = errors.New("itilium: BaseURL пуст")
	ErrEmptyLogin   = errors.New("itilium: Login пуст")
	ErrBadStatus    = errors.New("itilium: неожиданный HTTP-статус")
)

// APIError — ответ с HTTP-статусом вне 2xx (тело сохраняется для диагностики).
type APIError struct {
	Method string
	URL    string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	body := e.Body
	if len(body) > 512 {
		body = body[:512] + "…"
	}
	return fmt.Sprintf("itilium: %s %s → HTTP %d: %s", e.Method, e.URL, e.Status, body)
}

func (e *APIError) Is(target error) bool { return target == ErrBadStatus }

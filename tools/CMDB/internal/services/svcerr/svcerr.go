// Package svcerr — общие ошибки сервисного слоя, по которым контроллеры выбирают HTTP-статус
// (404 / 409 / 422), не разбирая тексты ошибок БД.
package svcerr

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotFound — запись не существует.
	ErrNotFound = errors.New("запись не найдена")
	// ErrVersionConflict — row_version в БД не совпал с ожидаемым: запись изменил кто-то другой
	// (оптимистическая блокировка, E2.8).
	ErrVersionConflict = errors.New("запись изменена другим пользователем")
	// ErrInvalidState — операция недопустима в текущем состоянии записи.
	ErrInvalidState = errors.New("операция недопустима в текущем состоянии")
	// ErrNoChanges — в запросе на изменение не указано ни одного поля.
	ErrNoChanges = errors.New("не указано ни одного изменяемого поля")
	// ErrPeriodClosed — изменение с датой учёта (effective_at) в закрытом учётном периоде (E6.5).
	ErrPeriodClosed = errors.New("учётный период закрыт")
)

// Conflict оборачивает ErrVersionConflict текущей версией, чтобы UI мог предложить перезагрузку.
func Conflict(entity string, id int64, expected, actual int) error {
	return fmt.Errorf("%w: %s id=%d, ожидалась версия %d, в БД %d", ErrVersionConflict, entity, id, expected, actual)
}

// NotFound оборачивает ErrNotFound названием сущности.
func NotFound(entity string, id int64) error {
	return fmt.Errorf("%w: %s id=%d", ErrNotFound, entity, id)
}

// ErrDuplicate — нарушена уникальность (дубликат имени, адреса, пересечение периода связи).
var ErrDuplicate = errors.New("такая запись уже существует")

// ViolatesConstraint сообщает, что ошибка БД вызвана нарушением ограничения с указанным именем.
// Драйвер pgx кладёт имя ограничения в текст ошибки, поэтому достаточно поиска по подстроке;
// имена ограничений задаются в миграциях явно.
func ViolatesConstraint(err error, constraint string) bool {
	return err != nil && strings.Contains(err.Error(), constraint)
}

// ErrInvalidInput — значение не прошло проверку формата (адрес, WWN, MAC и т.п.).
var ErrInvalidInput = errors.New("некорректное значение")

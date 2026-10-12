// Package logging настраивает структурные JSON-логи (slog) для cmdb-web и cmdb-worker.
package logging

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

// Setup переключает slog на JSON-хендлер с заданным уровнем и делает его логгером по умолчанию.
func Setup(levelName string) *slog.Logger {
	var level slog.Level
	switch levelName {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// WithRequestID возвращает контекст с привязанным идентификатором запроса.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestID достаёт идентификатор запроса из контекста, если он был установлен.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// FromContext возвращает логгер по умолчанию с полем request_id, если оно есть в контексте.
func FromContext(ctx context.Context) *slog.Logger {
	if id := RequestID(ctx); id != "" {
		return slog.Default().With("request_id", id)
	}
	return slog.Default()
}

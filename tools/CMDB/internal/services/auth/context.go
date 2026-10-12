package auth

import "context"

type identityKey struct{}

// WithIdentity кладёт пользователя запроса в контекст (его ставит фильтр аутентификации).
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFromContext достаёт пользователя запроса; ok=false — запрос не аутентифицирован.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}

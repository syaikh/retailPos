package middleware

import (
	"context"
)

type ctxKey string

const (
	CtxKeyUserID   ctxKey = "userID"
	CtxKeyUsername ctxKey = "username"
	CtxKeyRole     ctxKey = "role"
)

func UserIDFromContext(ctx context.Context) *int {
	if v := ctx.Value(CtxKeyUserID); v != nil {
		if id, ok := v.(int); ok && id > 0 {
			return &id
		}
	}
	return nil
}

func UsernameFromContext(ctx context.Context) string {
	if v := ctx.Value(CtxKeyUsername); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func RoleFromContext(ctx context.Context) string {
	if v := ctx.Value(CtxKeyRole); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

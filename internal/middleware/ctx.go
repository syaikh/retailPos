package middleware

import (
	"context"

	"retail-pos-system/internal/shared"
)

type ctxKey string

const (
	CtxKeyUserID      ctxKey = "userID"
	CtxKeyUsername    ctxKey = "username"
	CtxKeyRole        ctxKey = "role"
	CtxKeyStoreID     ctxKey = "storeID"
	CtxKeyPermissions ctxKey = "permissions"
	CtxKeyReportsToID ctxKey = "reportsToID"
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

func IPAddressFromContext(ctx context.Context) string {
	if v := ctx.Value(shared.CtxKeyIPAddress); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func StoreIDFromContext(ctx context.Context) *int {
	if v := ctx.Value(CtxKeyStoreID); v != nil {
		if id, ok := v.(*int); ok {
			return id
		}
	}
	return nil
}

func PermissionsFromContext(ctx context.Context) []string {
	if v := ctx.Value(CtxKeyPermissions); v != nil {
		if perms, ok := v.([]string); ok {
			return perms
		}
	}
	return nil
}

func ReportsToIDFromContext(ctx context.Context) *int {
	if v := ctx.Value(CtxKeyReportsToID); v != nil {
		if id, ok := v.(*int); ok {
			return id
		}
	}
	return nil
}

func UserAgentFromContext(ctx context.Context) string {
	if v := ctx.Value(shared.CtxKeyUserAgent); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

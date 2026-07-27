package middleware

import (
	"context"
	"testing"

	"retail-pos-system/internal/shared"
)

func intPtr(v int) *int { return &v }

func TestUserIDFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUserID, 42)
	got := UserIDFromContext(ctx)
	if got == nil || *got != 42 {
		t.Errorf("expected 42, got %v", got)
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	got := UserIDFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestUserIDFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUserID, "not-int")
	got := UserIDFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestUserIDFromContext_NegativeOrZero(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUserID, 0)
	got := UserIDFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil for zero, got %v", got)
	}
	ctx2 := context.WithValue(context.Background(), CtxKeyUserID, -1)
	got2 := UserIDFromContext(ctx2)
	if got2 != nil {
		t.Errorf("expected nil for negative, got %v", got2)
	}
}

func TestUsernameFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUsername, "admin")
	if got := UsernameFromContext(ctx); got != "admin" {
		t.Errorf("expected admin, got %s", got)
	}
}

func TestUsernameFromContext_Missing(t *testing.T) {
	if got := UsernameFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestUsernameFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUsername, 123)
	if got := UsernameFromContext(ctx); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestRoleFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyRole, "cashier")
	if got := RoleFromContext(ctx); got != "cashier" {
		t.Errorf("expected cashier, got %s", got)
	}
}

func TestRoleFromContext_Missing(t *testing.T) {
	if got := RoleFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestRoleFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyRole, 42)
	if got := RoleFromContext(ctx); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestIPAddressFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), shared.CtxKeyIPAddress, "10.0.0.1")
	if got := IPAddressFromContext(ctx); got != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", got)
	}
}

func TestIPAddressFromContext_Missing(t *testing.T) {
	if got := IPAddressFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestStoreIDFromContext_Present(t *testing.T) {
	sid := intPtr(5)
	ctx := context.WithValue(context.Background(), CtxKeyStoreID, sid)
	got := StoreIDFromContext(ctx)
	if got == nil || *got != 5 {
		t.Errorf("expected 5, got %v", got)
	}
}

func TestStoreIDFromContext_Missing(t *testing.T) {
	got := StoreIDFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestStoreIDFromContext_NilPtr(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyStoreID, (*int)(nil))
	got := StoreIDFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestStoreIDFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyStoreID, 42)
	got := StoreIDFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil for wrong type, got %v", got)
	}
}

func TestUserAgentFromContext_Present(t *testing.T) {
	ctx := context.WithValue(context.Background(), shared.CtxKeyUserAgent, "Mozilla/5.0")
	if got := UserAgentFromContext(ctx); got != "Mozilla/5.0" {
		t.Errorf("expected Mozilla/5.0, got %s", got)
	}
}

func TestUserAgentFromContext_Missing(t *testing.T) {
	if got := UserAgentFromContext(context.Background()); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestUserAgentFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), shared.CtxKeyUserAgent, 123)
	if got := UserAgentFromContext(ctx); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestReportsToIDFromContext_Present(t *testing.T) {
	sid := intPtr(7)
	ctx := context.WithValue(context.Background(), CtxKeyReportsToID, sid)
	got := ReportsToIDFromContext(ctx)
	if got == nil || *got != 7 {
		t.Errorf("expected 7, got %v", got)
	}
}

func TestReportsToIDFromContext_Missing(t *testing.T) {
	got := ReportsToIDFromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestReportsToIDFromContext_NilPtr(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyReportsToID, (*int)(nil))
	got := ReportsToIDFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil for nil pointer, got %v", got)
	}
}

func TestReportsToIDFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyReportsToID, "not-int")
	got := ReportsToIDFromContext(ctx)
	if got != nil {
		t.Errorf("expected nil for wrong type, got %v", got)
	}
}

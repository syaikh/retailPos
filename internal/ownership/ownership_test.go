package ownership

import (
	"testing"

	"retail-pos-system/internal/permissions"
)

func TestResolve_AllAccess(t *testing.T) {
	scope := Resolve(7, true, nil)
	if scope.UserID != nil {
		t.Fatalf("all-access with no filter: UserID = %v, want nil", *scope.UserID)
	}
	if !scope.CanAccess(1) || !scope.CanAccess(99) {
		t.Fatalf("all-access scope must allow any owner")
	}
	if _, restricted := scope.OwnID(); restricted {
		t.Fatalf("all-access scope must not be restricted")
	}
}

func TestResolve_AllAccessWithFilter(t *testing.T) {
	scope := Resolve(7, true, intPtr(42))
	if scope.UserID == nil || *scope.UserID != 42 {
		t.Fatalf("all-access caller filter: UserID = %v, want 42", scope.UserID)
	}
	if !scope.CanAccess(42) {
		t.Fatalf("scope must allow filtered owner 42")
	}
	if scope.CanAccess(1) {
		t.Fatalf("scope must not allow owner 1 when filtered to 42")
	}
}

func TestResolve_RestrictedClampsFilter(t *testing.T) {
	scope := Resolve(7, false, intPtr(42))
	if scope.UserID == nil || *scope.UserID != 7 {
		t.Fatalf("restricted caller must be clamped to own user, got %v", scope.UserID)
	}
	if !scope.CanAccess(7) {
		t.Fatalf("restricted caller must access own rows")
	}
	if scope.CanAccess(42) {
		t.Fatalf("restricted caller filter must never widen access to owner 42")
	}
	if scope.CanAccess(99) {
		t.Fatalf("restricted caller must not access owner 99")
	}
}

func TestResolve_RestrictedNoFilter(t *testing.T) {
	scope := Resolve(7, false, nil)
	if scope.UserID == nil || *scope.UserID != 7 {
		t.Fatalf("restricted caller without filter: UserID = %v, want 7", scope.UserID)
	}
	if _, restricted := scope.OwnID(); !restricted {
		t.Fatalf("restricted caller must be restricted")
	}
}

func TestCanAccessAll(t *testing.T) {
	perms := []string{"dashboard.view", "shift.view", "shift.create"}
	if CanAccessAll(perms, permissions.ShiftReview) {
		t.Fatalf("cashier perms must not grant shift.review all-access")
	}
	if !CanAccessAll(perms, permissions.ShiftView) {
		t.Fatalf("shift.view must be detected in perms")
	}
	managerPerms := []string{"shift.view", "shift.review"}
	if !CanAccessAll(managerPerms, permissions.ShiftReview) {
		t.Fatalf("shift.review must grant all-access")
	}
	if CanAccessAll(nil, permissions.ShiftReview) {
		t.Fatalf("nil perms must not grant all-access")
	}
}

func intPtr(v int) *int {
	return &v
}

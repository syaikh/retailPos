// Package ownership provides reusable row-level scope resolution for
// ownership-scoped resources (shifts today; sales, stock opnames, ... later).
//
// The invariant enforced here: a caller without the all-access permission for
// a resource may only see rows it owns. The all-access permission is chosen
// per resource (e.g. shift.review) and passed in explicitly so the same
// helper stays usable across modules.
package ownership

import "retail-pos-system/internal/permissions"

// Scope is the row-level visibility constraint for an ownership-scoped
// resource.
//
//   - UserID == nil: no user restriction (caller has all-access).
//   - UserID == &X: the caller may only access rows owned by user X.
//   - StoreID == nil: no store restriction (caller is superadmin).
//   - StoreID == &X: the caller may only access rows in store X.
//
// The two dimensions are independent and both must pass: a supervisor with
// shift.review holds an all-access permission on the user dimension but is
// still scoped to its own store, so it is unrestricted across users and
// restricted across stores. Adding the field rather than widening UserID
// keeps the dimensions from being conflated, and keeps the existing callers
// of Resolve/OwnID/CanAccessAll compiling unchanged.
type Scope struct {
	UserID  *int
	StoreID *int
}

// ResolveStore returns the scope's store restriction, or nil when the caller
// is unrestricted. Restricted callers must apply it in their query.
func (s Scope) ResolveStore() *int {
	return s.StoreID
}

// CanAccessStore reports whether the caller may access a row in storeID.
//
// A row with no store (store_id IS NULL) belongs to nobody in particular and
// is in reach of every store-scoped caller. That matches ListUsers and
// ListShifts, which have always shown store-less rows to store-scoped callers;
// rejecting them here would make this filter stricter than the list endpoints
// the same caller can already see, so a row could be visible in a list and
// invisible when opened.
func (s Scope) CanAccessStore(storeID *int) bool {
	if s.StoreID == nil {
		return true
	}
	if storeID == nil {
		return true
	}
	return *storeID == *s.StoreID
}

// Resolve computes the effective row-level scope for a request.
//
// currentUserID is the authenticated caller. canAccessAll indicates whether
// the caller may read every row (typically derived from a resource's
// all-access permission). requestedUserID is the optional ownership filter
// the caller asked for (e.g. a user_id query parameter); it is honored only
// when the caller has all-access, otherwise the scope is clamped to the
// caller's own user so an ownership-filter request can never widen access.
func Resolve(currentUserID int, canAccessAll bool, requestedUserID *int) Scope {
	if canAccessAll {
		return Scope{UserID: requestedUserID}
	}
	return Scope{UserID: &currentUserID}
}

// CanAccessAll reports whether the caller's permission list includes the code
// that grants access to all rows of the resource.
func CanAccessAll(userPerms []string, allAccessPermission permissions.Code) bool {
	for _, p := range userPerms {
		if p == string(allAccessPermission) {
			return true
		}
	}
	return false
}

// CanAccess reports whether the caller may access a row owned by ownerID.
func (s Scope) CanAccess(ownerID int) bool {
	if s.UserID == nil {
		return true
	}
	return *s.UserID == ownerID
}

// OwnID returns the owner the caller is restricted to, and whether a
// restriction is in place. Restricted callers must pass the returned id into
// their query; otherwise the returned bool is false and no restriction applies.
func (s Scope) OwnID() (int, bool) {
	if s.UserID == nil {
		return 0, false
	}
	return *s.UserID, true
}

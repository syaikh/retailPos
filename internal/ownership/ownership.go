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
type Scope struct {
	UserID *int
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

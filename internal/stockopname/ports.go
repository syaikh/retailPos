package stockopname

import (
	"context"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockApplier is the consumer-side port for the inventory subsystem's
// product_stock writes performed when a stock opname posts its adjustment.
// Posting is a Unit of Work (ADR_Cross_Module_Transaction_Strategy), so the
// implementation MUST run against the caller's tx to preserve atomicity — a
// session must never post while its stock write fails. internal/inventory is the
// canonical single-writer of product_stock and provides the production
// implementation; the composition root MUST wire it via SetStockApplier before
// any posting path runs — an unwired service fails fast at runtime.
type StockApplier interface {
	SetProductStock(ctx context.Context, tx pgx.Tx, item shared.StockSetItem) error
	ReconcileLocationStock(ctx context.Context, tx pgx.Tx, reconcile shared.LocationStockReconcile) error
}

// UsernameProvider resolves usernames for stock opname reads (assignments,
// count history, adjustment created-by). The users table is owned by the
// platform bounded context (internal/user); stockopname routes the read here
// instead of a direct LEFT JOIN (ADR §2.8 Platform). The composition root MUST
// wire it via SetUsernameProvider before any such read — an unwired repository
// fails fast at runtime.
type UsernameProvider interface {
	UsernamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error)
}

// AssignableUserProvider lists active users eligible for assignment to a stock
// opname session (counters and supervisors). users/roles are owned by
// internal/user, which owns the eligibility (role, active, deleted) and
// username/email search filtering. stockopname maps the result into its own
// AssignableUser read model. SetAssignableUserProvider MUST be wired before
// ListAssignableUsers runs.
type AssignableUserProvider interface {
	AssignableUsers(ctx context.Context, db shared.DBPool, search string) ([]shared.UserRoleRef, error)
}

// UserRoleNameProvider resolves the role name of a single user. ok is false
// when the user does not exist or is inactive. SetUserRoleNameProvider MUST be
// wired before GetUserRoleName runs.
type UserRoleNameProvider interface {
	RoleNameByUserID(ctx context.Context, db shared.DBPool, userID int) (roleName string, ok bool, err error)
}

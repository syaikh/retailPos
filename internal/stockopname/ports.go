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

// ScopeRef identifies a session scope by type and id. The human-readable name
// lives in the owner module of the referenced table (ADR Modular_Monolith_Module_Boundaries
// §2.8), so stockopname only carries the reference.
type ScopeRef struct {
	ScopeType string
	ScopeID   int64
}

// ScopeNameResolver is the consumer-side port that resolves session scope names
// (store/warehouse/category/brand/supplier/product/location). Each referenced
// table is owned by another bounded context, so stockopname routes the read
// here instead of the legacy correlated-subquery over cross-context tables. The
// composition root MUST wire it via SetScopeNameResolver before any session
// read that renders a scope name — an unwired repository fails fast at runtime.
type ScopeNameResolver interface {
	ScopeNames(ctx context.Context, db shared.DBPool, refs []ScopeRef) (map[ScopeRef]string, error)
}

// LocationScopeProvider is the consumer-side port that resolves the warehouse,
// store, and active state of a storage location for location-scoped sessions.
// storage_locations is owned by internal/storagelocation, whose RackProvider
// satisfies this interface (shared.LocationRack contract). SetLocationScopeProvider
// MUST be wired before GetLocationScope runs.
type LocationScopeProvider interface {
	GetRack(ctx context.Context, db shared.DBPool, locationID int) (*shared.LocationRack, error)
}

// WarehouseStoreIDProvider is the consumer-side port that resolves the store_id
// linked to a warehouse for warehouse-scoped sessions. warehouses is owned by
// internal/store. SetWarehouseStoreIDProvider MUST be wired before
// GetWarehouseStoreID runs.
type WarehouseStoreIDProvider interface {
	WarehouseStoreID(ctx context.Context, db shared.DBPool, warehouseID int) (*int, error)
}

// StockLocker is the consumer-side port that locks product_stock rows ahead of
// stock opname posting so concurrent sessions cannot double-count the same
// stock. Posting is a Unit of Work (ADR_Cross_Module_Transaction_Strategy), so
// the locks MUST be taken on the caller's tx to be released on commit/rollback.
// internal/inventory is the canonical owner of product_stock
// (ADR Modular_Monolith_Module_Boundaries §2.8) and provides the production
// implementation; the composition root MUST wire it via SetStockLocker before
// any posting path runs — an unwired repository fails fast at runtime.
type StockLocker interface {
	// LockProductStock locks the global product_stock rows of the given
	// products within the caller's tx and returns their current quantities.
	LockProductStock(ctx context.Context, tx pgx.Tx, productIDs []int) (map[int]int, error)
	// LockLocationStock locks the rack product_stock rows of the given
	// products on a location within the caller's tx and returns the current
	// rack quantities (0 when no rack row exists).
	LockLocationStock(ctx context.Context, tx pgx.Tx, productIDs []int, locationID int) (map[int]int, error)
}

package user

import (
	"context"
	"fmt"

	"retail-pos-system/internal/shared"
)

// StoreStaffCounts is the user-owned implementation of the store module's
// consumer-side StaffCountsProvider port (structural typing — no import of
// internal/store needed). internal/user is the canonical owner of the
// users/roles tables (ADR Modular_Monolith_Module_Boundaries §2.8 Platform),
// so the onboarding readiness check resolves active staff per role here rather
// than via a direct JOIN from internal/store.
type StoreStaffCounts struct{}

// StaffCountsByStore returns active, non-deleted users per role name for one
// store. Accounts without a store (HQ roles) are intentionally excluded: a
// store is only staffed when the account is assigned to it. Zero-count roles
// are absent from the map.
func (StoreStaffCounts) StaffCountsByStore(ctx context.Context, db shared.DBPool, storeID int) (map[string]int, error) {
	rows, err := db.Query(ctx, `
		SELECT r.name, COUNT(*)::int
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL AND u.is_active = true AND u.store_id = $1
		GROUP BY r.name`, storeID)
	if err != nil {
		return nil, fmt.Errorf("count store staff by role: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var roleName string
		var count int
		if err := rows.Scan(&roleName, &count); err != nil {
			return nil, fmt.Errorf("scan store staff count: %w", err)
		}
		counts[roleName] = count
	}
	return counts, rows.Err()
}

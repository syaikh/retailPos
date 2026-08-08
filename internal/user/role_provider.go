package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// AssignableUsersProvider is the user-owned implementation of the stockopname
// module's consumer-side AssignableUserProvider port (structural typing — no
// import of internal/stockopname needed). internal/user is the canonical owner
// of users/roles (ADR Modular_Monolith_Module_Boundaries §2.8 Platform), so
// stock opname assignment reads resolve eligible users here rather than via a
// direct JOIN on users/roles.
type AssignableUsersProvider struct{}

// AssignableUsers returns active users eligible for assignment to a stock
// opname session (counters and supervisors), optionally filtered by a
// username/email search. Superadmins are excluded as they sit outside the
// day-to-day assignment flow.
func (AssignableUsersProvider) AssignableUsers(ctx context.Context, db shared.DBPool, search string) ([]shared.UserRoleRef, error) {
	rows, err := db.Query(ctx, `
		SELECT u.id, u.username, u.email, u.role_id, r.name
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL AND u.is_active = true
		  AND r.name IN ('cashier', 'staff', 'manager', 'admin')
		  AND ($1 = '' OR u.username ILIKE '%' || $1 || '%' OR u.email ILIKE '%' || $1 || '%')
		ORDER BY u.username ASC`, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]shared.UserRoleRef, 0)
	for rows.Next() {
		var u shared.UserRoleRef
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.RoleID, &u.RoleName); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// RoleNameProvider is the user-owned implementation of the stockopname module's
// consumer-side UserRoleNameProvider port. ok is false when the user does not
// exist or is inactive.
type RoleNameProvider struct{}

// RoleNameByUserID returns the role name of an active user. ok is false when
// the user row is missing or the user is inactive/deleted.
func (RoleNameProvider) RoleNameByUserID(ctx context.Context, db shared.DBPool, userID int) (string, bool, error) {
	var roleName string
	err := db.QueryRow(ctx, `
		SELECT r.name
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL AND u.is_active = true`, userID).Scan(&roleName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return roleName, true, nil
}

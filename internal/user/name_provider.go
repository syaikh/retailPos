package user

import (
	"context"

	"retail-pos-system/internal/shared"
)

// UsernamesProvider is the user-owned implementation of the shift module's
// consumer-side port (shift.UsernameProvider, structural typing — no import of
// internal/shift needed). internal/user is the canonical owner of the users
// table (ADR Modular_Monolith_Module_Boundaries §2.8 Platform), so shift
// listing/detail reads resolve cashier usernames here rather than via a direct
// JOIN on users.
type UsernamesProvider struct{}

// UsernamesByIDs returns a map of user id -> username for the given ids. IDs
// without a user row are absent from the result map.
func (UsernamesProvider) UsernamesByIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	names := make(map[int]string, len(ids))
	rows, err := db.Query(ctx, `
		SELECT id, username
		FROM users
		WHERE id = ANY($1)
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var username string
		if err := rows.Scan(&id, &username); err != nil {
			return nil, err
		}
		names[id] = username
	}
	return names, rows.Err()
}

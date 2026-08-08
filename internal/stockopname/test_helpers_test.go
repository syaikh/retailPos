package stockopname

import (
	"retail-pos-system/internal/user"
)

// newTestRepository returns a Repository wired with the user-owned read ports,
// mirroring the internal/wiring composition. Tests exercise the same provider
// implementations that run in production.
func newTestRepository() *Repository {
	repo := NewRepository(dbPool)
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetAssignableUserProvider(user.AssignableUsersProvider{})
	repo.SetUserRoleNameProvider(user.RoleNameProvider{})
	return repo
}

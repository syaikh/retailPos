package shared

// UserRoleRef is a cross-module DTO describing a user with its role. It is the
// contract between internal/stockopname (consumer) and internal/user (canonical
// owner of users/roles, see ADR_Modular_Monolith_Module_Boundaries §2.8
// Platform) for assignable-user and role-name reads.
type UserRoleRef struct {
	ID       int
	Username string
	Email    string
	RoleID   int
	RoleName string
}

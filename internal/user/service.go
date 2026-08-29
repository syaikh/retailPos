package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrManagerNotFound = errors.New("manager not found")

type Repo interface {
	GetByID(ctx context.Context, id int) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetAllUsers(ctx context.Context, limit, offset int, search string, sortBy string, sortDir string, roleID int, isActive *bool) ([]User, int, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	UpdatePreferences(ctx context.Context, userID int, language, theme string) error
	DeleteUser(ctx context.Context, id int) error
	CountUsersByRole(ctx context.Context, roleID int) (int, error)
	GetAllRoles(ctx context.Context) ([]Role, error)
	GetRoleByID(ctx context.Context, id int) (*Role, error)
	GetRolePermissions(ctx context.Context, roleID int) ([]Permission, error)
	GetAllPermissions(ctx context.Context) ([]Permission, error)
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error
	DeleteRole(ctx context.Context, id int) error
	GetManager(ctx context.Context, userID int) (*User, error)
	GetOrgChart(ctx context.Context) ([]User, error)
	GetSubordinates(ctx context.Context, managerID int) ([]User, error)
	IsSubordinate(ctx context.Context, managerID, userID int) (bool, error)
}

type service struct {
	repo *Repository
}

func NewService(repo *Repository) Service {
	return &service{repo: repo}
}

// ==================== USER ====================

func (s *service) GetUserByID(ctx context.Context, id int) (*User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) GetAllUsers(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
	rid := 0
	if roleID != nil {
		rid = *roleID
	}
	users, total, err := s.repo.GetAllUsers(ctx, limit, offset, search, sortBy, sortDir, rid, isActive)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (s *service) CreateUser(ctx context.Context, user *User) error {
	return s.repo.CreateUser(ctx, user)
}

func (s *service) UpdateUser(ctx context.Context, user *User) error {
	return s.repo.UpdateUser(ctx, user)
}

// InTx runs fn inside a single transaction on the user database, committing on
// success and rolling back on error. Used to make a user mutation and its audit
// log atomic.
func (s *service) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.repo.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateUserTx updates the user within an existing transaction.
func (s *service) UpdateUserTx(ctx context.Context, tx pgx.Tx, user *User) error {
	return s.repo.UpdateUserTx(ctx, tx, user)
}

func (s *service) UpdatePreferences(ctx context.Context, userID int, language, theme string) error {
	return s.repo.UpdatePreferences(ctx, userID, language, theme)
}

func (s *service) DeleteUser(ctx context.Context, id int) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *service) GetSubordinates(ctx context.Context, managerID int) ([]User, error) {
	return s.repo.GetSubordinates(ctx, managerID)
}

func (s *service) GetManager(ctx context.Context, userID int) (*User, error) {
	return s.repo.GetManager(ctx, userID)
}

func (s *service) GetOrgChart(ctx context.Context) ([]User, error) {
	return s.repo.GetOrgChart(ctx)
}

func (s *service) IsSubordinate(ctx context.Context, managerID, userID int) (bool, error) {
	return s.repo.IsSubordinate(ctx, managerID, userID)
}

// ==================== ROLE ====================

func (s *service) GetRoleByID(ctx context.Context, id int) (*Role, error) {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (s *service) GetAllRoles(ctx context.Context) ([]Role, error) {
	roles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *service) CreateRole(ctx context.Context, role *Role) error {
	return s.repo.CreateRole(ctx, role)
}

func (s *service) UpdateRole(ctx context.Context, role *Role) error {
	return s.repo.UpdateRole(ctx, role)
}

func (s *service) DeleteRole(ctx context.Context, id int) error {
	return s.repo.DeleteRole(ctx, id)
}

func (s *service) GetRolePermissions(ctx context.Context, roleID int) ([]Permission, error) {
	perms, err := s.repo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (s *service) UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error {
	return s.repo.UpdateRolePermissions(ctx, roleID, permissionIDs)
}

// UpdateRolePermissionsTx replaces a role's permissions within an existing
// transaction.
func (s *service) UpdateRolePermissionsTx(ctx context.Context, tx pgx.Tx, roleID int, permissionIDs []int) error {
	return s.repo.updateRolePermissionsTx(ctx, tx, roleID, permissionIDs)
}

func (s *service) CountUsersByRole(ctx context.Context, roleID int) (int, error) {
	return s.repo.CountUsersByRole(ctx, roleID)
}

func (s *service) GetAllPermissions(ctx context.Context) ([]Permission, error) {
	perms, err := s.repo.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

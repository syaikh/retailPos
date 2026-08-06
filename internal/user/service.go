package user

import (
	"context"
	"errors"
)

var ErrManagerNotFound = errors.New("manager not found")

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

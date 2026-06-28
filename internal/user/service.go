package user

import (
	"context"

	"retail-pos-system/internal/eventbus"
)

type Service struct {
	repo     *Repository
	eventBus EventBus
}

func NewService(repo *Repository, eventBus EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

// ==================== USER ====================

func (s *Service) GetUserByID(ctx context.Context, id int) (*User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) GetAllUsers(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error) {
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

func (s *Service) CreateUser(ctx context.Context, user *User) error {
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return err
	}
	if err := s.eventBus.Publish(ctx, "user.created", user); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateUser(ctx context.Context, user *User) error {
	old, err := s.repo.GetByID(user.ID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}
	if err := s.eventBus.Publish(ctx, "user.updated", eventbus.UpdatePayload{Old: old, New: user}); err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, id int) error {
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		return err
	}
	if err := s.eventBus.Publish(ctx, "user.deleted", id); err != nil {
		return err
	}
	return nil
}

// ==================== ROLE ====================

func (s *Service) GetRoleByID(ctx context.Context, id int) (*Role, error) {
	role, err := s.repo.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (s *Service) GetAllRoles(ctx context.Context) ([]Role, error) {
	roles, err := s.repo.GetAllRoles(ctx)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *Service) CreateRole(ctx context.Context, role *Role) error {
	if err := s.repo.CreateRole(ctx, role); err != nil {
		return err
	}
	if err := s.eventBus.Publish(ctx, "role.created", role); err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateRole(ctx context.Context, role *Role) error {
	old, err := s.repo.GetRoleByID(ctx, role.ID)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return err
	}
	if err := s.eventBus.Publish(ctx, "role.updated", eventbus.UpdatePayload{Old: old, New: role}); err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteRole(ctx context.Context, id int) error {
	if err := s.repo.DeleteRole(ctx, id); err != nil {
		return err
	}
	if err := s.eventBus.Publish(ctx, "role.deleted", id); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetRolePermissions(ctx context.Context, roleID int) ([]Permission, error) {
	perms, err := s.repo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

func (s *Service) UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error {
	return s.repo.UpdateRolePermissions(ctx, roleID, permissionIDs)
}

func (s *Service) CountUsersByRole(ctx context.Context, roleID int) (int, error) {
	return s.repo.CountUsersByRole(ctx, roleID)
}

func (s *Service) GetAllPermissions(ctx context.Context) ([]Permission, error) {
	perms, err := s.repo.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	return perms, nil
}

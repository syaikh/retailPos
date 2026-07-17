package customergroup

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]CustomerGroup, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive)
}

func (s *Service) GetByID(ctx context.Context, id int) (*CustomerGroup, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAllActive(ctx context.Context) ([]CustomerGroup, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *Service) Create(ctx context.Context, req CustomerGroupCreateRequest) (*CustomerGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	exists, err := s.repo.NameExists(ctx, name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("customer group name already exists")
	}

	cg := &CustomerGroup{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		IsActive:    true,
	}
	if err := s.repo.Create(ctx, cg); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, cg.ID)
}

func (s *Service) Update(ctx context.Context, id int, req CustomerGroupUpdateRequest) (*CustomerGroup, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("customer group not found")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		exists, err := s.repo.NameExists(ctx, name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("customer group name already exists")
		}
		existing.Name = name
	}
	if req.Description != nil {
		existing.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("customer group not found")
	}
	return s.repo.Delete(ctx, id)
}

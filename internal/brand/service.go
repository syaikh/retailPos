package brand

import (
	"context"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetByID(ctx context.Context, id int) (*Brand, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAll(ctx context.Context) ([]Brand, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
	return s.repo.GetAllPaginated(ctx, limit, offset, search)
}

func (s *Service) GetIDByName(ctx context.Context, name string) (int, error) {
	return s.repo.GetIDByName(ctx, name)
}

func (s *Service) Create(ctx context.Context, req *BrandCreateRequest) (*Brand, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	brand := &Brand{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    isActive,
	}
	if err := s.repo.Create(ctx, brand); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, brand.ID)
}

func (s *Service) Update(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
	brand, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	brand.Name = req.Name
	brand.Description = req.Description
	if req.IsActive != nil {
		brand.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, brand); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

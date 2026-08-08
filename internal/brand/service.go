package brand

import (
	"context"
)

type Repo interface {
	GetByID(ctx context.Context, id int) (*Brand, error)
	GetAll(ctx context.Context) ([]Brand, error)
	GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]Brand, int, error)
	GetIDByName(ctx context.Context, name string) (int, error)
	Create(ctx context.Context, brand *Brand) error
	Update(ctx context.Context, brand *Brand) error
	Delete(ctx context.Context, id int) error
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetByID(ctx context.Context, id int) (*Brand, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetAll(ctx context.Context) ([]Brand, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) GetAllPaginated(ctx context.Context, limit, offset int, search string) ([]Brand, int, error) {
	return s.repo.GetAllPaginated(ctx, limit, offset, search)
}

func (s *service) GetIDByName(ctx context.Context, name string) (int, error) {
	return s.repo.GetIDByName(ctx, name)
}

func (s *service) Create(ctx context.Context, req *CreateRequest) (*Brand, error) {
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

func (s *service) Update(ctx context.Context, id int, req *UpdateRequest) (*Brand, error) {
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

func (s *service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

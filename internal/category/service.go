package category

import (
	"context"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *Service) GetCategoryByID(ctx context.Context, id int) (*Category, error) {
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *Service) GetAllCategories(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
	return s.repo.GetAllCategories(ctx, limit, offset, search)
}

func (s *Service) CreateCategory(ctx context.Context, req *CategoryCreateRequest) (*Category, error) {
	category := &Category{
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		IsActive:    true,
	}
	if err := s.repo.CreateCategory(ctx, category); err != nil {
		return nil, err
	}
	return s.repo.GetCategoryByID(ctx, category.ID)
}

func (s *Service) UpdateCategory(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error) {
	category, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	category.Name = req.Name
	category.Description = req.Description
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	category.Slug = generateSlug(category.Name)
	if err := s.repo.UpdateCategory(ctx, category); err != nil {
		return nil, err
	}
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *Service) DeleteCategory(ctx context.Context, id int) error {
	return s.repo.DeleteCategory(ctx, id)
}

func (s *Service) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	return s.repo.SlugExists(ctx, slug, excludeID)
}

func (s *Service) HasActiveProducts(ctx context.Context, categoryID int) (bool, error) {
	return s.repo.HasActiveProducts(ctx, categoryID)
}

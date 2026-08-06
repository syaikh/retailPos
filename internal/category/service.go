package category

import (
	"context"
)

type service struct {
	repo *Repository
}

func NewService(repo *Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *service) GetCategoryByID(ctx context.Context, id int) (*Category, error) {
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *service) GetAllCategories(ctx context.Context, limit, offset int, search string) ([]Category, int, error) {
	return s.repo.GetAllCategories(ctx, limit, offset, search)
}

func (s *service) CreateCategory(ctx context.Context, req *CreateRequest) (*Category, error) {
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

func (s *service) UpdateCategory(ctx context.Context, id int, req *UpdateRequest) (*Category, error) {
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

func (s *service) DeleteCategory(ctx context.Context, id int) error {
	return s.repo.DeleteCategory(ctx, id)
}

func (s *service) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	return s.repo.SlugExists(ctx, slug, excludeID)
}

func (s *service) HasActiveProducts(ctx context.Context, categoryID int) (bool, error) {
	return s.repo.HasActiveProducts(ctx, categoryID)
}

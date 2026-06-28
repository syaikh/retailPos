package category

import (
	"context"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/importutil"
)

type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}

type Service struct {
	repo     *Repository
	eventBus EventBus
}

func NewService(repo *Repository, eventBus EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
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
	if err := s.eventBus.Publish(ctx, "category.created", category); err != nil {
		return nil, err
	}
	return s.repo.GetCategoryByID(ctx, category.ID)
}

func (s *Service) UpdateCategory(ctx context.Context, id int, req *CategoryUpdateRequest) (*Category, error) {
	category, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	old := *category
	category.Name = req.Name
	category.Description = req.Description
	if req.IsActive != nil {
		category.IsActive = *req.IsActive
	}
	category.Slug = generateSlug(category.Name)
	if err := s.repo.UpdateCategory(ctx, category); err != nil {
		return nil, err
	}
	if err := s.eventBus.Publish(ctx, "category.updated", eventbus.UpdatePayload{Old: &old, New: category}); err != nil {
		return nil, err
	}
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *Service) DeleteCategory(ctx context.Context, id int) error {
	if err := s.repo.DeleteCategory(ctx, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "category.deleted", id)
}

func (s *Service) SlugExists(ctx context.Context, slug string, excludeID int) (bool, error) {
	return s.repo.SlugExists(ctx, slug, excludeID)
}

func (s *Service) HasActiveProducts(ctx context.Context, categoryID int) (bool, error) {
	return s.repo.HasActiveProducts(ctx, categoryID)
}

func (s *Service) GetAllCategoriesForExport(ctx context.Context) ([]Category, error) {
	return s.repo.GetAllCategoriesForExport(ctx)
}

func (s *Service) ImportCategories(ctx context.Context, records []CategoryImportRow) importutil.ImportResult {
	return s.repo.BulkUpsertCategories(ctx, records)
}

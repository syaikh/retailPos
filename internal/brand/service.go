package brand

import (
	"context"

	"retail-pos-system/internal/eventbus"
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

func (s *Service) GetByID(ctx context.Context, id int) (*Brand, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAll(ctx context.Context) ([]Brand, error) {
	return s.repo.GetAll(ctx)
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
	if err := s.eventBus.Publish(ctx, "brand.created", brand); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, brand.ID)
}

func (s *Service) Update(ctx context.Context, id int, req *BrandUpdateRequest) (*Brand, error) {
	brand, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	old := *brand
	brand.Name = req.Name
	brand.Description = req.Description
	if req.IsActive != nil {
		brand.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, brand); err != nil {
		return nil, err
	}
	if err := s.eventBus.Publish(ctx, "brand.updated", eventbus.UpdatePayload{Old: &old, New: brand}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "brand.deleted", id)
}



package uom

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

func (s *Service) GetByID(ctx context.Context, id int) (*UnitOfMeasure, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAll(ctx context.Context) ([]UnitOfMeasure, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) GetIDByCode(ctx context.Context, code string) (int, error) {
	return s.repo.GetIDByCode(ctx, code)
}

func (s *Service) Create(ctx context.Context, req *UOMCreateRequest) (*UnitOfMeasure, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	uom := &UnitOfMeasure{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		IsActive:    isActive,
	}
	if err := s.repo.Create(ctx, uom); err != nil {
		return nil, err
	}
	if err := s.eventBus.Publish(ctx, "uom.created", uom); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, uom.ID)
}

func (s *Service) Update(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
	uom, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	old := *uom
	uom.Code = req.Code
	uom.Name = req.Name
	uom.Description = req.Description
	if req.IsActive != nil {
		uom.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, uom); err != nil {
		return nil, err
	}
	if err := s.eventBus.Publish(ctx, "uom.updated", eventbus.UpdatePayload{Old: &old, New: uom}); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "uom.deleted", id)
}



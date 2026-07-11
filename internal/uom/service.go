package uom

import (
	"context"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
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
	return s.repo.GetByID(ctx, uom.ID)
}

func (s *Service) Update(ctx context.Context, id int, req *UOMUpdateRequest) (*UnitOfMeasure, error) {
	uom, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	uom.Code = req.Code
	uom.Name = req.Name
	uom.Description = req.Description
	if req.IsActive != nil {
		uom.IsActive = *req.IsActive
	}
	if err := s.repo.Update(ctx, uom); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

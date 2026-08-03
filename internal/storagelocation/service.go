package storagelocation

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

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]StorageLocation, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive)
}

func (s *Service) GetByID(ctx context.Context, id int) (*StorageLocation, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetAllActive(ctx context.Context) ([]StorageLocation, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *Service) Create(ctx context.Context, req StorageLocationCreateRequest) (*StorageLocation, error) {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.WarehouseID == nil && req.StoreID == nil {
		return nil, fmt.Errorf("warehouse_id or store_id is required")
	}

	if err := s.validateScope(ctx, req.WarehouseID, req.StoreID); err != nil {
		return nil, err
	}

	exists, err := s.repo.CodeExists(ctx, code, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("storage location code already exists")
	}

	sl := &StorageLocation{
		Code:        code,
		Name:        name,
		WarehouseID: req.WarehouseID,
		StoreID:     req.StoreID,
		Notes:       strings.TrimSpace(req.Notes),
		IsActive:    true,
	}
	if err := s.repo.Create(ctx, sl); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, sl.ID)
}

func (s *Service) Update(ctx context.Context, id int, req StorageLocationUpdateRequest) (*StorageLocation, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage location not found")
	}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" {
			return nil, fmt.Errorf("code cannot be empty")
		}
		exists, err := s.repo.CodeExists(ctx, code, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("storage location code already exists")
		}
		existing.Code = code
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		existing.Name = name
	}
	warehouseID := existing.WarehouseID
	storeID := existing.StoreID
	if req.WarehouseID != nil {
		warehouseID = req.WarehouseID
	}
	if req.StoreID != nil {
		storeID = req.StoreID
	}
	if warehouseID == nil && storeID == nil {
		return nil, fmt.Errorf("warehouse_id or store_id is required")
	}
	if err := s.validateScope(ctx, warehouseID, storeID); err != nil {
		return nil, err
	}
	existing.WarehouseID = warehouseID
	existing.StoreID = storeID
	if req.Notes != nil {
		existing.Notes = strings.TrimSpace(*req.Notes)
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
		return fmt.Errorf("storage location not found")
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no IDs provided")
	}
	return s.repo.BulkUpdate(ctx, ids, isActive)
}

func (s *Service) BulkDelete(ctx context.Context, ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no IDs provided")
	}
	return s.repo.BulkDelete(ctx, ids)
}

func (s *Service) validateScope(ctx context.Context, warehouseID, storeID *int) error {
	if warehouseID != nil {
		exists, err := s.repo.WarehouseExists(ctx, *warehouseID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("warehouse not found")
		}
	}
	if storeID != nil {
		exists, err := s.repo.StoreExists(ctx, *storeID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("store not found")
		}
	}
	return nil
}

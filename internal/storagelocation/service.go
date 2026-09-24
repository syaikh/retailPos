package storagelocation

import (
	"context"
	"fmt"
	"strings"
)

type Repo interface {
	GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]StorageLocation, int, error)
	GetByID(ctx context.Context, id int) (*StorageLocation, error)
	GetByIDs(ctx context.Context, ids []int) ([]StorageLocation, error)
	CodeExists(ctx context.Context, code string, excludeID int) (bool, error)
	WarehouseExists(ctx context.Context, id int) (bool, error)
	WarehouseStoreID(ctx context.Context, id int) (*int, error)
	StoreExists(ctx context.Context, id int) (bool, error)
	Create(ctx context.Context, sl *StorageLocation) error
	Update(ctx context.Context, sl *StorageLocation) error
	Delete(ctx context.Context, id int) error
	BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error)
	BulkDelete(ctx context.Context, ids []int) (int, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]StorageLocation, int, error) {
	return s.repo.GetAll(ctx, limit, offset, search, isActive, storeID)
}

// ensureStoreScope enforces the store boundary on a row the caller is about
// to read or write. A nil caller store (superadmin) bypasses the check;
// otherwise the row must belong to the caller's store — either directly via
// store_id, or through a warehouse linked to that store. Warehouses with no
// store (central/unassigned) are superadmin-only.
func (s *Service) ensureStoreScope(ctx context.Context, row *StorageLocation, storeID *int) error {
	if storeID == nil {
		return nil
	}
	if row.StoreID != nil {
		if *row.StoreID == *storeID {
			return nil
		}
		return ErrStoreForbidden
	}
	if row.WarehouseID != nil {
		warehouseStoreID, err := s.repo.WarehouseStoreID(ctx, *row.WarehouseID)
		if err != nil {
			return err
		}
		if warehouseStoreID != nil && *warehouseStoreID == *storeID {
			return nil
		}
	}
	return ErrStoreForbidden
}

// ensureScopeForRequest checks the scope values a create/update request
// targets (before/after merge) against the caller's store.
func (s *Service) ensureScopeForRequest(ctx context.Context, warehouseID, storeID *int, callerStoreID *int) error {
	if callerStoreID == nil {
		return nil
	}
	if storeID != nil && *storeID != *callerStoreID {
		return ErrStoreForbidden
	}
	if warehouseID != nil {
		warehouseStoreID, err := s.repo.WarehouseStoreID(ctx, *warehouseID)
		if err != nil {
			return err
		}
		if warehouseStoreID == nil || *warehouseStoreID != *callerStoreID {
			return ErrStoreForbidden
		}
	}
	if storeID == nil && warehouseID == nil {
		return ErrStoreForbidden
	}
	return nil
}

func (s *Service) GetByID(ctx context.Context, id int, storeID *int) (*StorageLocation, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureStoreScope(ctx, row, storeID); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest, storeID *int) (*StorageLocation, error) {
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
	if err := s.ensureScopeForRequest(ctx, req.WarehouseID, req.StoreID, storeID); err != nil {
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
	return s.GetByID(ctx, sl.ID, storeID)
}

func (s *Service) Update(ctx context.Context, id int, req UpdateRequest, storeID *int) (*StorageLocation, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("storage location not found")
	}
	if err := s.ensureStoreScope(ctx, existing, storeID); err != nil {
		return nil, err
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
	newStoreID := existing.StoreID
	if req.WarehouseID != nil {
		warehouseID = req.WarehouseID
	}
	if req.StoreID != nil {
		newStoreID = req.StoreID
	}
	if warehouseID == nil && newStoreID == nil {
		return nil, fmt.Errorf("warehouse_id or store_id is required")
	}
	if err := s.validateScope(ctx, warehouseID, newStoreID); err != nil {
		return nil, err
	}
	if err := s.ensureScopeForRequest(ctx, warehouseID, newStoreID, storeID); err != nil {
		return nil, err
	}
	existing.WarehouseID = warehouseID
	existing.StoreID = newStoreID
	if req.Notes != nil {
		existing.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id, storeID)
}

func (s *Service) Delete(ctx context.Context, id int, storeID *int) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("storage location not found")
	}
	if err := s.ensureStoreScope(ctx, existing, storeID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) BulkUpdate(ctx context.Context, ids []int, isActive bool, storeID *int) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no IDs provided")
	}
	if err := s.ensureBulkScope(ctx, ids, storeID); err != nil {
		return 0, err
	}
	return s.repo.BulkUpdate(ctx, ids, isActive)
}

func (s *Service) BulkDelete(ctx context.Context, ids []int, storeID *int) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("no IDs provided")
	}
	if err := s.ensureBulkScope(ctx, ids, storeID); err != nil {
		return 0, err
	}
	return s.repo.BulkDelete(ctx, ids)
}

// ensureBulkScope applies the store boundary to every id that exists. One
// foreign row aborts the whole bulk operation (strict semantics — no partial
// write); ids that do not exist are ignored, matching the repository's
// rows-affected behaviour.
func (s *Service) ensureBulkScope(ctx context.Context, ids []int, storeID *int) error {
	if storeID == nil {
		return nil
	}
	rows, err := s.repo.GetByIDs(ctx, ids)
	if err != nil {
		return err
	}
	for i := range rows {
		if err := s.ensureStoreScope(ctx, &rows[i], storeID); err != nil {
			return err
		}
	}
	return nil
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

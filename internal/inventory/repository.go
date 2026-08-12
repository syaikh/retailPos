package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"retail-pos-system/internal/shared"
)

type Repository struct {
	db           shared.DBPool
	stockSyncer  StockSyncer
	locProvider  LocationRackProvider
	metaProvider ProductMetaProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// SetStockSyncer wires the products.stock mirror port, implemented by
// internal/product (see StockSyncer). It MUST be called before
// AdjustStock/AdjustStockBatch run; an unwired repository fails fast at the
// sync point.
func (r *Repository) SetStockSyncer(s StockSyncer) {
	r.stockSyncer = s
}

// SetLocationRackProvider wires the storage_locations read port, implemented
// by internal/storagelocation (see LocationRackProvider). It MUST be called
// before rack-stock paths run; an unwired repository fails fast at the load
// point.
func (r *Repository) SetLocationRackProvider(p LocationRackProvider) {
	r.locProvider = p
}

// SetProductMetaProvider wires the products sku/name read port, implemented by
// internal/product (see ProductMetaProvider). It MUST be called before
// ListLocationStock runs; an unwired repository fails fast at the enrichment
// point.
func (r *Repository) SetProductMetaProvider(p ProductMetaProvider) {
	r.metaProvider = p
}

func (r *Repository) locationProvider() LocationRackProvider {
	if r.locProvider == nil {
		panic("inventory.Repository: LocationRackProvider not wired (SetLocationRackProvider)")
	}
	return r.locProvider
}

func (r *Repository) productMetaProvider() ProductMetaProvider {
	if r.metaProvider == nil {
		panic("inventory.Repository: ProductMetaProvider not wired (SetProductMetaProvider)")
	}
	return r.metaProvider
}

func (r *Repository) GetStockByProductID(ctx context.Context, productID int) (*ProductStock, error) {
	var ps ProductStock
	var storeID, warehouseID sql.NullInt64
	var reorderPoint, reorderQuantity sql.NullInt64
	var lastRestockedAt sql.NullTime
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, warehouse_id, store_id, quantity, reorder_point, reorder_quantity, last_restocked_at, created_at, updated_at
		FROM product_stock
		WHERE product_id = $1
		ORDER BY id ASC LIMIT 1
	`, productID).Scan(&ps.ID, &ps.ProductID, &warehouseID, &storeID, &ps.Quantity, &reorderPoint, &reorderQuantity, &lastRestockedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if warehouseID.Valid {
		v := int(warehouseID.Int64)
		ps.WarehouseID = &v
	}
	if storeID.Valid {
		v := int(storeID.Int64)
		ps.StoreID = &v
	}
	if reorderPoint.Valid {
		v := int(reorderPoint.Int64)
		ps.ReorderPoint = v
	}
	if reorderQuantity.Valid {
		v := int(reorderQuantity.Int64)
		ps.ReorderQuantity = v
	}
	if lastRestockedAt.Valid {
		ps.LastRestockedAt = lastRestockedAt.Time.In(shared.JakartaLocation()).Format(time.RFC3339)
	}
	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	ps.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &ps, nil
}

// AdjustStock applies a signed delta to a product's stock. A non-nil storeID
// (store-scoped manager/staff) routes the delta to that store's product_stock
// row after validating the product belongs to the store; nil storeID
// (superadmin/admin) keeps the global bucket. The products.stock mirror is
// synced to the adjusted bucket's new value.
func (r *Repository) AdjustStock(ctx context.Context, productID int, quantityChange int, storeID *int, userID *int, notes string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if storeID != nil {
		if err := r.checkProductStore(ctx, tx, productID, storeID); err != nil {
			return err
		}
	}

	if err := r.adjustStockInTx(ctx, tx, productID, quantityChange, storeID, userID, notes); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit stock adjustment: %w", err)
	}

	return nil
}

// checkProductStore verifies a store-scoped adjustment only touches a product
// that actually belongs to the caller's store. The product ownership read goes
// through the ProductMetaProvider port so internal/inventory never queries the
// Katalog-owned products table directly. The db parameter is the connection
// pool or an in-flight transaction; passing the transaction makes the
// ownership check atomic with the subsequent stock write.
func (r *Repository) checkProductStore(ctx context.Context, db shared.DBPool, productID int, storeID *int) error {
	metas, err := r.productMetaProvider().ProductMetasByIDs(ctx, db, []int{productID})
	if err != nil {
		return fmt.Errorf("failed to load product store: %w", err)
	}
	meta, ok := metas[productID]
	if !ok {
		return fmt.Errorf("product not found")
	}
	if meta.StoreID == nil || *meta.StoreID != *storeID {
		return ErrStoreForbidden
	}
	return nil
}

// AdjustStockBatch applies all stock deltas in a single transaction so a
// multi-item goods receipt is no longer one transaction per product. Each
// store-scoped adjustment is validated against product ownership before it is
// applied, matching the single-item AdjustStock authorization rule.
func (r *Repository) AdjustStockBatch(ctx context.Context, adjustments []StockAdjustment, userID *int, notes string) error {
	if len(adjustments) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, adj := range adjustments {
		if adj.StoreID != nil {
			if err := r.checkProductStore(ctx, tx, adj.ProductID, adj.StoreID); err != nil {
				return err
			}
		}
		if err := r.adjustStockInTx(ctx, tx, adj.ProductID, adj.QuantityChange, adj.StoreID, userID, notes); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit stock adjustment batch: %w", err)
	}

	return nil
}

// adjustStockInTx is the single store-aware stock deducer shared by
// AdjustStock and AdjustStockBatch. A non-nil storeID targets the per-store row
// (product_id, store_id, warehouse_id IS NULL, location_id IS NULL); a nil
// storeID targets the global bucket. The product_stock row is created on first
// use and the products.stock mirror is synced to the adjusted bucket's value.
func (r *Repository) adjustStockInTx(ctx context.Context, tx pgx.Tx, productID int, quantityChange int, storeID *int, userID *int, notes string) error {
	if quantityChange == 0 {
		return fmt.Errorf("quantity change must not be zero")
	}

	var currentStock int
	var err error
	if storeID != nil {
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(quantity, 0) FROM product_stock
			WHERE product_id = $1 AND store_id = $2 AND warehouse_id IS NULL AND location_id IS NULL
			FOR UPDATE
		`, productID, *storeID).Scan(&currentStock)
	} else {
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(quantity, 0) FROM product_stock
			WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
			FOR UPDATE
		`, productID).Scan(&currentStock)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			currentStock = 0
		} else {
			return fmt.Errorf("failed to load product stock: %w", err)
		}
	}

	newStock := currentStock + quantityChange
	if newStock < 0 {
		return fmt.Errorf("insufficient stock: current %d, requested %d", currentStock, quantityChange)
	}

	var tag pgconn.CommandTag
	if storeID != nil {
		tag, err = tx.Exec(ctx, `
			UPDATE product_stock SET quantity = $1, updated_at = NOW()
			WHERE product_id = $2 AND store_id = $3 AND warehouse_id IS NULL AND location_id IS NULL
		`, newStock, productID, *storeID)
		if err != nil {
			return fmt.Errorf("failed to update stock: %w", err)
		}
		if tag.RowsAffected() == 0 {
			_, err = tx.Exec(ctx, `
				INSERT INTO product_stock (product_id, store_id, quantity, updated_at)
				VALUES ($1, $2, $3, NOW())
			`, productID, *storeID, newStock)
			if err != nil {
				return fmt.Errorf("failed to insert stock: %w", err)
			}
		}
	} else {
		tag, err = tx.Exec(ctx, `
			UPDATE product_stock SET quantity = $1, updated_at = NOW()
			WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
		`, newStock, productID)
		if err != nil {
			return fmt.Errorf("failed to update stock: %w", err)
		}
		if tag.RowsAffected() == 0 {
			_, err = tx.Exec(ctx, `
				INSERT INTO product_stock (product_id, quantity, updated_at)
				VALUES ($1, $2, NOW())
			`, productID, newStock)
			if err != nil {
				return fmt.Errorf("failed to insert stock: %w", err)
			}
		}
	}
	if r.stockSyncer == nil {
		return errors.New("inventory repository: product stock syncer not wired; call SetStockSyncer")
	}
	if err := r.stockSyncer.SyncStock(ctx, tx, productID, newStock); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes)
		VALUES ($1, $2, $3, NULL, NULL, $4, $5)
	`, productID, quantityChange, "adjustment", userID, notes)
	if err != nil {
		return fmt.Errorf("failed to record inventory movement: %w", err)
	}

	return nil
}

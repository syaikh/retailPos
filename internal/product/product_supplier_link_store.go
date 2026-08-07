package product

import (
	"context"
	"fmt"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

// ProductSupplierLinkStore is the product-owned implementation of the supplier
// module's consumer-side port (supplier.ProductSupplierStore, structural
// typing — no import of internal/supplier needed). internal/product is the
// canonical owner of the product_suppliers link table (ADR
// Modular_Monolith_Module_Boundaries §2.8 Katalog), so every read and write
// that internal/supplier performs on that table is computed here rather than
// via direct SQL inside internal/supplier.
type ProductSupplierLinkStore struct{}

// CreateLink inserts a new product-supplier link row.
func (ProductSupplierLinkStore) CreateLink(ctx context.Context, db shared.DBPool, ps *shared.ProductSupplier) error {
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		INSERT INTO product_suppliers (product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, ps.ProductID, ps.SupplierID, ps.SupplierSKU, ps.UnitCost, ps.LeadTimeDays, ps.IsPreferred,
	).Scan(&ps.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("link product supplier: %w", err)
	}
	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

// DeleteLink removes the product-supplier link row.
func (ProductSupplierLinkStore) DeleteLink(ctx context.Context, db shared.DBPool, productID, supplierID int) error {
	_, err := db.Exec(ctx, `
		DELETE FROM product_suppliers WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID)
	if err != nil {
		return fmt.Errorf("unlink product supplier: %w", err)
	}
	return nil
}

// GetLink returns a single link row, or shared.ErrProductSupplierNotFound.
func (ProductSupplierLinkStore) GetLink(ctx context.Context, db shared.DBPool, productID, supplierID int) (*shared.ProductSupplier, error) {
	var ps shared.ProductSupplier
	var createdAt time.Time

	err := db.QueryRow(ctx, `
		SELECT id, product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, created_at
		FROM product_suppliers WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID).Scan(
		&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
		&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.ErrProductSupplierNotFound
		}
		return nil, err
	}

	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &ps, nil
}

// GetPreferredLink returns the preferred link row for a product, or
// shared.ErrProductSupplierNotFound when none is preferred.
func (ProductSupplierLinkStore) GetPreferredLink(ctx context.Context, db shared.DBPool, productID int) (*shared.ProductSupplier, error) {
	var ps shared.ProductSupplier
	var createdAt time.Time

	err := db.QueryRow(ctx, `
		SELECT id, product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, created_at
		FROM product_suppliers WHERE product_id = $1 AND is_preferred = true
	`, productID).Scan(
		&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
		&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, shared.ErrProductSupplierNotFound
		}
		return nil, err
	}

	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &ps, nil
}

// SetPreferredLink makes the given link the product's single preferred one
// (clearing any other preferred row for the product first).
func (ProductSupplierLinkStore) SetPreferredLink(ctx context.Context, db shared.DBPool, productID, supplierID int) error {
	_, err := db.Exec(ctx, `
		UPDATE product_suppliers SET is_preferred = false
		WHERE product_id = $1 AND is_preferred = true
	`, productID)
	if err != nil {
		return fmt.Errorf("clear preferred supplier: %w", err)
	}

	_, err = db.Exec(ctx, `
		UPDATE product_suppliers SET is_preferred = true
		WHERE product_id = $1 AND supplier_id = $2
	`, productID, supplierID)
	if err != nil {
		return fmt.Errorf("set preferred supplier: %w", err)
	}
	return nil
}

// UpdateLink updates the per-supplier metadata of an existing link row.
func (ProductSupplierLinkStore) UpdateLink(ctx context.Context, db shared.DBPool, ps *shared.ProductSupplier) error {
	_, err := db.Exec(ctx, `
		UPDATE product_suppliers
		SET supplier_sku = $1, unit_cost = $2, lead_time_days = $3, is_preferred = $4, updated_at = NOW()
		WHERE product_id = $5 AND supplier_id = $6
	`, ps.SupplierSKU, ps.UnitCost, ps.LeadTimeDays, ps.IsPreferred, ps.ProductID, ps.SupplierID)
	if err != nil {
		return fmt.Errorf("update product supplier: %w", err)
	}
	return nil
}

// ListLinksByProduct returns the raw link rows of a product ordered by
// is_preferred DESC then supplier ID. Supplier enrichment (name/code) is the
// consumer's responsibility on its own suppliers table.
func (ProductSupplierLinkStore) ListLinksByProduct(ctx context.Context, db shared.DBPool, productID int) ([]shared.ProductSupplier, error) {
	rows, err := db.Query(ctx, `
		SELECT id, product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, created_at
		FROM product_suppliers
		WHERE product_id = $1
		ORDER BY is_preferred DESC, supplier_id ASC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []shared.ProductSupplier
	for rows.Next() {
		var ps shared.ProductSupplier
		var createdAt time.Time

		if err := rows.Scan(
			&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
			&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
		); err != nil {
			return nil, err
		}
		ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// ListLinksBySupplier returns the link rows of a supplier with the joined
// product name/SKU (both product_suppliers and products are katalog-owned).
func (ProductSupplierLinkStore) ListLinksBySupplier(ctx context.Context, db shared.DBPool, supplierID int) ([]shared.ProductSupplier, error) {
	rows, err := db.Query(ctx, `
		SELECT ps.id, ps.product_id, ps.supplier_id, ps.supplier_sku, ps.unit_cost, ps.lead_time_days, ps.is_preferred, ps.created_at,
		       p.name, p.sku
		FROM product_suppliers ps
		JOIN products p ON ps.product_id = p.id AND p.deleted_at IS NULL
		WHERE ps.supplier_id = $1
		ORDER BY p.name ASC
	`, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []shared.ProductSupplier
	for rows.Next() {
		var ps shared.ProductSupplier
		var createdAt time.Time
		var productName, productSKU string

		if err := rows.Scan(
			&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
			&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &createdAt,
			&productName, &productSKU,
		); err != nil {
			return nil, err
		}
		ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		ps.ProductName = &productName
		ps.ProductSKU = &productSKU
		result = append(result, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// HasPreferredLink reports whether the product has a preferred supplier.
func (ProductSupplierLinkStore) HasPreferredLink(ctx context.Context, db shared.DBPool, productID int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM product_suppliers WHERE product_id = $1 AND is_preferred = true)
	`, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check preferred supplier: %w", err)
	}
	return exists, nil
}

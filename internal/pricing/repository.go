package pricing

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db shared.DBPool
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetBasePrice(ctx context.Context, productID int) (int, error) {
	var price int
	err := r.db.QueryRow(ctx, `
		SELECT price FROM products WHERE id = $1 AND deleted_at IS NULL
	`, productID).Scan(&price)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, ErrProductNotFound
		}
		return 0, err
	}
	return price, nil
}

func (r *Repository) GetProductScope(ctx context.Context, productID int) (categoryID *int, brandID *int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT category_id, brand_id FROM products WHERE id = $1 AND deleted_at IS NULL
	`, productID).Scan(&categoryID, &brandID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrProductNotFound
		}
		return nil, nil, err
	}
	return categoryID, brandID, nil
}

func (r *Repository) GetBasePricesBatch(ctx context.Context, productIDs []int) (map[int]int, error) {
	if len(productIDs) == 0 {
		return map[int]int{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, price FROM products WHERE id = ANY($1) AND deleted_at IS NULL
	`, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]int, len(productIDs))
	for rows.Next() {
		var id, price int
		if err := rows.Scan(&id, &price); err != nil {
			return nil, err
		}
		result[id] = price
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) GetProductScopesBatch(ctx context.Context, productIDs []int) (map[int]ProductScope, error) {
	if len(productIDs) == 0 {
		return map[int]ProductScope{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, category_id, brand_id FROM products WHERE id = ANY($1) AND deleted_at IS NULL
	`, productIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int]ProductScope, len(productIDs))
	for rows.Next() {
		var id int
		var categoryID, brandID *int
		if err := rows.Scan(&id, &categoryID, &brandID); err != nil {
			return nil, err
		}
		result[id] = ProductScope{CategoryID: categoryID, BrandID: brandID}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type ProductScope struct {
	CategoryID *int
	BrandID    *int
}

func (r *Repository) GetByID(ctx context.Context, id int) (*PricingRule, error) {
	var rule PricingRule
	var createdAt, updatedAt time.Time
	var effectiveFrom, effectiveUntil sql.NullTime
	var timeFrom, timeTo sql.NullString
	var recurrenceDays []string

	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules WHERE id = $1
	`, id).Scan(
		&rule.ID, &rule.ProductID, &rule.CategoryID, &rule.BrandID,
		&rule.PricingType, &rule.PricingMethod, &rule.PricingValue,
		&rule.Name, &rule.MinimumQuantity, &rule.MaximumQuantity,
		&rule.Priority, &rule.CustomerGroupID, &rule.StoreID,
		&recurrenceDays, &timeFrom, &timeTo,
		&rule.AllowCombine, &rule.IsActive, &rule.Status,
		&effectiveFrom, &effectiveUntil, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}

	if effectiveFrom.Valid {
		rule.EffectiveFrom = &effectiveFrom.Time
	}
	if effectiveUntil.Valid {
		rule.EffectiveUntil = &effectiveUntil.Time
	}
	if timeFrom.Valid {
		rule.TimeFrom = &timeFrom.String
	}
	if timeTo.Valid {
		rule.TimeTo = &timeTo.String
	}
	if recurrenceDays != nil {
		rule.RecurrenceDays = recurrenceDays
	}
	rule.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	rule.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &rule, nil
}

func (r *Repository) GetByProductID(ctx context.Context, productID int) ([]PricingRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules WHERE product_id = $1 ORDER BY priority DESC, pricing_value ASC, id ASC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRules(rows)
}

func (r *Repository) GetActiveRules(ctx context.Context, productID int, categoryID, brandID *int, now time.Time, customerGroupID, storeID *int) ([]PricingRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE is_active = true
		  AND status = 'approved'
		  AND (product_id = $1 OR (category_id IS NOT NULL AND category_id = $2) OR (brand_id IS NOT NULL AND brand_id = $3))
		  AND (effective_from IS NULL OR effective_from <= $4)
		  AND (effective_until IS NULL OR effective_until >= $4)
		  AND (customer_group_id IS NULL OR customer_group_id = $5)
		  AND (store_id IS NULL OR store_id = $6)
		ORDER BY priority DESC, pricing_value ASC, id ASC
	`, productID, categoryID, brandID, now, customerGroupID, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRules(rows)
}

func (r *Repository) GetActiveRulesBatch(ctx context.Context, productIDs []int, now time.Time) (map[int][]PricingRule, error) {
	if len(productIDs) == 0 {
		return map[int][]PricingRule{}, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT p.id AS matched_product_id, pr.id, pr.product_id, pr.category_id, pr.brand_id, pr.pricing_type, pr.pricing_method,
		       pr.pricing_value, pr.name, pr.minimum_quantity, pr.maximum_quantity, pr.priority,
		       pr.customer_group_id, pr.store_id, pr.recurrence_days, pr.time_from, pr.time_to,
		       pr.allow_combine, pr.is_active, pr.status, pr.effective_from, pr.effective_until, pr.created_at, pr.updated_at
		FROM pricing_rules pr
		JOIN products p ON (pr.product_id = p.id OR pr.category_id = p.category_id OR pr.brand_id = p.brand_id)
		WHERE p.id = ANY($1)
		  AND p.deleted_at IS NULL
		  AND pr.is_active = true
		  AND pr.status = 'approved'
		  AND (pr.effective_from IS NULL OR pr.effective_from <= $2)
		  AND (pr.effective_until IS NULL OR pr.effective_until >= $2)
		ORDER BY p.id, pr.priority DESC, pr.pricing_value ASC, pr.id ASC
	`, productIDs, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]PricingRule)
	for rows.Next() {
		var matchedProductID int
		var rule PricingRule
		var createdAt, updatedAt time.Time
		var effectiveFrom, effectiveUntil sql.NullTime
		var timeFrom, timeTo sql.NullString
		var recurrenceDays []string

		err := rows.Scan(
			&matchedProductID, &rule.ID, &rule.ProductID, &rule.CategoryID, &rule.BrandID,
			&rule.PricingType, &rule.PricingMethod, &rule.PricingValue,
			&rule.Name, &rule.MinimumQuantity, &rule.MaximumQuantity,
			&rule.Priority, &rule.CustomerGroupID, &rule.StoreID,
			&recurrenceDays, &timeFrom, &timeTo,
			&rule.AllowCombine, &rule.IsActive, &rule.Status,
			&effectiveFrom, &effectiveUntil, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if effectiveFrom.Valid {
			rule.EffectiveFrom = &effectiveFrom.Time
		}
		if effectiveUntil.Valid {
			rule.EffectiveUntil = &effectiveUntil.Time
		}
		if timeFrom.Valid {
			rule.TimeFrom = &timeFrom.String
		}
		if timeTo.Valid {
			rule.TimeTo = &timeTo.String
		}
		if recurrenceDays != nil {
			rule.RecurrenceDays = recurrenceDays
		}
		rule.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		rule.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

		result[matchedProductID] = append(result[matchedProductID], rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, rule *PricingRule) error {
	if rule.Status == "" {
		rule.Status = StatusApproved
	}
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO pricing_rules (product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, created_at, updated_at
	`, rule.ProductID, rule.CategoryID, rule.BrandID, rule.PricingType, rule.PricingMethod,
		rule.PricingValue, rule.Name, rule.MinimumQuantity, rule.MaximumQuantity,
		rule.Priority, rule.CustomerGroupID, rule.StoreID,
		rule.RecurrenceDays, rule.TimeFrom, rule.TimeTo,
		rule.AllowCombine, rule.IsActive, rule.Status, rule.EffectiveFrom, rule.EffectiveUntil,
	).Scan(&rule.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert pricing rule: %w", err)
	}
	rule.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	rule.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, rule *PricingRule) error {
	_, err := r.db.Exec(ctx, `
		UPDATE pricing_rules
		SET product_id = $1, category_id = $2, brand_id = $3, pricing_type = $4, pricing_method = $5,
		    pricing_value = $6, name = $7, minimum_quantity = $8, maximum_quantity = $9,
		    priority = $10, customer_group_id = $11, store_id = $12,
		    recurrence_days = $13, time_from = $14, time_to = $15,
		    allow_combine = $16, is_active = $17, status = $18, effective_from = $19, effective_until = $20,
		    updated_at = NOW()
		WHERE id = $21
	`, rule.ProductID, rule.CategoryID, rule.BrandID, rule.PricingType, rule.PricingMethod,
		rule.PricingValue, rule.Name, rule.MinimumQuantity, rule.MaximumQuantity,
		rule.Priority, rule.CustomerGroupID, rule.StoreID,
		rule.RecurrenceDays, rule.TimeFrom, rule.TimeTo,
		rule.AllowCombine, rule.IsActive, rule.Status, rule.EffectiveFrom, rule.EffectiveUntil,
		rule.ID)
	if err != nil {
		return fmt.Errorf("update pricing rule: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM pricing_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete pricing rule: %w", err)
	}
	return nil
}

func (r *Repository) NameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM pricing_rules WHERE LOWER(TRIM(name)) = LOWER(TRIM($1)) AND id != $2`,
		name, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check name exists: %w", err)
	}
	return count > 0, nil
}

// FindConflicts returns active rules that overlap with the given rule's scope
// (product/category/brand), pricing type, priority, and quantity range.
func (r *Repository) FindConflicts(ctx context.Context, rule *PricingRule, excludeID int) ([]PricingRule, error) {
	query := `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE is_active = true
		  AND id != $1
		  AND pricing_type = $2
		  AND priority = $3
		  AND (
		    product_id = $4
		    OR (category_id IS NOT NULL AND category_id = $5)
		    OR (brand_id IS NOT NULL AND brand_id = $6)
		  )
		  AND (
		    maximum_quantity IS NULL OR minimum_quantity <= $8
		  )
		  AND (
		    $7 <= maximum_quantity OR maximum_quantity IS NULL
		  )
		ORDER BY priority DESC, id ASC
	`

	var productID, categoryID, brandID interface{}
	if rule.ProductID != nil {
		productID = *rule.ProductID
	}
	if rule.CategoryID != nil {
		categoryID = *rule.CategoryID
	}
	if rule.BrandID != nil {
		brandID = *rule.BrandID
	}

	maxQty := rule.MinimumQuantity
	if rule.MaximumQuantity != nil {
		maxQty = *rule.MaximumQuantity
	}

	rows, err := r.db.Query(ctx, query, excludeID, rule.PricingType, rule.Priority,
		productID, categoryID, brandID, rule.MinimumQuantity, maxQty)
	if err != nil {
		return nil, fmt.Errorf("find conflicts: %w", err)
	}
	defer rows.Close()

	return scanRules(rows)
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]PricingRule, int, error) {
	countQuery := `SELECT COUNT(*) FROM pricing_rules WHERE 1=1`
	dataQuery := `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules WHERE 1=1`

	var args []interface{}
	argIdx := 1

	if search != "" {
		searchFilter := fmt.Sprintf(` AND (
			name ILIKE $%d
			OR pricing_type ILIKE $%d
			OR EXISTS (SELECT 1 FROM products p WHERE p.id = pricing_rules.product_id AND p.deleted_at IS NULL AND p.name ILIKE $%d)
			OR EXISTS (SELECT 1 FROM categories c WHERE c.id = pricing_rules.category_id AND c.name ILIKE $%d)
			OR EXISTS (SELECT 1 FROM brands b WHERE b.id = pricing_rules.brand_id AND b.name ILIKE $%d)
		)`, argIdx, argIdx, argIdx, argIdx, argIdx)
		countQuery += searchFilter
		dataQuery += searchFilter
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if productID != nil {
		filter := fmt.Sprintf(" AND product_id = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *productID)
		argIdx++
	}
	if pricingType != "" {
		filter := fmt.Sprintf(" AND pricing_type = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, pricingType)
		argIdx++
	}
	if pricingMethod != "" {
		filter := fmt.Sprintf(" AND pricing_method = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, pricingMethod)
		argIdx++
	}
	if categoryID != nil {
		filter := fmt.Sprintf(" AND category_id = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *categoryID)
		argIdx++
	}
	if brandID != nil {
		filter := fmt.Sprintf(" AND brand_id = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *brandID)
		argIdx++
	}
	if customerGroupID != nil {
		filter := fmt.Sprintf(" AND customer_group_id = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *customerGroupID)
		argIdx++
	}
	if storeID != nil {
		filter := fmt.Sprintf(" AND store_id = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *storeID)
		argIdx++
	}
	if isActive != nil {
		filter := fmt.Sprintf(" AND is_active = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *isActive)
		argIdx++
	}
	if status != "" {
		filter := fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, status)
		argIdx++
	}

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rules, err := scanRules(rows)
	if err != nil {
		return nil, 0, err
	}
	return rules, total, nil
}

func scanRules(rows pgx.Rows) ([]PricingRule, error) {
	var rules []PricingRule
	for rows.Next() {
		var rule PricingRule
		var createdAt, updatedAt time.Time
		var effectiveFrom, effectiveUntil sql.NullTime
		var timeFrom, timeTo sql.NullString
		var recurrenceDays []string

		err := rows.Scan(
			&rule.ID, &rule.ProductID, &rule.CategoryID, &rule.BrandID,
			&rule.PricingType, &rule.PricingMethod, &rule.PricingValue,
			&rule.Name, &rule.MinimumQuantity, &rule.MaximumQuantity,
			&rule.Priority, &rule.CustomerGroupID, &rule.StoreID,
			&recurrenceDays, &timeFrom, &timeTo,
			&rule.AllowCombine, &rule.IsActive, &rule.Status,
			&effectiveFrom, &effectiveUntil, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if effectiveFrom.Valid {
			rule.EffectiveFrom = &effectiveFrom.Time
		}
		if effectiveUntil.Valid {
			rule.EffectiveUntil = &effectiveUntil.Time
		}
		if timeFrom.Valid {
			rule.TimeFrom = &timeFrom.String
		}
		if timeTo.Valid {
			rule.TimeTo = &timeTo.String
		}
		if recurrenceDays != nil {
			rule.RecurrenceDays = recurrenceDays
		}
		rule.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		rule.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

type PricingRuleImportRow struct {
	Row              int
	ProductID        *int
	CategoryID       *int
	BrandID          *int
	PricingType      string
	PricingMethod    string
	PricingValue     float64
	Name             string
	MinimumQuantity  int
	MaximumQuantity  *int
	Priority         int
	IsActive         bool
	EffectiveFrom    *time.Time
	EffectiveUntil   *time.Time
	CustomerGroupID  *int
	StoreID          *int
	RecurrenceDays   []string
	TimeFrom         *string
	TimeTo           *string
	AllowCombine     bool
}

type PricingRuleImportPayload struct {
	ProductID        *int
	CategoryID       *int
	BrandID          *int
	PricingType      string
	PricingMethod    string
	PricingValue     float64
	Name             string
	MinimumQuantity  int
	MaximumQuantity  *int
	Priority         int
	IsActive         bool
	EffectiveFrom    *time.Time
	EffectiveUntil   *time.Time
	CustomerGroupID  *int
	StoreID          *int
	RecurrenceDays   []string
	TimeFrom         *string
	TimeTo           *string
	AllowCombine     bool
}

func (r *Repository) BulkInsertPricingRules(ctx context.Context, payloads []PricingRuleImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	count := 0
	for _, p := range payloads {
		_, err := tx.Exec(ctx, `
			INSERT INTO pricing_rules (product_id, category_id, brand_id, pricing_type, pricing_method,
			       pricing_value, name, minimum_quantity, maximum_quantity, priority,
			       customer_group_id, store_id, recurrence_days, time_from, time_to,
			       allow_combine, is_active, effective_from, effective_until)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		`, p.ProductID, p.CategoryID, p.BrandID, p.PricingType, p.PricingMethod,
			p.PricingValue, p.Name, p.MinimumQuantity, p.MaximumQuantity,
			p.Priority, p.CustomerGroupID, p.StoreID,
			p.RecurrenceDays, p.TimeFrom, p.TimeTo,
			p.AllowCombine, p.IsActive, p.EffectiveFrom, p.EffectiveUntil)
		if err != nil {
			return count, fmt.Errorf("insert pricing rule: %w", err)
		}
		count++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

func (r *Repository) BulkUpdatePricingRules(ctx context.Context, payloads []PricingRuleImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	count := 0
	for _, p := range payloads {
		tag, err := tx.Exec(ctx, `
			UPDATE pricing_rules
			SET pricing_method = $1, pricing_value = $2, minimum_quantity = $3, maximum_quantity = $4,
			    priority = $5, is_active = $6, effective_from = $7, effective_until = $8,
			    updated_at = NOW()
			WHERE product_id = $9 AND pricing_type = $10 AND name = $11
		`, p.PricingMethod, p.PricingValue, p.MinimumQuantity, p.MaximumQuantity,
			p.Priority, p.IsActive, p.EffectiveFrom, p.EffectiveUntil,
			p.ProductID, p.PricingType, p.Name)
		if err != nil {
			return count, fmt.Errorf("update pricing rule: %w", err)
		}
		if tag.RowsAffected() > 0 {
			count++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return count, nil
}

func (r *Repository) GetAllForExport(ctx context.Context) ([]PricingRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

// SearchProducts searches products by name, SKU, or barcode for autocomplete.
func (r *Repository) SearchProducts(ctx context.Context, query string, limit int) ([]ProductSearchResult, error) {
	if query == "" {
		return []ProductSearchResult{}, nil
	}
	like := "%" + strings.ToLower(query) + "%"
	rows, err := r.db.Query(ctx, `
		SELECT id, name, sku, price FROM products
		WHERE deleted_at IS NULL AND status = 'active'
		  AND (LOWER(name) LIKE $1 OR LOWER(sku) LIKE $1 OR LOWER(barcode) LIKE $1)
		ORDER BY name ASC
		LIMIT $2
	`, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProductSearchResult
	for rows.Next() {
		var p ProductSearchResult
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Price); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

type ProductSearchResult struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	SKU   string `json:"sku"`
	Price int    `json:"price"`
}

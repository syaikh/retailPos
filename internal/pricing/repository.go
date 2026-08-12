package pricing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db                     shared.DBPool
	productPricingProvider ProductPricingProvider
	categorySearchProvider CategoryNameSearchProvider
	brandSearchProvider    BrandNameSearchProvider
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// SetProductPricingProvider wires the product-owned implementation of the
// ProductPricingProvider port (see ports.go). products/tax_classes are owned by
// internal/product (ADR §2.8 Katalog); pricing routes base-price, scope,
// cost/tax, and autocomplete reads through this port instead of querying those
// tables directly.
func (r *Repository) SetProductPricingProvider(p ProductPricingProvider) {
	r.productPricingProvider = p
}

// SetCategorySearchProvider wires the category-owned implementation of the
// CategoryNameSearchProvider port (see ports.go). categories is owned by
// internal/category; pricing routes the rule-listing category-name search
// through this port instead of a categories EXISTS clause.
func (r *Repository) SetCategorySearchProvider(p CategoryNameSearchProvider) {
	r.categorySearchProvider = p
}

// SetBrandSearchProvider wires the brand-owned implementation of the
// BrandNameSearchProvider port (see ports.go). brands is owned by
// internal/brand; pricing routes the rule-listing brand-name search through
// this port instead of a brands EXISTS clause.
func (r *Repository) SetBrandSearchProvider(p BrandNameSearchProvider) {
	r.brandSearchProvider = p
}

func (r *Repository) basePricesByIDs(ctx context.Context, ids []int) (map[int]int, error) {
	if r.productPricingProvider == nil {
		return nil, fmt.Errorf("pricing repository: product pricing provider not wired; call SetProductPricingProvider")
	}
	return r.productPricingProvider.BasePricesByIDs(ctx, r.db, ids)
}

func (r *Repository) productScopesByIDs(ctx context.Context, ids []int) (map[int]ProductScope, error) {
	if r.productPricingProvider == nil {
		return nil, fmt.Errorf("pricing repository: product pricing provider not wired; call SetProductPricingProvider")
	}
	return r.productPricingProvider.ProductScopesByIDs(ctx, r.db, ids)
}

func (r *Repository) productCostTaxesByIDs(ctx context.Context, ids []int) (map[int]ProductCostTax, error) {
	if r.productPricingProvider == nil {
		return nil, fmt.Errorf("pricing repository: product pricing provider not wired; call SetProductPricingProvider")
	}
	return r.productPricingProvider.ProductCostTaxesByIDs(ctx, r.db, ids)
}

func (r *Repository) productIDsByName(ctx context.Context, search string) ([]int, error) {
	if r.productPricingProvider == nil {
		return nil, fmt.Errorf("pricing repository: product pricing provider not wired; call SetProductPricingProvider")
	}
	return r.productPricingProvider.ProductIDsByName(ctx, r.db, search)
}

func (r *Repository) categoryIDsByName(ctx context.Context, search string) ([]int, error) {
	if r.categorySearchProvider == nil {
		return nil, fmt.Errorf("pricing repository: category search provider not wired; call SetCategorySearchProvider")
	}
	return r.categorySearchProvider.CategoryIDsByName(ctx, r.db, search)
}

func (r *Repository) brandIDsByName(ctx context.Context, search string) ([]int, error) {
	if r.brandSearchProvider == nil {
		return nil, fmt.Errorf("pricing repository: brand search provider not wired; call SetBrandSearchProvider")
	}
	return r.brandSearchProvider.BrandIDsByName(ctx, r.db, search)
}

func (r *Repository) searchPricingProducts(ctx context.Context, query string, limit int) ([]ProductSearchResult, error) {
	if r.productPricingProvider == nil {
		return nil, fmt.Errorf("pricing repository: product pricing provider not wired; call SetProductPricingProvider")
	}
	return r.productPricingProvider.SearchPricingProducts(ctx, r.db, query, limit)
}

func (r *Repository) GetBasePrice(ctx context.Context, productID int) (int, error) {
	prices, err := r.basePricesByIDs(ctx, []int{productID})
	if err != nil {
		return 0, err
	}
	price, ok := prices[productID]
	if !ok {
		return 0, ErrProductNotFound
	}
	return price, nil
}

func (r *Repository) GetProductScope(ctx context.Context, productID int) (categoryID *int, brandID *int, err error) {
	scopes, err := r.productScopesByIDs(ctx, []int{productID})
	if err != nil {
		return nil, nil, err
	}
	scope, ok := scopes[productID]
	if !ok {
		return nil, nil, ErrProductNotFound
	}
	return scope.CategoryID, scope.BrandID, nil
}

func (r *Repository) GetProductCostAndTax(ctx context.Context, productID int) (ProductCostTax, error) {
	costs, err := r.productCostTaxesByIDs(ctx, []int{productID})
	if err != nil {
		return ProductCostTax{}, err
	}
	ct, ok := costs[productID]
	if !ok {
		return ProductCostTax{}, ErrProductNotFound
	}
	return ct, nil
}

func (r *Repository) GetProductCostAndTaxBatch(ctx context.Context, productIDs []int) (map[int]ProductCostTax, error) {
	return r.productCostTaxesByIDs(ctx, productIDs)
}

func (r *Repository) GetBasePricesBatch(ctx context.Context, productIDs []int) (map[int]int, error) {
	return r.basePricesByIDs(ctx, productIDs)
}

func (r *Repository) GetProductScopesBatch(ctx context.Context, productIDs []int) (map[int]ProductScope, error) {
	return r.productScopesByIDs(ctx, productIDs)
}

// ProductScope is the category/brand membership of a product. It is an alias
// of shared.ProductScope, the cross-module contract produced by the
// product-owned ProductPricingProvider port.
type ProductScope = shared.ProductScope

func (r *Repository) GetByID(ctx context.Context, id int) (*Rule, error) {
	var rule Rule
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
		&rule.Type, &rule.Method, &rule.PricingValue,
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

func (r *Repository) GetByProductID(ctx context.Context, productID int) ([]Rule, error) {
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

func (r *Repository) GetActiveRules(ctx context.Context, productID int, categoryID, brandID *int, now time.Time, customerGroupID, storeID *int) ([]Rule, error) {
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

// GetActiveRulesBatch returns the active approved rules that match each
// product. Product identity (id, category, brand, non-deleted) is resolved via
// the product-owned ProductPricingProvider; rules are then matched in Go
// because pricing_rules scope columns reference products it no longer queries
// directly (ADR §2.8 Katalog). Per-product ordering matches the previous
// SQL ordering (priority DESC, pricing_value ASC, id ASC).
func (r *Repository) GetActiveRulesBatch(ctx context.Context, productIDs []int, now time.Time) (map[int][]Rule, error) {
	if len(productIDs) == 0 {
		return map[int][]Rule{}, nil
	}

	scopes, err := r.productScopesByIDs(ctx, productIDs)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return map[int][]Rule{}, nil
	}

	activeProductIDs := make([]int, 0, len(scopes))
	catIDSet := make(map[int]struct{})
	brandIDSet := make(map[int]struct{})
	for id, scope := range scopes {
		activeProductIDs = append(activeProductIDs, id)
		if scope.CategoryID != nil {
			catIDSet[*scope.CategoryID] = struct{}{}
		}
		if scope.BrandID != nil {
			brandIDSet[*scope.BrandID] = struct{}{}
		}
	}
	catIDs := make([]int, 0, len(catIDSet))
	for id := range catIDSet {
		catIDs = append(catIDs, id)
	}
	brandIDs := make([]int, 0, len(brandIDSet))
	for id := range brandIDSet {
		brandIDs = append(brandIDs, id)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE is_active = true
		  AND status = 'approved'
		  AND (effective_from IS NULL OR effective_from <= $1)
		  AND (effective_until IS NULL OR effective_until >= $1)
		  AND (
		    (product_id IS NOT NULL AND product_id = ANY($2))
		    OR (category_id IS NOT NULL AND category_id = ANY($3))
		    OR (brand_id IS NOT NULL AND brand_id = ANY($4))
		  )
		ORDER BY priority DESC, pricing_value ASC, id ASC
	`, now, activeProductIDs, catIDs, brandIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules, err := scanRules(rows)
	if err != nil {
		return nil, err
	}

	result := make(map[int][]Rule)
	for productID, scope := range scopes {
		for _, rule := range rules {
			if ruleMatchesProduct(rule, productID, scope) {
				result[productID] = append(result[productID], rule)
			}
		}
	}
	return result, nil
}

// ruleMatchesProduct reports whether a rule applies to a product given its
// category/brand membership (product-scope wins; category/brand only when set).
func ruleMatchesProduct(rule Rule, productID int, scope ProductScope) bool {
	if rule.ProductID != nil && *rule.ProductID == productID {
		return true
	}
	if rule.CategoryID != nil && scope.CategoryID != nil && *rule.CategoryID == *scope.CategoryID {
		return true
	}
	if rule.BrandID != nil && scope.BrandID != nil && *rule.BrandID == *scope.BrandID {
		return true
	}
	return false
}

func (r *Repository) Create(ctx context.Context, rule *Rule) error {
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
	`, rule.ProductID, rule.CategoryID, rule.BrandID, rule.Type, rule.Method,
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

func (r *Repository) Update(ctx context.Context, rule *Rule) error {
	_, err := r.db.Exec(ctx, `
		UPDATE pricing_rules
		SET product_id = $1, category_id = $2, brand_id = $3, pricing_type = $4, pricing_method = $5,
		    pricing_value = $6, name = $7, minimum_quantity = $8, maximum_quantity = $9,
		    priority = $10, customer_group_id = $11, store_id = $12,
		    recurrence_days = $13, time_from = $14, time_to = $15,
		    allow_combine = $16, is_active = $17, status = $18, effective_from = $19, effective_until = $20,
		    updated_at = NOW()
		WHERE id = $21
	`, rule.ProductID, rule.CategoryID, rule.BrandID, rule.Type, rule.Method,
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
func (r *Repository) FindConflicts(ctx context.Context, rule *Rule, excludeID int) ([]Rule, error) {
	query := `
		SELECT id, product_id, category_id, brand_id, pricing_type, pricing_method,
		       pricing_value, name, minimum_quantity, maximum_quantity, priority,
		       customer_group_id, store_id, recurrence_days, time_from, time_to,
		       allow_combine, is_active, status, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE is_active = true
		  AND status = 'approved'
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

	rows, err := r.db.Query(ctx, query, excludeID, rule.Type, rule.Priority,
		productID, categoryID, brandID, rule.MinimumQuantity, maxQty)
	if err != nil {
		return nil, fmt.Errorf("find conflicts: %w", err)
	}
	defer rows.Close()

	return scanRules(rows)
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType, pricingMethod string, categoryID, brandID, customerGroupID, storeID *int, isActive *bool, status string) ([]Rule, int, error) {
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
		pattern := "%" + search + "%"
		productIDs, err := r.productIDsByName(ctx, pattern)
		if err != nil {
			return nil, 0, err
		}
		categoryIDs, err := r.categoryIDsByName(ctx, pattern)
		if err != nil {
			return nil, 0, err
		}
		brandIDs, err := r.brandIDsByName(ctx, pattern)
		if err != nil {
			return nil, 0, err
		}
		if productIDs == nil {
			productIDs = []int{}
		}
		if categoryIDs == nil {
			categoryIDs = []int{}
		}
		if brandIDs == nil {
			brandIDs = []int{}
		}

		searchFilter := fmt.Sprintf(` AND (
			name ILIKE $%d
			OR pricing_type ILIKE $%d
			OR (product_id IS NOT NULL AND product_id = ANY($%d))
			OR (category_id IS NOT NULL AND category_id = ANY($%d))
			OR (brand_id IS NOT NULL AND brand_id = ANY($%d))
		)`, argIdx, argIdx, argIdx+1, argIdx+2, argIdx+3)
		countQuery += searchFilter
		dataQuery += searchFilter
		args = append(args, pattern, productIDs, categoryIDs, brandIDs)
		argIdx += 4
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

func scanRules(rows pgx.Rows) ([]Rule, error) {
	var rules []Rule
	for rows.Next() {
		var rule Rule
		var createdAt, updatedAt time.Time
		var effectiveFrom, effectiveUntil sql.NullTime
		var timeFrom, timeTo sql.NullString
		var recurrenceDays []string

		err := rows.Scan(
			&rule.ID, &rule.ProductID, &rule.CategoryID, &rule.BrandID,
			&rule.Type, &rule.Method, &rule.PricingValue,
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

type RuleImportRow struct {
	Row             int
	ProductID       *int
	CategoryID      *int
	BrandID         *int
	Type            string
	Method          string
	PricingValue    float64
	Name            string
	MinimumQuantity int
	MaximumQuantity *int
	Priority        int
	IsActive        bool
	EffectiveFrom   *time.Time
	EffectiveUntil  *time.Time
	CustomerGroupID *int
	StoreID         *int
	RecurrenceDays  []string
	TimeFrom        *string
	TimeTo          *string
	AllowCombine    bool
}

type RuleImportPayload struct {
	ProductID       *int
	CategoryID      *int
	BrandID         *int
	Type            string
	Method          string
	PricingValue    float64
	Name            string
	MinimumQuantity int
	MaximumQuantity *int
	Priority        int
	IsActive        bool
	EffectiveFrom   *time.Time
	EffectiveUntil  *time.Time
	CustomerGroupID *int
	StoreID         *int
	RecurrenceDays  []string
	TimeFrom        *string
	TimeTo          *string
	AllowCombine    bool
}

func (r *Repository) BulkInsertPricingRules(ctx context.Context, payloads []RuleImportPayload) (int, error) {
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
			       allow_combine, is_active, status, effective_from, effective_until)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		`, p.ProductID, p.CategoryID, p.BrandID, p.Type, p.Method,
			p.PricingValue, p.Name, p.MinimumQuantity, p.MaximumQuantity,
			p.Priority, p.CustomerGroupID, p.StoreID,
			p.RecurrenceDays, p.TimeFrom, p.TimeTo,
			p.AllowCombine, p.IsActive, StatusApproved, p.EffectiveFrom, p.EffectiveUntil)
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

func (r *Repository) BulkUpdatePricingRules(ctx context.Context, payloads []RuleImportPayload) (int, error) {
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
			SET category_id = $1, brand_id = $2, pricing_method = $3, pricing_value = $4, minimum_quantity = $5, maximum_quantity = $6,
			    priority = $7, is_active = $8, effective_from = $9, effective_until = $10,
			    customer_group_id = $11, store_id = $12, recurrence_days = $13, time_from = $14, time_to = $15,
			    allow_combine = $16, status = $17, updated_at = NOW()
			WHERE product_id = $18 AND pricing_type = $19 AND name = $20
		`, p.CategoryID, p.BrandID, p.Method, p.PricingValue, p.MinimumQuantity, p.MaximumQuantity,
			p.Priority, p.IsActive, p.EffectiveFrom, p.EffectiveUntil,
			p.CustomerGroupID, p.StoreID, p.RecurrenceDays, p.TimeFrom, p.TimeTo,
			p.AllowCombine, StatusApproved,
			p.ProductID, p.Type, p.Name)
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

func (r *Repository) GetAllForExport(ctx context.Context) ([]Rule, error) {
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
// Routed through the product-owned ProductPricingProvider port.
func (r *Repository) SearchProducts(ctx context.Context, query string, limit int) ([]ProductSearchResult, error) {
	return r.searchPricingProducts(ctx, query, limit)
}

// ProductSearchResult is an autocomplete hit for product search. It is an
// alias of shared.ProductSearchResult, the cross-module contract produced by
// the product-owned ProductPricingProvider port.
type ProductSearchResult = shared.ProductSearchResult

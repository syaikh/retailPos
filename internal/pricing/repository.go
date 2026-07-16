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

func (r *Repository) GetByID(ctx context.Context, id int) (*PricingRule, error) {
	var rule PricingRule
	var createdAt, updatedAt time.Time
	var effectiveFrom, effectiveUntil sql.NullTime

	err := r.db.QueryRow(ctx, `
		SELECT id, product_id, pricing_type, name, price, minimum_quantity,
		       priority, is_active, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules WHERE id = $1
	`, id).Scan(
		&rule.ID, &rule.ProductID, &rule.PricingType, &rule.Name,
		&rule.Price, &rule.MinimumQuantity, &rule.Priority, &rule.IsActive,
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
	rule.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	rule.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &rule, nil
}

func (r *Repository) GetByProductID(ctx context.Context, productID int) ([]PricingRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, pricing_type, name, price, minimum_quantity,
		       priority, is_active, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules WHERE product_id = $1 ORDER BY priority DESC, price ASC, id ASC
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRules(rows)
}

func (r *Repository) GetActiveRules(ctx context.Context, productID int, now time.Time) ([]PricingRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, pricing_type, name, price, minimum_quantity,
		       priority, is_active, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE product_id = $1
		  AND is_active = true
		  AND (effective_from IS NULL OR effective_from <= $2)
		  AND (effective_until IS NULL OR effective_until >= $2)
		ORDER BY priority DESC, price ASC, id ASC
	`, productID, now)
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
		SELECT id, product_id, pricing_type, name, price, minimum_quantity,
		       priority, is_active, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules
		WHERE product_id = ANY($1)
		  AND is_active = true
		  AND (effective_from IS NULL OR effective_from <= $2)
		  AND (effective_until IS NULL OR effective_until >= $2)
		ORDER BY product_id, priority DESC, price ASC, id ASC
	`, productIDs, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]PricingRule)
	for rows.Next() {
		var rule PricingRule
		var createdAt, updatedAt time.Time
		var effectiveFrom, effectiveUntil sql.NullTime

		err := rows.Scan(
			&rule.ID, &rule.ProductID, &rule.PricingType, &rule.Name,
			&rule.Price, &rule.MinimumQuantity, &rule.Priority, &rule.IsActive,
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
		rule.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		rule.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)

		result[rule.ProductID] = append(result[rule.ProductID], rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, rule *PricingRule) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO pricing_rules (product_id, pricing_type, name, price, minimum_quantity, priority, is_active, effective_from, effective_until)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, rule.ProductID, rule.PricingType, rule.Name, rule.Price,
		rule.MinimumQuantity, rule.Priority, rule.IsActive,
		rule.EffectiveFrom, rule.EffectiveUntil,
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
		SET pricing_type = $1, name = $2, price = $3, minimum_quantity = $4,
		    priority = $5, is_active = $6, effective_from = $7, effective_until = $8,
		    updated_at = NOW()
		WHERE id = $9
	`, rule.PricingType, rule.Name, rule.Price, rule.MinimumQuantity,
		rule.Priority, rule.IsActive, rule.EffectiveFrom, rule.EffectiveUntil,
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

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, productID *int, pricingType string, isActive *bool) ([]PricingRule, int, error) {
	countQuery := `SELECT COUNT(*) FROM pricing_rules WHERE 1=1`
	dataQuery := `
		SELECT id, product_id, pricing_type, name, price, minimum_quantity,
		       priority, is_active, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules WHERE 1=1`

	var args []interface{}
	argIdx := 1

	if search != "" {
		filter := fmt.Sprintf(" AND (name ILIKE $%d OR pricing_type ILIKE $%d)", argIdx, argIdx+1)
		countQuery += filter
		dataQuery += filter
		args = append(args, "%"+search+"%", "%"+search+"%")
		argIdx += 2
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
	if isActive != nil {
		filter := fmt.Sprintf(" AND is_active = $%d", argIdx)
		countQuery += filter
		dataQuery += filter
		args = append(args, *isActive)
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

		err := rows.Scan(
			&rule.ID, &rule.ProductID, &rule.PricingType, &rule.Name,
			&rule.Price, &rule.MinimumQuantity, &rule.Priority, &rule.IsActive,
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
	Row             int
	ProductID       int
	PricingType     string
	Name            string
	Price           int
	MinimumQuantity int
	Priority        int
	IsActive        bool
	EffectiveFrom   *time.Time
	EffectiveUntil  *time.Time
}

type PricingRuleImportPayload struct {
	ProductID       int
	PricingType     string
	Name            string
	Price           int
	MinimumQuantity int
	Priority        int
	IsActive        bool
	EffectiveFrom   *time.Time
	EffectiveUntil  *time.Time
}

func (r *Repository) BulkInsertPricingRules(ctx context.Context, payloads []PricingRuleImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	count := 0
	for _, p := range payloads {
		_, err := tx.Exec(ctx, `
			INSERT INTO pricing_rules (product_id, pricing_type, name, price, minimum_quantity, priority, is_active, effective_from, effective_until)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, p.ProductID, p.PricingType, p.Name, p.Price, p.MinimumQuantity, p.Priority, p.IsActive, p.EffectiveFrom, p.EffectiveUntil)
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
	defer tx.Rollback(ctx)

	count := 0
	for _, p := range payloads {
		tag, err := tx.Exec(ctx, `
			UPDATE pricing_rules
			SET price = $1, minimum_quantity = $2, priority = $3, is_active = $4,
			    effective_from = $5, effective_until = $6, updated_at = NOW()
			WHERE product_id = $7 AND pricing_type = $8 AND name = $9
		`, p.Price, p.MinimumQuantity, p.Priority, p.IsActive, p.EffectiveFrom, p.EffectiveUntil,
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
		SELECT id, product_id, pricing_type, name, price, minimum_quantity,
		       priority, is_active, effective_from, effective_until, created_at, updated_at
		FROM pricing_rules ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRules(rows)
}

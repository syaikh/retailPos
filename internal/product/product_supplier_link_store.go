package product

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

// SupplierLinkStore is the product-owned implementation of the supplier
// module's consumer-side port (supplier.ProductSupplierStore, structural
// typing — no import of internal/supplier needed). internal/product is the
// canonical owner of the product_suppliers link table (ADR
// Modular_Monolith_Module_Boundaries §2.8 Katalog), so every read and write
// that internal/supplier performs on that table is computed here rather than
// via direct SQL inside internal/supplier.
//
// # Store scoping
//
// Migration 058_supplier_terms_store_scope.sql (audit D3, Option C) moves the
// store boundary from suppliers (a trading partner's identity, correctly
// global) down to product_suppliers (the commercial terms, correctly per
// store). A link row's store_id names the store whose terms it carries, and
// NULL means *global* terms that apply everywhere — the value every row already
// had, so existing data keeps working as a shared default.
//
// Every method takes storeID, the caller's resolved scope, following the
// pricing and shift convention: nil means unrestricted (superadmin), non-nil
// means one store. Handlers reject a foreign store and fail closed on a missing
// claim before calling in here, so a non-nil storeID always means "this store".
//
// Reads and writes are deliberately asymmetric, matching pricing:
//
//   - Reads see `store_id IS NULL OR store_id = $n` — a store inherits the
//     global default. List endpoints then collapse the pair so a store that has
//     overridden a supplier sees only its own row, not the shadowed global one.
//   - Writes match the store column *exactly*, so a store-scoped caller can
//     never mutate or delete the global default. That row belongs to the whole
//     estate; changing one store's negotiated cost must not reprice the others.
//     A store expresses an override by creating its own link row.
//
// Two helpers below build the two predicates. They are helpers rather than
// inline fragments because every method needs one and a method that quietly
// picked the wrong one is either a cross-store cost leak or a silently ignored
// write.
type SupplierLinkStore struct{}

// Both helpers keep the store placeholder in the statement whatever the scope,
// so the caller can always pass the argument and the two can never drift apart.
//
// That constancy is not cosmetic. A helper that dropped the placeholder for the
// nil case would leave every statement taking one fewer argument than it passed,
// and pgx rejects that with "mismatched param and argument count" -- which reads
// like a driver bug rather than the store predicate it actually is. Letting the
// SQL absorb the nil instead keeps one code path per query:
//
//	$1 = NULL  ->  TRUE                          (unrestricted read)
//	$1 = 5     ->  (false OR ... OR store_id = 5) (inherits the global default)
//	store_id IS NOT DISTINCT FROM NULL  ->  the global row (unscoped write)
//	store_id IS NOT DISTINCT FROM 5     ->  this store's own row
//
// argOffset is the $n the store argument occupies.
func visibleSQL(column string, argOffset int) string {
	return fmt.Sprintf("($%d::int IS NULL OR %s IS NULL OR %s = $%d::int)",
		argOffset, column, column, argOffset)
}

// ownSQL renders the write predicate: only the caller's own store. A nil scope
// (superadmin) targets the global row, which is what an unscoped management edit
// means. NOT DISTINCT FROM is what makes that nil match global rows rather than
// nothing at all.
func ownSQL(column string, argOffset int) string {
	// The cast is required, not decorative: unlike `col = $n`, `IS NOT DISTINCT
	// FROM` gives PostgreSQL no type context for an untyped NULL parameter, so
	// an unscoped write would fail with "could not determine data type of
	// parameter $n" instead of quietly matching the global row.
	return fmt.Sprintf("%s IS NOT DISTINCT FROM $%d::int", column, argOffset)
}

// linkColumns is the shared projection, kept in one place because a read that
// scanned a different column order from its SELECT would fill StoreID with
// created_at and produce a plausible-looking wrong answer.
const linkColumns = `id, product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, store_id, created_at`

// CreateLink inserts a new product-supplier link row. ps.StoreID is taken from
// the caller, which has already pinned it: a store-scoped caller writes its own
// store, a superadmin writes NULL for global terms or an explicit store.
func (SupplierLinkStore) CreateLink(ctx context.Context, db shared.DBPool, ps *shared.ProductSupplier) error {
	var createdAt time.Time
	err := db.QueryRow(ctx, `
		INSERT INTO product_suppliers (product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, store_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, ps.ProductID, ps.SupplierID, ps.SupplierSKU, ps.UnitCost, ps.LeadTimeDays, ps.IsPreferred, ps.StoreID,
	).Scan(&ps.ID, &createdAt)
	if err != nil {
		return fmt.Errorf("link product supplier: %w", err)
	}
	ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

// DeleteLink removes the caller's own product-supplier link row, or
// shared.ErrProductSupplierNotFound when the caller has none.
func (SupplierLinkStore) DeleteLink(ctx context.Context, db shared.DBPool, productID, supplierID int, storeID *int) error {
	tag, err := db.Exec(ctx, `
		DELETE FROM product_suppliers
		WHERE product_id = $1 AND supplier_id = $2 AND `+ownSQL("store_id", 3),
		productID, supplierID, storeID)
	if err != nil {
		return fmt.Errorf("unlink product supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// The link the caller sees is inherited from the global default and it
		// does not own it. Reporting that is better than a 200 that changed
		// nothing, and better than deleting a row priced for every store.
		return fmt.Errorf("unlink product supplier: product %d has no supplier link owned by this store: %w",
			productID, shared.ErrProductSupplierNotFound)
	}
	return nil
}

// GetLink returns a single link row visible in the caller's scope, or
// shared.ErrProductSupplierNotFound.
//
// This is a visibility read, so a store-scoped caller asking about a pair it has
// not overridden gets the inherited global row. Handlers use it to display a
// link before deciding whether the caller may edit it, so it deliberately does
// not enforce ownership.
//
// The ORDER BY is load-bearing, not cosmetic. Once a store may override a pair,
// the predicate matches two rows, and QueryRow without an order returns whichever
// the planner liked — so the same request could report the store's negotiated cost
// or the global default at random. Own row first for a store, global default first
// when there is no store to speak for, matching GetPreferredLink.
func (SupplierLinkStore) GetLink(ctx context.Context, db shared.DBPool, productID, supplierID int, storeID *int) (*shared.ProductSupplier, error) {
	var ps shared.ProductSupplier
	var createdAt time.Time

	order := "(store_id IS NULL) ASC, id ASC" // own store first, inherited default last
	if storeID == nil {
		order = "(store_id IS NOT NULL) ASC, id ASC" // global default first
	}

	err := db.QueryRow(ctx, `
		SELECT `+linkColumns+`
		FROM product_suppliers
		WHERE product_id = $1 AND supplier_id = $2 AND `+visibleSQL("store_id", 3)+`
		ORDER BY `+order+`
		LIMIT 1
	`, productID, supplierID, storeID).Scan(
		&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
		&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &ps.StoreID, &createdAt,
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

// GetPreferredLink returns the preferred link row in effect for the caller's
// scope, or shared.ErrProductSupplierNotFound when none is preferred.
//
// is_preferred is store-relative, so a product can have a global preferred
// supplier *and* one per store. The ORDER BY is what makes the answer
// deterministic: a store's own choice wins, and the global default is the
// fallback it has not overridden. Without it the predicate matches two rows and
// QueryRow returns whichever the planner liked first, so the same request could
// report two different preferred suppliers.
func (SupplierLinkStore) GetPreferredLink(ctx context.Context, db shared.DBPool, productID int, storeID *int) (*shared.ProductSupplier, error) {
	var ps shared.ProductSupplier
	var createdAt time.Time

	scopeArg := 2
	// Both branches have to be ordered, because an unscoped read matches every
	// store's row and a scoped one matches the shadowed global row too. Without
	// ORDER BY the predicate matches two rows and QueryRow returns whichever the
	// planner liked, so the same request could report two different preferred
	// suppliers.
	order := "(store_id IS NULL) ASC, id ASC" // own store first, inherited default last
	if storeID == nil {
		// Unscoped there is no store to prefer, so the global default leads: it
		// is the estate-wide answer, and a store row is only a fallback. Still
		// ordered, so two store rows never swap places between calls.
		order = "(store_id IS NOT NULL) ASC, id ASC"
	}

	err := db.QueryRow(ctx, `
		SELECT `+linkColumns+`
		FROM product_suppliers
		WHERE product_id = $1 AND is_preferred = true AND `+visibleSQL("store_id", scopeArg)+`
		ORDER BY `+order+`
		LIMIT 1
	`, productID, storeID).Scan(
		&ps.ID, &ps.ProductID, &ps.SupplierID, &ps.SupplierSKU,
		&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &ps.StoreID, &createdAt,
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

// SetPreferredLink makes the given link the product's single preferred one for
// the caller's own store, clearing that store's previous choice first.
//
// Both statements match the store column exactly. Clearing across scopes would
// demote the global default whenever any store changed its mind, and under the
// new partial index each store's choice is a separate slot anyway.
func (SupplierLinkStore) SetPreferredLink(ctx context.Context, db shared.DBPool, productID, supplierID int, storeID *int) error {
	// Two statements, so two placeholder layouts: placeholders are positional, so
	// the clear cannot borrow $3 from the statement that follows it without
	// leaving a gap PostgreSQL cannot type ("could not determine data type of
	// parameter $2").
	if _, err := db.Exec(ctx, `
		UPDATE product_suppliers SET is_preferred = false
		WHERE product_id = $1 AND is_preferred = true AND `+ownSQL("store_id", 2),
		productID, storeID); err != nil {
		return fmt.Errorf("clear preferred supplier: %w", err)
	}

	tag, err := db.Exec(ctx, `
		UPDATE product_suppliers SET is_preferred = true
		WHERE product_id = $1 AND supplier_id = $2 AND `+ownSQL("store_id", 3),
		productID, supplierID, storeID)
	if err != nil {
		return fmt.Errorf("set preferred supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// The caller named a link it does not own. Succeeding silently would
		// leave it with no preferred supplier and a 200.
		return fmt.Errorf("set preferred supplier: product %d has no supplier link owned by this store: %w",
			productID, shared.ErrProductSupplierNotFound)
	}
	return nil
}

// UpdateLink updates the per-supplier metadata of the caller's own link row.
func (SupplierLinkStore) UpdateLink(ctx context.Context, db shared.DBPool, ps *shared.ProductSupplier) error {
	tag, err := db.Exec(ctx, `
		UPDATE product_suppliers
		SET supplier_sku = $1, unit_cost = $2, lead_time_days = $3, is_preferred = $4, updated_at = NOW()
		WHERE product_id = $5 AND supplier_id = $6 AND `+ownSQL("store_id", 7),
		ps.SupplierSKU, ps.UnitCost, ps.LeadTimeDays, ps.IsPreferred, ps.ProductID, ps.SupplierID, ps.StoreID)
	if err != nil {
		return fmt.Errorf("update product supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update product supplier: %w", shared.ErrProductSupplierNotFound)
	}
	return nil
}

// ListLinksByProduct returns the link rows of a product in effect for the
// caller's scope. A store-scoped caller sees one row per supplier — its own
// where it has one, the inherited global default otherwise — so overriding a
// supplier does not leave the shadowed default sitting next to it in the UI.
// Supplier enrichment (name/code) is the consumer's responsibility on its own
// suppliers table.
func (SupplierLinkStore) ListLinksByProduct(ctx context.Context, db shared.DBPool, productID int, storeID *int) ([]shared.ProductSupplier, error) {
	query := `
		SELECT ` + linkColumns + `
		FROM product_suppliers
		WHERE product_id = $1 AND ` + visibleSQL("store_id", 2) + `
		ORDER BY is_preferred DESC, supplier_id ASC, store_id ASC NULLS FIRST
	`
	if storeID != nil {
		// Collapse each (product, supplier) pair to the row actually in effect:
		// this store's own, else the global default. DISTINCT ON forces the
		// ORDER BY to lead with the DISTINCT columns, so the preferred-first
		// display order is restored in Go afterwards rather than fought for here.
		query = `
			SELECT DISTINCT ON (product_id, supplier_id) ` + linkColumns + `
			FROM product_suppliers
			WHERE product_id = $1 AND ` + visibleSQL("store_id", 2) + `
			ORDER BY product_id, supplier_id, (store_id IS NULL) ASC
		`
	}

	rows, err := db.Query(ctx, query, productID, storeID)
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
			&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &ps.StoreID, &createdAt,
		); err != nil {
			return nil, err
		}
		ps.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		result = append(result, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if storeID != nil {
		sortPreferredFirst(result)
	}
	return result, nil
}

// ListLinksBySupplier returns the link rows of a supplier in effect for the
// caller's scope, with the joined product name/SKU (both product_suppliers and
// products are katalog-owned).
func (SupplierLinkStore) ListLinksBySupplier(ctx context.Context, db shared.DBPool, supplierID int, storeID *int) ([]shared.ProductSupplier, error) {
	query := `
		SELECT ` + prefixedLinkColumns("ps") + `, p.name, p.sku
		FROM product_suppliers ps
		JOIN products p ON ps.product_id = p.id AND p.deleted_at IS NULL
		WHERE ps.supplier_id = $1 AND ` + visibleSQL("ps.store_id", 2) + `
		ORDER BY p.name ASC, ps.product_id ASC, ps.store_id ASC NULLS FIRST
	`
	if storeID != nil {
		query = `
			SELECT DISTINCT ON (ps.product_id, ps.supplier_id) ` + prefixedLinkColumns("ps") + `, p.name, p.sku
			FROM product_suppliers ps
			JOIN products p ON ps.product_id = p.id AND p.deleted_at IS NULL
			WHERE ps.supplier_id = $1 AND ` + visibleSQL("ps.store_id", 2) + `
			ORDER BY ps.product_id, ps.supplier_id, (ps.store_id IS NULL) ASC
		`
	}

	rows, err := db.Query(ctx, query, supplierID, storeID)
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
			&ps.UnitCost, &ps.LeadTimeDays, &ps.IsPreferred, &ps.StoreID, &createdAt,
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

	if storeID != nil {
		// Restore the by-product-name ordering that DISTINCT ON displaced.
		sort.SliceStable(result, func(i, j int) bool {
			return strings.Compare(derefStr(result[i].ProductName), derefStr(result[j].ProductName)) < 0
		})
	}
	return result, nil
}

// HasPreferredLink reports whether a preferred supplier is in effect for the
// caller's scope: the store's own choice if it has made one, otherwise the
// global default.
func (SupplierLinkStore) HasPreferredLink(ctx context.Context, db shared.DBPool, productID int, storeID *int) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM product_suppliers
			WHERE product_id = $1 AND is_preferred = true AND `+visibleSQL("store_id", 2)+`
		)
	`, productID, storeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check preferred supplier: %w", err)
	}
	return exists, nil
}

// CountLinksBySupplier counts every product_suppliers row that references the
// supplier, across every store. It deliberately takes no store scope: the
// supplier delete guard is estate-wide, and a link priced for one store is
// still a link the delete would cascade away, so a scoped count would understate
// the blast radius. product_suppliers has no soft-delete column, so every row
// counts.
func (SupplierLinkStore) CountLinksBySupplier(ctx context.Context, db shared.DBPool, supplierID int) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM product_suppliers WHERE supplier_id = $1
	`, supplierID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count product_suppliers by supplier: %w", err)
	}
	return count, nil
}

// sortPreferredFirst orders links the way the product detail screen has always
// shown them: the store's preferred supplier first, then by supplier ID. The
// unscoped branch gets this from SQL; the collapsed branch cannot, because
// DISTINCT ON fixes the sort keys.
func sortPreferredFirst(links []shared.ProductSupplier) {
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].IsPreferred != links[j].IsPreferred {
			return links[i].IsPreferred
		}
		return links[i].SupplierID < links[j].SupplierID
	})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// prefixedLinkColumns renders the link projection with a table alias, for the
// queries that join products.
func prefixedLinkColumns(alias string) string {
	cols := []string{"id", "product_id", "supplier_id", "supplier_sku", "unit_cost", "lead_time_days", "is_preferred", "store_id", "created_at"}
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = alias + "." + col
	}
	return strings.Join(out, ", ")
}

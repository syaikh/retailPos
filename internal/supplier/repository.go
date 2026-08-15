package supplier

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db shared.DBPool

	// psStore is the product-owned port for the product_suppliers link table
	// (katalog-owned, ADR Modular_Monolith_Module_Boundaries §2.8). It MUST be
	// wired via SetProductSupplierStore by the composition root.
	psStore ProductSupplierStore
}

func NewRepository(db shared.DBPool) *Repository {
	return &Repository{db: db}
}

// GetNextSupplierCode returns the next auto-generated supplier code (SUP-%06d)
// from supplier_seq. Used by Create when the payload omits a code.
func (r *Repository) GetNextSupplierCode(ctx context.Context) (string, error) {
	var seq int
	err := r.db.QueryRow(ctx, `SELECT nextval('supplier_seq')`).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("failed to get next supplier code: %w", err)
	}
	return fmt.Sprintf("SUP-%06d", seq), nil
}

// SetProductSupplierStore wires the product-owned port that backs all
// product_suppliers reads and writes. Calls to the linked-product methods
// fail fast until it is set.
func (r *Repository) SetProductSupplierStore(ps ProductSupplierStore) {
	r.psStore = ps
}

// linkStore returns the wired product_suppliers port, failing fast when the
// composition root has not wired it.
func (r *Repository) linkStore() ProductSupplierStore {
	if r.psStore == nil {
		panic("supplier.Repository: ProductSupplierStore is not wired — set it via SetProductSupplierStore")
	}
	return r.psStore
}

func (r *Repository) GetByID(ctx context.Context, id int) (*Supplier, error) {
	var s Supplier
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, is_consignment, created_at, updated_at
		FROM suppliers WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&s.ID, &s.Name, &s.Code, &s.ContactName,
		&s.Email, &s.Phone, &s.Address, &s.Notes,
		&s.IsActive, &s.IsConsignment, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}

	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}

func (r *Repository) GetByIDs(ctx context.Context, ids []int) ([]Supplier, error) {
	if len(ids) == 0 {
		return []Supplier{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, is_consignment, created_at, updated_at
		FROM suppliers WHERE id IN (%s) AND deleted_at IS NULL`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch get suppliers by ids: %w", err)
	}
	defer rows.Close()

	return scanSuppliers(rows)
}

func (r *Repository) GetIDsByName(ctx context.Context, name string) ([]int, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM suppliers WHERE name ILIKE $1 AND deleted_at IS NULL`, "%"+name+"%")
	if err != nil {
		return nil, fmt.Errorf("get supplier ids by name: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	var s Supplier
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, is_consignment, created_at, updated_at
		FROM suppliers WHERE code = $1 AND deleted_at IS NULL
	`, code).Scan(
		&s.ID, &s.Name, &s.Code, &s.ContactName,
		&s.Email, &s.Phone, &s.Address, &s.Notes,
		&s.IsActive, &s.IsConsignment, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSupplierNotFound
		}
		return nil, err
	}

	s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return &s, nil
}

func (r *Repository) Create(ctx context.Context, supplier *Supplier) error {
	var createdAt, updatedAt time.Time
	err := r.db.QueryRow(ctx, `
		INSERT INTO suppliers (name, code, contact_name, email, phone, address, notes, is_active, is_consignment)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, supplier.Name, supplier.Code, supplier.ContactName,
		supplier.Email, supplier.Phone, supplier.Address, supplier.Notes, supplier.IsActive, supplier.IsConsignment,
	).Scan(&supplier.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert supplier: %w", err)
	}
	supplier.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	supplier.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
	return nil
}

func (r *Repository) Update(ctx context.Context, supplier *Supplier) error {
	_, err := r.db.Exec(ctx, `
		UPDATE suppliers
		SET name = $1, code = $2, contact_name = $3, email = $4, phone = $5,
		    address = $6, notes = $7, is_active = $8, is_consignment = $9, updated_at = NOW()
		WHERE id = $10 AND deleted_at IS NULL
	`, supplier.Name, supplier.Code, supplier.ContactName,
		supplier.Email, supplier.Phone, supplier.Address,
		supplier.Notes, supplier.IsActive, supplier.IsConsignment, supplier.ID)
	if err != nil {
		return fmt.Errorf("update supplier: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE suppliers SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("delete supplier: %w", err)
	}
	return nil
}

func (r *Repository) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
	countQuery := `SELECT COUNT(*) FROM suppliers WHERE deleted_at IS NULL`
	dataQuery := `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, is_consignment, created_at, updated_at
		FROM suppliers WHERE deleted_at IS NULL`

	var args []interface{}
	argIdx := 1

	if search != "" {
		filter := fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d OR contact_name ILIKE $%d)", argIdx, argIdx+1, argIdx+2)
		countQuery += filter
		dataQuery += filter
		args = append(args, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		argIdx += 3
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

	dataQuery += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	suppliers, err := scanSuppliers(rows)
	if err != nil {
		return nil, 0, err
	}
	return suppliers, total, nil
}

func (r *Repository) LinkProduct(ctx context.Context, ps *ProductSupplier) error {
	return r.linkStore().CreateLink(ctx, r.db, ps)
}

func (r *Repository) UnlinkProduct(ctx context.Context, productID, supplierID int) error {
	return r.linkStore().DeleteLink(ctx, r.db, productID, supplierID)
}

func (r *Repository) GetProductSupplier(ctx context.Context, productID, supplierID int) (*ProductSupplier, error) {
	return r.linkStore().GetLink(ctx, r.db, productID, supplierID)
}

func (r *Repository) GetPreferredSupplier(ctx context.Context, productID int) (*ProductSupplier, error) {
	return r.linkStore().GetPreferredLink(ctx, r.db, productID)
}

func (r *Repository) SetPreferredSupplier(ctx context.Context, productID, supplierID int) error {
	return r.linkStore().SetPreferredLink(ctx, r.db, productID, supplierID)
}

func (r *Repository) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	return r.linkStore().UpdateLink(ctx, r.db, ps)
}

// GetSuppliersByProductID returns the product-supplier links of a product,
// enriched with the supplier name/code. The link rows come from the
// product-owned port; supplier enrichment is computed here on the suppliers
// table (referensi-owned), preserving the previous JOIN's ordering.
func (r *Repository) GetSuppliersByProductID(ctx context.Context, productID int) ([]ProductSupplier, error) {
	links, err := r.linkStore().ListLinksByProduct(ctx, r.db, productID)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []ProductSupplier{}, nil
	}

	ids := make([]int, len(links))
	for i, l := range links {
		ids[i] = l.SupplierID
	}
	suppliers, err := r.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]Supplier, len(suppliers))
	for _, s := range suppliers {
		byID[s.ID] = s
	}
	for i := range links {
		s, ok := byID[links[i].SupplierID]
		if !ok {
			continue
		}
		name, code := s.Name, s.Code
		links[i].SupplierName = &name
		links[i].SupplierCode = &code
	}

	sort.SliceStable(links, func(i, j int) bool {
		if links[i].IsPreferred != links[j].IsPreferred {
			return links[i].IsPreferred
		}
		ni, nj := "", ""
		if links[i].SupplierName != nil {
			ni = *links[i].SupplierName
		}
		if links[j].SupplierName != nil {
			nj = *links[j].SupplierName
		}
		return ni < nj
	})
	return links, nil
}

func (r *Repository) GetProductsBySupplierID(ctx context.Context, supplierID int) ([]ProductSupplier, error) {
	return r.linkStore().ListLinksBySupplier(ctx, r.db, supplierID)
}

func (r *Repository) HasPreferredSupplier(ctx context.Context, productID int) (bool, error) {
	return r.linkStore().HasPreferredLink(ctx, r.db, productID)
}

func scanSuppliers(rows pgx.Rows) ([]Supplier, error) {
	var suppliers []Supplier
	for rows.Next() {
		var s Supplier
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&s.ID, &s.Name, &s.Code, &s.ContactName,
			&s.Email, &s.Phone, &s.Address, &s.Notes,
			&s.IsActive, &s.IsConsignment, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}
		s.CreatedAt = createdAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		s.UpdatedAt = updatedAt.In(shared.JakartaLocation()).Format(time.RFC3339)
		suppliers = append(suppliers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suppliers, nil
}

type ImportRow struct {
	Row         int
	Code        string
	Name        string
	ContactName string
	Phone       string
	Email       string
	Address     string
	Notes       string
	IsActive    bool
}

type ImportPayload struct {
	Code        string
	Name        string
	ContactName *string
	Phone       *string
	Email       *string
	Address     *string
	Notes       *string
	IsActive    bool
}

func (r *Repository) BulkInsertSuppliers(ctx context.Context, payloads []ImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	rows := make([][]interface{}, len(payloads))
	for i, p := range payloads {
		rows[i] = []interface{}{p.Name, p.Code, p.ContactName, p.Email, p.Phone, p.Address, p.Notes, p.IsActive}
	}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"suppliers"},
		[]string{"name", "code", "contact_name", "email", "phone", "address", "notes", "is_active"},
		pgx.CopyFromRows(rows))
	if err != nil {
		return 0, fmt.Errorf("bulk insert suppliers: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return len(payloads), nil
}

func (r *Repository) BulkUpdateSuppliers(ctx context.Context, payloads []ImportPayload) (int, error) {
	if len(payloads) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const maxBatchSize = 1000
	var totalRowsAffected int64
	for start := 0; start < len(payloads); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(payloads) {
			end = len(payloads)
		}
		chunk := payloads[start:end]

		valueStrings := make([]string, 0, len(chunk))
		valueArgs := make([]interface{}, 0, len(chunk)*8)
		for i, p := range chunk {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d::text,$%d::text,$%d::text,$%d::text,$%d::text,$%d::text,$%d::boolean,$%d::text)",
				i*8+1, i*8+2, i*8+3, i*8+4, i*8+5, i*8+6, i*8+7, i*8+8))
			valueArgs = append(valueArgs, p.Name, p.ContactName, p.Email, p.Phone, p.Address, p.Notes, p.IsActive, p.Code)
		}

		query := fmt.Sprintf(`
			UPDATE suppliers s
			SET name = v.name, contact_name = v.contact_name, email = v.email,
			    phone = v.phone, address = v.address, notes = v.notes,
			    is_active = v.is_active, updated_at = NOW()
			FROM (VALUES %s) AS v(name, contact_name, email, phone, address, notes, is_active, code)
			WHERE s.code = v.code AND s.deleted_at IS NULL
		`, strings.Join(valueStrings, ","))

		tag, err := tx.Exec(ctx, query, valueArgs...)
		if err != nil {
			return 0, fmt.Errorf("bulk update suppliers: %w", err)
		}
		totalRowsAffected += tag.RowsAffected()
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return int(totalRowsAffected), nil
}

func (r *Repository) GetAllForExport(ctx context.Context) ([]Supplier, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, code, contact_name, email, phone, address, notes, is_active, is_consignment, created_at, updated_at
		FROM suppliers WHERE deleted_at IS NULL ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSuppliers(rows)
}

func (r *Repository) BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx, `
		UPDATE suppliers SET is_active = $1, updated_at = NOW()
		WHERE id = ANY($2) AND deleted_at IS NULL
	`, isActive, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk update suppliers: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *Repository) BulkDelete(ctx context.Context, ids []int) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	count := 0
	for _, id := range ids {
		tag, err := tx.Exec(ctx, `
			UPDATE suppliers SET deleted_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND deleted_at IS NULL
		`, id)
		if err != nil {
			return count, fmt.Errorf("delete supplier: %w", err)
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

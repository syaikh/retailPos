package consignment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/supplier"
	"retail-pos-system/internal/user"
)

var (
	dbPool *pgxpool.Pool
	seq    atomic.Int64
)

func uniqueSuffix() string {
	return fmt.Sprintf("%d", seq.Add(1))
}

func TestMain(m *testing.M) {
	pool, err := shared.NewTestDB()
	if err != nil {
		os.Exit(1)
	}
	dbPool = pool
	defer pool.Close()

	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		os.Exit(1)
	}

	if err := shared.TruncateTestData(pool); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// newTestService wires a real Repository with the same providers the
// composition root uses (inventory adjuster, supplier store, product meta,
// usernames) plus a payment-method provider backed by the test DB.
func newTestService(t *testing.T) *Service {
	t.Helper()
	repo := NewRepository(dbPool)
	repo.SetStockAdjuster(inventory.ConsignmentAdjuster{})
	repo.SetSupplierStore(supplier.ConsignmentSupplierProvider{})
	repo.SetProductMetaProvider(product.MetaLookup{})
	repo.SetUsernameProvider(user.UsernamesProvider{})
	repo.SetPaymentMethods(dbPaymentMethods{})
	return NewService(repo)
}

// dbPaymentMethods answers payment-method queries straight from the test DB,
// mirroring the wiring's paymentMethodAdapter without needing internal/sale.
type dbPaymentMethods struct{}

func (dbPaymentMethods) ActivePaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	rows, err := dbPool.Query(ctx, `SELECT id, code, name FROM payment_methods WHERE is_active = true ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PaymentMethod
	for rows.Next() {
		var m PaymentMethod
		if err := rows.Scan(&m.ID, &m.Code, &m.Name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (dbPaymentMethods) PaymentMethodByID(ctx context.Context, id int) (*PaymentMethod, error) {
	var m PaymentMethod
	err := dbPool.QueryRow(ctx, `SELECT id, code, name FROM payment_methods WHERE id = $1 AND is_active = true`, id).
		Scan(&m.ID, &m.Code, &m.Name)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func insertTestStore(ctx context.Context, t *testing.T) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `INSERT INTO stores (name) VALUES ($1) RETURNING id`, "Consignment Test Store").Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestSupplier(ctx context.Context, t *testing.T, name string, consignment bool) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO suppliers (name, code, is_active, is_consignment)
		VALUES ($1, $2, true, $3)
		RETURNING id
	`, name, name+"-"+uniqueSuffix(), consignment).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestUser(ctx context.Context, t *testing.T) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, role_id)
		VALUES ($1, $2, 'hash', 1)
		ON CONFLICT (username) DO UPDATE SET email = EXCLUDED.email
		RETURNING id
	`, "consignment_user", "consignment@test.com").Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestPaymentMethod(ctx context.Context, t *testing.T, code string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO payment_methods (code, name, is_active, requires_reference)
		VALUES ($1, $1, true, false)
		ON CONFLICT (code) DO UPDATE SET is_active = true
		RETURNING id
	`, code).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertTestProduct(ctx context.Context, t *testing.T, sku string) int {
	t.Helper()
	var id int
	err := dbPool.QueryRow(ctx, `
		INSERT INTO products (sku, name, price, cost, status)
		VALUES ($1, $2, 10000, 5000, 'active')
		RETURNING id
	`, sku, "Test Product "+sku).Scan(&id)
	require.NoError(t, err)
	return id
}

// globalStockQty reads the store-owned global product_stock bucket for a product.
func globalStockQty(ctx context.Context, t *testing.T, productID int) int {
	t.Helper()
	var qty int
	err := dbPool.QueryRow(ctx, `
		SELECT COALESCE(quantity, 0) FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, productID).Scan(&qty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0
		}
		require.NoError(t, err)
	}
	return qty
}

// setupArrangement creates a consignment supplier, store, arrangement, and
// terms for the given products, returning the service and ids.
func setupArrangement(t *testing.T, products ...int) (*Service, int, int) {
	t.Helper()
	ctx := context.Background()
	userID := insertTestUser(ctx, t)
	supplierID := insertTestSupplier(ctx, t, "Konsinyasi Test Supplier", true)
	storeID := insertTestStore(ctx, t)

	svc := newTestService(t)
	arr, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: supplierID, StoreID: storeID}, userID, nil)
	require.NoError(t, err)
	require.Greater(t, arr.ID, 0)

	terms := make([]SetTermsRequest, 0, len(products))
	for _, pid := range products {
		terms = append(terms, SetTermsRequest{ProductID: pid, Price: 10000, StoreShareType: ShareTypePercentage, StoreShareValue: 20})
	}
	created, err := svc.SetTerms(ctx, arr.ID, terms, userID, nil)
	require.NoError(t, err)
	require.Len(t, created, len(products))
	return svc, supplierID, storeID
}

// setupArrangementNoTerms creates a consignment arrangement without setting any
// terms. Useful when the test needs to set terms separately (e.g. to test
// ownership conflicts).
func setupArrangementNoTerms(t *testing.T) (*Service, int, int, int) {
	t.Helper()
	ctx := context.Background()
	userID := insertTestUser(ctx, t)
	supplierID := insertTestSupplier(ctx, t, "Konsinyasi Test Supplier", true)
	storeID := insertTestStore(ctx, t)
	svc := newTestService(t)
	arr, err := svc.CreateArrangement(ctx, &CreateArrangementRequest{SupplierID: supplierID, StoreID: storeID}, userID, nil)
	require.NoError(t, err)
	require.Greater(t, arr.ID, 0)
	return svc, supplierID, storeID, userID
}

package customer

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func customerRow(id int, name, phone string, storeID int, isActive, isWalkIn bool, now time.Time) *pgxmock.Rows {
	return pgxmock.NewRows([]string{
		"id", "name", "phone", "email", "address", "tax_id",
		"loyalty_points", "total_spent", "last_purchase_at", "note",
		"is_active", "is_walk_in", "store_id", "created_at", "updated_at",
	}).AddRow(
		id, name, phone, "", "", "",
		0, 0.0, nil, "",
		isActive, isWalkIn, storeID, now, now,
	)
}

func TestRepository_GetByPhone(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := customerRow(1, "John", "081234", 1, true, false, now)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE phone = \\$1").WithArgs("081234").WillReturnRows(rows)

	repo := NewRepository(mock)
	c, err := repo.GetByPhone(context.Background(), "081234", nil)
	require.NoError(t, err)
	assert.Equal(t, "John", c.Name)
	require.NotNil(t, c.Phone)
	assert.Equal(t, "081234", *c.Phone)
	require.NotNil(t, c.StoreID)
	assert.Equal(t, 1, *c.StoreID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByPhone_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	sid := 1
	rows := customerRow(1, "John", "081234", 1, true, false, now)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE phone = \\$1 AND").WithArgs("081234", sid).WillReturnRows(rows)

	repo := NewRepository(mock)
	c, err := repo.GetByPhone(context.Background(), "081234", &sid)
	require.NoError(t, err)
	assert.Equal(t, "John", c.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByPhone_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM customers WHERE phone = \\$1").WithArgs("000").WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetByPhone(context.Background(), "000", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCustomerByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := customerRow(5, "Jane", "089999", 2, true, false, now)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id = \\$1").WithArgs(5).WillReturnRows(rows)

	repo := NewRepository(mock)
	c, err := repo.GetCustomerByID(context.Background(), 5, nil)
	require.NoError(t, err)
	assert.Equal(t, "Jane", c.Name)
	require.NotNil(t, c.StoreID)
	assert.Equal(t, 2, *c.StoreID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCustomerByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id = \\$1").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetCustomerByID(context.Background(), 999, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCustomerByID_WalkIn(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := customerRow(10, "Walk-in", "", 1, true, true, now)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE id = \\$1").WithArgs(10).WillReturnRows(rows)

	repo := NewRepository(mock)
	c, err := repo.GetCustomerByID(context.Background(), 10, nil)
	require.NoError(t, err)
	assert.True(t, c.IsWalkIn)
	assert.Nil(t, c.StoreID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteCustomer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE customers SET is_active = false").
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	err = repo.DeleteCustomer(context.Background(), 1, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteCustomer_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	sid := 1
	mock.ExpectExec("UPDATE customers SET is_active = false").
		WithArgs(1, sid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	err = repo.DeleteCustomer(context.Background(), 1, &sid)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpdateCustomersStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE customers SET is_active = \\$1").
		WithArgs(true, []int{1, 2}).
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	repo := NewRepository(mock)
	err = repo.BulkUpdateCustomersStatus(context.Background(), []int{1, 2}, true, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkDeleteCustomers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE customers SET is_active = false").
		WithArgs([]int{1, 2, 3}).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	repo := NewRepository(mock)
	err = repo.BulkDeleteCustomers(context.Background(), []int{1, 2, 3}, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsertCustomers_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsertCustomers(context.Background(), nil, nil)
	assert.Equal(t, 0, result.Inserted)
}

func TestRepository_BulkUpsertCustomers_NameRequired(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsertCustomers(context.Background(), []CustomerImportRow{
		{Row: 1, Name: ""},
	}, nil)
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Name is required")
}

func TestRepository_BulkUpsertCustomers_AllInvalid(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsertCustomers(context.Background(), []CustomerImportRow{
		{Row: 1, Name: ""},
		{Row: 2, Name: ""},
	}, nil)
	assert.Equal(t, 0, result.Inserted)
	assert.Equal(t, 0, result.Updated)
}

func TestRepository_GetAllCustomersForExport(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := customerRow(1, "Cust", "08111", 1, true, false, now)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE is_walk_in = false ORDER BY name").WillReturnRows(rows)

	repo := NewRepository(mock)
	customers, err := repo.GetAllCustomersForExport(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, customers, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCustomersForExport_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	sid := 1
	rows := customerRow(1, "Cust", "08111", sid, true, false, now)
	mock.ExpectQuery("SELECT (.+) FROM customers WHERE is_walk_in = false AND store_id").WithArgs(sid).WillReturnRows(rows)

	repo := NewRepository(mock)
	customers, err := repo.GetAllCustomersForExport(context.Background(), &sid)
	require.NoError(t, err)
	assert.Len(t, customers, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateCustomer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(42, now, now)
	phone := strPtr("08111")
	mock.ExpectQuery("INSERT INTO customers").
		WithArgs("New", phone, (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), true, false, 1).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	c := &Customer{Name: "New", Phone: phone, IsActive: true, IsWalkIn: false}
	err = repo.CreateCustomer(context.Background(), c)
	require.NoError(t, err)
	assert.Equal(t, 42, c.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateCustomer_NilStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(43, now, now)
	phone := strPtr("08222")
	mock.ExpectQuery("INSERT INTO customers").
		WithArgs("NoStore", phone, (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), true, false, 1).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	c := &Customer{Name: "NoStore", Phone: phone, IsActive: true, IsWalkIn: false, StoreID: nil}
	err = repo.CreateCustomer(context.Background(), c)
	require.NoError(t, err)
	assert.Equal(t, 43, c.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateCustomer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	phone := strPtr("08333")
	email := strPtr("u@t.com")
	mock.ExpectExec("UPDATE customers").
		WithArgs("Upd", phone, email, (*string)(nil), (*string)(nil), (*string)(nil), true, 1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	c := &Customer{Name: "Upd", Phone: phone, Email: email, IsActive: true}
	err = repo.UpdateCustomer(context.Background(), c, 1, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateCustomer_WithStoreID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	sid := 1
	phone := strPtr("08333")
	mock.ExpectExec("UPDATE customers").
		WithArgs("Upd", phone, (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), true, 1, sid).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	c := &Customer{Name: "Upd", Phone: phone, IsActive: true}
	err = repo.UpdateCustomer(context.Background(), c, 1, &sid)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCustomers_NoFilters(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	dataRows := customerRow(1, "Cust", "08111", 1, true, false, now)
	mock.ExpectQuery("SELECT id, name, phone").WithArgs(10, 0).WillReturnRows(dataRows)

	repo := NewRepository(mock)
	customers, total, err := repo.GetAllCustomers(context.Background(), 10, 0, "", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, customers, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCustomers_SearchAndFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	active := true
	sid := 1
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(sid, "%search%", active).
		WillReturnRows(countRows)

	dataRows := pgxmock.NewRows([]string{"id", "name", "phone", "email", "address", "tax_id", "loyalty_points", "total_spent", "last_purchase_at", "note", "is_active", "is_walk_in", "store_id", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, name, phone").
		WithArgs(sid, "%search%", active, 10, 0).
		WillReturnRows(dataRows)

	repo := NewRepository(mock)
	_, _, err = repo.GetAllCustomers(context.Background(), 10, 0, "search", &active, &sid)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCustomers_StoreIDFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	sid := 2
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(sid).
		WillReturnRows(countRows)

	dataRows := pgxmock.NewRows([]string{"id", "name", "phone", "email", "address", "tax_id", "loyalty_points", "total_spent", "last_purchase_at", "note", "is_active", "is_walk_in", "store_id", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, name, phone").
		WithArgs(sid, 10, 0).
		WillReturnRows(dataRows)

	repo := NewRepository(mock)
	_, _, err = repo.GetAllCustomers(context.Background(), 10, 0, "", nil, &sid)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCustomers_FilterOnly(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	active := false
	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(active).
		WillReturnRows(countRows)

	dataRows := pgxmock.NewRows([]string{"id", "name", "phone", "email", "address", "tax_id", "loyalty_points", "total_spent", "last_purchase_at", "note", "is_active", "is_walk_in", "store_id", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT id, name, phone").
		WithArgs(active, 10, 0).
		WillReturnRows(dataRows)

	repo := NewRepository(mock)
	_, _, err = repo.GetAllCustomers(context.Background(), 10, 0, "", &active, nil)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

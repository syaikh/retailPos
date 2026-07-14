package shared

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanRow_String(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name", "phone"}).
		AddRow(1, "John", "081234")
	mock.ExpectQuery("SELECT (.+)").WillReturnRows(rows)

	r := mock.QueryRow(context.Background(), "SELECT 1")
	var id int
	var name, phone string
	err = ScanRow(r, &id, &name, &phone)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Equal(t, "John", name)
	assert.Equal(t, "081234", phone)
}

func TestScanRow_PointerToString(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name", "phone"}).
		AddRow(1, "John", "081234")
	mock.ExpectQuery("SELECT (.+)").WillReturnRows(rows)

	r := mock.QueryRow(context.Background(), "SELECT 1")
	var id int
	var name string
	var phone *string
	err = ScanRow(r, &id, &name, &phone)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Equal(t, "John", name)
	require.NotNil(t, phone)
	assert.Equal(t, "081234", *phone)
}

func TestScanRow_PointerToString_Nil(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "phone"}).
		AddRow(1, nil)
	mock.ExpectQuery("SELECT (.+)").WillReturnRows(rows)

	r := mock.QueryRow(context.Background(), "SELECT 1")
	var id int
	var phone *string
	err = ScanRow(r, &id, &phone)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Nil(t, phone)
}

func TestScanRow_MixedTypes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := "2026-01-01"
	rows := pgxmock.NewRows([]string{"id", "name", "phone", "email", "address", "created_at"}).
		AddRow(5, "Jane", "089999", "jane@test.com", "123 St", now)
	mock.ExpectQuery("SELECT (.+)").WillReturnRows(rows)

	r := mock.QueryRow(context.Background(), "SELECT 1")
	var id int
	var name, email string
	var phone, address, createdAt *string
	err = ScanRow(r, &id, &name, &phone, &email, &address, &createdAt)
	require.NoError(t, err)
	assert.Equal(t, 5, id)
	assert.Equal(t, "Jane", name)
	require.NotNil(t, phone)
	assert.Equal(t, "089999", *phone)
	assert.Equal(t, "jane@test.com", email)
	require.NotNil(t, address)
	assert.Equal(t, "123 St", *address)
	require.NotNil(t, createdAt)
	assert.Equal(t, now, *createdAt)
}

func TestScanRow_StandardTypes(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "active"}).
		AddRow(10, true)
	mock.ExpectQuery("SELECT (.+)").WillReturnRows(rows)

	r := mock.QueryRow(context.Background(), "SELECT 1")
	var id int
	var active bool
	err = ScanRow(r, &id, &active)
	require.NoError(t, err)
	assert.Equal(t, 10, id)
	assert.True(t, active)
}

func TestScanRow_WithRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "phone"}).
		AddRow(1, "08111").
		AddRow(2, "08222")
	mock.ExpectQuery("SELECT (.+)").WillReturnRows(rows)

	rs, err := mock.Query(context.Background(), "SELECT 1")
	require.NoError(t, err)
	defer rs.Close()

	for rs.Next() {
		var id int
		var phone *string
		err = ScanRow(rs, &id, &phone)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
		assert.NotNil(t, phone)
	}
}

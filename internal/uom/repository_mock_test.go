package uom

import (
	"context"
	"fmt"
	"testing"
	"time"

	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "description", "is_active", "created_at"}).
		AddRow(1, "KG", "Kilogram", "mass unit", true, now)
	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE id = \\$1").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	u, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "KG", u.Code)
	assert.Equal(t, "Kilogram", u.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE id = \\$1").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetByID(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAll(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "description", "is_active", "created_at"}).
		AddRow(1, "KG", "Kilogram", "", true, now).
		AddRow(2, "GM", "Gram", "", true, now)
	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE is_active").WillReturnRows(rows)

	repo := NewRepository(mock)
	units, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, units, 2)
	assert.Equal(t, "KG", units[0].Code)
	assert.Equal(t, "GM", units[1].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at"}).AddRow(42, now)
	mock.ExpectQuery("INSERT INTO units_of_measure").
		WithArgs("L", "Liter", "", true).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	u := &UnitOfMeasure{Code: "L", Name: "Liter", IsActive: true}
	err = repo.Create(context.Background(), u)
	require.NoError(t, err)
	assert.Equal(t, 42, u.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE units_of_measure SET").
		WithArgs("KG", "Kilogram", "", true, 1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	u := &UnitOfMeasure{ID: 1, Code: "KG", Name: "Kilogram", IsActive: true}
	err = repo.Update(context.Background(), u)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM units_of_measure").
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewRepository(mock)
	err = repo.Delete(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsert(context.Background(), nil)
	assert.Equal(t, 0, result.Inserted)
}

func TestRepository_BulkUpsert_CodeRequired(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsert(context.Background(), []ImportRow{
		{Row: 1, Code: "", Name: "test"},
	})
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Code is required")
}

func TestRepository_BulkUpsert_NameRequired(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsert(context.Background(), []ImportRow{
		{Row: 1, Code: "KG", Name: ""},
	})
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Name is required")
}

func TestRepository_SetCache(t *testing.T) {
	repo := NewRepository(nil)
	c := cache.New(5*time.Minute, 10*time.Minute)
	repo.SetCache(c)
	assert.NotNil(t, repo.cache)
}

func TestRepository_GetIDsByCodes_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result, err := repo.GetIDsByCodes(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRepository_GetIDsByCodes_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "code"}).
		AddRow(1, "KG").AddRow(2, "LT")
	mock.ExpectQuery("SELECT id, code FROM units_of_measure").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	result, err := repo.GetIDsByCodes(context.Background(), []string{"KG", "LT"})
	require.NoError(t, err)
	assert.Equal(t, 1, result["KG"])
	assert.Equal(t, 2, result["LT"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetIDsByCodes_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, code FROM units_of_measure").WithArgs(pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetIDsByCodes(context.Background(), []string{"KG"})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE units_of_measure SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("update failed"))

	repo := NewRepository(mock)
	u := &UnitOfMeasure{ID: 1, Code: "KG", Name: "Kilogram", IsActive: true}
	err = repo.Update(context.Background(), u)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("uom:1", UnitOfMeasure{ID: 1})
	c.Set("uoms:all", []UnitOfMeasure{{ID: 1}})
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("UPDATE units_of_measure SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	u := &UnitOfMeasure{ID: 1, Code: "KG", Name: "Kilogram", IsActive: true}
	err = repo.Update(context.Background(), u)
	assert.NoError(t, err)

	_, ok1 := c.Get("uom:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("uoms:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM units_of_measure").WithArgs(pgxmock.AnyArg()).WillReturnError(fmt.Errorf("fk violation"))

	repo := NewRepository(mock)
	err = repo.Delete(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("uom:1", UnitOfMeasure{ID: 1})
	c.Set("uoms:all", []UnitOfMeasure{{ID: 1}})
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("DELETE FROM units_of_measure").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), 1)
	assert.NoError(t, err)

	_, ok1 := c.Get("uom:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("uoms:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	c.Set("uom:1", UnitOfMeasure{ID: 1, Code: "KG", Name: "Kilogram"})
	c.Wait()

	u, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "KG", u.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE id = \\$1").WithArgs(1).WillReturnError(fmt.Errorf("db lost"))

	repo := NewRepository(mock)
	_, err = repo.GetByID(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db lost")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_DBCacheSet(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "description", "is_active", "created_at"}).
		AddRow(1, "KG", "Kilogram", "mass", true, now)
	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE id = \\$1").WithArgs(1).WillReturnRows(rows)

	u, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "KG", u.Code)

	c.Wait()
	v, ok := c.Get("uom:1")
	assert.True(t, ok)
	assert.Equal(t, "KG", v.(UnitOfMeasure).Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAll_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	units := []UnitOfMeasure{{ID: 1, Code: "KG"}}
	c.Set("uoms:all", units)
	c.Wait()

	result, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "KG", result[0].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAll_CacheSet(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "description", "is_active", "created_at"}).
		AddRow(1, "KG", "Kilogram", "", true, now)
	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE is_active").WillReturnRows(rows)

	result, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)

	c.Wait()
	v, ok := c.Get("uoms:all")
	assert.True(t, ok)
	assert.Equal(t, "KG", v.([]UnitOfMeasure)[0].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAll_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM units_of_measure WHERE is_active").WillReturnError(fmt.Errorf("query failed"))

	repo := NewRepository(mock)
	_, err = repo.GetAll(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_CacheFlush(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("uom:1", UnitOfMeasure{ID: 1})
	c.Set("uoms:all", []UnitOfMeasure{{ID: 1}})
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at"}).AddRow(10, now)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	u := &UnitOfMeasure{Code: "L", Name: "Liter", IsActive: true}
	err = repo.Create(context.Background(), u)
	require.NoError(t, err)

	_, ok1 := c.Get("uom:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("uoms:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("duplicate key"))

	repo := NewRepository(mock)
	u := &UnitOfMeasure{Code: "KG", Name: "Kilogram", IsActive: true}
	err = repo.Create(context.Background(), u)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllForExport_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "code", "name", "description", "is_active", "created_at"}).
		AddRow(1, "KG", "Kilogram", "mass", true, now)
	mock.ExpectQuery("SELECT (.+) FROM units_of_measure ORDER BY code").WillReturnRows(rows)

	repo := NewRepository(mock)
	result, err := repo.GetAllForExport(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "KG", result[0].Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllForExport_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM units_of_measure ORDER BY code").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetAllForExport(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).
		AddRow(true).AddRow(false)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnRows(rows)

	repo := NewRepository(mock)
	result := repo.BulkUpsert(context.Background(), []ImportRow{
		{Row: 1, Code: "L", Name: "Liter", IsActive: true},
		{Row: 2, Code: "KG", Name: "Kilogram", IsActive: true},
	})
	assert.Equal(t, 1, result.Inserted)
	assert.Equal(t, 1, result.Updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db down"))

	repo := NewRepository(mock)
	result := repo.BulkUpsert(context.Background(), []ImportRow{
		{Row: 1, Code: "KG", Name: "Kilogram", IsActive: true},
	})
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "batch upsert failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(nil)
	mock.ExpectQuery("INSERT INTO units_of_measure").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	result := repo.BulkUpsert(context.Background(), []ImportRow{
		{Row: 1, Code: "KG", Name: "Kilogram", IsActive: true},
	})
	assert.Equal(t, 0, result.Inserted)
	assert.Equal(t, 1, result.Updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

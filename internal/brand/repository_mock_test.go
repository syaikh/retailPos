package brand

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
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Test Brand", "desc", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM brands WHERE id = \\$1").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	b, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Test Brand", b.Name)
	assert.Equal(t, "desc", b.Description)
	assert.True(t, b.IsActive)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM brands WHERE id = \\$1").WithArgs(999).WillReturnError(pgx.ErrNoRows)

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
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Brand A", "a", true, now, now).
		AddRow(2, "Brand B", "b", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM brands WHERE is_active").WillReturnRows(rows)

	repo := NewRepository(mock)
	brands, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, brands, 2)
	assert.Equal(t, "Brand A", brands[0].Name)
	assert.Equal(t, "Brand B", brands[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow(10, now, now)
	mock.ExpectQuery("INSERT INTO brands").WithArgs("New Brand", "new", true).WillReturnRows(rows)

	repo := NewRepository(mock)
	b := &Brand{Name: "New Brand", Description: "new", IsActive: true}
	err = repo.Create(context.Background(), b)
	require.NoError(t, err)
	assert.Equal(t, 10, b.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE brands SET").
		WithArgs("Updated", "upd", false, 1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	b := &Brand{ID: 1, Name: "Updated", Description: "upd", IsActive: false}
	err = repo.Update(context.Background(), b)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM brands").
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewRepository(mock)
	err = repo.Delete(context.Background(), 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllForExport(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Export Brand", "export", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM brands ORDER BY name").WillReturnRows(rows)

	repo := NewRepository(mock)
	brands, err := repo.GetAllForExport(context.Background())
	require.NoError(t, err)
	assert.Len(t, brands, 1)
	assert.Equal(t, "Export Brand", brands[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsert(context.Background(), nil)
	assert.Equal(t, 0, result.Inserted)
	assert.Equal(t, 0, result.Updated)
}

func TestRepository_BulkUpsert_NameRequired(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsert(context.Background(), []BrandImportRow{
		{Row: 1, Name: "", IsActive: true},
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

func TestRepository_GetByID_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	c.Set("brand:1", Brand{ID: 1, Name: "Cached Brand", Description: "cached", IsActive: true, CreatedAt: now.Format(time.RFC3339)})
	c.Wait()

	b, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Cached Brand", b.Name)
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
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Brand", "desc", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM brands WHERE id = \\$1").WithArgs(1).WillReturnRows(rows)

	b, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Brand", b.Name)

	c.Wait()
	v, ok := c.Get("brand:1")
	assert.True(t, ok)
	assert.Equal(t, "Brand", v.(Brand).Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM brands WHERE id = \\$1").WithArgs(1).WillReturnError(fmt.Errorf("db connection lost"))

	repo := NewRepository(mock)
	_, err = repo.GetByID(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAll_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	brands := []Brand{{ID: 1, Name: "Cached"}}
	c.Set("brands:all", brands)
	c.Wait()

	result, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Cached", result[0].Name)
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
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Brand", "desc", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM brands WHERE is_active").WillReturnRows(rows)

	result, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)

	c.Wait()
	v, ok := c.Get("brands:all")
	assert.True(t, ok)
	assert.Equal(t, "Brand", v.([]Brand)[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAll_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM brands WHERE is_active").WillReturnError(fmt.Errorf("query failed"))

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
	c.Set("brand:1", Brand{ID: 1})
	c.Set("brands:all", []Brand{{ID: 1}})
	c.Wait()
	repo := NewRepository(mock)
	repo.SetCache(c)

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(10, now, now)
	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	b := &Brand{Name: "New", Description: "new", IsActive: true}
	err = repo.Create(context.Background(), b)
	require.NoError(t, err)

	c.Wait()
	_, ok1 := c.Get("brand:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("brands:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Create_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("duplicate key"))

	repo := NewRepository(mock)
	b := &Brand{Name: "Dup", IsActive: true}
	err = repo.Create(context.Background(), b)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE brands SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("update failed"))

	repo := NewRepository(mock)
	b := &Brand{ID: 1, Name: "X", IsActive: true}
	err = repo.Update(context.Background(), b)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Update_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("brand:1", Brand{ID: 1})
	c.Set("brands:all", []Brand{{ID: 1}})
	c.Wait()
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("UPDATE brands SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	b := &Brand{ID: 1, Name: "X", IsActive: true}
	err = repo.Update(context.Background(), b)
	assert.NoError(t, err)

	c.Wait()
	_, ok1 := c.Get("brand:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("brands:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_Delete_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM brands").WithArgs(pgxmock.AnyArg()).WillReturnError(fmt.Errorf("fk violation"))

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
	c.Set("brand:1", Brand{ID: 1})
	c.Set("brands:all", []Brand{{ID: 1}})
	c.Wait()
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("DELETE FROM brands").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), 1)
	assert.NoError(t, err)

	c.Wait()
	_, ok1 := c.Get("brand:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("brands:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetIDsByNames_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result, err := repo.GetIDsByNames(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRepository_GetIDsByNames_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Samsung").AddRow(2, "Apple")
	mock.ExpectQuery("SELECT id, name FROM brands").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	result, err := repo.GetIDsByNames(context.Background(), []string{"Samsung", "Apple"})
	require.NoError(t, err)
	assert.Equal(t, 1, result["Samsung"])
	assert.Equal(t, 2, result["Apple"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetIDsByNames_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, name FROM brands").WithArgs(pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetIDsByNames(context.Background(), []string{"X"})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllForExport_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM brands ORDER BY name").WillReturnError(fmt.Errorf("db error"))

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
	mock.ExpectQuery("INSERT INTO brands").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnRows(rows)

	repo := NewRepository(mock)
	result := repo.BulkUpsert(context.Background(), []BrandImportRow{
		{Row: 1, Name: "New", Description: "new", IsActive: true},
		{Row: 2, Name: "Existing", Description: "upd", IsActive: false},
	})
	assert.Equal(t, 1, result.Inserted)
	assert.Equal(t, 1, result.Updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db down"))

	repo := NewRepository(mock)
	result := repo.BulkUpsert(context.Background(), []BrandImportRow{
		{Row: 1, Name: "Brand", IsActive: true},
	})
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "batch upsert failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_WithCache(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("brand:1", Brand{ID: 1})
	c.Set("brands:all", []Brand{{ID: 1}})
	c.Wait()
	repo := NewRepository(mock)
	repo.SetCache(c)

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	result := repo.BulkUpsert(context.Background(), []BrandImportRow{
		{Row: 1, Name: "New", IsActive: true},
	})
	assert.Equal(t, 1, result.Inserted)

	c.Wait()
	_, ok1 := c.Get("brand:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("brands:all")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsert_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(nil)
	mock.ExpectQuery("INSERT INTO brands").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	result := repo.BulkUpsert(context.Background(), []BrandImportRow{
		{Row: 1, Name: "Brand", IsActive: true},
	})
	assert.Equal(t, 0, result.Inserted)
	assert.Equal(t, 1, result.Updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

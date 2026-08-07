package category

import (
	"context"
	"fmt"
	"testing"
	"time"

	"retail-pos-system/internal/shared"
	"retail-pos-system/pkg/cache"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubProductQueryProvider struct {
	counts map[int]int
	err    error
}

func (s stubProductQueryProvider) CountActiveByCategoryIDs(ctx context.Context, db shared.DBPool, ids []int) (map[int]int, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[int]int, len(ids))
	for id, n := range s.counts {
		out[id] = n
	}
	return out, nil
}

func TestRepository_GetCategoryByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Food", "food", "All food", true, now, now)
	mock.ExpectQuery("SELECT (.+) FROM categories WHERE id = \\$1").WithArgs(1).WillReturnRows(rows)

	repo := NewRepository(mock)
	c, err := repo.GetCategoryByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Food", c.Name)
	assert.Equal(t, "food", c.Slug)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCategoryByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM categories WHERE id = \\$1").WithArgs(999).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err = repo.GetCategoryByID(context.Background(), 999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ListCategories(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at"}).
		AddRow(1, "Drinks", "drinks", "", true, now).
		AddRow(2, "Snacks", "snacks", "", true, now)
	mock.ExpectQuery("SELECT (.+) FROM categories WHERE is_active = true").WillReturnRows(rows)

	repo := NewRepository(mock)
	cats, err := repo.ListCategories(context.Background())
	require.NoError(t, err)
	assert.Len(t, cats, 2)
	assert.Equal(t, "Drinks", cats[0].Name)
	assert.Equal(t, "Snacks", cats[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCategoryIDsByNames_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result, err := repo.GetCategoryIDsByNames(context.Background(), nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRepository_SlugExists(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery("SELECT EXISTS").WithArgs("food", 0).WillReturnRows(rows)

	repo := NewRepository(mock)
	exists, err := repo.SlugExists(context.Background(), "food", 0)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteCategory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM categories WHERE id = \\$1").WithArgs(3).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewRepository(mock)
	err = repo.DeleteCategory(context.Background(), 3)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsertCategories_Empty(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsertCategories(context.Background(), nil)
	assert.Equal(t, 0, result.Inserted)
}

func TestRepository_BulkUpsertCategories_NameRequired(t *testing.T) {
	repo := NewRepository(nil)
	result := repo.BulkUpsertCategories(context.Background(), []ImportRow{
		{Row: 1, Name: ""},
	})
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "Name is required")
}

func TestRepository_GenerateSlug(t *testing.T) {
	assert.Equal(t, "hello-world", generateSlug("Hello World"))
	assert.Equal(t, "food-and-drink", generateSlug("Food & Drink"))
	assert.Equal(t, "its-fine", generateSlug("It's fine"))
	assert.Equal(t, "a-b", generateSlug("a/b"))
}

func TestRepository_SetCache(t *testing.T) {
	repo := NewRepository(nil)
	c := cache.New(5*time.Minute, 10*time.Minute)
	repo.SetCache(c)
	assert.NotNil(t, repo.cache)
}

func TestRepository_DeleteCategory_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM categories WHERE id = \\$1").WithArgs(pgxmock.AnyArg()).WillReturnError(fmt.Errorf("fk violation"))

	repo := NewRepository(mock)
	err = repo.DeleteCategory(context.Background(), 1)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteCategory_CacheDelete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("category:1", Category{ID: 1})
	c.Set("categories:list", []Category{{ID: 1}})
	repo := NewRepository(mock)
	repo.SetCache(c)

	mock.ExpectExec("DELETE FROM categories WHERE id = \\$1").WithArgs(pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeleteCategory(context.Background(), 1)
	assert.NoError(t, err)

	_, ok1 := c.Get("category:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("categories:list")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategoriesForExport_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at"}).
		AddRow(1, "Food", "food", "All food", true, now)
	mock.ExpectQuery("SELECT id, name, COALESCE").WillReturnRows(rows)

	repo := NewRepository(mock)
	cats, err := repo.GetAllCategoriesForExport(context.Background())
	require.NoError(t, err)
	assert.Len(t, cats, 1)
	assert.Equal(t, "Food", cats[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategoriesForExport_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, name, COALESCE").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetAllCategoriesForExport(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsertCategories_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true).AddRow(false)
	mock.ExpectQuery("INSERT INTO categories").WithArgs(
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
	).WillReturnRows(rows)

	repo := NewRepository(mock)
	result := repo.BulkUpsertCategories(context.Background(), []ImportRow{
		{Row: 1, Name: "New Cat", Description: "new", IsActive: true},
		{Row: 2, Name: "Existing Cat", Description: "upd", IsActive: false},
	})
	assert.Equal(t, 1, result.Inserted)
	assert.Equal(t, 1, result.Updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsertCategories_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("INSERT INTO categories").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db down"))

	repo := NewRepository(mock)
	result := repo.BulkUpsertCategories(context.Background(), []ImportRow{
		{Row: 1, Name: "Cat", IsActive: true},
	})
	assert.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "batch upsert failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsertCategories_ScanError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(nil)
	mock.ExpectQuery("INSERT INTO categories").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	result := repo.BulkUpsertCategories(context.Background(), []ImportRow{
		{Row: 1, Name: "Cat", IsActive: true},
	})
	assert.Equal(t, 0, result.Inserted)
	assert.Equal(t, 1, result.Updated)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_BulkUpsertCategories_WithCache(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	c.Set("category:1", Category{ID: 1})
	c.Set("categories:list", []Category{{ID: 1}})
	repo := NewRepository(mock)
	repo.SetCache(c)

	rows := pgxmock.NewRows([]string{"is_insert"}).AddRow(true)
	mock.ExpectQuery("INSERT INTO categories").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(rows)

	result := repo.BulkUpsertCategories(context.Background(), []ImportRow{
		{Row: 1, Name: "New Cat", IsActive: true},
	})
	assert.Equal(t, 1, result.Inserted)

	_, ok1 := c.Get("category:1")
	assert.False(t, ok1)
	_, ok2 := c.Get("categories:list")
	assert.False(t, ok2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCategoryIDsByNames_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Food").AddRow(2, "Drinks")
	mock.ExpectQuery("SELECT id, name FROM categories").WithArgs(pgxmock.AnyArg()).WillReturnRows(rows)

	repo := NewRepository(mock)
	result, err := repo.GetCategoryIDsByNames(context.Background(), []string{"Food", "Drinks"})
	require.NoError(t, err)
	assert.Equal(t, 1, result["Food"])
	assert.Equal(t, 2, result["Drinks"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCategoryIDsByNames_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT id, name FROM categories").WithArgs(pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.GetCategoryIDsByNames(context.Background(), []string{"Food"})
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ListCategories_CacheHit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	c := cache.New(5*time.Minute, 10*time.Minute)
	repo := NewRepository(mock)
	repo.SetCache(c)

	cats := []Category{{ID: 1, Name: "Cached"}}
	c.Set("categories:list", cats)
	c.Wait()

	result, err := repo.ListCategories(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Cached", result[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_ListCategories_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM categories WHERE is_active = true").WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.ListCategories(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetCategoryByID_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT (.+) FROM categories WHERE id = \\$1").WithArgs(1).WillReturnError(fmt.Errorf("db lost"))

	repo := NewRepository(mock)
	_, err = repo.GetCategoryByID(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db lost")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_SlugExists_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	_, err = repo.SlugExists(context.Background(), "food", 0)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	now := time.Now()
	dataRows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Food", "food", "All food", true, now, now)
	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(dataRows)

	repo := NewRepository(mock)
	repo.SetProductQueryProvider(stubProductQueryProvider{counts: map[int]int{1: 5}})
	cats, total, err := repo.GetAllCategories(context.Background(), 10, 0, "")
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, cats, 1)
	assert.Equal(t, "Food", cats[0].Name)
	assert.Equal(t, 5, cats[0].ProductCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_WithSearch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT COUNT").WithArgs("%food%").WillReturnRows(countRows)

	dataRows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at", "updated_at"})
	mock.ExpectQuery("SELECT c.id, c.name").WithArgs("%food%", 10, 0).WillReturnRows(dataRows)

	repo := NewRepository(mock)
	repo.SetProductQueryProvider(stubProductQueryProvider{counts: map[int]int{}})
	_, _, err = repo.GetAllCategories(context.Background(), 10, 0, "food")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_ProductCountError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	now := time.Now()
	dataRows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Food", "food", "All food", true, now, now)
	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(dataRows)

	repo := NewRepository(mock)
	repo.SetProductQueryProvider(stubProductQueryProvider{err: fmt.Errorf("count error")})
	_, _, err = repo.GetAllCategories(context.Background(), 10, 0, "")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_NotWired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)

	now := time.Now()
	dataRows := pgxmock.NewRows([]string{"id", "name", "slug", "description", "is_active", "created_at", "updated_at"}).
		AddRow(1, "Food", "food", "All food", true, now, now)
	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(dataRows)

	repo := NewRepository(mock)
	_, _, err = repo.GetAllCategories(context.Background(), 10, 0, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not wired")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_CountError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT COUNT").WillReturnError(fmt.Errorf("count error"))

	repo := NewRepository(mock)
	_, _, err = repo.GetAllCategories(context.Background(), 10, 0, "")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetAllCategories_QueryError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	countRows := pgxmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(countRows)
	mock.ExpectQuery("SELECT c.id, c.name").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("query error"))

	repo := NewRepository(mock)
	_, _, err = repo.GetAllCategories(context.Background(), 10, 0, "")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateCategory_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs("food", 0).WillReturnRows(
		pgxmock.NewRows([]string{"exists"}).AddRow(false))

	now := time.Now()
	mock.ExpectQuery("INSERT INTO categories").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(10, now, now))

	repo := NewRepository(mock)
	cat := &Category{Name: "Food", Description: "All food", IsActive: true}
	err = repo.CreateCategory(context.Background(), cat)
	require.NoError(t, err)
	assert.Equal(t, 10, cat.ID)
	assert.Equal(t, "food", cat.Slug)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateCategory_SlugCollision(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs("food", 0).WillReturnRows(
		pgxmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS").WithArgs("food-2", 0).WillReturnRows(
		pgxmock.NewRows([]string{"exists"}).AddRow(false))

	now := time.Now()
	mock.ExpectQuery("INSERT INTO categories").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(10, now, now))

	repo := NewRepository(mock)
	cat := &Category{Name: "Food", Description: "", IsActive: true}
	err = repo.CreateCategory(context.Background(), cat)
	require.NoError(t, err)
	assert.Equal(t, "food-2", cat.Slug)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateCategory_SlugCheckError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	cat := &Category{Name: "Food", IsActive: true}
	err = repo.CreateCategory(context.Background(), cat)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateCategory_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(
		pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("INSERT INTO categories").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	cat := &Category{Name: "Food", IsActive: true}
	err = repo.CreateCategory(context.Background(), cat)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateCategory_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE categories SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	cat := &Category{ID: 1, Name: "Food", Slug: "food", Description: "updated", IsActive: true}
	err = repo.UpdateCategory(context.Background(), cat)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateCategory_SlugChanged(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs("new-name", 1).WillReturnRows(
		pgxmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec("UPDATE categories SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	cat := &Category{ID: 1, Name: "New Name", Slug: "old-name", IsActive: true}
	err = repo.UpdateCategory(context.Background(), cat)
	assert.NoError(t, err)
	assert.Equal(t, "new-name", cat.Slug)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateCategory_SlugCheckError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT EXISTS").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("db error"))

	repo := NewRepository(mock)
	cat := &Category{ID: 1, Name: "New", Slug: "old", IsActive: true}
	err = repo.UpdateCategory(context.Background(), cat)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateCategory_ExecError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE categories SET").WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(fmt.Errorf("update failed"))

	repo := NewRepository(mock)
	cat := &Category{ID: 1, Name: "Food", Slug: "food", IsActive: true}
	err = repo.UpdateCategory(context.Background(), cat)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

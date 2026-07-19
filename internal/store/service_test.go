package store

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/shared"
)

func TestService_GetAll(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc List Store", Address: "Addr", Phone: "111", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	stores, total, err := svc.GetAll(ctx, 10, 0, "", nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(stores), 1)
}

func TestService_GetAll_Search(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Store{Name: "UniqueAlphaStore", IsActive: true}))

	stores, total, err := svc.GetAll(ctx, 10, 0, "UniqueAlpha", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "UniqueAlphaStore", stores[0].Name)
}

func TestService_GetAll_IsActiveFilter(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Store{Name: "Active Filtered", IsActive: true}))
	require.NoError(t, repo.Create(ctx, &Store{Name: "Inactive Filtered", IsActive: false}))

	tf := true
	stores, _, err := svc.GetAll(ctx, 10, 0, "", &tf)
	require.NoError(t, err)
	for _, s := range stores {
		assert.True(t, s.IsActive)
	}

	ff := false
	stores2, _, err := svc.GetAll(ctx, 10, 0, "", &ff)
	require.NoError(t, err)
	for _, s := range stores2 {
		assert.False(t, s.IsActive)
	}
}

func TestService_GetByID(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc GetByID", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	fetched, err := svc.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Svc GetByID", fetched.Name)
}

func TestService_GetByID_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, 999999)
	assert.Error(t, err)
}

func TestService_GetAllActive(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &Store{Name: "Svc Active A", IsActive: true}))
	require.NoError(t, repo.Create(ctx, &Store{Name: "Svc Inactive B", IsActive: false}))

	stores, err := svc.GetAllActive(ctx)
	require.NoError(t, err)
	for _, s := range stores {
		assert.True(t, s.IsActive)
	}
	assert.GreaterOrEqual(t, len(stores), 1)
}

func TestService_Create(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	st, err := svc.Create(ctx, StoreCreateRequest{
		Name:    "Svc Create Store",
		Address: "Svc Addr",
		Phone:   "123",
	})
	require.NoError(t, err)
	assert.Greater(t, st.ID, 0)
	assert.Equal(t, "Svc Create Store", st.Name)
	assert.Equal(t, "Svc Addr", st.Address)
	assert.True(t, st.IsActive)
}

func TestService_Create_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, StoreCreateRequest{Name: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestService_Create_WhitespaceName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, StoreCreateRequest{Name: "   "})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestService_Update(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc Update Me", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	newName := "Svc Updated"
	newAddr := "New Addr"
	updated, err := svc.Update(ctx, s.ID, StoreUpdateRequest{
		Name:    &newName,
		Address: &newAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, "Svc Updated", updated.Name)
	assert.Equal(t, "New Addr", updated.Address)
}

func TestService_Update_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	newName := "nope"
	_, err := svc.Update(ctx, 999999, StoreUpdateRequest{Name: &newName})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store not found")
}

func TestService_Update_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc Update Empty", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	emptyName := ""
	_, err := svc.Update(ctx, s.ID, StoreUpdateRequest{Name: &emptyName})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestService_Delete(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc Delete Me", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	err := svc.Delete(ctx, s.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, s.ID)
	assert.Error(t, err)
}

func TestService_Delete_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	err := svc.Delete(ctx, 999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store not found")
}

func TestService_Update_IsActive(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc Update Active", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	ff := false
	updated, err := svc.Update(ctx, s.ID, StoreUpdateRequest{IsActive: &ff})
	require.NoError(t, err)
	assert.False(t, updated.IsActive)

	tt := true
	updated2, err := svc.Update(ctx, s.ID, StoreUpdateRequest{IsActive: &tt})
	require.NoError(t, err)
	assert.True(t, updated2.IsActive)
}

func TestService_Create_TrimmedFields(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	st, err := svc.Create(ctx, StoreCreateRequest{
		Name:    "  Trimmed Name  ",
		Address: "  Trimmed Addr  ",
		Phone:   "  999  ",
	})
	require.NoError(t, err)
	assert.Equal(t, "Trimmed Name", st.Name)
	assert.Equal(t, "Trimmed Addr", st.Address)
	assert.Equal(t, "999", st.Phone)
}

func TestService_Update_TrimmedFields(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc Trim Test", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	newName := "  Trimmed Update  "
	newAddr := "  Trimmed Addr  "
	updated, err := svc.Update(ctx, s.ID, StoreUpdateRequest{
		Name:    &newName,
		Address: &newAddr,
	})
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace("  Trimmed Update  "), updated.Name)
	assert.Equal(t, strings.TrimSpace("  Trimmed Addr  "), updated.Address)
}

func TestService_Update_PhoneOnly(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Store{Name: "Svc Phone Update", Address: "Keep Addr", IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	phone := "555"
	updated, err := svc.Update(ctx, s.ID, StoreUpdateRequest{Phone: &phone})
	require.NoError(t, err)
	assert.Equal(t, "555", updated.Phone)
	assert.Equal(t, "Keep Addr", updated.Address)
}

func TestService_GetAll_Empty(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	stores, total, err := svc.GetAll(ctx, 10, 0, "", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.NotNil(t, stores)
}

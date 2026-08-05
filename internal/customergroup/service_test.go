package customergroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_GetAll(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "SvcGetAll-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))
	defer func() { _ = repo.Delete(ctx, cg.ID) }()

	groups, total, err := svc.GetAll(ctx, 10, 0, "", nil, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.GreaterOrEqual(t, len(groups), 1)
}

func TestService_GetByID_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "SvcGetByID-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))
	defer func() { _ = repo.Delete(ctx, cg.ID) }()

	got, err := svc.GetByID(ctx, cg.ID)
	require.NoError(t, err)
	assert.Equal(t, cg.Name, got.Name)
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
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "SvcActive-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))
	defer func() { _ = repo.Delete(ctx, cg.ID) }()

	groups, err := svc.GetAllActive(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(groups), 1)
}

func TestService_Create_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	name := "SvcCreate-" + t.Name()
	result, err := svc.Create(ctx, CustomerGroupCreateRequest{Name: name, Description: "test"})
	require.NoError(t, err)
	assert.Equal(t, name, result.Name)
	assert.Greater(t, result.ID, 0)
	defer func() { _ = repo.Delete(ctx, result.ID) }()
}

func TestService_Create_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.Create(ctx, CustomerGroupCreateRequest{Name: "  ", Description: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestService_Create_DuplicateName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	name := "SvcDup-" + t.Name()
	result, err := svc.Create(ctx, CustomerGroupCreateRequest{Name: name})
	require.NoError(t, err)
	defer func() { _ = repo.Delete(ctx, result.ID) }()

	_, err = svc.Create(ctx, CustomerGroupCreateRequest{Name: name})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_Update_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "SvcUpd-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))
	defer func() { _ = repo.Delete(ctx, cg.ID) }()

	newName := "SvcUpd-Updated-" + t.Name()
	updated, err := svc.Update(ctx, cg.ID, CustomerGroupUpdateRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
}

func TestService_Update_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	name := "nope"
	_, err := svc.Update(ctx, 999999, CustomerGroupUpdateRequest{Name: &name})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_Update_EmptyName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "SvcUpdEmpty-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))
	defer func() { _ = repo.Delete(ctx, cg.ID) }()

	empty := "  "
	_, err := svc.Update(ctx, cg.ID, CustomerGroupUpdateRequest{Name: &empty})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestService_Update_DuplicateName(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg1 := &CustomerGroup{Name: "SvcUpdDup1-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg1))
	defer func() { _ = repo.Delete(ctx, cg1.ID) }()

	cg2 := &CustomerGroup{Name: "SvcUpdDup2-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg2))
	defer func() { _ = repo.Delete(ctx, cg2.ID) }()

	_, err := svc.Update(ctx, cg1.ID, CustomerGroupUpdateRequest{Name: &cg2.Name})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestService_Delete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg := &CustomerGroup{Name: "SvcDel-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg))

	err := svc.Delete(ctx, cg.ID)
	assert.NoError(t, err)

	_, err = repo.GetByID(ctx, cg.ID)
	assert.Error(t, err)
}

func TestService_Delete_NotFound(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	err := svc.Delete(ctx, 999999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_BulkUpdate_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg1 := &CustomerGroup{Name: "SvcBulkUpd1-" + t.Name(), IsActive: true}
	cg2 := &CustomerGroup{Name: "SvcBulkUpd2-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg1))
	require.NoError(t, repo.Create(ctx, cg2))
	defer func() { _ = repo.Delete(ctx, cg1.ID) }()
	defer func() { _ = repo.Delete(ctx, cg2.ID) }()

	updated, err := svc.BulkUpdate(ctx, []int{cg1.ID, cg2.ID}, false)
	require.NoError(t, err)
	assert.Equal(t, 2, updated)
}

func TestService_BulkUpdate_EmptyIDs(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.BulkUpdate(ctx, []int{}, true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no IDs provided")
}

func TestService_BulkDelete_Success(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	cg1 := &CustomerGroup{Name: "SvcBulkDel1-" + t.Name(), IsActive: true}
	cg2 := &CustomerGroup{Name: "SvcBulkDel2-" + t.Name(), IsActive: true}
	require.NoError(t, repo.Create(ctx, cg1))
	require.NoError(t, repo.Create(ctx, cg2))

	deleted, err := svc.BulkDelete(ctx, []int{cg1.ID, cg2.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, deleted)
}

func TestService_BulkDelete_EmptyIDs(t *testing.T) {
	skipIfNoDB(t)
	repo := NewRepository(dbPool)
	svc := NewService(repo)
	ctx := context.Background()

	_, err := svc.BulkDelete(ctx, []int{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no IDs provided")
}

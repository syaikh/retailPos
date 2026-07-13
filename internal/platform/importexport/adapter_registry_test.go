package importexport

import (
	"context"
	"testing"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAdapter struct {
	module string
}

func (a *stubAdapter) ModuleName() string { return a.module }
func (a *stubAdapter) ValidateBusiness(_ context.Context, _ importexportshared.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}
func (a *stubAdapter) MapToEntity(_ context.Context, _ importexportshared.ModuleSchema, _ map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (a *stubAdapter) Repository() importexportshared.RepositoryActions { return nil }

func TestAdapterRegistry_RegisterAndGet(t *testing.T) {
	r := NewAdapterRegistry()
	a := &stubAdapter{module: "products"}
	require.NoError(t, r.Register(a))

	got, err := r.Get("products")
	require.NoError(t, err)
	assert.Equal(t, a, got)
}

func TestAdapterRegistry_RegisterEmptyName(t *testing.T) {
	r := NewAdapterRegistry()
	err := r.Register(&stubAdapter{module: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module_name is required")
}

func TestAdapterRegistry_RegisterDuplicate(t *testing.T) {
	r := NewAdapterRegistry()
	require.NoError(t, r.Register(&stubAdapter{module: "products"}))
	err := r.Register(&stubAdapter{module: "products"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestAdapterRegistry_GetNotFound(t *testing.T) {
	r := NewAdapterRegistry()
	_, err := r.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no adapter registered")
}

func TestAdapterRegistry_Modules(t *testing.T) {
	r := NewAdapterRegistry()
	require.NoError(t, r.Register(&stubAdapter{module: "products"}))
	require.NoError(t, r.Register(&stubAdapter{module: "categories"}))
	require.NoError(t, r.Register(&stubAdapter{module: "brands"}))

	modules := r.Modules()
	assert.Len(t, modules, 3)
	assert.Contains(t, modules, "products")
	assert.Contains(t, modules, "categories")
	assert.Contains(t, modules, "brands")
}

func TestAdapterRegistry_ModulesEmpty(t *testing.T) {
	r := NewAdapterRegistry()
	modules := r.Modules()
	assert.Empty(t, modules)
}

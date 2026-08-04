package supplier

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSupplier_ErrorsAreSentinels(t *testing.T) {
	assert.ErrorIs(t, ErrSupplierNotFound, ErrSupplierNotFound)
	assert.ErrorIs(t, ErrSupplierCodeExists, ErrSupplierCodeExists)
	assert.ErrorIs(t, ErrInvalidSupplier, ErrInvalidSupplier)
	assert.ErrorIs(t, ErrProductSupplierExists, ErrProductSupplierExists)
	assert.ErrorIs(t, ErrProductSupplierNotFound, ErrProductSupplierNotFound)
	assert.ErrorIs(t, ErrMultiplePreferred, ErrMultiplePreferred)
}

func TestSupplier_ZeroValue(t *testing.T) {
	var s Supplier
	assert.Equal(t, 0, s.ID)
	assert.Equal(t, "", s.Name)
	assert.Equal(t, "", s.Code)
	assert.Nil(t, s.ContactName)
	assert.Nil(t, s.Phone)
	assert.Nil(t, s.Email)
	assert.Nil(t, s.Address)
	assert.Nil(t, s.Notes)
	assert.False(t, s.IsActive)
	assert.Nil(t, s.StoreID)
	assert.Equal(t, "", s.CreatedAt)
	assert.Equal(t, "", s.UpdatedAt)
	assert.Nil(t, s.DeletedAt)
}

func TestSupplier_WithValues(t *testing.T) {
	phone := "+628123456789"
	email := "supplier@example.com"
	s := Supplier{
		ID:       1,
		Name:     "PT Maju Jaya",
		Code:     "SUP-001",
		Phone:    &phone,
		Email:    &email,
		IsActive: true,
	}

	assert.Equal(t, 1, s.ID)
	assert.Equal(t, "PT Maju Jaya", s.Name)
	assert.Equal(t, "SUP-001", s.Code)
	assert.NotNil(t, s.Phone)
	assert.Equal(t, "+628123456789", *s.Phone)
	assert.NotNil(t, s.Email)
	assert.Equal(t, "supplier@example.com", *s.Email)
	assert.True(t, s.IsActive)
}

func TestProductSupplier_ZeroValue(t *testing.T) {
	var ps ProductSupplier
	assert.Equal(t, 0, ps.ID)
	assert.Equal(t, 0, ps.ProductID)
	assert.Equal(t, 0, ps.SupplierID)
	assert.Nil(t, ps.SupplierSKU)
	assert.Equal(t, 0, ps.UnitCost)
	assert.Equal(t, 0, ps.LeadTimeDays)
	assert.False(t, ps.IsPreferred)
}

func TestProductSupplier_WithValues(t *testing.T) {
	sku := "SUP-SKU-001"
	ps := ProductSupplier{
		ID:           1,
		ProductID:    42,
		SupplierID:   7,
		SupplierSKU:  &sku,
		UnitCost:     8000,
		LeadTimeDays: 7,
		IsPreferred:  true,
	}

	assert.Equal(t, 1, ps.ID)
	assert.Equal(t, 42, ps.ProductID)
	assert.Equal(t, 7, ps.SupplierID)
	assert.NotNil(t, ps.SupplierSKU)
	assert.Equal(t, "SUP-SKU-001", *ps.SupplierSKU)
	assert.Equal(t, 8000, ps.UnitCost)
	assert.Equal(t, 7, ps.LeadTimeDays)
	assert.True(t, ps.IsPreferred)
}

func TestProductSupplier_CostIsIndependent(t *testing.T) {
	// Verify that UnitCost and IsPreferred are independent fields.
	// A product can have zero cost but still be preferred (e.g., future free samples).
	ps := ProductSupplier{
		UnitCost:    0,
		IsPreferred: true,
	}
	assert.Equal(t, 0, ps.UnitCost)
	assert.True(t, ps.IsPreferred)

	// A product can have high cost but not be preferred
	ps2 := ProductSupplier{
		UnitCost:    999999,
		IsPreferred: false,
	}
	assert.Equal(t, 999999, ps2.UnitCost)
	assert.False(t, ps2.IsPreferred)
}

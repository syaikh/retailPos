package supplier

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSupplier(t *testing.T) {
	tests := []struct {
		name      string
		supplier  *Supplier
		expectErr bool
	}{
		{
			name: "valid supplier",
			supplier: &Supplier{
				Name: "PT Maju Jaya",
				Code: "SUP-001",
			},
			expectErr: false,
		},
		{
			name: "empty name",
			supplier: &Supplier{
				Name: "",
				Code: "SUP-001",
			},
			expectErr: true,
		},
		{
			name: "empty code",
			supplier: &Supplier{
				Name: "PT Maju Jaya",
				Code: "",
			},
			expectErr: true,
		},
		{
			name: "both empty",
			supplier: &Supplier{
				Name: "",
				Code: "",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSupplier(tt.supplier)
			if tt.expectErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidSupplier))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProductSupplier(t *testing.T) {
	tests := []struct {
		name      string
		ps        *ProductSupplier
		expectErr bool
	}{
		{
			name: "valid product supplier",
			ps: &ProductSupplier{
				ProductID:  1,
				SupplierID: 1,
				UnitCost:   8000,
			},
			expectErr: false,
		},
		{
			name: "valid with zero cost",
			ps: &ProductSupplier{
				ProductID:  1,
				SupplierID: 1,
				UnitCost:   0,
			},
			expectErr: false,
		},
		{
			name: "zero product_id",
			ps: &ProductSupplier{
				ProductID:  0,
				SupplierID: 1,
				UnitCost:   8000,
			},
			expectErr: true,
		},
		{
			name: "negative product_id",
			ps: &ProductSupplier{
				ProductID:  -1,
				SupplierID: 1,
				UnitCost:   8000,
			},
			expectErr: true,
		},
		{
			name: "zero supplier_id",
			ps: &ProductSupplier{
				ProductID:  1,
				SupplierID: 0,
				UnitCost:   8000,
			},
			expectErr: true,
		},
		{
			name: "negative supplier_id",
			ps: &ProductSupplier{
				ProductID:  1,
				SupplierID: -1,
				UnitCost:   8000,
			},
			expectErr: true,
		},
		{
			name: "negative unit_cost",
			ps: &ProductSupplier{
				ProductID:  1,
				SupplierID: 1,
				UnitCost:   -1,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProductSupplier(tt.ps)
			if tt.expectErr {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, ErrInvalidSupplier))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

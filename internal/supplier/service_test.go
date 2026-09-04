package supplier

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name: "valid supplier with email",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Email: strPtr("contact@majujaya.co.id"),
			},
			expectErr: false,
		},
		{
			name: "valid supplier with phone",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Phone: strPtr("+62812345678"),
			},
			expectErr: false,
		},
		{
			name: "valid supplier with formatted phone",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Phone: strPtr("(021) 555-1234"),
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
		{
			name: "invalid email format",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Email: strPtr("not-an-email"),
			},
			expectErr: true,
		},
		{
			name: "invalid email missing @",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Email: strPtr("userexample.com"),
			},
			expectErr: true,
		},
		{
			name: "empty email is valid",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Email: strPtr(""),
			},
			expectErr: false,
		},
		{
			name: "invalid phone too short",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Phone: strPtr("123"),
			},
			expectErr: true,
		},
		{
			name: "invalid phone letters",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Phone: strPtr("+62abc123"),
			},
			expectErr: true,
		},
		{
			name: "empty phone is valid",
			supplier: &Supplier{
				Name:  "PT Maju Jaya",
				Code:  "SUP-001",
				Phone: strPtr(""),
			},
			expectErr: false,
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

func TestService_GetByID(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC GetByID",
		Code:     "SVC-GID-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	got, err := svc.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.Name, got.Name)

	_, err = svc.GetByID(ctx, -1)
	assert.Error(t, err)
}

func TestService_GetByCode(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	code := "SVC-GBC-" + time.Now().Format("0102150405")
	s := &Supplier{Name: "SVC GetByCode", Code: code, IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	got, err := svc.GetByCode(ctx, code)
	require.NoError(t, err)
	assert.Equal(t, s.Name, got.Name)

	_, err = svc.GetByCode(ctx, "NONEXISTENT")
	assert.Error(t, err)
}

func TestService_GetAll(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{Name: "SVC-GetAll", Code: "SUP-SVC-GA-" + time.Now().Format("0102150405"), IsActive: true}
	require.NoError(t, repo.Create(ctx, s))

	suppliers, total, err := svc.GetAll(ctx, 10, 0, "", nil, nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
	assert.NotEmpty(t, suppliers)
}

func TestService_Create_And_Delete(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC Create",
		Code:     "SVC-CR-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	err := svc.Create(ctx, s)
	require.NoError(t, err)
	assert.Greater(t, s.ID, 0)

	err = svc.Delete(ctx, s.ID)
	require.NoError(t, err)
}

func TestService_Create_ValidationFails(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{Name: "", Code: ""}
	err := svc.Create(ctx, s)
	assert.Error(t, err)
}

func TestService_Create_AutoGeneratesCode(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{Name: "SVC Auto Code", IsActive: true}
	err := svc.Create(ctx, s)
	require.NoError(t, err)
	assert.Greater(t, s.ID, 0)
	assert.Regexp(t, `^SUP-\d+$`, s.Code, "blank code should be auto-generated as SUP-%06d")

	got, err := svc.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.Code, got.Code)
}

func TestService_Create_KeepsProvidedCode(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	code := "SVC-KEEP-" + time.Now().Format("0102150405")
	s := &Supplier{Name: "SVC Keep Code", Code: code, IsActive: true}
	err := svc.Create(ctx, s)
	require.NoError(t, err)
	assert.Equal(t, code, s.Code, "provided code must be preserved, not overwritten")
}

func TestService_Update(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC Before Update",
		Code:     "SVC-UPD-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	s.Name = "SVC After Update"
	err := svc.Update(ctx, s)
	require.NoError(t, err)

	got, err := svc.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "SVC After Update", got.Name)
}

func TestService_Update_ValidationFails(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{ID: 1, Name: "", Code: ""}
	err := svc.Update(ctx, s)
	assert.Error(t, err)
}

func TestService_LinkProduct(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC Link Supplier",
		Code:     "SVC-LNK-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "SVC-LP-"+time.Now().Format("0102150405"), "SVC Link Product", 5000)

	ps := &ProductSupplier{
		ProductID:  productID,
		SupplierID: s.ID,
		UnitCost:   4000,
	}
	err := svc.LinkProduct(ctx, ps)
	require.NoError(t, err)
	assert.Greater(t, ps.ID, 0)

	err = svc.UnlinkProduct(ctx, productID, s.ID)
	require.NoError(t, err)
}

func TestService_LinkProduct_ValidationFails(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	ps := &ProductSupplier{ProductID: 0, SupplierID: 0, UnitCost: 0}
	err := svc.LinkProduct(ctx, ps)
	assert.Error(t, err)
}

func TestService_GetProductSupplier(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC GetPS Supplier",
		Code:     "SVC-GPS-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "SVC-GPS-P-"+time.Now().Format("0102150405"), "SVC GetPS Product", 6000)

	ps := &ProductSupplier{
		ProductID:  productID,
		SupplierID: s.ID,
		UnitCost:   3000,
	}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	got, err := svc.GetProductSupplier(ctx, productID, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 3000, got.UnitCost)

	_, err = svc.GetProductSupplier(ctx, -1, -1)
	assert.Error(t, err)

	require.NoError(t, svc.UnlinkProduct(ctx, productID, s.ID))
}

func TestService_GetPreferredSupplier(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC Pref Supplier",
		Code:     "SVC-PREF-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "SVC-PREF-P-"+time.Now().Format("0102150405"), "SVC Pref Product", 7000)

	ps := &ProductSupplier{
		ProductID:   productID,
		SupplierID:  s.ID,
		UnitCost:    5000,
		IsPreferred: true,
	}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	got, err := svc.GetPreferredSupplier(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, got.SupplierID)

	require.NoError(t, svc.UnlinkProduct(ctx, productID, s.ID))
}

func TestService_SetPreferredSupplier(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s1 := &Supplier{
		Name:     "SVC SetPref1",
		Code:     "SVC-SP1-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	s2 := &Supplier{
		Name:     "SVC SetPref2",
		Code:     "SVC-SP2-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	productID := insertTestProduct(ctx, t, "SVC-SP-P-"+time.Now().Format("0102150405"), "SVC SetPref Product", 8000)

	ps1 := &ProductSupplier{ProductID: productID, SupplierID: s1.ID, UnitCost: 5000, IsPreferred: true}
	ps2 := &ProductSupplier{ProductID: productID, SupplierID: s2.ID, UnitCost: 6000, IsPreferred: false}
	require.NoError(t, repo.LinkProduct(ctx, ps1))
	require.NoError(t, repo.LinkProduct(ctx, ps2))

	err := svc.SetPreferredSupplier(ctx, productID, s2.ID)
	require.NoError(t, err)

	got, err := svc.GetPreferredSupplier(ctx, productID)
	require.NoError(t, err)
	assert.Equal(t, s2.ID, got.SupplierID)

	require.NoError(t, svc.UnlinkProduct(ctx, productID, s1.ID))
	require.NoError(t, svc.UnlinkProduct(ctx, productID, s2.ID))
}

func TestService_UpdateProductSupplier(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC UpdPS Supplier",
		Code:     "SVC-UPS-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "SVC-UPS-P-"+time.Now().Format("0102150405"), "SVC UpdPS Product", 9000)

	ps := &ProductSupplier{ProductID: productID, SupplierID: s.ID, UnitCost: 4000}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	ps.UnitCost = 5500
	ps.LeadTimeDays = 10
	err := svc.UpdateProductSupplier(ctx, ps)
	require.NoError(t, err)

	got, err := svc.GetProductSupplier(ctx, productID, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 5500, got.UnitCost)
	assert.Equal(t, 10, got.LeadTimeDays)

	require.NoError(t, svc.UnlinkProduct(ctx, productID, s.ID))
}

func TestService_UpdateProductSupplier_ValidationFails(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	ps := &ProductSupplier{ProductID: 0, SupplierID: 0, UnitCost: -1}
	err := svc.UpdateProductSupplier(ctx, ps)
	assert.Error(t, err)
}

func TestService_GetSuppliersByProductID(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC ByProd Supplier",
		Code:     "SVC-BP-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "SVC-BP-P-"+time.Now().Format("0102150405"), "SVC ByProd Product", 10000)

	ps := &ProductSupplier{ProductID: productID, SupplierID: s.ID, UnitCost: 7000}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	suppliers, err := svc.GetSuppliersByProductID(ctx, productID)
	require.NoError(t, err)
	assert.NotEmpty(t, suppliers)

	require.NoError(t, svc.UnlinkProduct(ctx, productID, s.ID))
}

func TestService_GetProductsBySupplierID(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s := &Supplier{
		Name:     "SVC BySup Supplier",
		Code:     "SVC-BS-" + time.Now().Format("0102150405"),
		IsActive: true,
	}
	require.NoError(t, repo.Create(ctx, s))

	productID := insertTestProduct(ctx, t, "SVC-BS-P-"+time.Now().Format("0102150405"), "SVC BySup Product", 11000)

	ps := &ProductSupplier{ProductID: productID, SupplierID: s.ID, UnitCost: 8000}
	require.NoError(t, repo.LinkProduct(ctx, ps))

	products, err := svc.GetProductsBySupplierID(ctx, s.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, products)

	require.NoError(t, svc.UnlinkProduct(ctx, productID, s.ID))
}

func TestService_BulkUpdate(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s1 := &Supplier{Name: "SVC BU1", Code: "SVC-BU1-" + time.Now().Format("0102150405"), IsActive: true}
	s2 := &Supplier{Name: "SVC BU2", Code: "SVC-BU2-" + time.Now().Format("0102150405"), IsActive: true}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	count, err := svc.BulkUpdate(ctx, []int{s1.ID, s2.ID}, false)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	count, err = svc.BulkUpdate(ctx, []int{s1.ID, s2.ID}, true)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestService_BulkDelete(t *testing.T) {
	skipIfNoDB(t)
	repo := newTestRepo(t)
	svc := NewService(repo)
	ctx := context.Background()

	s1 := &Supplier{Name: "SVC BD1", Code: "SVC-BD1-" + time.Now().Format("0102150405"), IsActive: true}
	s2 := &Supplier{Name: "SVC BD2", Code: "SVC-BD2-" + time.Now().Format("0102150405"), IsActive: true}
	require.NoError(t, repo.Create(ctx, s1))
	require.NoError(t, repo.Create(ctx, s2))

	count, err := svc.BulkDelete(ctx, []int{s1.ID, s2.ID})
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = svc.GetByID(ctx, s1.ID)
	assert.Error(t, err)
}

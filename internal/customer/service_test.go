package customer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomerService_CreateCustomerNilError(t *testing.T) {
	svc := NewService(NewRepository(dbPool))
	ctx := context.Background()

	err := svc.CreateCustomer(ctx, nil, nil)
	assert.ErrorContains(t, err, "customer cannot be nil")
}

func TestCustomerService_UpdateCustomerNilError(t *testing.T) {
	svc := NewService(NewRepository(dbPool))
	ctx := context.Background()

	err := svc.UpdateCustomer(ctx, nil, 1, nil)
	assert.ErrorContains(t, err, "customer cannot be nil")
}

func TestCustomerService_ReadOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	phone := "087777777774"
	c := &Customer{
		Name:     "Service Read Test",
		Phone:    &phone,
		Email: ptr("test@example.com"),
		IsActive: true,
	}
	require.NoError(t, svc.CreateCustomer(ctx, c, nil))

	t.Run("GetByPhone", func(t *testing.T) {
		got, err := svc.GetByPhone(ctx, phone, nil)
		require.NoError(t, err)
		assert.Equal(t, c.Name, got.Name)
	})

	t.Run("GetCustomerByID", func(t *testing.T) {
		got, err := svc.GetCustomerByID(ctx, c.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, c.Name, got.Name)
	})

	t.Run("GetAllCustomers", func(t *testing.T) {
		customers, total, err := svc.GetAllCustomers(ctx, 10, 0, "", nil, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, 1)
		assert.GreaterOrEqual(t, len(customers), 1)
	})
}

func TestCustomerService_BulkOperations(t *testing.T) {
	repo := NewRepository(dbPool)

	svc := NewService(repo)
	ctx := context.Background()

	phone1 := "087777777775"
	phone2 := "087777777776"
	c1 := &Customer{Name: "Svc Bulk 1", Phone: &phone1, Email: ptr("test@example.com"), IsActive: true}
	c2 := &Customer{Name: "Svc Bulk 2", Phone: &phone2, Email: ptr("test@example.com"), IsActive: true}
	require.NoError(t, svc.CreateCustomer(ctx, c1, nil))
	require.NoError(t, svc.CreateCustomer(ctx, c2, nil))

	t.Run("BulkUpdateCustomersStatus", func(t *testing.T) {
		err := svc.BulkUpdateCustomersStatus(ctx, []int{c1.ID, c2.ID}, false, nil)
		require.NoError(t, err)

		got1, _ := repo.GetCustomerByID(ctx, c1.ID, nil)
		got2, _ := repo.GetCustomerByID(ctx, c2.ID, nil)
		assert.False(t, got1.IsActive)
		assert.False(t, got2.IsActive)
	})

	t.Run("BulkDeleteCustomers", func(t *testing.T) {
		phone3 := "087777777777"
		c3 := &Customer{Name: "Svc Bulk 3", Phone: &phone3, Email: ptr("test@example.com"), IsActive: true}
		require.NoError(t, svc.CreateCustomer(ctx, c3, nil))

		err := svc.BulkDeleteCustomers(ctx, []int{c3.ID}, nil)
		require.NoError(t, err)

		got, _ := repo.GetCustomerByID(ctx, c3.ID, nil)
		assert.False(t, got.IsActive)
	})
}

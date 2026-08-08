package customer

import (
	"context"
	"errors"
	"strings"
)

type Repo interface {
	GetByPhone(ctx context.Context, phone string, storeID *int) (*Customer, error)
	GetCustomerByID(ctx context.Context, id int, storeID *int) (*Customer, error)
	GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int, customerGroupID *int) ([]Customer, int, error)
	CreateCustomer(ctx context.Context, customer *Customer) error
	UpdateCustomer(ctx context.Context, customer *Customer, id int, storeID *int) error
	DeleteCustomer(ctx context.Context, id int, storeID *int) error
	BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error
	BulkDeleteCustomers(ctx context.Context, ids []int, storeID *int) error
}

type service struct {
	repo Repo
}

func NewService(repo Repo) Service {
	return &service{repo: repo}
}

func (s *service) GetByPhone(ctx context.Context, phone string, storeID *int) (*Customer, error) {
	return s.repo.GetByPhone(ctx, phone, storeID)
}

func (s *service) GetCustomerByID(ctx context.Context, id int, storeID *int) (*Customer, error) {
	return s.repo.GetCustomerByID(ctx, id, storeID)
}

func (s *service) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int, customerGroupID *int) ([]Customer, int, error) {
	return s.repo.GetAllCustomers(ctx, limit, offset, search, isActive, storeID, customerGroupID)
}

func (s *service) CreateCustomer(ctx context.Context, customer *Customer, storeID *int) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	customer.Name = strings.TrimSpace(customer.Name)
	customer.StoreID = storeID
	return s.repo.CreateCustomer(ctx, customer)
}

func (s *service) UpdateCustomer(ctx context.Context, customer *Customer, id int, storeID *int) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	if customer.Name != "" {
		customer.Name = strings.TrimSpace(customer.Name)
	}
	old, err := s.repo.GetCustomerByID(ctx, id, storeID)
	if err != nil {
		return err
	}
	if customer.Name == "" {
		customer.Name = old.Name
	}
	if customer.Phone == nil {
		customer.Phone = old.Phone
	}
	if customer.Email == nil {
		customer.Email = old.Email
	}
	if customer.Address == nil {
		customer.Address = old.Address
	}
	if customer.TaxID == nil {
		customer.TaxID = old.TaxID
	}
	if customer.Note == nil {
		customer.Note = old.Note
	}
	if customer.CustomerGroupID == nil {
		customer.CustomerGroupID = old.CustomerGroupID
	}
	return s.repo.UpdateCustomer(ctx, customer, id, storeID)
}

func (s *service) DeleteCustomer(ctx context.Context, id int, storeID *int) error {
	return s.repo.DeleteCustomer(ctx, id, storeID)
}

func (s *service) BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	return s.repo.BulkUpdateCustomersStatus(ctx, ids, isActive, storeID)
}

func (s *service) BulkDeleteCustomers(ctx context.Context, ids []int, storeID *int) error {
	return s.repo.BulkDeleteCustomers(ctx, ids, storeID)
}

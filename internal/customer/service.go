package customer

import (
	"context"
	"errors"

	"retail-pos-system/internal/eventbus"
)

type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
}

type Service struct {
	repo     *Repository
	eventBus EventBus
}

func NewService(repo *Repository, eventBus EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) GetByPhone(ctx context.Context, phone string, storeID *int) (*Customer, error) {
	return s.repo.GetByPhone(ctx, phone, storeID)
}

func (s *Service) GetCustomerByID(ctx context.Context, id int, storeID *int) (*Customer, error) {
	return s.repo.GetCustomerByID(ctx, id, storeID)
}

func (s *Service) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int) ([]Customer, int, error) {
	return s.repo.GetAllCustomers(ctx, limit, offset, search, isActive, storeID)
}

func (s *Service) CreateCustomer(ctx context.Context, customer *Customer, storeID *int) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	customer.StoreID = storeID
	if err := s.repo.CreateCustomer(ctx, customer); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "customer.created", customer)
}

func (s *Service) UpdateCustomer(ctx context.Context, customer *Customer, id int, storeID *int) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	old, err := s.repo.GetCustomerByID(ctx, id, storeID)
	if err != nil {
		return err
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
	if err := s.repo.UpdateCustomer(ctx, customer, id, storeID); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "customer.updated", eventbus.UpdatePayload{Old: old, New: customer})
}

func (s *Service) DeleteCustomer(ctx context.Context, id int, storeID *int) error {
	if err := s.repo.DeleteCustomer(ctx, id, storeID); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "customer.deleted", id)
}

func (s *Service) BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error {
	return s.repo.BulkUpdateCustomersStatus(ctx, ids, isActive, storeID)
}

func (s *Service) BulkDeleteCustomers(ctx context.Context, ids []int, storeID *int) error {
	return s.repo.BulkDeleteCustomers(ctx, ids, storeID)
}



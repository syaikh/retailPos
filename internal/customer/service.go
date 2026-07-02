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

func (s *Service) GetByPhone(ctx context.Context, phone string) (*Customer, error) {
	return s.repo.GetByPhone(ctx, phone)
}

func (s *Service) GetCustomerByID(ctx context.Context, id int) (*Customer, error) {
	return s.repo.GetCustomerByID(ctx, id)
}

func (s *Service) GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Customer, int, error) {
	return s.repo.GetAllCustomers(ctx, limit, offset, search, isActive)
}

func (s *Service) CreateCustomer(ctx context.Context, customer *Customer) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	if err := s.repo.CreateCustomer(ctx, customer); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "customer.created", customer)
}

func (s *Service) UpdateCustomer(ctx context.Context, customer *Customer, id int) error {
	if customer == nil {
		return errors.New("customer cannot be nil")
	}
	old, err := s.repo.GetCustomerByID(ctx, id)
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
	if err := s.repo.UpdateCustomer(ctx, customer, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "customer.updated", eventbus.UpdatePayload{Old: old, New: customer})
}

func (s *Service) DeleteCustomer(ctx context.Context, id int) error {
	if err := s.repo.DeleteCustomer(ctx, id); err != nil {
		return err
	}
	return s.eventBus.Publish(ctx, "customer.deleted", id)
}

func (s *Service) BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool) error {
	return s.repo.BulkUpdateCustomersStatus(ctx, ids, isActive)
}

func (s *Service) BulkDeleteCustomers(ctx context.Context, ids []int) error {
	return s.repo.BulkDeleteCustomers(ctx, ids)
}



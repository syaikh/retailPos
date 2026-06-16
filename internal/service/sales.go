package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"
	"retail-pos-system/pkg/websocket"
)

type SalesService struct {
	db         *pgxpool.Pool
	repo       repository.SaleRepository
	productRepo repository.ProductRepository
	hub        *websocket.Hub
}

func NewSalesService(db *pgxpool.Pool, repo repository.SaleRepository, productRepo repository.ProductRepository, hub *websocket.Hub) *SalesService {
	return &SalesService{
		db:         db,
		repo:       repo,
		productRepo: productRepo,
		hub:        hub,
	}
}

func (s *SalesService) CreateSale(ctx context.Context, sale *domain.Sale, items []domain.SaleItem) error {
	// Validate items
	for _, item := range items {
		product, err := s.productRepo.GetProductByID(ctx, item.ProductID, sale.StoreID)
		if err != nil {
			return err
		}
		if product.Stock < item.Quantity {
			return ErrInsufficientStock
		}
	}

	// Use repository transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	if err := s.repo.CreateSale(ctx, tx, sale, items); err != nil {
		tx.Rollback(ctx)
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *SalesService) GetSaleByID(ctx context.Context, id int) (*domain.Sale, error) {
	return s.repo.GetSaleByID(ctx, id)
}

func (s *SalesService) ListSales(ctx context.Context, limit, offset int, search string, storeID *int) ([]domain.Sale, int, error) {
	return s.repo.GetAllSales(ctx, limit, offset, search, "created_at", "DESC", "", "", storeID, "", nil, nil)
}

var (
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrUnsupportedRepository = errors.New("unsupported repository type")
)

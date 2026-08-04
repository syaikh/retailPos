package shift

import (
	"context"
	"fmt"

	"retail-pos-system/internal/ownership"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
	if openingBalance <= 0 {
		return nil, fmt.Errorf("opening balance must be greater than zero")
	}
	return s.repo.OpenShift(ctx, userID, storeID, openingBalance)
}

func (s *Service) CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	if closingBalance < 0 {
		return nil, fmt.Errorf("closing balance must not be negative")
	}
	return s.repo.CloseShift(ctx, shiftID, userID, closingBalance, notes)
}

func (s *Service) GetActiveShift(ctx context.Context, userID int) (*Shift, error) {
	return s.repo.GetActiveShiftByUserID(ctx, userID)
}

func (s *Service) ListShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
	return s.repo.ListShifts(ctx, scope, status, needsReview, discrepancyFilter, limit, offset, sortBy, sortDir)
}

func (s *Service) ExportShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
	const maxExportRows = 10000
	shifts, _, err := s.repo.ListShifts(ctx, scope, status, needsReview, discrepancyFilter, maxExportRows, 0, "opened_at", "DESC")
	if err != nil {
		return nil, err
	}
	return shifts, nil
}

func (s *Service) GetShiftByID(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
	return s.repo.GetShiftByID(ctx, scope, shiftID)
}

func (s *Service) ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
	return s.repo.ReviewShift(ctx, shiftID, reviewerID)
}

func (s *Service) AuditShift(ctx context.Context, shiftID int) (*Shift, int, error) {
	return s.repo.GetShiftWithLiveSales(ctx, shiftID)
}

func (s *Service) CloseAll(ctx context.Context, userID int) ([]int, error) {
	return s.repo.CloseAll(ctx, userID)
}

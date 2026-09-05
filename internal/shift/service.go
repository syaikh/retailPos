package shift

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/ownership"
)

type Repo interface {
	OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error)
	CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error)
	ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error)
	FlagForReview(ctx context.Context, shiftID int) error
	GetActiveShiftByUserID(ctx context.Context, userID int) (*Shift, error)
	GetShiftByID(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error)
	GetShiftWithLiveSales(ctx context.Context, shiftID int) (*Shift, int, error)
	ListShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error)
	CreateCashMovement(ctx context.Context, tx pgx.Tx, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error)
	ListCashMovements(ctx context.Context, shiftID int) ([]CashMovement, error)
	ShiftCashMovementSummary(ctx context.Context, tx pgx.Tx, shiftID int) (CashMovementSummary, error)
	GetShiftReportData(ctx context.Context, shiftID int) (*ShiftReportData, error)
}

type SettingsProvider interface {
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
}

type service struct {
	repo     *Repository
	settings SettingsProvider
}

func NewService(repo *Repository) Service {
	return &service{repo: repo}
}

func (s *service) SetSettingsProvider(p SettingsProvider) {
	s.settings = p
}

const defaultDiscrepancyThreshold = 50000

func (s *service) getDiscrepancyThreshold(ctx context.Context) int {
	if s.settings == nil {
		return defaultDiscrepancyThreshold
	}
	settings, err := s.settings.GetMultiple(ctx, []string{"shift_discrepancy_threshold"})
	if err != nil {
		return defaultDiscrepancyThreshold
	}
	if v, ok := settings["shift_discrepancy_threshold"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultDiscrepancyThreshold
}

func (s *service) GetDiscrepancyThreshold(ctx context.Context) int {
	return s.getDiscrepancyThreshold(ctx)
}

// InTx runs fn inside a single transaction on the shift database, committing on
// success and rolling back on error. Used to make a shift mutation and its
// audit log atomic.
func (s *service) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.repo.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// OpenShiftTx opens a shift within an existing transaction.
func (s *service) OpenShiftTx(ctx context.Context, tx pgx.Tx, userID int, storeID *int, openingBalance int) (*Shift, error) {
	if openingBalance <= 0 {
		return nil, fmt.Errorf("opening balance must be greater than zero")
	}
	return s.repo.OpenShiftTx(ctx, tx, userID, storeID, openingBalance)
}

// CloseShiftTx closes a shift within an existing transaction.
func (s *service) CloseShiftTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	if closingBalance < 0 {
		return nil, fmt.Errorf("closing balance must not be negative")
	}
	return s.repo.CloseShiftTx(ctx, tx, shiftID, userID, closingBalance, notes)
}

func (s *service) OpenShift(ctx context.Context, userID int, storeID *int, openingBalance int) (*Shift, error) {
	if openingBalance <= 0 {
		return nil, fmt.Errorf("opening balance must be greater than zero")
	}
	return s.repo.OpenShift(ctx, userID, storeID, openingBalance)
}

func (s *service) CloseShift(ctx context.Context, shiftID, userID int, closingBalance int, notes *string) (*Shift, error) {
	if closingBalance < 0 {
		return nil, fmt.Errorf("closing balance must not be negative")
	}
	return s.repo.CloseShift(ctx, shiftID, userID, closingBalance, notes)
}

func (s *service) GetActiveShift(ctx context.Context, userID int) (*Shift, error) {
	return s.repo.GetActiveShiftByUserID(ctx, userID)
}

func (s *service) ListShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string, limit, offset int, sortBy, sortDir string) ([]Shift, int, error) {
	return s.repo.ListShifts(ctx, scope, status, needsReview, discrepancyFilter, limit, offset, sortBy, sortDir)
}

func (s *service) ExportShifts(ctx context.Context, scope ownership.Scope, status string, needsReview *bool, discrepancyFilter string) ([]Shift, error) {
	const maxExportRows = 10000
	shifts, _, err := s.repo.ListShifts(ctx, scope, status, needsReview, discrepancyFilter, maxExportRows, 0, "opened_at", "DESC")
	if err != nil {
		return nil, err
	}
	return shifts, nil
}

func (s *service) GetShiftByID(ctx context.Context, scope ownership.Scope, shiftID int) (*Shift, error) {
	return s.repo.GetShiftByID(ctx, scope, shiftID)
}

func (s *service) ReviewShift(ctx context.Context, shiftID, reviewerID int) (*Shift, error) {
	return s.repo.ReviewShift(ctx, shiftID, reviewerID)
}

func (s *service) FlagForReview(ctx context.Context, shiftID int) error {
	return s.repo.FlagForReview(ctx, shiftID)
}

func (s *service) CreateCashMovement(ctx context.Context, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	return s.repo.CreateCashMovement(ctx, nil, shiftID, userID, movementType, amount, description)
}

func (s *service) CreateCashMovementTx(ctx context.Context, tx pgx.Tx, shiftID, userID int, movementType string, amount int, description *string) (*CashMovement, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}
	return s.repo.CreateCashMovement(ctx, tx, shiftID, userID, movementType, amount, description)
}

func (s *service) ListCashMovements(ctx context.Context, shiftID int) ([]CashMovement, error) {
	return s.repo.ListCashMovements(ctx, shiftID)
}

func (s *service) ShiftCashMovementSummary(ctx context.Context, tx pgx.Tx, shiftID int) (CashMovementSummary, error) {
	return s.repo.ShiftCashMovementSummary(ctx, tx, shiftID)
}

func (s *service) AuditShift(ctx context.Context, shiftID int) (*Shift, int, error) {
	return s.repo.GetShiftWithLiveSales(ctx, shiftID)
}

func (s *service) GetShiftReportData(ctx context.Context, shiftID int) (*ShiftReportData, error) {
	return s.repo.GetShiftReportData(ctx, shiftID)
}

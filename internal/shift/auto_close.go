package shift

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

const defaultAutoCloseHours = 24

// AutoCloser periodically closes abandoned shifts that have been open
// longer than the configured threshold.
type AutoCloser struct {
	repo     *Repository
	settings SettingsProvider
}

func NewAutoCloser(repo *Repository, settings SettingsProvider) *AutoCloser {
	return &AutoCloser{repo: repo, settings: settings}
}

func (a *AutoCloser) Run(ctx context.Context) error {
	hours := a.getHours(ctx)
	if hours <= 0 {
		return nil
	}

	threshold := time.Now().Add(-time.Duration(hours) * time.Hour)

	shifts, err := a.repo.ListOpenShiftsOlderThan(ctx, threshold)
	if err != nil {
		return fmt.Errorf("auto-close: failed to list abandoned shifts: %w", err)
	}

	if len(shifts) == 0 {
		return nil
	}

	slog.Info("auto-close: closing abandoned shifts", "count", len(shifts), "threshold_hours", hours)

	for _, s := range shifts {
		summary, err := a.repo.shiftSalesSummary(ctx, s.ID)
		if err != nil {
			slog.Error("auto-close: failed to get summary", "shift_id", s.ID, "error", err)
			continue
		}

		expectedCash := s.OpeningBalance + summary.TotalCashSales

		notes := fmt.Sprintf("Auto-closed after %d hours of inactivity", hours)
		_, err = a.repo.CloseShift(ctx, s.ID, s.UserID, expectedCash, &notes)
		if err != nil {
			slog.Error("auto-close: failed to close shift", "shift_id", s.ID, "error", err)
			continue
		}

		slog.Info("auto-close: shift closed", "shift_id", s.ID, "user_id", s.UserID)
	}

	return nil
}

func (a *AutoCloser) getHours(ctx context.Context) int {
	if a.settings == nil {
		return defaultAutoCloseHours
	}
	settings, err := a.settings.GetMultiple(ctx, []string{"shift_auto_close_hours"})
	if err != nil {
		return defaultAutoCloseHours
	}
	if v, ok := settings["shift_auto_close_hours"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultAutoCloseHours
}

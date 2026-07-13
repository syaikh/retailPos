package audit

import (
	"context"
	"time"

	"retail-pos-system/internal/shared"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	return s.repo.CreateAuditLog(ctx, log)
}

func (s *Service) GetEntityTypes(ctx context.Context) ([]string, error) {
	return s.repo.GetDistinctEntityTypes(ctx)
}

func (s *Service) GetAuditLogByID(ctx context.Context, id int) (*AuditLog, error) {
	return s.repo.GetAuditLogByID(ctx, id)
}

func (s *Service) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search, action, entityType, startDate, endDate string) ([]AuditLogListItem, int, error) {
	var start, end *time.Time
	if startDate != "" {
		t, err := time.Parse(time.RFC3339, startDate)
		if err != nil {
			t, err = time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation())
			if err != nil {
				return nil, 0, err
			}
		}
		start = &t
	}
	if endDate != "" {
		t, err := time.Parse(time.RFC3339, endDate)
		if err != nil {
			t, err = time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation())
			if err != nil {
				return nil, 0, err
			}
		}
		end = &t
	}
	return s.repo.GetAuditLogs(ctx, limit, offset, userID, search, action, entityType, start, end)
}

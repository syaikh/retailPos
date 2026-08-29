package audit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

type Repo interface {
	CreateAuditLog(ctx context.Context, log *Log) error
	CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *Log) error
	GetAuditLogByID(ctx context.Context, id int) (*Log, error)
	GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search string, action string, entityType string, entityID *int, startDate *time.Time, endDate *time.Time) ([]LogListItem, int, error)
	GetDistinctEntityTypes(ctx context.Context) ([]string, error)
}

type Service struct {
	repo Repo
}

func NewService(repo Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAuditLog(ctx context.Context, log *Log) error {
	return s.repo.CreateAuditLog(ctx, log)
}

func (s *Service) CreateAuditLogTx(ctx context.Context, tx pgx.Tx, log *Log) error {
	return s.repo.CreateAuditLogTx(ctx, tx, log)
}

func (s *Service) GetEntityTypes(ctx context.Context) ([]string, error) {
	return s.repo.GetDistinctEntityTypes(ctx)
}

func (s *Service) GetAuditLogByID(ctx context.Context, id int) (*Log, error) {
	return s.repo.GetAuditLogByID(ctx, id)
}

func (s *Service) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search, action, entityType string, entityID *int, startDate, endDate string) ([]LogListItem, int, error) {
	start, end, err := parseDateRange(startDate, endDate)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.GetAuditLogs(ctx, limit, offset, userID, search, action, entityType, entityID, start, end)
}

func parseDateRange(startDate, endDate string) (start, end *time.Time, err error) {
	if startDate != "" {
		t, e := time.Parse(time.RFC3339, startDate)
		if e != nil {
			t, e = time.ParseInLocation("2006-01-02", startDate, shared.JakartaLocation())
			if e != nil {
				return nil, nil, e
			}
		}
		start = &t
	}
	if endDate != "" {
		t, e := time.Parse(time.RFC3339, endDate)
		if e != nil {
			t, e = time.ParseInLocation("2006-01-02", endDate, shared.JakartaLocation())
			if e != nil {
				return nil, nil, e
			}
		}
		end = &t
	}
	return start, end, nil
}

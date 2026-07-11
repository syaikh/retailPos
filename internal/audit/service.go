package audit

import (
	"context"
	"log"
	"time"
)

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Warning: failed to load Asia/Jakarta timezone: %v. Falling back to UTC.", err)
		jakartaLoc = time.UTC
	}
}

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

func (s *Service) GetAuditLogs(ctx context.Context, limit, offset int, userID *int, search, action, entityType, startDate, endDate string) ([]AuditLog, int, error) {
	var start, end *time.Time
	if startDate != "" {
		t, err := time.Parse(time.RFC3339, startDate)
		if err != nil {
			t, err = time.ParseInLocation("2006-01-02", startDate, jakartaLoc)
			if err != nil {
				return nil, 0, err
			}
		}
		start = &t
	}
	if endDate != "" {
		t, err := time.Parse(time.RFC3339, endDate)
		if err != nil {
			t, err = time.ParseInLocation("2006-01-02", endDate, jakartaLoc)
			if err != nil {
				return nil, 0, err
			}
		}
		end = &t
	}
	return s.repo.GetAuditLogs(ctx, limit, offset, userID, search, action, entityType, start, end)
}

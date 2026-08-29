package appsettings

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Service provides business-logic operations for application settings.
type Service struct {
	repo *Repository
}

// NewService returns a new Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetAll returns every setting as a map.
func (s *Service) GetAll(ctx context.Context) (map[string]string, error) {
	return s.repo.GetAll(ctx)
}

// GetMultiple returns only the requested setting keys.
func (s *Service) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return s.repo.GetMultiple(ctx, keys)
}

// Upsert validates and persists the provided settings.
func (s *Service) Upsert(ctx context.Context, settings map[string]string) error {
	cleaned := make(map[string]string, len(settings))
	for k, v := range settings {
		cleaned[k] = strings.TrimSpace(v)
	}

	if name, ok := cleaned["store_name"]; ok && name == "" {
		return fmt.Errorf("store_name must not be empty")
	}

	return s.repo.UpsertMultiple(ctx, cleaned)
}

// InTx runs fn inside a single transaction on the settings database. The
// transaction is committed if fn returns nil and rolled back otherwise. It is
// used to keep a settings mutation and its audit log atomic.
func (s *Service) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpsertTx validates and persists settings within an existing transaction.
func (s *Service) UpsertTx(ctx context.Context, tx pgx.Tx, settings map[string]string) error {
	cleaned := make(map[string]string, len(settings))
	for k, v := range settings {
		cleaned[k] = strings.TrimSpace(v)
	}

	if name, ok := cleaned["store_name"]; ok && name == "" {
		return fmt.Errorf("store_name must not be empty")
	}

	return s.repo.upsertMultipleTx(ctx, tx, cleaned)
}

// SaveLogoPath persists the logo path setting.
func (s *Service) SaveLogoPath(ctx context.Context, logoPath string) error {
	return s.repo.UpsertMultiple(ctx, map[string]string{"logo_path": logoPath})
}

// LogoPathForFile returns the absolute filesystem path for a logo with the
// given extension, stored under uploads/logos/.
func LogoPathForFile(ext string) string {
	return filepath.Join("uploads", "logos", "logo"+ext)
}

// ValidateLogoExtension checks that the extension is one of the allowed types.
// Returns the normalised extension (lowercase, dot-prefixed) or an error.
func ValidateLogoExtension(ext string) (string, error) {
	ext = strings.ToLower(strings.TrimSpace(ext))
	allowed := map[string]bool{".png": true, ".jpg": true, ".jpeg": true}
	if !allowed[ext] {
		return "", fmt.Errorf("unsupported image type %q; allowed: png, jpg", ext)
	}
	return ext, nil
}

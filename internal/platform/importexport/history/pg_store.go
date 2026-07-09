package history

import (
	"context"
	"encoding/json"
	"fmt"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) SaveSnapshot(ctx context.Context, jobID int64, moduleSchema schema.ModuleSchema, rows []map[string]interface{}, result *importexport.PreviewResult) error {
	rowsJSON, err := json.Marshal(rows)
	if err != nil {
		return fmt.Errorf("marshal snapshot rows: %w", err)
	}
	schemaJSON, err := json.Marshal(moduleSchema)
	if err != nil {
		return fmt.Errorf("marshal schema snapshot: %w", err)
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO import_snapshots (import_job_id, rows_data, schema_snapshot)
		VALUES ($1, $2, $3)
	`, jobID, rowsJSON, schemaJSON)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func (s *Store) SaveRow(ctx context.Context, jobID int64, rowNumber int, status string, entityID *int, oldValues, newValues map[string]interface{}) error {
	var oldJSON, newJSON []byte
	if len(oldValues) > 0 {
		oldJSON, _ = json.Marshal(oldValues)
	}
	if len(newValues) > 0 {
		newJSON, _ = json.Marshal(newValues)
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO import_rows (import_job_id, row_number, status, entity_id, old_values, new_values)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, jobID, rowNumber, status, entityID, oldJSON, newJSON)
	if err != nil {
		return fmt.Errorf("save row: %w", err)
	}
	return nil
}

func (s *Store) SaveError(ctx context.Context, jobID int64, rowNumber int, field, value, reason, suggestion, stage string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO import_errors (import_job_id, row_number, field, value, reason, suggestion, stage)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, jobID, rowNumber, strPtr(field), strPtr(value), reason, strPtr(suggestion), stage)
	if err != nil {
		return fmt.Errorf("save error: %w", err)
	}
	return nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

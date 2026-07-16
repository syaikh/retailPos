package history

import (
	"context"
	"encoding/json"
	"fmt"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/shared"
	importexportshared "retail-pos-system/internal/shared/importexport"

	"github.com/jackc/pgx/v5"
)

type Store struct {
	db shared.DBPool
}

func NewStore(db shared.DBPool) *Store {
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

type SnapshotData struct {
	RowsData       []map[string]interface{} `json:"rows_data"`
	SchemaSnapshot map[string]interface{}   `json:"schema_snapshot"`
}

type RowWithErrors struct {
	RowNumber int                                  `json:"row_number"`
	Status    string                               `json:"status"`
	EntityID  *int                                 `json:"entity_id,omitempty"`
	OldValues map[string]interface{}               `json:"old_values,omitempty"`
	NewValues map[string]interface{}               `json:"new_values,omitempty"`
	Errors    []importexportshared.ValidationError `json:"errors,omitempty"`
}

func (s *Store) GetSnapshot(ctx context.Context, jobID int64) (*SnapshotData, error) {
	var rowsData, schemaSnapshot []byte
	err := s.db.QueryRow(ctx, `
		SELECT rows_data, schema_snapshot
		FROM import_snapshots
		WHERE import_job_id = $1
		ORDER BY id DESC
		LIMIT 1
	`, jobID).Scan(&rowsData, &schemaSnapshot)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("snapshot not found for job %d", jobID)
		}
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var result SnapshotData
	if err := json.Unmarshal(rowsData, &result.RowsData); err != nil {
		return nil, fmt.Errorf("unmarshal rows_data: %w", err)
	}
	if err := json.Unmarshal(schemaSnapshot, &result.SchemaSnapshot); err != nil {
		return nil, fmt.Errorf("unmarshal schema_snapshot: %w", err)
	}
	return &result, nil
}

func (s *Store) GetRows(ctx context.Context, jobID int64) ([]RowWithErrors, error) {
	rows, err := s.db.Query(ctx, `
		SELECT ir.row_number, ir.status, ir.entity_id, ir.old_values, ir.new_values
		FROM import_rows ir
		WHERE ir.import_job_id = $1
		ORDER BY ir.row_number
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query rows: %w", err)
	}
	defer rows.Close()

	rowMap := make(map[int]*RowWithErrors)
	for rows.Next() {
		r := &RowWithErrors{}
		var oldVal, newVal []byte
		if err := rows.Scan(&r.RowNumber, &r.Status, &r.EntityID, &oldVal, &newVal); err != nil {
			continue
		}
		if len(oldVal) > 0 {
			_ = json.Unmarshal(oldVal, &r.OldValues)
		}
		if len(newVal) > 0 {
			_ = json.Unmarshal(newVal, &r.NewValues)
		}
		rowMap[r.RowNumber] = r
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	errRows, err := s.db.Query(ctx, `
		SELECT row_number, field, value, reason, suggestion, stage
		FROM import_errors
		WHERE import_job_id = $1
		ORDER BY row_number
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query errors: %w", err)
	}
	defer errRows.Close()

	for errRows.Next() {
		var rowNum int
		var field, value, reason, suggestion, stage string
		var fieldPtr, valuePtr, suggestionPtr *string
		if err := errRows.Scan(&rowNum, &fieldPtr, &valuePtr, &reason, &suggestionPtr, &stage); err != nil {
			continue
		}
		if fieldPtr != nil {
			field = *fieldPtr
		}
		if valuePtr != nil {
			value = *valuePtr
		}
		if suggestionPtr != nil {
			suggestion = *suggestionPtr
		}
		verr := importexportshared.ValidationError{
			Row:        rowNum,
			Field:      field,
			Value:      value,
			Reason:     reason,
			Suggestion: suggestion,
			Stage:      importexportshared.ErrorStage(stage),
		}
		if r, ok := rowMap[rowNum]; ok {
			r.Errors = append(r.Errors, verr)
		}
	}
	if err := errRows.Err(); err != nil {
		return nil, fmt.Errorf("error rows iteration: %w", err)
	}

	result := make([]RowWithErrors, 0, len(rowMap))
	for _, r := range rowMap {
		result = append(result, *r)
	}
	return result, nil
}

func (s *Store) GetErrors(ctx context.Context, jobID int64) ([]importexportshared.ValidationError, error) {
	rows, err := s.db.Query(ctx, `
		SELECT row_number, COALESCE(field, ''), COALESCE(value, ''), reason, COALESCE(suggestion, ''), stage
		FROM import_errors
		WHERE import_job_id = $1
		ORDER BY row_number
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query errors: %w", err)
	}
	defer rows.Close()

	var result []importexportshared.ValidationError
	for rows.Next() {
		var ve importexportshared.ValidationError
		if err := rows.Scan(&ve.Row, &ve.Field, &ve.Value, &ve.Reason, &ve.Suggestion, &ve.Stage); err != nil {
			continue
		}
		result = append(result, ve)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error rows iteration: %w", err)
	}
	return result, nil
}

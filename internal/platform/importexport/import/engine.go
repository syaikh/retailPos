package importer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	importexportshared "retail-pos-system/internal/shared/importexport"
	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/history"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/validation"

	"github.com/xuri/excelize/v2"
)

type PreviewState struct {
	Module   string
	Schema   schema.ModuleSchema
	Rows     []map[string]interface{}
	Result   *importexport.PreviewResult
	FileName string
	UserID   int
	StoreID  int
	Created  time.Time
}

type Engine struct {
	schemaReg    *schema.Registry
	validator    *validation.Pipeline
	adapterReg   *importexport.AdapterRegistry
	progressEng  *progress.Engine
	historyStore *history.Store
	previews     map[string]*PreviewState
	mu           sync.RWMutex
}

func NewEngine(schemaReg *schema.Registry, v *validation.Pipeline, adapterReg *importexport.AdapterRegistry, progressEng *progress.Engine, historyStore *history.Store) *Engine {
	return &Engine{
		schemaReg:    schemaReg,
		validator:    v,
		adapterReg:   adapterReg,
		progressEng:  progressEng,
		historyStore: historyStore,
		previews:     make(map[string]*PreviewState),
	}
}

func (e *Engine) Preview(ctx context.Context, module string, filename string, file io.Reader) (*importexport.PreviewResult, error) {
	s, err := e.schemaReg.Get(module)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		if err := validateMetaSheet(bytes.NewReader(data), s); err != nil {
			return nil, err
		}
	}

	rows, err := ParseFile(filename, bytes.NewReader(data), s)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if len(rows) == 0 {
		return &importexport.PreviewResult{
			Module:    module,
			TotalRows: 0,
		}, nil
	}

	refs := make(map[string][]importexportshared.ReferenceItem)
	if adapter, err := e.adapterReg.Get(module); err == nil {
		if loaded, err := adapter.Repository().LoadReferences(ctx, s); err == nil {
			refs = loaded
		}
	}
	errs := e.validator.Run(ctx, s, rows, refs)

	preview := GeneratePreview(s, rows, errs)

	token := fmt.Sprintf("pv_%s_%d_%d", module, len(rows), time.Now().UnixNano())
	e.StorePreview(token, &PreviewState{
		Module:   module,
		Schema:   s,
		Rows:     rows,
		Result:   preview,
		FileName: filename,
		Created:  time.Now(),
	})

	preview.Token = token
	return preview, nil
}

func (e *Engine) StorePreview(token string, state *PreviewState) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.previews[token] = state
}

func (e *Engine) GetPreview(token string) *PreviewState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.previews[token]
}

func (e *Engine) DeletePreview(token string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.previews, token)
}

func (e *Engine) isCancelled(ctx context.Context, jobID int64) bool {
	cancelled, err := e.progressEng.IsCancelRequested(ctx, jobID)
	return err == nil && cancelled
}

func (e *Engine) executeImport(ctx context.Context, jobID int64, state *PreviewState, adapter importexportshared.Adapter) {
	repo := adapter.Repository()

	var insertEntities, updateEntities []interface{}
	for _, pr := range state.Result.Rows {
		rowIdx := pr.RowNumber - 2
		if rowIdx < 0 || rowIdx >= len(state.Rows) {
			continue
		}
		if e.isCancelled(ctx, jobID) {
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusCancelled)
			return
		}
		if state.StoreID > 0 {
			state.Rows[rowIdx]["_store_id"] = state.StoreID
		}

		status := pr.Status
		if status == "error" {
			if e.historyStore != nil {
				for _, verr := range pr.Errors {
					_ = e.historyStore.SaveError(ctx, jobID, verr.Row, verr.Field, verr.Value, verr.Reason, verr.Suggestion, string(verr.Stage))
				}
			}
			continue
		}

		entity, err := adapter.MapToEntity(ctx, state.Schema, state.Rows[rowIdx])
		if err != nil {
			continue
		}
		switch status {
		case "insert":
			insertEntities = append(insertEntities, entity)
		case "update":
			updateEntities = append(updateEntities, entity)
		}

		if e.historyStore != nil {
			_ = e.historyStore.SaveRow(ctx, jobID, pr.RowNumber, status, nil, pr.OldValues, pr.NewValues)
		}

		processed := len(insertEntities) + len(updateEntities)
		_ = e.progressEng.UpdateProgress(ctx, jobID, processed, state.Result.TotalRows, state.Result.ErrorCount, len(insertEntities), len(updateEntities))
	}

	if e.isCancelled(ctx, jobID) {
		_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusCancelled)
		return
	}

	totalInsert := len(insertEntities)
	totalUpdate := len(updateEntities)
	errors := state.Result.ErrorCount

	if len(insertEntities) > 0 {
		n, err := repo.Insert(ctx, insertEntities)
		_ = e.progressEng.UpdateProgress(ctx, jobID, n, state.Result.TotalRows, errors, n, totalUpdate)
		if err != nil {
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusFailed)
			return
		}
	}

	if len(updateEntities) > 0 {
		n, err := repo.Update(ctx, updateEntities)
		_ = e.progressEng.UpdateProgress(ctx, jobID, totalInsert+n, state.Result.TotalRows, errors, totalInsert, n)
		if err != nil {
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusFailed)
			return
		}
	}

	processed := totalInsert + totalUpdate
	_ = e.progressEng.UpdateProgress(ctx, jobID, processed, state.Result.TotalRows, errors, totalInsert, totalUpdate)
	_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusCompleted)
}

func (e *Engine) Execute(ctx context.Context, token string) (*importexport.ImportResult, error) {
	state := e.GetPreview(token)
	if state == nil {
		return nil, fmt.Errorf("preview state not found for token %q", token)
	}

	adapter, err := e.adapterReg.Get(state.Module)
	if err != nil {
		return nil, fmt.Errorf("adapter: %w", err)
	}

	jobID, err := e.progressEng.CreateJob(ctx, state.Module, state.Schema.SchemaVersion, state.FileName, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusImporting)
	_ = e.progressEng.UpdateProgress(ctx, jobID, 0, state.Result.TotalRows, state.Result.ErrorCount, 0, 0)

	if e.historyStore != nil {
		_ = e.historyStore.SaveSnapshot(ctx, jobID, state.Schema, state.Rows, state.Result)
	}

	e.executeImport(ctx, jobID, state, adapter)

	e.DeletePreview(token)

	p, err := e.progressEng.GetProgress(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	}

	return &importexport.ImportResult{
		JobID:     jobID,
		Module:    state.Module,
		Status:    string(p.Status),
		TotalRows: p.TotalRows,
		Inserted:  p.Processed - state.Result.ErrorCount,
		Updated:   0,
		Skipped:   0,
		Errors:    p.Errors,
		DurationMs: p.DurationMs,
	}, nil
}

func (e *Engine) StartImport(ctx context.Context, token string, userID, storeID int) (int64, error) {
	state := e.GetPreview(token)
	if state == nil {
		return 0, fmt.Errorf("preview state not found for token %q", token)
	}

	state.UserID = userID
	state.StoreID = storeID

	adapter, err := e.adapterReg.Get(state.Module)
	if err != nil {
		return 0, fmt.Errorf("adapter: %w", err)
	}

	jobID, err := e.progressEng.CreateJob(ctx, state.Module, state.Schema.SchemaVersion, state.FileName, userID, storeID)
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}

	_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusImporting)
	_ = e.progressEng.UpdateProgress(ctx, jobID, 0, state.Result.TotalRows, state.Result.ErrorCount, 0, 0)

	if e.historyStore != nil {
		_ = e.historyStore.SaveSnapshot(ctx, jobID, state.Schema, state.Rows, state.Result)
	}

	go func() {
		e.executeImport(context.Background(), jobID, state, adapter)
		e.DeletePreview(token)
	}()

	return jobID, nil
}

func validateMetaSheet(r io.Reader, s schema.ModuleSchema) error {
	wb, err := excelize.OpenReader(r)
	if err != nil {
		return nil
	}
	defer wb.Close()

	idx, err := wb.GetSheetIndex("_Meta")
	if err != nil || idx < 0 {
		return nil
	}

	rows, err := wb.GetRows("_Meta")
	if err != nil {
		return nil
	}

	var version string
	for _, row := range rows {
		if len(row) >= 2 && row[0] == "SchemaVersion" {
			version = row[1]
			break
		}
	}
	if version == "" {
		return nil
	}
	if version != s.SchemaVersion {
		return fmt.Errorf("template schema version mismatch: expected %q, got %q", s.SchemaVersion, version)
	}
	return nil
}

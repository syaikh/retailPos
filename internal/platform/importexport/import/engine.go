package importer

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/validation"
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
	schemaReg   *schema.Registry
	validator   *validation.Pipeline
	adapterReg  *importexport.AdapterRegistry
	progressEng *progress.Engine
	previews    map[string]*PreviewState
	mu          sync.RWMutex
}

func NewEngine(schemaReg *schema.Registry, v *validation.Pipeline, adapterReg *importexport.AdapterRegistry, progressEng *progress.Engine) *Engine {
	return &Engine{
		schemaReg:   schemaReg,
		validator:   v,
		adapterReg:  adapterReg,
		progressEng: progressEng,
		previews:    make(map[string]*PreviewState),
	}
}

func (e *Engine) Preview(ctx context.Context, module string, filename string, file io.Reader) (*importexport.PreviewResult, error) {
	s, err := e.schemaReg.Get(module)
	if err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}

	rows, err := ParseFile(filename, file, s)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if len(rows) == 0 {
		return &importexport.PreviewResult{
			Module:    module,
			TotalRows: 0,
		}, nil
	}

	refs := make(map[string][]importexport.ReferenceItem)
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

func (e *Engine) executeImport(ctx context.Context, jobID int64, state *PreviewState, adapter importexport.Adapter) {
	repo := adapter.Repository()

	var insertEntities, updateEntities []interface{}
	for _, pr := range state.Result.Rows {
		if pr.Status == "error" {
			continue
		}
		rowIdx := pr.RowNumber - 2
		if rowIdx < 0 || rowIdx >= len(state.Rows) {
			continue
		}
		if e.isCancelled(ctx, jobID) {
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusCancelled)
			return
		}
		entity, err := adapter.MapToEntity(ctx, state.Schema, state.Rows[rowIdx])
		if err != nil {
			continue
		}
		switch pr.Status {
		case "insert":
			insertEntities = append(insertEntities, entity)
		case "update":
			updateEntities = append(updateEntities, entity)
		}
		processed := len(insertEntities) + len(updateEntities)
		_ = e.progressEng.UpdateProgress(ctx, jobID, processed, state.Result.TotalRows, state.Result.ErrorCount)
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
		_ = e.progressEng.UpdateProgress(ctx, jobID, n, state.Result.TotalRows, errors)
		if err != nil {
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusFailed)
			return
		}
	}

	if len(updateEntities) > 0 {
		n, err := repo.Update(ctx, updateEntities)
		_ = e.progressEng.UpdateProgress(ctx, jobID, totalInsert+n, state.Result.TotalRows, errors)
		if err != nil {
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusFailed)
			return
		}
	}

	processed := totalInsert + totalUpdate
	_ = e.progressEng.UpdateProgress(ctx, jobID, processed, state.Result.TotalRows, errors)
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
	_ = e.progressEng.UpdateProgress(ctx, jobID, 0, state.Result.TotalRows, state.Result.ErrorCount)

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

	adapter, err := e.adapterReg.Get(state.Module)
	if err != nil {
		return 0, fmt.Errorf("adapter: %w", err)
	}

	jobID, err := e.progressEng.CreateJob(ctx, state.Module, state.Schema.SchemaVersion, state.FileName, userID, storeID)
	if err != nil {
		return 0, fmt.Errorf("create job: %w", err)
	}

	_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusImporting)
	_ = e.progressEng.UpdateProgress(ctx, jobID, 0, state.Result.TotalRows, state.Result.ErrorCount)

	go func() {
		e.executeImport(context.Background(), jobID, state, adapter)
		e.DeletePreview(token)
	}()

	return jobID, nil
}

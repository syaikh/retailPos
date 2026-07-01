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
	}

	totalInsert := len(insertEntities)
	totalUpdate := len(updateEntities)
	processed := totalInsert + totalUpdate
	errors := state.Result.ErrorCount

	if len(insertEntities) > 0 {
		n, err := repo.Insert(ctx, insertEntities)
		if err != nil {
			_ = e.progressEng.UpdateProgress(ctx, jobID, n, state.Result.TotalRows, errors)
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusFailed)
			return &importexport.ImportResult{
				JobID:       jobID,
				Module:      state.Module,
				Status:      string(progress.StatusFailed),
				TotalRows:   state.Result.TotalRows,
				Inserted:    n,
				Updated:     0,
				Skipped:     totalUpdate,
				Errors:      errors,
				ErrorReport: fmt.Sprintf("insert: %v", err),
			}, nil
		}
		_ = e.progressEng.UpdateProgress(ctx, jobID, n, state.Result.TotalRows, errors)
	}

	if len(updateEntities) > 0 {
		n, err := repo.Update(ctx, updateEntities)
		if err != nil {
			_ = e.progressEng.UpdateProgress(ctx, jobID, totalInsert+n, state.Result.TotalRows, errors)
			_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusFailed)
			return &importexport.ImportResult{
				JobID:       jobID,
				Module:      state.Module,
				Status:      string(progress.StatusFailed),
				TotalRows:   state.Result.TotalRows,
				Inserted:    totalInsert,
				Updated:     n,
				Errors:      errors,
				ErrorReport: fmt.Sprintf("update: %v", err),
			}, nil
		}
		_ = e.progressEng.UpdateProgress(ctx, jobID, totalInsert+n, state.Result.TotalRows, errors)
	}

	_ = e.progressEng.UpdateProgress(ctx, jobID, processed, state.Result.TotalRows, errors)
	_ = e.progressEng.SetStatus(ctx, jobID, progress.StatusCompleted)

	e.DeletePreview(token)

	return &importexport.ImportResult{
		JobID:     jobID,
		Module:    state.Module,
		Status:    string(progress.StatusCompleted),
		TotalRows: state.Result.TotalRows,
		Inserted:  totalInsert,
		Updated:   totalUpdate,
		Skipped:   state.Result.TotalRows - processed - errors,
		Errors:    errors,
	}, nil
}

package importer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/validation"
	importexportshared "retail-pos-system/internal/shared/importexport"
)

type mockRepo struct {
	insertFn func(ctx context.Context, entities []interface{}) (int, error)
	updateFn func(ctx context.Context, entities []interface{}) (int, error)
}

func (m *mockRepo) Insert(ctx context.Context, entities []interface{}) (int, error) {
	if m.insertFn != nil {
		return m.insertFn(ctx, entities)
	}
	return len(entities), nil
}

func (m *mockRepo) Update(ctx context.Context, entities []interface{}) (int, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, entities)
	}
	return len(entities), nil
}

func (m *mockRepo) ExportData(ctx context.Context, s schema.ModuleSchema) ([]map[string]interface{}, error) {
	return nil, nil
}

func (m *mockRepo) LoadReferences(ctx context.Context, s schema.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, nil
}

type mockAdapter struct {
	moduleName    string
	mapToEntityFn func(ctx context.Context, s schema.ModuleSchema, row map[string]interface{}) (interface{}, error)
	repo          *mockRepo
}

func (a *mockAdapter) ModuleName() string { return a.moduleName }

func (a *mockAdapter) ValidateBusiness(ctx context.Context, s schema.ModuleSchema, rows []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (a *mockAdapter) MapToEntity(ctx context.Context, s schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	if a.mapToEntityFn != nil {
		return a.mapToEntityFn(ctx, s, row)
	}
	return row, nil
}

func (a *mockAdapter) Repository() importexportshared.RepositoryActions {
	return a.repo
}

func newTestEngine() (*Engine, *progress.Engine) {
	reg := schema.NewRegistry()
	_ = reg.Register(testSchema)
	v := validation.NewDefaultPipeline()
	adapterReg := importexport.NewAdapterRegistry()
	progStore := progress.NewInMemoryStore()
	progEng := progress.NewEngine(progStore)
	e := NewEngine(reg, v, adapterReg, progEng, nil)
	return e, progEng
}

func registerMockAdapter(e *Engine, adapter *mockAdapter) {
	_ = e.adapterReg.Register(adapter)
}

func newInsertRows(count int) ([]map[string]interface{}, []importexport.PreviewRow) {
	rows := make([]map[string]interface{}, count)
	previewRows := make([]importexport.PreviewRow, count)
	for i := 0; i < count; i++ {
		rows[i] = map[string]interface{}{
			"Code":  fmt.Sprintf("CODE-%d", i+1),
			"Name":  fmt.Sprintf("Item %d", i+1),
			"Price": float64(100 * (i + 1)),
		}
		previewRows[i] = importexport.PreviewRow{
			RowNumber: i + 2,
			Status:    "insert",
			NewValues: rows[i],
		}
	}
	return rows, previewRows
}

func TestExecuteImport_InsertSuccess(t *testing.T) {
	e, progEng := newTestEngine()

	inserted := 0
	adapter := &mockAdapter{
		moduleName: "test",
		repo: &mockRepo{
			insertFn: func(ctx context.Context, entities []interface{}) (int, error) {
				inserted = len(entities)
				return len(entities), nil
			},
		},
	}
	registerMockAdapter(e, adapter)

	rows, previewRows := newInsertRows(3)
	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 3, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	if err != nil {
		t.Fatal(err)
	}

	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusCompleted {
		t.Fatalf("expected status completed, got %s", p.Status)
	}
	if inserted != 3 {
		t.Fatalf("expected 3 inserts, got %d", inserted)
	}
	if p.Inserted != 3 {
		t.Fatalf("progress Inserted = %d, want 3", p.Inserted)
	}
}

func TestExecuteImport_UpdateSuccess(t *testing.T) {
	e, progEng := newTestEngine()

	updated := 0
	adapter := &mockAdapter{
		moduleName: "test",
		repo: &mockRepo{
			updateFn: func(ctx context.Context, entities []interface{}) (int, error) {
				updated = len(entities)
				return len(entities), nil
			},
		},
	}
	registerMockAdapter(e, adapter)

	rows := make([]map[string]interface{}, 2)
	previewRows := make([]importexport.PreviewRow, 2)
	for i := 0; i < 2; i++ {
		rows[i] = map[string]interface{}{
			"Code":  fmt.Sprintf("CODE-%d", i+1),
			"Name":  fmt.Sprintf("Item %d", i+1),
			"Price": float64(100 * (i + 1)),
		}
		previewRows[i] = importexport.PreviewRow{
			RowNumber: i + 2,
			Status:    "update",
			NewValues: rows[i],
		}
	}

	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 2, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusCompleted {
		t.Fatalf("expected completed, got %s", p.Status)
	}
	if updated != 2 {
		t.Fatalf("expected 2 updates, got %d", updated)
	}
	if p.Updated != 2 {
		t.Fatalf("progress Updated = %d, want 2", p.Updated)
	}
}

func TestExecuteImport_InsertFailure(t *testing.T) {
	e, progEng := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		repo: &mockRepo{
			insertFn: func(ctx context.Context, entities []interface{}) (int, error) {
				return 0, fmt.Errorf("duplicate key")
			},
		},
	}
	registerMockAdapter(e, adapter)

	rows, previewRows := newInsertRows(2)
	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 2, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusFailed {
		t.Fatalf("expected failed, got %s", p.Status)
	}
	if p.ErrorReport == "" {
		t.Fatal("expected error report for failed insert")
	}
}

func TestExecuteImport_UpdateFailure(t *testing.T) {
	e, progEng := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		repo: &mockRepo{
			updateFn: func(ctx context.Context, entities []interface{}) (int, error) {
				return 0, fmt.Errorf("update constraint")
			},
		},
	}
	registerMockAdapter(e, adapter)

	rows := make([]map[string]interface{}, 1)
	previewRows := []importexport.PreviewRow{
		{RowNumber: 2, Status: "update", NewValues: map[string]interface{}{"Code": "A", "Name": "B", "Price": 1.0}},
	}
	rows[0] = map[string]interface{}{"Code": "A", "Name": "B", "Price": 1.0}

	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 1, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusFailed {
		t.Fatalf("expected failed, got %s", p.Status)
	}
	if p.ErrorReport == "" {
		t.Fatal("expected error report for failed update")
	}
}

func TestExecuteImport_MapToEntityError(t *testing.T) {
	e, progEng := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		mapToEntityFn: func(ctx context.Context, s schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("invalid entity data")
		},
		repo: &mockRepo{},
	}
	registerMockAdapter(e, adapter)

	rows, previewRows := newInsertRows(2)
	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 2, ErrorCount: 2, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusFailed {
		t.Fatalf("expected failed, got %s", p.Status)
	}
	if p.ErrorReport == "" {
		t.Fatal("expected error report for MapToEntity failures")
	}
}

func TestExecuteImport_ErrorRowsSkipped(t *testing.T) {
	e, progEng := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		repo:       &mockRepo{},
	}
	registerMockAdapter(e, adapter)

	rows := []map[string]interface{}{
		{"Code": "A", "Name": "Good", "Price": 100.0},
		{"Code": "B", "Name": "Bad", "Price": 50.0},
	}
	previewRows := []importexport.PreviewRow{
		{RowNumber: 2, Status: "insert", NewValues: rows[0]},
		{RowNumber: 3, Status: "error", Errors: []importexportshared.ValidationError{
			{Row: 3, Field: "Name", Reason: "required"},
		}},
	}

	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 2, ErrorCount: 1, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusCompleted {
		t.Fatalf("expected completed, got %s", p.Status)
	}
	if p.Inserted != 1 {
		t.Fatalf("expected 1 insert (error row skipped), got %d", p.Inserted)
	}
}

func TestExecuteImport_StoreIDInjected(t *testing.T) {
	e, progEng := newTestEngine()

	var capturedRow map[string]interface{}
	adapter := &mockAdapter{
		moduleName: "test",
		mapToEntityFn: func(ctx context.Context, s schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
			capturedRow = row
			return row, nil
		},
		repo: &mockRepo{},
	}
	registerMockAdapter(e, adapter)

	rows, previewRows := newInsertRows(1)
	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 1, Rows: previewRows},
		StoreID: 42,
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 42)
	e.executeImport(ctx, jobID, state, adapter)

	if capturedRow == nil {
		t.Fatal("expected MapToEntity to be called")
	}
	if capturedRow["_store_id"] != 42 {
		t.Fatalf("_store_id = %v, want 42", capturedRow["_store_id"])
	}
}

func TestExecuteImport_OutOfBoundsRowSkipped(t *testing.T) {
	e, progEng := newTestEngine()

	entityCount := 0
	adapter := &mockAdapter{
		moduleName: "test",
		mapToEntityFn: func(ctx context.Context, s schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
			entityCount++
			return row, nil
		},
		repo: &mockRepo{},
	}
	registerMockAdapter(e, adapter)

	previewRows := []importexport.PreviewRow{
		{RowNumber: 99, Status: "insert", NewValues: map[string]interface{}{"Code": "X"}},
		{RowNumber: -1, Status: "insert", NewValues: map[string]interface{}{"Code": "Y"}},
	}

	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    []map[string]interface{}{},
		Result:  &importexport.PreviewResult{TotalRows: 2, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	if entityCount != 0 {
		t.Fatalf("expected 0 entities (all out of bounds), got %d", entityCount)
	}
}

func TestExecuteImport_MixedInsertAndError(t *testing.T) {
	e, progEng := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		repo:       &mockRepo{},
	}
	registerMockAdapter(e, adapter)

	rows := []map[string]interface{}{
		{"Code": "A", "Name": "Good", "Price": 100.0},
		{"Code": "B", "Name": "Bad", "Price": 50.0},
		{"Code": "C", "Name": "Also Good", "Price": 200.0},
	}
	previewRows := []importexport.PreviewRow{
		{RowNumber: 2, Status: "insert", NewValues: rows[0]},
		{RowNumber: 3, Status: "error", Errors: []importexportshared.ValidationError{
			{Row: 3, Field: "Name", Reason: "empty"},
		}},
		{RowNumber: 4, Status: "insert", NewValues: rows[2]},
	}

	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 3, ErrorCount: 1, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)
	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusCompleted {
		t.Fatalf("expected completed, got %s", p.Status)
	}
	if p.Inserted != 2 {
		t.Fatalf("expected 2 inserts, got %d", p.Inserted)
	}
}

func TestExecuteImport_CancellationDuringLoop(t *testing.T) {
	e, progEng := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		repo:       &mockRepo{},
	}
	registerMockAdapter(e, adapter)

	rows, previewRows := newInsertRows(5)
	state := &PreviewState{
		Module:  "test",
		Schema:  testSchema,
		Rows:    rows,
		Result:  &importexport.PreviewResult{TotalRows: 5, Rows: previewRows},
		Created: time.Now(),
	}

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "test", "1.0.0", "data.csv", 1, 0)

	_ = progEng.RequestCancel(ctx, jobID)

	e.executeImport(ctx, jobID, state, adapter)

	p, _ := progEng.GetProgress(ctx, jobID)
	if p.Status != progress.StatusCancelled {
		t.Fatalf("expected cancelled, got %s", p.Status)
	}
}

func TestExecuteImport_ExecuteHappyPath(t *testing.T) {
	e, _ := newTestEngine()

	adapter := &mockAdapter{
		moduleName: "test",
		repo: &mockRepo{
			insertFn: func(ctx context.Context, entities []interface{}) (int, error) {
				return len(entities), nil
			},
		},
	}
	registerMockAdapter(e, adapter)

	rows, previewRows := newInsertRows(2)
	state := &PreviewState{
		Module:   "test",
		Schema:   testSchema,
		Rows:     rows,
		Result:   &importexport.PreviewResult{TotalRows: 2, InsertCount: 2, Rows: previewRows},
		FileName: "data.csv",
		UserID:   1,
		Created:  time.Now(),
	}
	e.StorePreview("pv_test_ok", state)

	result, err := e.Execute(context.Background(), "pv_test_ok")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != string(progress.StatusCompleted) {
		t.Fatalf("expected completed, got %s", result.Status)
	}
	if result.Inserted != 2 {
		t.Fatalf("expected 2 inserted, got %d", result.Inserted)
	}

	if e.GetPreview("pv_test_ok") != nil {
		t.Fatal("preview should be deleted after Execute")
	}
}

func TestExecuteImport_ExecuteNoAdapter(t *testing.T) {
	e, _ := newTestEngine()

	state := &PreviewState{
		Module:  "unknown-module",
		Schema:  testSchema,
		Rows:    []map[string]interface{}{},
		Result:  &importexport.PreviewResult{TotalRows: 0},
		Created: time.Now(),
	}
	e.StorePreview("pv_no_adapter", state)

	_, err := e.Execute(context.Background(), "pv_no_adapter")
	if err == nil {
		t.Fatal("expected error for missing adapter")
	}
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/export"
	"retail-pos-system/internal/platform/importexport/history"
	importer "retail-pos-system/internal/platform/importexport/import"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"
	"retail-pos-system/internal/platform/importexport/validation"
	importexportshared "retail-pos-system/internal/shared/importexport"
)

var testSchema = schema.ModuleSchema{
	ModuleName:    "categories",
	DisplayName:   "Test Module",
	SchemaVersion: "1.0.0",
	Description:   "Test module for handler tests",
	PrimaryKey:    "id",
	BusinessKeys:  []string{"Code"},
	Columns: []schema.ColumnSchema{
		{Name: "Code", Type: schema.ColString, Label: "Code", Required: true, MaxLength: schema.IntPtr(20), Editable: false, Exportable: true, Template: true},
		{Name: "Name", Type: schema.ColString, Label: "Name", Required: true, MaxLength: schema.IntPtr(100), Editable: true, Exportable: true, Template: true},
	},
	Features: schema.ModuleFeatures{
		ImportEnabled:   true,
		ExportEnabled:   true,
		TemplateEnabled: true,
		SupportsPreview: true,
		MassInsert:      true,
		MassUpdate:      true,
	},
}

type mockTestRepo struct {
	exportData []map[string]interface{}
}

func (m *mockTestRepo) Insert(_ context.Context, entities []interface{}) (int, error) {
	return len(entities), nil
}

func (m *mockTestRepo) Update(_ context.Context, entities []interface{}) (int, error) {
	return len(entities), nil
}

func (m *mockTestRepo) ExportData(_ context.Context, _ schema.ModuleSchema) ([]map[string]interface{}, error) {
	if m.exportData == nil {
		return []map[string]interface{}{}, nil
	}
	return m.exportData, nil
}

func (m *mockTestRepo) LoadReferences(_ context.Context, _ schema.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, nil
}

type mockTestAdapter struct {
	repo *mockTestRepo
}

func (m *mockTestAdapter) ModuleName() string { return "categories" }

func (m *mockTestAdapter) ValidateBusiness(_ context.Context, _ schema.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}

func (m *mockTestAdapter) MapToEntity(_ context.Context, _ schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	return row, nil
}

func (m *mockTestAdapter) Repository() importexportshared.RepositoryActions {
	return m.repo
}

func setupTestHandler() (*Handler, *gin.Engine, *progress.Engine) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&mockTestAdapter{repo: &mockTestRepo{}})

	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", 0)
		c.Next()
	}
	perm := func(_ permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	return h, r, progEng
}

func TestHandler_ListModules(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/modules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp, 1)
	assert.Equal(t, "categories", resp[0]["name"])
	assert.Equal(t, "Test Module", resp[0]["displayName"])
}

func TestHandler_DownloadTemplate(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/template/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "categories-template.xlsx")
}

func TestHandler_DownloadTemplate_UnknownModule(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/template/bogus", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Preview(t *testing.T) {
	_, r, _ := setupTestHandler()

	csv := "Code,Name\nA1,Widget\nA2,Gadget\n"
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.csv")
	_, _ = part.Write([]byte(csv))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/categories", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "categories", result["module"])
	assert.Equal(t, float64(2), result["total_rows"])
	assert.NotEmpty(t, result["token"])
}

func TestHandler_Preview_UnknownModule(t *testing.T) {
	_, r, _ := setupTestHandler()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.csv")
	_, _ = part.Write([]byte("Code,Name\nA1,Widget\n"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/bogus", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Preview_NoFile(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Confirm(t *testing.T) {
	_, r, progEng := setupTestHandler()

	csv := "Code,Name\nA1,Widget\nA2,Gadget\n"
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.csv")
	_, _ = part.Write([]byte(csv))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/categories", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var previewResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &previewResp))
	token, ok := previewResp["token"].(string)
	require.True(t, ok, "preview should return a token")
	require.NotEmpty(t, token)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/import-export/confirm/categories?token="+token, nil)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var result map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &result)
	require.NoError(t, err)
	jobID := int64(result["job_id"].(float64))
	assert.Equal(t, "importing", result["status"])

	ctx := context.Background()
	var p *progress.Progress
	for i := 0; i < 50; i++ {
		p, err = progEng.GetProgress(ctx, jobID)
		require.NoError(t, err)
		if p.Status != progress.StatusImporting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, progress.StatusCompleted, p.Status)
	assert.Equal(t, 2, p.TotalRows)
}

func TestHandler_Confirm_NoToken(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/confirm/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Confirm_BadToken(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/confirm/categories?token=bogus-token", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetProgress(t *testing.T) {
	_, r, progEng := setupTestHandler()

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)
	require.NoError(t, err)

	_ = progEng.SetStatus(ctx, jobID, progress.StatusImporting)
	_ = progEng.UpdateProgress(ctx, jobID, 5, 10, 0, 0, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/progress/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var p map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &p)
	require.NoError(t, err)
	assert.Equal(t, float64(jobID), p["job_id"])
	assert.Equal(t, "importing", p["status"])
	assert.Equal(t, float64(5), p["processed"])
	assert.Equal(t, float64(10), p["total_rows"])
}

func TestHandler_GetProgress_NotFound(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/progress/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetProgress_InvalidJobID(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/progress/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CancelImport(t *testing.T) {
	_, r, progEng := setupTestHandler()

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/import-export/cancel/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "cancellation requested", result["status"])

	cancelled, err := progEng.IsCancelRequested(ctx, jobID)
	require.NoError(t, err)
	assert.True(t, cancelled)

	p, err := progEng.GetProgress(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, progress.StatusQueued, p.Status)
}

func TestHandler_CancelImport_NotFound(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/cancel/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CancelImport_InvalidJobID(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/cancel/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func setupStoreScopedHandler(claimStore int) (*gin.Engine, *progress.Engine) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&mockTestAdapter{repo: &mockTestRepo{}})

	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

	r := gin.New()
	sid := claimStore
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", &sid)
		c.Next()
	}
	perm := func(_ permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	return r, progEng
}

func TestHandler_GetProgress_StoreScoped_OwnStore(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/progress/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetProgress_StoreScoped_OtherStore(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 9)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/progress/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_CancelImport_StoreScoped_OwnStore(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/import-export/cancel/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CancelImport_StoreScoped_OtherStore(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 9)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/import-export/cancel/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_ListImportHistory_StoreScoped(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	_, err := progEng.CreateJob(ctx, "categories", "1.0.0", "a.csv", 1, 5)
	require.NoError(t, err)
	_, err = progEng.CreateJob(ctx, "categories", "1.0.0", "b.csv", 1, 9)
	require.NoError(t, err)
	_, err = progEng.CreateJob(ctx, "categories", "1.0.0", "c.csv", 1, 5)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
}

func TestHandler_GetImportDetail_StoreScoped_OtherStore(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 9)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_GetImportRows_StoreScoped_OtherStore(t *testing.T) {
	r, progEng := setupStoreScopedHandler(5)

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 9)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d/rows", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_ExportCSV(t *testing.T) {
	setupExportTest := func() *gin.Engine {
		gin.SetMode(gin.TestMode)
		schemaReg := schema.NewRegistry()
		_ = schemaReg.Register(testSchema)

		adapterReg := importexport.NewAdapterRegistry()
		_ = adapterReg.Register(&mockTestAdapter{
			repo: &mockTestRepo{
				exportData: []map[string]interface{}{
					{"Code": "A1", "Name": "Widget"},
					{"Code": "A2", "Name": "Gadget"},
				},
			},
		})

		val := validation.NewDefaultPipeline()
		progEng := progress.NewEngine(progress.NewInMemoryStore())
		importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
		exportEng := export.NewEngine()
		templateEng := template.NewEngine()

		h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

		r := gin.New()
		auth := func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("storeID", 0)
			c.Next()
		}
		perm := func(_ permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		}
		h.RegisterRoutes(r.Group("/api"), auth, perm)
		return r
	}

	r := setupExportTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/export/categories?format=csv", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "categories.csv")

	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), "A1")
	assert.Contains(t, string(body), "Gadget")
}

func TestHandler_ExportXLSX(t *testing.T) {
	setupExportTest := func() *gin.Engine {
		gin.SetMode(gin.TestMode)
		schemaReg := schema.NewRegistry()
		_ = schemaReg.Register(testSchema)

		adapterReg := importexport.NewAdapterRegistry()
		_ = adapterReg.Register(&mockTestAdapter{
			repo: &mockTestRepo{
				exportData: []map[string]interface{}{
					{"Code": "A1", "Name": "Widget"},
				},
			},
		})

		val := validation.NewDefaultPipeline()
		progEng := progress.NewEngine(progress.NewInMemoryStore())
		importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
		exportEng := export.NewEngine()
		templateEng := template.NewEngine()

		h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

		r := gin.New()
		auth := func(c *gin.Context) {
			c.Set("userID", 1)
			c.Set("storeID", 0)
			c.Next()
		}
		perm := func(_ permissions.Code) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		}
		h.RegisterRoutes(r.Group("/api"), auth, perm)
		return r
	}

	r := setupExportTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/export/categories?format=xlsx", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "categories.xlsx")
}

func TestHandler_Export_UnknownModule(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/export/bogus?format=csv", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Export_InvalidFormat(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/export/categories?format=pdf", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "format must be")
}

func TestHandler_PermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(schema.ModuleSchema{
		ModuleName: "categories",
		Columns:    []schema.ColumnSchema{{Name: "X", Type: schema.ColString, Label: "X"}},
	})

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&mockTestAdapter{repo: &mockTestRepo{}})

	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", 0)
		c.Next()
	}
	permCalled := false
	perm := func(permStr permissions.Code) gin.HandlerFunc {
		permCalled = true
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: " + permStr})
		}
	}
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	t.Run("preview requires import permission", func(t *testing.T) {
		permCalled = false
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.csv")
		_, _ = part.Write([]byte("X\nval\n"))
		_ = writer.Close()

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/import-export/preview/categories", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, permCalled, "perm middleware should have been called")
	})

	t.Run("export requires export permission", func(t *testing.T) {
		permCalled = false
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/import-export/export/categories?format=csv", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, permCalled, "perm middleware should have been called")
	})

	t.Run("confirm requires import permission", func(t *testing.T) {
		permCalled = false
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/import-export/confirm/categories?token=abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, permCalled, "perm middleware should have been called")
	})
}

func TestHandler_ListModules_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	schemaReg := schema.NewRegistry()
	adapterReg := importexport.NewAdapterRegistry()
	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", 0)
		c.Next()
	}
	perm := func(_ permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/modules", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, "[]", w.Body.String())
}

func TestHandler_GetProgress_JSONStructure(t *testing.T) {
	_, r, progEng := setupTestHandler()

	ctx := context.Background()
	jobID, err := progEng.CreateJob(ctx, "categories", "1.0.0", "data.csv", 1, 0)
	require.NoError(t, err)

	_ = progEng.SetStatus(ctx, jobID, progress.StatusCompleted)
	_ = progEng.UpdateProgress(ctx, jobID, 10, 10, 2, 0, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/progress/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var p importexport.ImportProgress
	err = json.Unmarshal(w.Body.Bytes(), &p)
	require.NoError(t, err)
	assert.Equal(t, jobID, p.JobID)
	assert.Equal(t, "completed", p.Status)
	assert.Equal(t, 100, p.ProgressPct)
	assert.Equal(t, 10, p.TotalRows)
	assert.Equal(t, 10, p.Processed)
	assert.Equal(t, 2, p.Errors)
	assert.NotEmpty(t, p.StartedAt)
}

func TestHandler_Preview_EmptyFile(t *testing.T) {
	_, r, _ := setupTestHandler()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "empty.csv")
	_, _ = part.Write([]byte("Code,Name\n"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/categories", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Preview_MissingModuleInPerm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(schema.ModuleSchema{
		ModuleName: "custom",
		Columns:    []schema.ColumnSchema{{Name: "X", Type: schema.ColString, Label: "X"}},
	})

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&mockTestAdapter{repo: &mockTestRepo{}})

	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)

	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", 0)
		c.Next()
	}
	perm := func(_ permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.csv")
	_, _ = part.Write([]byte("X\nval\n"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/custom", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

var _ = importexportshared.ValidationError{}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"categories", "categories"},
		{"my file name", "my_file_name"},
		{"hello/world", "hello_world"},
		{"file-name_123", "file-name_123"},
		{"special!@#$chars", "special____chars"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestHandler_ListImportHistory(t *testing.T) {
	_, r, progEng := setupTestHandler()

	ctx := context.Background()
	_, _ = progEng.CreateJob(ctx, "categories", "1.0.0", "a.csv", 1, 0)
	_, _ = progEng.CreateJob(ctx, "categories", "1.0.0", "b.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var jobs []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &jobs)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestHandler_ListImportHistory_UnknownModule(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/bogus", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetImportDetail_UnknownModule(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/bogus/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetImportDetail_InvalidJobID(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/categories/abc", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetImportDetail_NilHistoryStore(t *testing.T) {
	_, r, progEng := setupTestHandler()

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetImportRows_UnknownModule(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/bogus/1/rows", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetImportRows_InvalidJobID(t *testing.T) {
	_, r, _ := setupTestHandler()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/categories/abc/rows", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetImportRows_NilHistoryStore(t *testing.T) {
	_, r, progEng := setupTestHandler()

	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d/rows", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_DownloadTemplate_TemplateDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(schema.ModuleSchema{
		ModuleName: "categories",
		Features: schema.ModuleFeatures{
			TemplateEnabled: false,
		},
		Columns: []schema.ColumnSchema{{Name: "Code", Type: schema.ColString, Label: "Code"}},
	})

	adapterReg := importexport.NewAdapterRegistry()
	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("storeID", 0)
		c.Next()
	}
	perm := func(_ permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/template/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "template not available")
}

func TestHandler_Preview_InvalidFormat(t *testing.T) {
	_, r, _ := setupTestHandler()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.pdf")
	_, _ = part.Write([]byte("%PDF-1.4 fake"))
	_ = writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/import-export/preview/categories", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

type failingLoadRepo struct {
	mockTestRepo
}

func (m *failingLoadRepo) LoadReferences(_ context.Context, _ schema.ModuleSchema) (map[string][]importexportshared.ReferenceItem, error) {
	return nil, fmt.Errorf("load references failed")
}

type failingExportRepo struct {
	mockTestRepo
}

func (m *failingExportRepo) ExportData(_ context.Context, _ schema.ModuleSchema) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("export data failed")
}

type failingLoadAdapter struct {
	repo *failingLoadRepo
}

func (m *failingLoadAdapter) ModuleName() string { return "categories" }
func (m *failingLoadAdapter) ValidateBusiness(_ context.Context, _ schema.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}
func (m *failingLoadAdapter) MapToEntity(_ context.Context, _ schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	return row, nil
}
func (m *failingLoadAdapter) Repository() importexportshared.RepositoryActions {
	return m.repo
}

type failingExportAdapter struct {
	repo *failingExportRepo
}

func (m *failingExportAdapter) ModuleName() string { return "categories" }
func (m *failingExportAdapter) ValidateBusiness(_ context.Context, _ schema.ModuleSchema, _ []map[string]interface{}) []importexportshared.ValidationError {
	return nil
}
func (m *failingExportAdapter) MapToEntity(_ context.Context, _ schema.ModuleSchema, row map[string]interface{}) (interface{}, error) {
	return row, nil
}
func (m *failingExportAdapter) Repository() importexportshared.RepositoryActions {
	return m.repo
}

type failingProgressStore struct {
	progress.InMemoryStore
	err error
}

func (f *failingProgressStore) ListJobs(_ context.Context, _ string, _ int, _ *int) ([]*progress.Progress, error) {
	return nil, f.err
}

func TestHandler_DownloadTemplate_AdapterError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(schema.ModuleSchema{
		ModuleName: "noadapter",
		Features:   schema.ModuleFeatures{TemplateEnabled: true},
		Columns:    []schema.ColumnSchema{{Name: "X", Type: schema.ColString, Label: "X"}},
	})

	adapterReg := importexport.NewAdapterRegistry()
	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/template/noadapter", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_DownloadTemplate_LoadRefError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&failingLoadAdapter{repo: &failingLoadRepo{}})

	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/template/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Export_AdapterError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)

	adapterReg := importexport.NewAdapterRegistry()
	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/export/categories?format=csv", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Export_ExportDataError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&failingExportAdapter{repo: &failingExportRepo{}})

	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/export/categories?format=csv", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_ListImportHistory_ListJobsError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)

	adapterReg := importexport.NewAdapterRegistry()
	val := validation.NewDefaultPipeline()
	failStore := &failingProgressStore{err: fmt.Errorf("store unavailable")}
	progEng := progress.NewEngine(failStore)
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, nil)
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Set("storeID", 0); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/categories", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

type mockHistoryStore struct {
	snapshot *history.SnapshotData
	rows     []history.RowWithErrors
	snapErr  error
	rowsErr  error
}

func (m *mockHistoryStore) GetSnapshot(_ context.Context, _ int64) (*history.SnapshotData, error) {
	return m.snapshot, m.snapErr
}

func (m *mockHistoryStore) GetRows(_ context.Context, _ int64) ([]history.RowWithErrors, error) {
	return m.rows, m.rowsErr
}

func setupHandlerWithHistory(hs HistoryReader) (*gin.Engine, *progress.Engine) {
	gin.SetMode(gin.TestMode)
	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)
	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(&mockTestAdapter{repo: &mockTestRepo{}})
	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, hs)
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Set("storeID", 0); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)
	return r, progEng
}

func TestHandler_GetImportDetail_SnapshotNotFound(t *testing.T) {
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{
		snapErr: fmt.Errorf("snapshot not found for job 999"),
	})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "snapshot not found")
}

func TestHandler_GetImportDetail_DBError(t *testing.T) {
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{
		snapErr: fmt.Errorf("get snapshot: connection refused"),
	})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetImportDetail_Success(t *testing.T) {
	snap := &history.SnapshotData{
		RowsData:       []map[string]interface{}{{"Code": "A1", "Name": "Widget"}},
		SchemaSnapshot: map[string]interface{}{"module_name": "categories"},
	}
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{snapshot: snap})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotNil(t, resp["snapshot"])
	assert.NotNil(t, resp["progress"])
}

func TestHandler_GetImportRows_RowsNotFound(t *testing.T) {
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{
		rowsErr: fmt.Errorf("rows not found for job 999"),
	})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d/rows", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "rows not found")
}

func TestHandler_GetImportRows_DBError(t *testing.T) {
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{
		rowsErr: fmt.Errorf("query rows: connection refused"),
	})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d/rows", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_GetImportRows_Success(t *testing.T) {
	rows := []history.RowWithErrors{
		{RowNumber: 1, Status: "created", EntityID: intPtr(10)},
		{RowNumber: 2, Status: "skipped"},
	}
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{rows: rows})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d/rows", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	rowList := resp["rows"].([]interface{})
	assert.Len(t, rowList, 2)
}

func intPtr(v int) *int { return &v }

func TestHandler_GetImportRows_NotFoundSubstring(t *testing.T) {
	r, progEng := setupHandlerWithHistory(&mockHistoryStore{
		rowsErr: fmt.Errorf("not found in table import_rows"),
	})
	ctx := context.Background()
	jobID, _ := progEng.CreateJob(ctx, "categories", "1.0.0", "test.csv", 1, 0)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/import-export/history/categories/%d/rows", jobID), nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "rows not found")
}

func TestHandler_GetImportDetail_GetProgressNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(testSchema)
	adapterReg := importexport.NewAdapterRegistry()
	val := validation.NewDefaultPipeline()
	progEng := progress.NewEngine(progress.NewInMemoryStore())
	importEng := importer.NewEngine(schemaReg, val, adapterReg, progEng, nil)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()

	h := NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, &mockHistoryStore{})
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("userID", 1); c.Next() }
	perm := func(_ permissions.Code) gin.HandlerFunc { return func(c *gin.Context) { c.Next() } }
	h.RegisterRoutes(r.Group("/api"), auth, perm)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/import-export/history/categories/999999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

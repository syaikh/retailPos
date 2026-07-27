package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/export"
	"retail-pos-system/internal/platform/importexport/history"
	importer "retail-pos-system/internal/platform/importexport/import"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

var modulePerms = map[string]string{
	"categories:import":   "category.import",
	"categories:export":   "category.export",
	"categories:history":  "category.import",
	"brands:import":       "product.import",
	"brands:export":       "product.export",
	"brands:history":      "product.import",
	"uoms:import":         "product.import",
	"uoms:export":         "product.export",
	"uoms:history":        "product.import",
	"customers:import":    "customer.import",
	"customers:export":    "customer.export",
	"customers:history":   "customer.import",
	"products:import":     "product.import",
	"products:export":     "product.export",
	"products:history":    "product.import",
	"pricing_rules:import":  "pricing.create",
	"pricing_rules:export":  "pricing.view",
	"pricing_rules:history": "pricing.view",
	"suppliers:import":      "supplier.create",
	"suppliers:export":      "supplier.view",
	"suppliers:history":     "supplier.view",
}

type Handler struct {
	schemaReg    *schema.Registry
	adapterReg   *importexport.AdapterRegistry
	importEng    *importer.Engine
	exportEng    *export.Engine
	templateEng  *template.Engine
	progressEng  *progress.Engine
	historyStore HistoryReader
	permFunc     func(string) gin.HandlerFunc
}

// HistoryReader abstracts the history store for testing.
type HistoryReader interface {
	GetSnapshot(ctx context.Context, jobID int64) (*history.SnapshotData, error)
	GetRows(ctx context.Context, jobID int64) ([]history.RowWithErrors, error)
}

func NewHandler(schemaReg *schema.Registry, adapterReg *importexport.AdapterRegistry, importEng *importer.Engine, exportEng *export.Engine, templateEng *template.Engine, progressEng *progress.Engine, historyStore HistoryReader) *Handler {
	return &Handler{
		schemaReg:    schemaReg,
		adapterReg:   adapterReg,
		importEng:    importEng,
		exportEng:    exportEng,
		templateEng:  templateEng,
		progressEng:  progressEng,
		historyStore: historyStore,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	h.permFunc = perm
	r := rg.Group("/import-export")
	r.Use(auth)
	{
		r.GET("/modules", h.ListModules)
		r.GET("/template/:module", h.DownloadTemplate)
		r.POST("/preview/:module", h.requirePerm("import"), h.Preview)
		r.POST("/confirm/:module", h.requirePerm("import"), h.Confirm)
		r.GET("/progress/:jobId", h.GetProgress)
		r.POST("/cancel/:jobId", h.CancelImport)
		r.GET("/history/:module", h.requirePerm("history"), h.ListImportHistory)
		r.GET("/history/:module/:jobId", h.requirePerm("history"), h.GetImportDetail)
		r.GET("/history/:module/:jobId/rows", h.requirePerm("history"), h.GetImportRows)
		r.GET("/export/:module", h.requirePerm("export"), h.Export)
	}
}

func (h *Handler) requirePerm(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		module := c.Param("module")
		if _, err := h.schemaReg.Get(module); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, shared.NewError(shared.ErrBadRequest, fmt.Sprintf("unknown module: %s", module)))
			return
		}
		key := module + ":" + action
		permStr, ok := modulePerms[key]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, shared.NewError(shared.ErrForbidden, "permission not defined for module"))
			return
		}
		h.permFunc(permStr)(c)
	}
}

func (h *Handler) ListModules(c *gin.Context) {
	allSchemas := h.schemaReg.All()
	modules := make([]gin.H, 0, len(allSchemas))
	for _, s := range allSchemas {
		modules = append(modules, gin.H{
			"name":        s.ModuleName,
			"displayName": s.DisplayName,
			"features": gin.H{
				"importEnabled":   s.Features.ImportEnabled,
				"exportEnabled":   s.Features.ExportEnabled,
				"templateEnabled": s.Features.TemplateEnabled,
			},
		})
	}
	c.JSON(http.StatusOK, modules)
}

func (h *Handler) DownloadTemplate(c *gin.Context) {
	module := c.Param("module")
	s, err := h.schemaReg.Get(module)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	if !s.Features.TemplateEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template not available for this module"})
		return
	}

	adapter, err := h.adapterReg.Get(module)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	refData, err := adapter.Repository().LoadReferences(c.Request.Context(), s)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	refDataFlat := make(map[string][]string)
	for k, items := range refData {
		vals := make([]string, len(items))
		for i, item := range items {
			vals[i] = item.Key
		}
		refDataFlat[k] = vals
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-template.xlsx"`, sanitizeFilename(module)))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := h.templateEng.Generate(s, refDataFlat, c.Writer); err != nil {
		shared.InternalError(c, err)
	}
}

func (h *Handler) Preview(c *gin.Context) {
	module := c.Param("module")
	_, err := h.schemaReg.Get(module)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<20)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required or too large (max 32MB)"})
		return
	}
	defer file.Close()

	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".csv") && !strings.HasSuffix(filename, ".xlsx") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file format. only .csv and .xlsx files are allowed"})
		return
	}

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	if n > 0 {
		mimeType := http.DetectContentType(buf[:n])
		isCSV := strings.HasPrefix(mimeType, "text/") || strings.Contains(mimeType, "csv") || strings.Contains(mimeType, "text")
		isXLSX := strings.Contains(mimeType, "officedocument") || strings.Contains(mimeType, "zip")
		if !isCSV && !isXLSX {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file content does not match expected format"})
			return
		}
	}
	if seeker, ok := file.(io.Seeker); ok {
		_, _ = seeker.Seek(0, io.SeekStart)
	}

	result, err := h.importEng.Preview(c.Request.Context(), module, header.Filename, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Confirm(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	userID, _ := c.Get("userID")
	userIDInt, _ := userID.(int)

	storeID := 0
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(*int); ok && v != nil {
			storeID = *v
		}
	}

	jobID, err := h.importEng.StartImport(c.Request.Context(), token, userIDInt, storeID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"job_id": jobID, "status": "importing"})
}

func (h *Handler) GetProgress(c *gin.Context) {
	var req struct {
		JobID int64 `uri:"jobId"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	p, err := h.progressEng.GetProgress(c.Request.Context(), req.JobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *Handler) CancelImport(c *gin.Context) {
	var req struct {
		JobID int64 `uri:"jobId"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	if err := h.progressEng.RequestCancel(c.Request.Context(), req.JobID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancellation requested"})
}

func (h *Handler) ListImportHistory(c *gin.Context) {
	module := c.Param("module")
	_, err := h.schemaReg.Get(module)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	jobs, err := h.progressEng.ListJobs(c.Request.Context(), module, 50)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) Export(c *gin.Context) {
	module := c.Param("module")
	s, err := h.schemaReg.Get(module)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	format := c.DefaultQuery("format", "csv")
	expFormat := export.Format(format)
	if expFormat != export.FormatCSV && expFormat != export.FormatXLSX {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format must be 'csv' or 'xlsx'"})
		return
	}

	adapter, err := h.adapterReg.Get(module)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	data, err := adapter.Repository().ExportData(c.Request.Context(), s)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	switch expFormat {
	case export.FormatCSV:
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, sanitizeFilename(module)))
		c.Header("Content-Type", "text/csv; charset=utf-8")
	case export.FormatXLSX:
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, sanitizeFilename(module)))
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	}

	if err := h.exportEng.Export(c.Writer, s, data, expFormat); err != nil {
		shared.InternalError(c, err)
		return
	}
}

func sanitizeFilename(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
}

func (h *Handler) GetImportDetail(c *gin.Context) {
	module := c.Param("module")
	if _, err := h.schemaReg.Get(module); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	jobID, err := strconv.ParseInt(c.Param("jobId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	jobProgress, err := h.progressEng.GetProgress(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if h.historyStore == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "history store not available"})
		return
	}

	snapshot, err := h.historyStore.GetSnapshot(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"progress": jobProgress,
		"snapshot": snapshot,
	})
}

func (h *Handler) GetImportRows(c *gin.Context) {
	module := c.Param("module")
	if _, err := h.schemaReg.Get(module); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	jobID, err := strconv.ParseInt(c.Param("jobId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	if h.historyStore == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "history store not available"})
		return
	}

	rows, err := h.historyStore.GetRows(c.Request.Context(), jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "rows not found"})
			return
		}
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"rows": rows})
}

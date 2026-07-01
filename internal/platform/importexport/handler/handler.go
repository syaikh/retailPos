package handler

import (
	"fmt"
	"net/http"

	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/export"
	importer "retail-pos-system/internal/platform/importexport/import"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"

	"github.com/gin-gonic/gin"
)

var modulePerms = map[string]string{
	"categories:import": "category:import",
	"categories:export": "category:export",
	"brands:import":     "product:import",
	"brands:export":     "product:export",
	"uoms:import":       "product:import",
	"uoms:export":       "product:export",
	"customers:import":  "customer:import",
	"customers:export":  "customer:export",
	"products:import":   "product:import",
	"products:export":   "product:export",
}

type Handler struct {
	schemaReg   *schema.Registry
	adapterReg  *importexport.AdapterRegistry
	importEng   *importer.Engine
	exportEng   *export.Engine
	templateEng *template.Engine
	progressEng *progress.Engine
	permFunc    func(string) gin.HandlerFunc
}

func NewHandler(schemaReg *schema.Registry, adapterReg *importexport.AdapterRegistry, importEng *importer.Engine, exportEng *export.Engine, templateEng *template.Engine, progressEng *progress.Engine) *Handler {
	return &Handler{
		schemaReg:   schemaReg,
		adapterReg:  adapterReg,
		importEng:   importEng,
		exportEng:   exportEng,
		templateEng: templateEng,
		progressEng: progressEng,
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
		r.GET("/export/:module", h.requirePerm("export"), h.Export)
	}
}

func (h *Handler) requirePerm(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		module := c.Param("module")
		key := module + ":" + action
		permStr, ok := modulePerms[key]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission not defined for module"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	refData, err := adapter.Repository().LoadReferences(c.Request.Context(), s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("load references: %v", err)})
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

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-template.xlsx"`, module))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := h.templateEng.Generate(s, refDataFlat, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("generate template: %v", err)})
	}
}

func (h *Handler) Preview(c *gin.Context) {
	module := c.Param("module")
	_, err := h.schemaReg.Get(module)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unknown module: %s", module)})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

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

	result, err := h.importEng.Execute(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
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

	_ = h.progressEng.SetStatus(c.Request.Context(), req.JobID, progress.StatusCancelled)

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	data, err := adapter.Repository().ExportData(c.Request.Context(), s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("export data: %v", err)})
		return
	}

	switch expFormat {
	case export.FormatCSV:
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, module))
		c.Header("Content-Type", "text/csv; charset=utf-8")
	case export.FormatXLSX:
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, module))
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	}

	if err := h.exportEng.Export(c.Writer, s, data, expFormat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("export: %v", err)})
	}
}

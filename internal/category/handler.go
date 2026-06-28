package category

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/importutil"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/categories", h.ListCategories)
	r.GET("/categories/manage", auth, perm("category:read"), h.ListCategoriesManagement)
	r.POST("/categories", auth, perm("category:create"), h.CreateCategoryHandler)
	r.PUT("/categories/:id", auth, perm("category:update"), h.UpdateCategoryHandler)
	r.DELETE("/categories/:id", auth, perm("category:delete"), h.DeleteCategoryHandler)
	r.GET("/categories/export", auth, perm("category:export"), h.ExportCategories)
	r.POST("/categories/import", auth, perm("category:import"), h.ImportCategories)
}

func (h *Handler) ListCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if categories == nil {
		categories = []Category{}
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}

func (h *Handler) ListCategoriesManagement(c *gin.Context) {
	limit := 50
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 {
		limit = l
	}
	if limit > 200 {
		limit = 200
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	search := c.Query("search")

	categories, total, err := h.svc.GetAllCategories(c.Request.Context(), limit, offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories, "total": total})
}

func (h *Handler) CreateCategoryHandler(c *gin.Context) {
	var req CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.svc.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": category})
}

func (h *Handler) UpdateCategoryHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req CategoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.svc.UpdateCategory(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": category})
}

func (h *Handler) DeleteCategoryHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteCategory(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ExportCategories(c *gin.Context) {
	format := c.Query("format")

	categories, err := h.svc.GetAllCategoriesForExport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch categories"})
		return
	}
	if categories == nil {
		categories = []Category{}
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "categories-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Categories"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Name", "Slug", "Description", "IsActive"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "D1", headerStyle)

		for i, cat := range categories {
			r := i + 2
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), cat.Name)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), cat.Slug)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), cat.Description)
			isActive := "false"
			if cat.IsActive {
				isActive = "true"
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), isActive)
		}

		colWidths := []float64{30, 25, 50, 12}
		for i, w := range colWidths {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetColWidth(sheet, col, col, w)
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))

		if err := wb.Write(c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write xlsx"})
			return
		}

	default:
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
		_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

		writer := csv.NewWriter(c.Writer)
		_ = writer.Write([]string{"Name", "Slug", "Description", "IsActive"})
		for _, cat := range categories {
			isActive := "false"
			if cat.IsActive {
				isActive = "true"
			}
			_ = writer.Write([]string{
				cat.Name,
				cat.Slug,
				cat.Description,
				isActive,
			})
		}
		writer.Flush()
	}
}

func (h *Handler) ImportCategories(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	headers, rows, err := importutil.ParseCSV(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nameIdx := importutil.HeaderIndex(headers, "Name")
	slugIdx := importutil.HeaderIndex(headers, "Slug")
	descIdx := importutil.HeaderIndex(headers, "Description")
	activeIdx := importutil.HeaderIndex(headers, "IsActive")

	if nameIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Name' is required"})
		return
	}

	var records []CategoryImportRow
	for i, row := range rows {
		if len(row) < len(headers) {
			continue
		}
		isActive := true
		if activeIdx != -1 {
			val := row[activeIdx]
			isActive = val == "" || val == "true" || val == "1" || val == "yes"
		}
		slug := ""
		if slugIdx != -1 {
			slug = row[slugIdx]
		}
		desc := ""
		if descIdx != -1 {
			desc = row[descIdx]
		}
		records = append(records, CategoryImportRow{
			Row:         i + 2,
			Name:        row[nameIdx],
			Slug:        slug,
			Description: desc,
			IsActive:    isActive,
		})
	}

	result := h.svc.ImportCategories(c.Request.Context(), records)
	c.JSON(http.StatusOK, result)
}

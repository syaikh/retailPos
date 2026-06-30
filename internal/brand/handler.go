package brand

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/importutil"
	"retail-pos-system/internal/shared"

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
	r.POST("/brands", auth, perm("product:create"), h.CreateBrand)
	r.PUT("/brands/:id", auth, perm("product:update"), h.UpdateBrand)
	r.DELETE("/brands/:id", auth, perm("product:delete"), h.DeleteBrand)
	r.GET("/brands/export", auth, perm("product:export"), h.ExportBrands)
	r.POST("/brands/import", auth, perm("product:import"), h.ImportBrands)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/brands", h.ListBrands)
}

func (h *Handler) ListBrands(c *gin.Context) {
	brands, err := h.svc.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brands"})
		return
	}
	if brands == nil {
		brands = []Brand{}
	}
	c.JSON(http.StatusOK, gin.H{"data": brands})
}

func (h *Handler) CreateBrand(c *gin.Context) {
	var req BrandCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	brand, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create brand"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": brand})
}

func (h *Handler) UpdateBrand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid brand id"})
		return
	}

	var req BrandUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	brand, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update brand"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": brand})
}

func (h *Handler) DeleteBrand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid brand id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete brand"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ExportBrands(c *gin.Context) {
	format := c.Query("format")

	brands, err := h.svc.GetAllForExport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brands"})
		return
	}
	if brands == nil {
		brands = []Brand{}
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "brands-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Brands"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Name", "Description", "IsActive"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "C1", headerStyle)

		for i, brand := range brands {
			r := i + 2
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), brand.Name)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), brand.Description)
			isActive := "false"
			if brand.IsActive {
				isActive = "true"
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), isActive)
		}

		colWidths := []float64{30, 50, 12}
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
		_ = shared.WriteCSVRow(writer, []string{"Name", "Description", "IsActive"})
		for _, brand := range brands {
			isActive := "false"
			if brand.IsActive {
				isActive = "true"
			}
			_ = shared.WriteCSVRow(writer, []string{
				brand.Name,
				brand.Description,
				isActive,
			})
		}
		writer.Flush()
	}
}

func (h *Handler) ImportBrands(c *gin.Context) {
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
	descIdx := importutil.HeaderIndex(headers, "Description")
	activeIdx := importutil.HeaderIndex(headers, "IsActive")

	if nameIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Name' is required"})
		return
	}

	var records []BrandImportRow
	for i, row := range rows {
		if len(row) < len(headers) {
			continue
		}
		isActive := true
		if activeIdx != -1 {
			val := row[activeIdx]
			isActive = val == "" || val == "true" || val == "1" || val == "yes"
		}
		desc := ""
		if descIdx != -1 {
			desc = row[descIdx]
		}
		records = append(records, BrandImportRow{
			Row:         i + 2,
			Name:        row[nameIdx],
			Description: desc,
			IsActive:    isActive,
		})
	}

	result := h.svc.Import(c.Request.Context(), records)
	c.JSON(http.StatusOK, result)
}

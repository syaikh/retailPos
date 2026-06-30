package uom

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
	r.POST("/units-of-measure", auth, perm("product:create"), h.CreateUnitOfMeasure)
	r.PUT("/units-of-measure/:id", auth, perm("product:update"), h.UpdateUnitOfMeasure)
	r.DELETE("/units-of-measure/:id", auth, perm("product:delete"), h.DeleteUnitOfMeasure)
	r.GET("/units-of-measure/export", auth, perm("product:export"), h.ExportUnitsOfMeasure)
	r.POST("/units-of-measure/import", auth, perm("product:import"), h.ImportUnitsOfMeasure)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/units-of-measure", h.ListUnitsOfMeasure)
}

func (h *Handler) ListUnitsOfMeasure(c *gin.Context) {
	units, err := h.svc.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch units of measure"})
		return
	}
	if units == nil {
		units = []UnitOfMeasure{}
	}
	c.JSON(http.StatusOK, gin.H{"data": units})
}

func (h *Handler) CreateUnitOfMeasure(c *gin.Context) {
	var req UOMCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uom, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create unit of measure"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": uom})
}

func (h *Handler) UpdateUnitOfMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit of measure id"})
		return
	}

	var req UOMUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uom, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update unit of measure"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": uom})
}

func (h *Handler) DeleteUnitOfMeasure(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid unit of measure id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete unit of measure"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ExportUnitsOfMeasure(c *gin.Context) {
	format := c.Query("format")

	units, err := h.svc.GetAllForExport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch units of measure"})
		return
	}
	if units == nil {
		units = []UnitOfMeasure{}
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "units-of-measure-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "UnitsOfMeasure"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Code", "Name", "Description", "IsActive"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "D1", headerStyle)

		for i, uom := range units {
			r := i + 2
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), uom.Code)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), uom.Name)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), uom.Description)
			isActive := "false"
			if uom.IsActive {
				isActive = "true"
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), isActive)
		}

		colWidths := []float64{15, 25, 50, 12}
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
		_ = shared.WriteCSVRow(writer, []string{"Code", "Name", "Description", "IsActive"})
		for _, uom := range units {
			isActive := "false"
			if uom.IsActive {
				isActive = "true"
			}
			_ = shared.WriteCSVRow(writer, []string{
				uom.Code,
				uom.Name,
				uom.Description,
				isActive,
			})
		}
		writer.Flush()
	}
}

func (h *Handler) ImportUnitsOfMeasure(c *gin.Context) {
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

	codeIdx := importutil.HeaderIndex(headers, "Code")
	nameIdx := importutil.HeaderIndex(headers, "Name")
	descIdx := importutil.HeaderIndex(headers, "Description")
	activeIdx := importutil.HeaderIndex(headers, "IsActive")

	if codeIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Code' is required"})
		return
	}
	if nameIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Name' is required"})
		return
	}

	var records []UOMImportRow
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
		records = append(records, UOMImportRow{
			Row:         i + 2,
			Code:        row[codeIdx],
			Name:        row[nameIdx],
			Description: desc,
			IsActive:    isActive,
		})
	}

	result := h.svc.Import(c.Request.Context(), records)
	c.JSON(http.StatusOK, result)
}

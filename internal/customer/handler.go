package customer

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	r.GET("/customers", auth, perm("customer:read"), h.GetCustomers)
	r.GET("/customers/:id", auth, perm("customer:read"), h.GetCustomerByID)
	r.POST("/customers", auth, perm("customer:create"), h.CreateCustomer)
	r.PUT("/customers/:id", auth, perm("customer:update"), h.UpdateCustomer)
	r.DELETE("/customers/:id", auth, perm("customer:delete"), h.DeleteCustomer)
	r.POST("/customers/bulk/status", auth, perm("customer:update"), h.BulkUpdateCustomerStatus)
	r.POST("/customers/bulk/delete", auth, perm("customer:delete"), h.BulkDeleteCustomers)
	r.GET("/customers/export", auth, perm("customer:export"), h.ExportCustomers)
	r.POST("/customers/import", auth, perm("customer:import"), h.ImportCustomers)
}

func (h *Handler) ExportCustomers(c *gin.Context) {
	format := c.Query("format")

	customers, err := h.svc.GetAllCustomersForExport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customers"})
		return
	}
	if customers == nil {
		customers = []Customer{}
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "customers-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Customers"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"Name", "Phone", "Email", "Address", "Note", "IsActive"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "F1", headerStyle)

		for i, c := range customers {
			r := i + 2
			phone := ""
			if c.Phone != nil {
				phone = *c.Phone
			}
			email := ""
			if c.Email != nil {
				email = *c.Email
			}
			address := ""
			if c.Address != nil {
				address = *c.Address
			}
			note := ""
			if c.Note != nil {
				note = *c.Note
			}
			isActive := "false"
			if c.IsActive {
				isActive = "true"
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), c.Name)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), phone)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), email)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), address)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("E%d", r), note)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("F%d", r), isActive)
		}

		colWidths := []float64{30, 20, 30, 50, 50, 12}
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
		_ = writer.Write([]string{"Name", "Phone", "Email", "Address", "Note", "IsActive"})
		for _, cust := range customers {
			phone := ""
			if cust.Phone != nil {
				phone = *cust.Phone
			}
			email := ""
			if cust.Email != nil {
				email = *cust.Email
			}
			address := ""
			if cust.Address != nil {
				address = *cust.Address
			}
			note := ""
			if cust.Note != nil {
				note = *cust.Note
			}
			isActive := "false"
			if cust.IsActive {
				isActive = "true"
			}
			_ = writer.Write([]string{
				cust.Name,
				phone,
				email,
				address,
				note,
				isActive,
			})
		}
		writer.Flush()
	}
}

func (h *Handler) ImportCustomers(c *gin.Context) {
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
	phoneIdx := importutil.HeaderIndex(headers, "Phone")
	emailIdx := importutil.HeaderIndex(headers, "Email")
	addressIdx := importutil.HeaderIndex(headers, "Address")
	noteIdx := importutil.HeaderIndex(headers, "Note")
	activeIdx := importutil.HeaderIndex(headers, "IsActive")

	if nameIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Name' is required"})
		return
	}

	var records []CustomerImportRow
	for i, row := range rows {
		if len(row) < len(headers) {
			continue
		}
		isActive := true
		if activeIdx != -1 {
			val := row[activeIdx]
			isActive = val == "" || val == "true" || val == "1" || val == "yes"
		}
		phone := ""
		if phoneIdx != -1 {
			phone = row[phoneIdx]
		}
		email := ""
		if emailIdx != -1 {
			email = row[emailIdx]
		}
		address := ""
		if addressIdx != -1 {
			address = row[addressIdx]
		}
		note := ""
		if noteIdx != -1 {
			note = row[noteIdx]
		}
		records = append(records, CustomerImportRow{
			Row:      i + 2,
			Name:     row[nameIdx],
			Phone:    phone,
			Email:    email,
			Address:  address,
			Note:     note,
			IsActive: isActive,
		})
	}

	result := h.svc.ImportCustomers(c.Request.Context(), records)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetCustomers(c *gin.Context) {
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
	search := strings.TrimSpace(c.Query("search"))

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			isActive = &b
		}
	}

	customers, total, err := h.svc.GetAllCustomers(c.Request.Context(), limit, offset, search, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if customers == nil {
		customers = []Customer{}
	}
	c.JSON(http.StatusOK, gin.H{"data": customers, "total": total})
}

func (h *Handler) GetCustomerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	customer, err := h.svc.GetCustomerByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	var req struct {
		Name     string  `json:"name"`
		Phone    string  `json:"phone"`
		Email    string  `json:"email"`
		Address  *string `json:"address"`
		IsActive bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := &Customer{
		Name:     req.Name,
		Phone:    &req.Phone,
		Email:    &req.Email,
		Address:  req.Address,
		IsActive: req.IsActive,
	}

	if err := h.svc.CreateCustomer(c.Request.Context(), customer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": customer})
}

func (h *Handler) UpdateCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name     string  `json:"name"`
		Phone    *string `json:"phone"`
		Email    *string `json:"email"`
		Address  *string `json:"address"`
		IsActive bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := &Customer{
		ID:       id,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		IsActive: req.IsActive,
	}

	if err := h.svc.UpdateCustomer(c.Request.Context(), customer, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *Handler) DeleteCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteCustomer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) BulkUpdateCustomerStatus(c *gin.Context) {
	var req struct {
		IDs      []int `json:"ids"`
		IsActive bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.BulkUpdateCustomersStatus(c.Request.Context(), req.IDs, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) BulkDeleteCustomers(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.BulkDeleteCustomers(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

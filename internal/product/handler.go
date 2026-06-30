package product

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
	r.GET("/products", h.GetProducts)
	r.GET("/products/next-sku", h.GetNextSKU)

	r.GET("/products/:id", auth, h.GetProductByID)
	r.POST("/products", auth, perm("product:create"), h.CreateProduct)
	r.PUT("/products/:id", auth, perm("product:update"), h.UpdateProduct)
	r.DELETE("/products/:id", auth, perm("product:delete"), h.DeleteProduct)
	r.POST("/products/bulk/status", auth, perm("product:update"), h.BulkUpdateProductStatus)
	r.GET("/products/export", auth, perm("product:export"), h.ExportProducts)
	r.POST("/products/import", auth, perm("product:import"), h.ImportProducts)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	r.GET("/tax-classes", h.ListTaxClasses)
	r.GET("/stock-thresholds", h.GetStockThresholds)
}

func (h *Handler) getStoreID(c *gin.Context) *int {
	if sid, exists := c.Get("storeID"); exists {
		if v, ok := sid.(int); ok {
			return &v
		}
	}
	return nil
}

func (h *Handler) GetProducts(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	search := c.Query("search")
	sortBy := c.Query("sortBy")
	sortDir := c.Query("sortDir")
	category := c.Query("category")
	brand := c.Query("brand")

	status := c.Query("status")
	var isActive *bool
	if status == "" {
		if s := c.Query("isActive"); s != "" {
			v := strings.EqualFold(s, "true") || s == "1"
			isActive = &v
		}
	}

	var minPrice, maxPrice *float64
	if mp := c.Query("minPrice"); mp != "" {
		if val, err := strconv.ParseFloat(mp, 64); err == nil {
			minPrice = &val
		}
	}
	if mxp := c.Query("maxPrice"); mxp != "" {
		if val, err := strconv.ParseFloat(mxp, 64); err == nil {
			maxPrice = &val
		}
	}

	var maxStock *int
	if ms := c.Query("maxStock"); ms != "" {
		if val, err := strconv.Atoi(ms); err == nil && val >= 0 {
			maxStock = &val
		}
	}

	storeID := h.getStoreID(c)

	products, total, err := h.svc.GetAllProducts(c.Request.Context(), limit, offset, search, sortBy, sortDir, category, brand, storeID, isActive, minPrice, maxPrice, maxStock, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	if products == nil {
		products = []Product{}
	}
	c.JSON(http.StatusOK, gin.H{"data": products, "total": total})
}

func (h *Handler) GetProductByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	storeID := h.getStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}

	product, err := h.svc.GetProductByID(c.Request.Context(), id, sid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.svc.CreateProduct(c.Request.Context(), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	product.ID = id

	if err := h.svc.UpdateProduct(c.Request.Context(), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	if err := h.svc.DeleteProduct(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) BulkUpdateProductStatus(c *gin.Context) {
	var req struct {
		IDs      []int `json:"ids"`
		IsActive bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no product IDs provided"})
		return
	}

	if err := h.svc.BulkUpdateProductStatus(c.Request.Context(), req.IDs, req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product statuses"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated", "updated_count": len(req.IDs)})
}

func (h *Handler) GetNextSKU(c *gin.Context) {
	sku, err := h.svc.GetNextSKU(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate SKU"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sku})
}

func (h *Handler) ListTaxClasses(c *gin.Context) {
	taxClasses, err := h.svc.GetAllTaxClasses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tax classes"})
		return
	}
	if taxClasses == nil {
		taxClasses = []TaxClass{}
	}
	c.JSON(http.StatusOK, gin.H{"data": taxClasses})
}

func (h *Handler) ExportProducts(c *gin.Context) {
	format := c.Query("format")

	products, err := h.svc.GetAllProductsForExport(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	if products == nil {
		products = []Product{}
	}

	cfg := config.Load()
	now := time.Now().In(cfg.Timezone)
	filename := "products-" + now.Format("2006-01-02")

	switch format {
	case "xlsx":
		wb := excelize.NewFile()
		sheet := "Products"
		_ = wb.SetSheetName("Sheet1", sheet)

		headers := []string{"SKU", "Name", "Barcode", "Category", "Brand", "Price", "Cost", "Stock", "Status", "UnitOfMeasure", "WeightGrams", "Description"}
		for i, hdr := range headers {
			col, _ := excelize.ColumnNumberToName(i + 1)
			_ = wb.SetCellValue(sheet, col+"1", hdr)
		}
		headerStyle, _ := wb.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"7C3AED"}},
		})
		_ = wb.SetCellStyle(sheet, "A1", "L1", headerStyle)

		for i, p := range products {
			r := i + 2
			barcode := ""
			if p.Barcode != nil {
				barcode = *p.Barcode
			}
			category := ""
			if p.CategoryName != nil {
				category = *p.CategoryName
			}
			brand := ""
			if p.BrandName != nil {
				brand = *p.BrandName
			}
			uom := ""
			if p.UnitOfMeasure != nil {
				uom = *p.UnitOfMeasure
			}
			weight := 0
			if p.WeightGrams != nil {
				weight = *p.WeightGrams
			}
			desc := ""
			if p.Description != nil {
				desc = *p.Description
			}
			_ = wb.SetCellValue(sheet, fmt.Sprintf("A%d", r), p.SKU)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("B%d", r), p.Name)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("C%d", r), barcode)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("D%d", r), category)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("E%d", r), brand)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("F%d", r), p.Price)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("G%d", r), p.Cost)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("H%d", r), p.Stock)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("I%d", r), p.Status)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("J%d", r), uom)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("K%d", r), weight)
			_ = wb.SetCellValue(sheet, fmt.Sprintf("L%d", r), desc)
		}

		colWidths := []float64{15, 30, 18, 20, 20, 12, 12, 10, 12, 15, 12, 50}
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
		_ = shared.WriteCSVRow(writer, []string{"SKU", "Name", "Barcode", "Category", "Brand", "Price", "Cost", "Stock", "Status", "UnitOfMeasure", "WeightGrams", "Description"})
		for _, p := range products {
			barcode := ""
			if p.Barcode != nil {
				barcode = *p.Barcode
			}
			category := ""
			if p.CategoryName != nil {
				category = *p.CategoryName
			}
			brand := ""
			if p.BrandName != nil {
				brand = *p.BrandName
			}
			uom := ""
			if p.UnitOfMeasure != nil {
				uom = *p.UnitOfMeasure
			}
			weight := 0
			if p.WeightGrams != nil {
				weight = *p.WeightGrams
			}
			desc := ""
			if p.Description != nil {
				desc = *p.Description
			}
			_ = shared.WriteCSVRow(writer, []string{
				p.SKU,
				p.Name,
				barcode,
				category,
				brand,
				fmt.Sprintf("%d", p.Price),
				fmt.Sprintf("%d", p.Cost),
				fmt.Sprintf("%d", p.Stock),
				p.Status,
				uom,
				fmt.Sprintf("%d", weight),
				desc,
			})
		}
		writer.Flush()
	}
}

func (h *Handler) ImportProducts(c *gin.Context) {
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

	skuIdx := importutil.HeaderIndex(headers, "SKU")
	nameIdx := importutil.HeaderIndex(headers, "Name")
	barcodeIdx := importutil.HeaderIndex(headers, "Barcode")
	categoryIdx := importutil.HeaderIndex(headers, "Category")
	brandIdx := importutil.HeaderIndex(headers, "Brand")
	priceIdx := importutil.HeaderIndex(headers, "Price")
	costIdx := importutil.HeaderIndex(headers, "Cost")
	stockIdx := importutil.HeaderIndex(headers, "Stock")
	statusIdx := importutil.HeaderIndex(headers, "Status")
	uomIdx := importutil.HeaderIndex(headers, "UnitOfMeasure")
	weightIdx := importutil.HeaderIndex(headers, "WeightGrams")
	descIdx := importutil.HeaderIndex(headers, "Description")

	if skuIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'SKU' is required"})
		return
	}
	if nameIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Name' is required"})
		return
	}
	if priceIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "column 'Price' is required"})
		return
	}

	var records []ProductImportRow
	for i, row := range rows {
		if len(row) < len(headers) {
			continue
		}
		price := 0
		if priceIdx != -1 {
			price, _ = strconv.Atoi(row[priceIdx])
		}
		cost := 0
		if costIdx != -1 {
			cost, _ = strconv.Atoi(row[costIdx])
		}
		stock := 0
		if stockIdx != -1 {
			stock, _ = strconv.Atoi(row[stockIdx])
		}
		weight := 0
		if weightIdx != -1 {
			weight, _ = strconv.Atoi(row[weightIdx])
		}
		barcode := ""
		if barcodeIdx != -1 {
			barcode = row[barcodeIdx]
		}
		category := ""
		if categoryIdx != -1 {
			category = row[categoryIdx]
		}
		brand := ""
		if brandIdx != -1 {
			brand = row[brandIdx]
		}
		status := "active"
		if statusIdx != -1 && row[statusIdx] != "" {
			status = row[statusIdx]
		}
		uom := ""
		if uomIdx != -1 {
			uom = row[uomIdx]
		}
		desc := ""
		if descIdx != -1 {
			desc = row[descIdx]
		}
		records = append(records, ProductImportRow{
			Row:           i + 2,
			SKU:           row[skuIdx],
			Name:          row[nameIdx],
			Barcode:       barcode,
			Category:      category,
			Brand:         brand,
			Price:         price,
			Cost:          cost,
			Stock:         stock,
			Status:        status,
			UnitOfMeasure: uom,
			WeightGrams:   weight,
			Description:   desc,
		})
	}

	result := h.svc.ImportProducts(c.Request.Context(), records)
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetStockThresholds(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"warning": 10, "critical": 5})
}

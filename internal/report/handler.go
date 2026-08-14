package report

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	_ "image/png"
)

type Service interface {
	GetDashboardStats(ctx context.Context, storeID int) (*DashboardStats, error)
	GetLiveDashboardStats(ctx context.Context, storeID int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error)
	GetHourlySales(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error)
	GetDailySales(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error)
	GetDualChartData(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (current, previous []ChartDataPoint, err error)
	GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (*PeriodComparison, error)
	GetSalesWeeklyReport(ctx context.Context, storeID int, start, end time.Time) ([]shared.WeeklyReportItem, error)
	GetSalesMonthlyReport(ctx context.Context, storeID int, start, end time.Time) ([]shared.MonthlyReportItem, error)
	GetDualMonthlyReport(ctx context.Context, storeID int, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous []shared.MonthlyReportItem, err error)
	GetAvailableYears(ctx context.Context, storeID int) ([]int, error)
	GetPricingBreakdown(ctx context.Context, start, end time.Time, storeID *int) ([]shared.PricingBreakdownItem, error)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/dashboard/stats", auth, perm(permissions.DashboardView), h.GetDashboardStats)
	r.GET("/dashboard/live", auth, perm(permissions.DashboardView), h.GetLiveDashboardStats)
	r.GET("/dashboard/chart", auth, perm(permissions.ReportView), h.GetSalesChartData)
	r.GET("/dashboard/chart/weekly", auth, perm(permissions.ReportView), h.GetSalesWeeklyReport)
	r.GET("/dashboard/chart/monthly", auth, perm(permissions.ReportView), h.GetSalesMonthlyReport)
	r.GET("/dashboard/comparison", auth, perm(permissions.ReportView), h.GetPeriodComparison)
	r.POST("/dashboard/export", auth, perm(permissions.ReportView), h.ExportDashboard)
	r.GET("/dashboard/years", auth, perm(permissions.ReportView), h.GetAvailableYears)
	r.GET("/dashboard/pricing-breakdown", auth, perm(permissions.ReportView), h.GetPricingBreakdown)
}

// GetDashboardStats godoc
// @Summary Get dashboard statistics
// @Description Get today's revenue, sales, total products, and low stock count
// @Tags reports
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/stats [get]
func (h *Handler) GetDashboardStats(c *gin.Context) {
	sid := shared.GetStoreIDInt(c)
	ctx := c.Request.Context()

	stats, err := h.svc.GetDashboardStats(ctx, sid)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	yesterdayRevenue := stats.TotalRevenue - stats.TodaysRevenue
	if yesterdayRevenue < 0 {
		yesterdayRevenue = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"todays_revenue":    stats.TodaysRevenue,
			"yesterday_revenue": yesterdayRevenue,
			"todays_sales":      stats.TodaysSales,
			"total_products":    stats.TotalProducts,
			"low_stock_count":   stats.LowStockCount,
		},
	})
}

func (h *Handler) GetLiveDashboardStats(c *gin.Context) {
	sid := shared.GetStoreIDInt(c)
	ctx := c.Request.Context()

	todaysRevenue, todaysSales, totalProducts, lowStockCount, err := h.svc.GetLiveDashboardStats(ctx, sid)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"todays_revenue":  todaysRevenue,
			"todays_sales":    todaysSales,
			"total_products":  totalProducts,
			"low_stock_count": lowStockCount,
		},
	})
}

// GetSalesChartData godoc
// @Summary Get sales chart data
// @Description Get daily or hourly sales data for charting. Returns hourly data when startDate equals endDate.
// @Tags reports
// @Accept json
// @Produce json
// @Param startDate query string false "Start date (YYYY-MM-DD, Jakarta time)" default(7 days ago)
// @Param endDate query string false "End date (YYYY-MM-DD, Jakarta time)" default(today)
// @Param prevStart query string false "Previous period start date for comparison"
// @Param prevEnd query string false "Previous period end date for comparison"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/chart [get]
func (h *Handler) GetSalesChartData(c *gin.Context) {
	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}
	ctx := c.Request.Context()

	dr, ok := parseDateRange(c, 7)
	if !ok {
		return
	}
	startDate, endDate := dr.Start, dr.End
	prevStartStr := c.Query("prevStart")
	prevEndStr := c.Query("prevEnd")

	if startDate.Equal(endDate) {
		data, err := h.svc.GetHourlySales(ctx, sid, startDate)
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		if prevStartStr != "" && prevEndStr != "" {
			prevStart, ok := parseDateParam(c, "prevStart")
			if !ok {
				return
			}
			prevData, err := h.svc.GetHourlySales(ctx, sid, prevStart)
			if err != nil {
				shared.InternalError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"current": data, "previous": prevData}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}

	endDate = endDate.Add(24 * time.Hour)

	if prevStartStr != "" && prevEndStr != "" {
		prevStart, ok := parseDateParam(c, "prevStart")
		if !ok {
			return
		}
		prevEnd, ok := parseDateParam(c, "prevEnd")
		if !ok {
			return
		}

		current, previous, err := h.svc.GetDualChartData(ctx, startDate, endDate.Add(-24*time.Hour), prevStart, prevEnd, storeID)
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"current": current, "previous": previous}})
		return
	}

	data, err := h.svc.GetDailySales(ctx, sid, startDate, endDate)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetSalesWeeklyReport(c *gin.Context) {
	sid := shared.GetStoreIDInt(c)
	ctx := c.Request.Context()

	dr, ok := parseDateRange(c, 84)
	if !ok {
		return
	}
	startDate, endDate := dr.Start, dr.End
	endDate = endDate.Add(24 * time.Hour)

	data, err := h.svc.GetSalesWeeklyReport(ctx, sid, startDate, endDate)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Handler) GetSalesMonthlyReport(c *gin.Context) {
	sid := shared.GetStoreIDInt(c)
	ctx := c.Request.Context()

	dr, ok := parseDateRange(c, 365)
	if !ok {
		return
	}
	startDate, endDate := dr.Start, dr.End
	prevStartStr := c.Query("prevStart")
	prevEndStr := c.Query("prevEnd")
	endDate = endDate.Add(24 * time.Hour)

	if prevStartStr != "" && prevEndStr != "" {
		prevStart, ok := parseDateParam(c, "prevStart")
		if !ok {
			return
		}
		prevEnd, ok := parseDateParam(c, "prevEnd")
		if !ok {
			return
		}
		prevEnd = prevEnd.Add(24 * time.Hour)

		current, previous, err := h.svc.GetDualMonthlyReport(ctx, sid, startDate, endDate, prevStart, prevEnd)
		if err != nil {
			shared.InternalError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"current": current, "previous": previous}})
		return
	}

	data, err := h.svc.GetSalesMonthlyReport(ctx, sid, startDate, endDate)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GetPeriodComparison godoc
// @Summary Get period comparison
// @Description Compare revenue, orders, and AOV between current and previous period
// @Tags reports
// @Accept json
// @Produce json
// @Param period query string false "Period type (daily, 7days, weekly, monthly, yearly)" default(daily)
// @Param mode query string false "Comparison mode (realtime, completed, 30days)" default(realtime)
// @Param date query string false "Reference date (YYYY-MM-DD, Jakarta time)" default(today)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dashboard/comparison [get]
func (h *Handler) GetPeriodComparison(c *gin.Context) {
	ctx := c.Request.Context()

	sid := shared.GetStoreIDInt(c)

	period := PeriodType(c.DefaultQuery("period", "daily"))
	mode := c.DefaultQuery("mode", "realtime")
	jakartaLoc := shared.JakartaLocation()
	dateStr := c.DefaultQuery("date", time.Now().In(jakartaLoc).Format("2006-01-02"))
	refDate, err := time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
		return
	}

	var pr PeriodRange
	switch mode {
	case "realtime":
		now := time.Now().In(jakartaLoc)
		pr = getRealtimeRanges(now)
	case "completed":
		pr = getComparisonRanges(period, refDate, true)
	case "30days":
		pr = get30DaysRanges(refDate)
	default:
		pr = getComparisonRanges(period, refDate, false)
	}

	comparison, err := h.svc.GetPeriodComparison(ctx, pr.CurrentStart, pr.CurrentEnd, pr.PreviousStart, pr.PreviousEnd, storeIDPtr(sid))
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": comparison,
		"meta": gin.H{
			"is_partial":     pr.IsPartial,
			"days_in_period": pr.DaysInPeriod,
			"period_type":    period,
			"current_start":  pr.CurrentStart.Format("2006-01-02 15:04:05"),
			"current_end":    pr.CurrentEnd.Format("2006-01-02 15:04:05"),
		},
	})
}

func (h *Handler) ExportDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	chartDataStr := c.PostForm("chartData")
	chartImageStr := c.PostForm("chartImage")
	period := c.PostForm("period")
	mode := c.PostForm("mode")
	jakartaLoc := shared.JakartaLocation()
	dateStr := c.DefaultPostForm("date", time.Now().In(jakartaLoc).Format("2006-01-02"))
	refDate, err := time.ParseInLocation("2006-01-02", dateStr, jakartaLoc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
		return
	}

	periodType := PeriodType(period)
	var pr PeriodRange
	switch mode {
	case "completed":
		pr = getComparisonRanges(periodType, refDate, true)
	default:
		pr = getComparisonRanges(periodType, refDate, false)
	}

	storeID := shared.GetStoreID(c)
	comparison, err := h.svc.GetPeriodComparison(ctx, pr.CurrentStart, pr.CurrentEnd, pr.PreviousStart, pr.PreviousEnd, storeID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	var chartDataPoints []ChartDataPoint
	if chartDataStr != "" {
		if len(chartDataStr) > 1<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chartData too large (max 1MB)"})
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(chartDataStr)
		if err == nil {
			if len(decoded) <= 2<<20 {
				_ = json.Unmarshal(decoded, &chartDataPoints)
			}
		} else {
			if err := json.Unmarshal([]byte(chartDataStr), &chartDataPoints); err != nil {
				chartDataPoints = nil
			}
		}
		if len(chartDataPoints) > 366 {
			chartDataPoints = chartDataPoints[:366]
		}
	}

	startDateStr := c.PostForm("startDate")
	endDateStr := c.PostForm("endDate")
	if startDateStr != "" && endDateStr != "" {
		if sd, err := time.ParseInLocation("2006-01-02", startDateStr, jakartaLoc); err == nil {
			if ed, err := time.ParseInLocation("2006-01-02", endDateStr, jakartaLoc); err == nil {
				if ed.After(sd) {
					pr.CurrentStart = sd
					pr.CurrentEnd = ed
					prevDuration := ed.Sub(sd)
					pr.PreviousStart = sd.Add(-prevDuration - time.Minute)
					pr.PreviousEnd = sd.Add(-time.Minute)
				}
			}
		}
	}

	selectedPeriodType := c.PostForm("selectedPeriodType")
	chartType := c.DefaultPostForm("chartType", "daily")
	currentTimeHour := c.PostForm("currentTimeHour")
	comparisonDateRange := c.PostForm("comparisonDateRange")
	comparisonLabel := c.PostForm("comparisonLabel")

	const maxFormJSON = 1 << 20
	var kpiDataMap map[string]interface{}
	if kpiDataStr := c.PostForm("kpiData"); kpiDataStr != "" {
		if len(kpiDataStr) > maxFormJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kpiData too large"})
			return
		}
		_ = json.Unmarshal([]byte(kpiDataStr), &kpiDataMap)
	}

	var bestPeriodMap map[string]interface{}
	if bestPeriodStr := c.PostForm("bestPeriod"); bestPeriodStr != "" {
		if len(bestPeriodStr) > maxFormJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bestPeriod too large"})
			return
		}
		_ = json.Unmarshal([]byte(bestPeriodStr), &bestPeriodMap)
	}

	var worstPeriodMap map[string]interface{}
	if worstPeriodStr := c.PostForm("worstPeriod"); worstPeriodStr != "" {
		if len(worstPeriodStr) > maxFormJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": "worstPeriod too large"})
			return
		}
		_ = json.Unmarshal([]byte(worstPeriodStr), &worstPeriodMap)
	}

	bestWorstHeading := c.PostForm("bestWorstHeading")

	var sortedRows []map[string]interface{}
	if sortedRowsStr := c.PostForm("sortedRows"); sortedRowsStr != "" {
		if len(sortedRowsStr) > maxFormJSON {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sortedRows too large"})
			return
		}
		_ = json.Unmarshal([]byte(sortedRowsStr), &sortedRows)
	}

	prCurrentStart := pr.CurrentStart.Format("2006-01-02")
	prCurrentEnd := pr.CurrentEnd.Format("2006-01-02")
	isSingleDay := selectedPeriodType == "realtime" || selectedPeriodType == "yesterday" || selectedPeriodType == "daily"
	var fileName string
	if isSingleDay {
		fileName = fmt.Sprintf("revenue-report-%s-%s.xlsx", selectedPeriodType, prCurrentStart)
	} else {
		fileName = fmt.Sprintf("revenue-report-%s-%s-to-%s.xlsx", selectedPeriodType, prCurrentStart, prCurrentEnd)
	}

	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "Report")
	_, _ = f.NewSheet("Data")

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "000000"},
	})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"7c3aed"}, Pattern: 1},
	})
	bodyStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 9},
		Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})

	_ = f.SetCellValue("Report", "A1", "Revenue Report")
	_ = f.SetCellStyle("Report", "A1", "A1", titleStyle)

	periodDesc := buildPeriodDescription(selectedPeriodType, currentTimeHour, pr)
	_ = f.SetCellValue("Report", "A2", fmt.Sprintf("Period: %s", periodDesc))

	granularity := "Hourly"
	if chartType == "daily" {
		granularity = "Daily"
	} else if chartType != "hourly" {
		granularity = "Periodic"
	}
	_ = f.SetCellValue("Report", "A3", fmt.Sprintf("Granularity: %s", granularity))

	if comparisonLabel != "" && comparisonDateRange != "" {
		_ = f.SetCellValue("Report", "A4", fmt.Sprintf("Comparison: %s \u00b7 %s", comparisonLabel, comparisonDateRange))
	}

	_ = f.SetCellValue("Report", "A6", "Metric")
	_ = f.SetCellValue("Report", "B6", "Current Period")
	_ = f.SetCellValue("Report", "C6", "Previous Period")
	_ = f.SetCellValue("Report", "D6", "Change")
	_ = f.SetCellStyle("Report", "A6", "D6", headerStyle)

	summaryRows := buildSummaryRows(chartType, comparison, kpiDataMap)
	currencyFmt := "Rp #,##0"
	numFmt := "#,##0"
	pctFmt := "0.0%"
	for i, row := range summaryRows {
		label := row[0].(string)
		_ = f.SetCellValue("Report", fmt.Sprintf("A%d", i+7), label)
		_ = f.SetCellStyle("Report", fmt.Sprintf("A%d", i+7), "A"+fmt.Sprintf("%d", i+7), bodyStyle)

		bVal := row[1]
		cVal := row[2]
		dVal := row[3]
		bFmt := currencyFmt
		cFmt := currencyFmt
		dFmt := pctFmt
		if label == "Orders" {
			bFmt = numFmt
			cFmt = numFmt
		}
		_ = f.SetCellValue("Report", fmt.Sprintf("B%d", i+7), bVal)
		_ = f.SetCellValue("Report", fmt.Sprintf("C%d", i+7), cVal)
		_ = f.SetCellValue("Report", fmt.Sprintf("D%d", i+7), dVal)
		bStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Size: 9},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			CustomNumFmt: &bFmt,
		})
		cStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Size: 9},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			CustomNumFmt: &cFmt,
		})
		dStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Size: 9},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			CustomNumFmt: &dFmt,
		})
		_ = f.SetCellStyle("Report", fmt.Sprintf("B%d", i+7), "B"+fmt.Sprintf("%d", i+7), bStyle)
		_ = f.SetCellStyle("Report", fmt.Sprintf("C%d", i+7), "C"+fmt.Sprintf("%d", i+7), cStyle)
		_ = f.SetCellStyle("Report", fmt.Sprintf("D%d", i+7), "D"+fmt.Sprintf("%d", i+7), dStyle)
	}

	textRow := 28
	if bestPeriodMap != nil {
		bestTotal := 0.0
		if v, ok := bestPeriodMap["total"].(float64); ok {
			bestTotal = v
		}
		bestLabel := getPeriodLabelFromMap(bestPeriodMap)
		if chartType == "hourly" {
			if hour, ok := bestPeriodMap["hour"].(float64); ok {
				startH := int(hour)
				endH := startH + 1
				bestLabel = fmt.Sprintf("%02d:00 - %02d:00", startH, endH)
			}
		}
		_ = f.SetCellValue("Report", fmt.Sprintf("A%d", textRow), fmt.Sprintf("Best %s:", bestWorstHeading))
		_ = f.SetCellValue("Report", fmt.Sprintf("B%d", textRow), bestLabel)
		_ = f.SetCellValue("Report", fmt.Sprintf("C%d", textRow), bestTotal)
		bwStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Size: 9},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})
		bwNumStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Size: 9},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
			CustomNumFmt: &currencyFmt,
		})
		_ = f.SetCellStyle("Report", fmt.Sprintf("A%d", textRow), fmt.Sprintf("B%d", textRow), bwStyle)
		_ = f.SetCellStyle("Report", fmt.Sprintf("C%d", textRow), fmt.Sprintf("C%d", textRow), bwNumStyle)
		textRow++
	}
	if worstPeriodMap != nil {
		worstTotal := 0.0
		if v, ok := worstPeriodMap["total"].(float64); ok {
			worstTotal = v
		}
		bestTotal := 0.0
		if bestPeriodMap != nil {
			if v, ok := bestPeriodMap["total"].(float64); ok {
				bestTotal = v
			}
		}
		if worstTotal != bestTotal {
			worstLabel := getPeriodLabelFromMap(worstPeriodMap)
			if chartType == "hourly" {
				if hour, ok := worstPeriodMap["hour"].(float64); ok {
					startH := int(hour)
					endH := startH + 1
					worstLabel = fmt.Sprintf("%02d:00 - %02d:00", startH, endH)
				}
			}
			_ = f.SetCellValue("Report", fmt.Sprintf("A%d", textRow), fmt.Sprintf("Worst %s:", bestWorstHeading))
			_ = f.SetCellValue("Report", fmt.Sprintf("B%d", textRow), worstLabel)
			_ = f.SetCellValue("Report", fmt.Sprintf("C%d", textRow), worstTotal)
			bwStyle, _ := f.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Size: 9},
				Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
				Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
			})
			bwNumStyle, _ := f.NewStyle(&excelize.Style{
				Font:      &excelize.Font{Size: 9},
				Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
				Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
				CustomNumFmt: &currencyFmt,
			})
			_ = f.SetCellStyle("Report", fmt.Sprintf("A%d", textRow), fmt.Sprintf("B%d", textRow), bwStyle)
			_ = f.SetCellStyle("Report", fmt.Sprintf("C%d", textRow), fmt.Sprintf("C%d", textRow), bwNumStyle)
			textRow++
			if chartType == "hourly" {
				italicStyle, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Size: 9, Italic: true},
					Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
					Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
				})
				_ = f.SetCellValue("Report", fmt.Sprintf("A%d", textRow), "(zero-revenue hours excluded)")
				_ = f.SetCellStyle("Report", fmt.Sprintf("A%d", textRow), fmt.Sprintf("A%d", textRow), italicStyle)
				textRow++
			}
		}
	}

	if len(sortedRows) > 0 {
		hasOrders := false
		for _, r := range sortedRows {
			if orderCount, ok := r["orderCount"]; ok && orderCount != nil {
				hasOrders = true
				break
			}
		}

		dataCurrencyFmt := "Rp #,##0"
		dataPctFmt := "0.0%"
		dataNumFmt := "#,##0"

		dataBodyStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Size: 9},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})
		dataColFmts := []string{"", dataCurrencyFmt, dataCurrencyFmt, dataPctFmt}
		if hasOrders {
			dataColFmts = append(dataColFmts, dataNumFmt)
		}
		dataColStyles := make([]int, len(dataColFmts))
		for colIdx, fmtStr := range dataColFmts {
			if fmtStr == "" {
				dataColStyles[colIdx] = dataBodyStyle
			} else {
				align := "right"
				if colIdx == 0 {
					align = "left"
				}
				styleID, _ := f.NewStyle(&excelize.Style{
					Font:      &excelize.Font{Size: 9},
					Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
					Alignment: &excelize.Alignment{Horizontal: align, Vertical: "center"},
					CustomNumFmt: &fmtStr,
				})
				dataColStyles[colIdx] = styleID
			}
		}

		headers := []string{"Period", "Revenue (Rp)", "Prev Period (Rp)", "Change %"}
		if hasOrders {
			headers = append(headers, "Orders")
		}
		for col, h := range headers {
			_ = f.SetCellValue("Data", fmt.Sprintf("%s1", string(rune('A'+col))), h)
			_ = f.SetCellStyle("Data", fmt.Sprintf("%s1", string(rune('A'+col))), fmt.Sprintf("%s1", string(rune('A'+col))), headerStyle)
		}

		for i, row := range sortedRows {
			col := 0
			_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), row["period"])
			_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), dataColStyles[col])
			col++

			if revenue, ok := row["revenue"].(float64); ok {
				_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), revenue)
			} else {
				_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), 0)
			}
			_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), dataColStyles[col])
			col++

		prevRevenue := 0.0
		hasPrevRevenue := false
		if pv, ok := row["prevRevenue"]; ok && pv != nil {
			if fv, ok := pv.(float64); ok && fv > 0 {
				prevRevenue = fv
				hasPrevRevenue = true
			}
		}
		_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), prevRevenue)
		_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), dataColStyles[col])
		col++

		change := 0.0
		if hasPrevRevenue {
			if rev, ok := row["revenue"].(float64); ok {
				change = (rev - prevRevenue) / prevRevenue
			}
		}
		_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), change)
			_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), dataColStyles[col])
			col++

			if hasOrders {
				if orderCount, ok := row["orderCount"]; ok && orderCount != nil {
					_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), orderCount)
				} else {
					_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), 0)
				}
				_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), fmt.Sprintf("%s%d", string(rune('A'+col)), i+2), dataColStyles[col])
				col++
			}
		}

		totalRow := len(sortedRows) + 2
		totalCol := 0
		_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), "TOTAL")
		_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), dataColStyles[totalCol])
		totalCol++

		tRev := 0.0
		for _, r := range sortedRows {
			if v, ok := r["revenue"].(float64); ok {
				tRev += v
			}
		}
		_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), tRev)
		_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), dataColStyles[totalCol])
		totalCol++

		tPrev := 0.0
		for _, r := range sortedRows {
			if v, ok := r["prevRevenue"].(float64); ok {
				tPrev += v
			}
		}
		_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), tPrev)
		_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), dataColStyles[totalCol])
		totalCol++

		if tPrev > 0 {
			tChg := (tRev - tPrev) / tPrev
			_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), tChg)
		} else {
			_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), 0)
		}
		_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), dataColStyles[totalCol])
		totalCol++

		if hasOrders {
			tOrders := 0
			for _, r := range sortedRows {
				if v, ok := r["orderCount"].(float64); ok {
					tOrders += int(v)
				}
			}
			_ = f.SetCellValue("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), tOrders)
			_ = f.SetCellStyle("Data", fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), dataColStyles[totalCol])
		}

		totalRowStyle, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"1e293b"}, Pattern: 1},
			Border:    []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}},
			Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		})
		_ = f.SetCellStyle("Data", fmt.Sprintf("A%d", totalRow), fmt.Sprintf("%s%d", string(rune('A'+totalCol)), totalRow), totalRowStyle)

		_ = f.SetColWidth("Data", "A", "A", 18)
		_ = f.SetColWidth("Data", "B", "B", 18)
		_ = f.SetColWidth("Data", "C", "C", 18)
		_ = f.SetColWidth("Data", "D", "D", 14)
		if hasOrders {
			_ = f.SetColWidth("Data", "E", "E", 12)
		}
	}

	_ = f.SetColWidth("Report", "A", "A", 22)
	_ = f.SetColWidth("Report", "B", "B", 18)
	_ = f.SetColWidth("Report", "C", "C", 18)
	_ = f.SetColWidth("Report", "D", "D", 14)

	if chartImageStr != "" {
		if len(chartImageStr) > 3<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chart image too large"})
			return
		}
		raw, err := base64.StdEncoding.DecodeString(chartImageStr)
		if err == nil && len(raw) > 0 {
			if len(raw) > 2<<20 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "chart image too large"})
				return
			}
			_ = f.AddPictureFromBytes("Report", "A12", &excelize.Picture{
				Extension: ".png",
				File:      raw,
			})
		}
	}

	f.SetActiveSheet(0)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	if err := f.Write(c.Writer); err != nil {
		shared.InternalError(c, err)
		return
	}
}

func buildPeriodDescription(selectedPeriodType string, currentTimeHour string, pr PeriodRange) string {
	formatDate := func(d time.Time) string {
		return fmt.Sprintf("%02d %s %d", d.Day(), d.Month().String()[:3], d.Year())
	}
	s := formatDate(pr.CurrentStart)
	e := formatDate(pr.CurrentEnd)
	switch selectedPeriodType {
	case "realtime":
		return fmt.Sprintf("Real-time (00:00 - %s)", currentTimeHour)
	case "yesterday":
		return fmt.Sprintf("Yesterday · %s", s)
	case "7days":
		return fmt.Sprintf("7 Days · %s - %s", s, e)
	case "30days":
		return fmt.Sprintf("30 Days · %s - %s", s, e)
	case "daily":
		return fmt.Sprintf("Daily · %s", s)
	case "weekly":
		return fmt.Sprintf("Weekly · %s - %s", s, e)
	case "monthly":
		return fmt.Sprintf("Monthly · %s - %s", s, e)
	case "yearly":
		return fmt.Sprintf("Yearly · %s - %s", s, e)
	default:
		return fmt.Sprintf("%s - %s", s, e)
	}
}

func buildSummaryRows(chartType string, comparison *PeriodComparison, kpiDataMap map[string]interface{}) [][]interface{} {
	getFloat := func(key string) float64 {
		if v, ok := kpiDataMap[key]; ok {
			if f, ok := v.(float64); ok {
				return f
			}
		}
		return 0
	}

	switch chartType {
	case "hourly":
		return [][]interface{}{
			{"Revenue (RP)", getFloat("totalRevenue"), float64(comparison.PreviousRevenue), pct(getFloat("totalRevenue"), float64(comparison.PreviousRevenue))},
			{"Orders", getFloat("totalOrders"), float64(comparison.PreviousOrders), pct(getFloat("totalOrders"), float64(comparison.PreviousOrders))},
			{"Avg Order Value (RP)", getFloat("avgOrderValue"), float64(comparison.PreviousAOV), pct(getFloat("avgOrderValue"), float64(comparison.PreviousAOV))},
			{"Peak Revenue Hour (RP)", getFloat("peakRevenueHour"), float64(comparison.PreviousPeakRevenue), pct(getFloat("peakRevenueHour"), float64(comparison.PreviousPeakRevenue))},
		}
	case "yearly":
		return [][]interface{}{
			{"Revenue (RP)", getFloat("totalRevenue"), float64(comparison.PreviousRevenue), pct(getFloat("totalRevenue"), float64(comparison.PreviousRevenue))},
			{"Orders", getFloat("totalOrders"), float64(comparison.PreviousOrders), pct(getFloat("totalOrders"), float64(comparison.PreviousOrders))},
			{"Avg Order Value (RP)", getFloat("avgOrderValue"), float64(comparison.PreviousAOV), pct(getFloat("avgOrderValue"), float64(comparison.PreviousAOV))},
			{"Peak Revenue Month (RP)", getFloat("peakRevenueMonth"), float64(comparison.PreviousPeakRevenueMonth), pct(getFloat("peakRevenueMonth"), float64(comparison.PreviousPeakRevenueMonth))},
			{"Avg. Revenue / Month (RP)", getFloat("revenuePerDay") * 30, float64(comparison.PreviousRevenuePerDay) * 30, pct(getFloat("revenuePerDay")*30, float64(comparison.PreviousRevenuePerDay)*30)},
		}
	default:
		return [][]interface{}{
			{"Revenue (RP)", getFloat("totalRevenue"), float64(comparison.PreviousRevenue), pct(getFloat("totalRevenue"), float64(comparison.PreviousRevenue))},
			{"Orders", getFloat("totalOrders"), float64(comparison.PreviousOrders), pct(getFloat("totalOrders"), float64(comparison.PreviousOrders))},
			{"Avg Order Value (RP)", getFloat("avgOrderValue"), float64(comparison.PreviousAOV), pct(getFloat("avgOrderValue"), float64(comparison.PreviousAOV))},
			{"Revenue per Day (RP)", getFloat("revenuePerDay"), float64(comparison.PreviousRevenuePerDay), pct(getFloat("revenuePerDay"), float64(comparison.PreviousRevenuePerDay))},
		}
	}
}

func pct(cur, prev float64) float64 {
	if prev == 0 {
		if cur > 0 {
			return 1
		}
		return 0
	}
	return (cur - prev) / prev
}

func getPeriodLabelFromMap(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	if hour, ok := m["hour"].(float64); ok {
		return fmt.Sprintf("%02d:00", int(hour))
	}
	if date, ok := m["date"].(string); ok && date != "" {
		return date
	}
	if monthStart, ok := m["month_start"].(string); ok && monthStart != "" {
		return monthStart
	}
	if label, ok := m["label"].(string); ok && label != "" {
		return label
	}
	return ""
}

func (h *Handler) GetAvailableYears(c *gin.Context) {
	sid := shared.GetStoreIDInt(c)
	ctx := c.Request.Context()

	years, err := h.svc.GetAvailableYears(ctx, sid)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": years})
}

func (h *Handler) GetPricingBreakdown(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	jakartaLoc := shared.JakartaLocation()
	now := time.Now().In(jakartaLoc)

	var start, end time.Time
	if startStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", startStr, jakartaLoc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date"})
			return
		}
		start = parsed
	} else {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, jakartaLoc)
	}
	if endStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", endStr, jakartaLoc)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date"})
			return
		}
		end = parsed.Add(24 * time.Hour)
	} else {
		end = now.Add(24 * time.Hour)
	}

	storeID := shared.GetStoreID(c)

	breakdown, err := h.svc.GetPricingBreakdown(c.Request.Context(), start, end, storeID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": breakdown})
}

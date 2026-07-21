package report

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ReportService interface {
	GetDashboardStats(ctx context.Context, storeID int) (*DashboardStats, error)
	GetLiveDashboardStats(ctx context.Context, storeID int) (todaysRevenue, todaysSales, totalProducts, lowStockCount int, err error)
	GetHourlySales(ctx context.Context, storeID int, date time.Time) ([]ChartDataPoint, error)
	GetDailySales(ctx context.Context, storeID int, start, end time.Time) ([]ChartDataPoint, error)
	GetDualChartData(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (current, previous []ChartDataPoint, err error)
	GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time, storeID *int) (*PeriodComparison, error)
	GetSalesWeeklyReport(ctx context.Context, storeID int, start, end time.Time) ([]WeeklyReportItem, error)
	GetSalesMonthlyReport(ctx context.Context, storeID int, start, end time.Time) ([]MonthlyReportItem, error)
	GetDualMonthlyReport(ctx context.Context, storeID int, currentStart, currentEnd, previousStart, previousEnd time.Time) (current, previous []MonthlyReportItem, err error)
	GetAvailableYears(ctx context.Context, storeID int) ([]int, error)
	GetPricingBreakdown(ctx context.Context, start, end time.Time, storeID *int) ([]PricingBreakdownItem, error)
}

type Handler struct {
	svc ReportService
}

func NewHandler(svc ReportService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/dashboard/stats", auth, perm("dashboard.view"), h.GetDashboardStats)
	r.GET("/dashboard/live", auth, perm("dashboard.view"), h.GetLiveDashboardStats)
	r.GET("/dashboard/chart", auth, perm("report.view"), h.GetSalesChartData)
	r.GET("/dashboard/chart/weekly", auth, perm("report.view"), h.GetSalesWeeklyReport)
	r.GET("/dashboard/chart/monthly", auth, perm("report.view"), h.GetSalesMonthlyReport)
	r.GET("/dashboard/comparison", auth, perm("report.view"), h.GetPeriodComparison)
	r.POST("/dashboard/export", auth, perm("report.view"), h.ExportDashboard)
	r.GET("/dashboard/years", auth, perm("report.view"), h.GetAvailableYears)
	r.GET("/dashboard/pricing-breakdown", auth, perm("report.view"), h.GetPricingBreakdown)
}

type PeriodType string

const (
	PeriodDaily   PeriodType = "daily"
	Period7Days   PeriodType = "7days"
	PeriodWeekly  PeriodType = "weekly"
	PeriodMonthly PeriodType = "monthly"
	PeriodYearly  PeriodType = "yearly"
)

type PeriodRange struct {
	CurrentStart  time.Time
	CurrentEnd    time.Time
	PreviousStart time.Time
	PreviousEnd   time.Time
	IsPartial     bool
	DaysInPeriod  int
}

func getComparisonRanges(
	periodType PeriodType,
	referenceDate time.Time,
	completedMode bool,
) PeriodRange {
	cfg := config.Load()
	refDate := time.Date(referenceDate.Year(), referenceDate.Month(),
		referenceDate.Day(), 0, 0, 0, 0, cfg.Timezone)

	var pr PeriodRange

	switch periodType {
	case PeriodDaily:
		pr = getDailyRanges(refDate, completedMode)
	case Period7Days:
		pr = get7DaysRanges(refDate, completedMode)
	case PeriodWeekly:
		pr = getWeeklyRanges(refDate, completedMode)
	case PeriodMonthly:
		pr = getMonthlyRanges(refDate, completedMode)
	case PeriodYearly:
		pr = getYearlyRanges(refDate, completedMode)
	default:
		pr = getDailyRanges(refDate, completedMode)
	}

	pr.DaysInPeriod = int(pr.CurrentEnd.Sub(pr.CurrentStart).Hours() / 24)
	pr.IsPartial = isPeriodIncomplete(periodType, refDate) && (periodType == PeriodWeekly || periodType == PeriodMonthly || completedMode)

	return pr
}

func getDailyRanges(refDate time.Time, completedMode bool) PeriodRange {
	if completedMode {
		sevenDaysAgo := refDate.AddDate(0, 0, -7)
		return PeriodRange{
			CurrentStart:  refDate,
			CurrentEnd:    refDate.AddDate(0, 0, 1),
			PreviousStart: sevenDaysAgo,
			PreviousEnd:   sevenDaysAgo.AddDate(0, 0, 1),
		}
	}

	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -6),
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -13),
		PreviousEnd:   refDate.AddDate(0, 0, -6),
	}
}

func get7DaysRanges(refDate time.Time, completedMode bool) PeriodRange {
	if completedMode {
		weekday := refDate.Weekday()
		daysSinceMonday := int(weekday - time.Monday)
		if weekday == time.Sunday {
			daysSinceMonday = 6
		}
		weekStart := refDate.AddDate(0, 0, -daysSinceMonday)
		return PeriodRange{
			CurrentStart:  weekStart.AddDate(0, 0, -7),
			CurrentEnd:    weekStart,
			PreviousStart: weekStart.AddDate(0, 0, -14),
			PreviousEnd:   weekStart.AddDate(0, 0, -7),
		}
	}
	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -6),
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -13),
		PreviousEnd:   refDate.AddDate(0, 0, -6),
	}
}

func get30DaysRanges(refDate time.Time) PeriodRange {
	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -29),
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -59),
		PreviousEnd:   refDate.AddDate(0, 0, -29),
	}
}

func getWeeklyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfWeek := refDate.AddDate(0, 0, -int(refDate.Weekday()-time.Monday))
	if refDate.Weekday() == time.Sunday {
		startOfWeek = refDate.AddDate(0, 0, -6)
	}

	if completedMode {
		endOfWeek := startOfWeek.AddDate(0, 0, 7)
		return PeriodRange{
			CurrentStart:  startOfWeek.AddDate(0, 0, -7),
			CurrentEnd:    endOfWeek,
			PreviousStart: startOfWeek.AddDate(0, 0, -14),
			PreviousEnd:   startOfWeek.AddDate(0, 0, -7),
		}
	}

	daysElapsed := int(refDate.Sub(startOfWeek).Hours()/24) + 1
	previousStart := startOfWeek.AddDate(0, 0, -7)

	return PeriodRange{
		CurrentStart:  startOfWeek,
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: previousStart,
		PreviousEnd:   previousStart.AddDate(0, 0, daysElapsed),
	}
}

func getRealtimeRanges(refDate time.Time) PeriodRange {
	currentHour := refDate.Hour()
	currentPeriodEnd := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), currentHour+1, 0, 0, 0, refDate.Location())

	yesterdayStart := refDate.AddDate(0, 0, -1)
	yesterdaySamePeriodEnd := time.Date(yesterdayStart.Year(), yesterdayStart.Month(), yesterdayStart.Day(), currentHour+1, 0, 0, 0, yesterdayStart.Location())

	return PeriodRange{
		CurrentStart:  time.Date(refDate.Year(), refDate.Month(), refDate.Day(), 0, 0, 0, 0, refDate.Location()),
		CurrentEnd:    currentPeriodEnd,
		PreviousStart: time.Date(yesterdayStart.Year(), yesterdayStart.Month(), yesterdayStart.Day(), 0, 0, 0, 0, yesterdayStart.Location()),
		PreviousEnd:   yesterdaySamePeriodEnd,
	}
}

func getMonthlyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfMonth := time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())

	if completedMode {
		lastMonthEnd := startOfMonth
		lastMonthStart := lastMonthEnd.AddDate(0, -1, 0)
		return PeriodRange{
			CurrentStart:  lastMonthStart,
			CurrentEnd:    lastMonthEnd,
			PreviousStart: lastMonthStart.AddDate(0, -1, 0),
			PreviousEnd:   lastMonthStart,
		}
	}

	daysElapsed := refDate.Day()
	previousStart := startOfMonth.AddDate(0, -1, 0)
	previousEnd := previousStart.AddDate(0, 0, daysElapsed)
	if previousEnd.After(startOfMonth) {
		previousEnd = startOfMonth
	}

	return PeriodRange{
		CurrentStart:  startOfMonth,
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: previousStart,
		PreviousEnd:   previousEnd,
	}
}

func getYearlyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfYear := time.Date(refDate.Year(), 1, 1, 0, 0, 0, 0, refDate.Location())

	if completedMode {
		lastYearEnd := startOfYear
		lastYearStart := lastYearEnd.AddDate(-1, 0, 0)
		return PeriodRange{
			CurrentStart:  lastYearStart,
			CurrentEnd:    lastYearEnd,
			PreviousStart: lastYearStart.AddDate(-1, 0, 0),
			PreviousEnd:   lastYearStart,
		}
	}

	nextDay := refDate.AddDate(0, 0, 1)
	return PeriodRange{
		CurrentStart:  startOfYear,
		CurrentEnd:    nextDay,
		PreviousStart: startOfYear.AddDate(-1, 0, 0),
		PreviousEnd:   nextDay.AddDate(-1, 0, 0),
	}
}

func isPeriodIncomplete(periodType PeriodType, refDate time.Time) bool {
	switch periodType {
	case PeriodWeekly:
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Weekday() != time.Monday
	case PeriodMonthly:
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Month() == refDate.Month()
	case PeriodYearly:
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Year() == refDate.Year() || refDate.Month() != time.December
	default:
		return false
	}
}

type dateRange struct {
	Start time.Time
	End   time.Time
}

func parseDateRange(c *gin.Context, defaultStartDaysAgo int) (dateRange, bool) {
	jakartaLoc := config.Load().Timezone
	now := time.Now().In(jakartaLoc)
	startDateStr := c.DefaultQuery("startDate", now.AddDate(0, 0, -defaultStartDaysAgo).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("endDate", now.Format("2006-01-02"))

	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, jakartaLoc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate"})
		return dateRange{}, false
	}
	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, jakartaLoc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate"})
		return dateRange{}, false
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endDate must not be before startDate"})
		return dateRange{}, false
	}
	if startDate.AddDate(0, 0, 366).Before(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date range must not exceed 366 days"})
		return dateRange{}, false
	}
	return dateRange{Start: startDate, End: endDate}, true
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
	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}
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
	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}
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
			prevStart, err := time.ParseInLocation("2006-01-02", prevStartStr, config.Load().Timezone)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prevStart"})
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
		prevStart, err := time.ParseInLocation("2006-01-02", prevStartStr, config.Load().Timezone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prevStart"})
			return
		}
		prevEnd, err := time.ParseInLocation("2006-01-02", prevEndStr, config.Load().Timezone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prevEnd"})
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
	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}
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
	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}
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
		prevStart, err := time.ParseInLocation("2006-01-02", prevStartStr, config.Load().Timezone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prevStart"})
			return
		}
		prevEnd, err := time.ParseInLocation("2006-01-02", prevEndStr, config.Load().Timezone)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prevEnd"})
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

	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}

	period := PeriodType(c.DefaultQuery("period", "daily"))
	mode := c.DefaultQuery("mode", "realtime")
	jakartaLoc := config.Load().Timezone
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
	period := c.PostForm("period")
	mode := c.PostForm("mode")
	jakartaLoc := config.Load().Timezone
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

	var chartData []ChartDataPoint
	if chartDataStr != "" {
		if len(chartDataStr) > 1<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "chartData too large (max 1MB)"})
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(chartDataStr)
		if err == nil && len(decoded) <= 2<<20 {
			_ = json.Unmarshal(decoded, &chartData)
		}
		if len(chartData) > 366 {
			chartData = chartData[:366]
		}
	}

	f := excelize.NewFile()
	_ = f.SetSheetName("Sheet1", "Dashboard")

	_ = f.SetCellValue("Dashboard", "A1", "Metric")
	_ = f.SetCellValue("Dashboard", "B1", "Current Period")
	_ = f.SetCellValue("Dashboard", "C1", "Previous Period")
	_ = f.SetCellValue("Dashboard", "A2", "Revenue")
	_ = f.SetCellValue("Dashboard", "B2", comparison.CurrentRevenue)
	_ = f.SetCellValue("Dashboard", "C2", comparison.PreviousRevenue)
	_ = f.SetCellValue("Dashboard", "A3", "Orders")
	_ = f.SetCellValue("Dashboard", "B3", comparison.CurrentOrders)
	_ = f.SetCellValue("Dashboard", "C3", comparison.PreviousOrders)
	_ = f.SetCellValue("Dashboard", "A4", "Average Order Value")
	_ = f.SetCellValue("Dashboard", "B4", comparison.CurrentAOV)
	_ = f.SetCellValue("Dashboard", "C4", comparison.PreviousAOV)
	_ = f.SetCellValue("Dashboard", "A5", "Revenue Per Day")
	_ = f.SetCellValue("Dashboard", "B5", comparison.RevenuePerDay)
	_ = f.SetCellValue("Dashboard", "C5", comparison.PreviousRevenuePerDay)
	_ = f.SetCellValue("Dashboard", "A6", "Peak Revenue Hour")
	_ = f.SetCellValue("Dashboard", "B6", comparison.PeakRevenueHour)
	_ = f.SetCellValue("Dashboard", "C6", comparison.PreviousPeakRevenue)

	_, _ = f.NewSheet("Summary")
	_ = f.SetCellValue("Summary", "A1", "Date")
	_ = f.SetCellValue("Summary", "B1", "Revenue")
	for i, d := range chartData {
		_ = f.SetCellValue("Summary", fmt.Sprintf("A%d", i+2), d.Date)
		_ = f.SetCellValue("Summary", fmt.Sprintf("B%d", i+2), d.Total)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=dashboard_export.xlsx")
	if err := f.Write(c.Writer); err != nil {
		shared.InternalError(c, err)
		return
	}
}

func (h *Handler) GetAvailableYears(c *gin.Context) {
	storeID := shared.GetStoreID(c)
	sid := 0
	if storeID != nil {
		sid = *storeID
	}
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

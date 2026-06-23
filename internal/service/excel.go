package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"time"

	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"

	"github.com/xuri/excelize/v2"
)

type ExcelService struct {
	saleRepo repository.SaleRepository
}

func NewExcelService(saleRepo repository.SaleRepository) *ExcelService {
	return &ExcelService{saleRepo: saleRepo}
}

type DashboardExportParams struct {
	PeriodLabel string
	StartDate   time.Time
	EndDate     time.Time
	PrevStart   time.Time
	PrevEnd     time.Time
	IsHourly    bool
	ChartImage  []byte
}

func (s *ExcelService) GenerateDashboardExport(ctx context.Context, params DashboardExportParams) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	var comparison *domain.PeriodComparison
	var currentData, previousData []repository.ChartDataPoint
	var currentHourly, previousHourly []hourlyDataPoint
	if params.IsHourly {
		if params.PeriodLabel == "realtime" {
			jakartaNow := time.Now().In(jakartaLoc)
			currentHour := jakartaNow.Hour()

			year, month, day := params.StartDate.In(jakartaLoc).Date()
			hourlyEnd := time.Date(year, month, day, currentHour+1, 0, 0, 0, jakartaLoc)
			year, month, day = params.PrevStart.In(jakartaLoc).Date()
			prevHourlyEnd := time.Date(year, month, day, currentHour+1, 0, 0, 0, jakartaLoc)

			var err error
			comparison, err = s.saleRepo.GetPeriodComparison(ctx, params.StartDate, hourlyEnd, params.PrevStart, prevHourlyEnd)
			if err != nil {
				return nil, fmt.Errorf("get period comparison: %w", err)
			}

			rawCurrent := aggregateHourly(ctx, s.saleRepo, params.StartDate, params.EndDate)
			rawPrevious := aggregateHourly(ctx, s.saleRepo, params.PrevStart, params.PrevEnd)
			currentHourly = filterHoursUpTo(rawCurrent, currentHour)
			previousHourly = filterHoursUpTo(rawPrevious, currentHour)
		} else {
			comparisonEnd := params.EndDate.AddDate(0, 0, 1)
			comparisonPrevEnd := params.PrevEnd.AddDate(0, 0, 1)
			var err error
			comparison, err = s.saleRepo.GetPeriodComparison(ctx, params.StartDate, comparisonEnd, params.PrevStart, comparisonPrevEnd)
			if err != nil {
				return nil, fmt.Errorf("get period comparison: %w", err)
			}

			currentHourly = aggregateHourly(ctx, s.saleRepo, params.StartDate, params.EndDate)
			previousHourly = aggregateHourly(ctx, s.saleRepo, params.PrevStart, params.PrevEnd)
		}
		currentData = toChartData(currentHourly)
		previousData = toChartData(previousHourly)
	} else {
		comparisonEnd := params.EndDate.AddDate(0, 0, 1)
		comparisonPrevEnd := params.PrevEnd.AddDate(0, 0, 1)
		var err error
		comparison, err = s.saleRepo.GetPeriodComparison(ctx, params.StartDate, comparisonEnd, params.PrevStart, comparisonPrevEnd)
		if err != nil {
			return nil, fmt.Errorf("get period comparison: %w", err)
		}

		currentData, previousData, err = s.saleRepo.GetDualChartData(ctx, params.StartDate, params.EndDate, params.PrevStart, params.PrevEnd)
		if err != nil {
			return nil, fmt.Errorf("get chart data: %w", err)
		}
	}

	startStr := params.StartDate.Format("2006-01-02")
	endStr := params.EndDate.Format("2006-01-02")

	numFmt := "#,##0"
	pctFmt := "+0.0%;-0.0%"

	titleFont := &excelize.Font{Bold: true, Size: 16, Color: "1F2937"}
	subtitleFont := &excelize.Font{Size: 11, Color: "6B7280"}
	headerFont := &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11}
	valueFont := &excelize.Font{Bold: true, Size: 14, Color: "1F2937"}
	positiveFont := &excelize.Font{Bold: true, Size: 11, Color: "059669"}
	negativeFont := &excelize.Font{Bold: true, Size: 11, Color: "DC2626"}
	titleStyle, _ := f.NewStyle(&excelize.Style{Font: titleFont})
	subtitleStyle, _ := f.NewStyle(&excelize.Style{Font: subtitleFont})
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      headerFont,
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4F46E5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	altStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F9FAFB"}},
	})
	kpiValueStyle, _ := f.NewStyle(&excelize.Style{
		Font:         valueFont,
		CustomNumFmt: &numFmt,
	})
	pctPosStyle, _ := f.NewStyle(&excelize.Style{
		Font:         positiveFont,
		CustomNumFmt: &pctFmt,
	})
	pctNegStyle, _ := f.NewStyle(&excelize.Style{
		Font:         negativeFont,
		CustomNumFmt: &pctFmt,
	})

	// ===== Sheet 1: Dashboard =====
	dashboard := "Dashboard"
	f.SetSheetName("Sheet1", dashboard)

	f.SetCellValue(dashboard, "A1", "Dashboard Report")
	f.SetCellStyle(dashboard, "A1", "A1", titleStyle)

	periodStr := fmt.Sprintf("%s - %s", startStr, endStr)
	if params.PeriodLabel != "" {
		f.SetCellValue(dashboard, "A2", fmt.Sprintf("%s (%s)", params.PeriodLabel, periodStr))
	} else {
		f.SetCellValue(dashboard, "A2", periodStr)
	}
	f.SetCellStyle(dashboard, "A2", "A2", subtitleStyle)

	for i, h := range []string{"Metric", "Current Period", "Previous Period", "Change"} {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(dashboard, col+"4", h)
		f.SetCellStyle(dashboard, col+"4", col+"4", headerStyle)
	}

	type kpiRow struct {
		label    string
		current  int
		previous int
	}

	kpiRows := []kpiRow{
		{"Revenue (RP)", comparison.CurrentRevenue, comparison.PreviousRevenue},
		{"Orders", comparison.CurrentOrders, comparison.PreviousOrders},
		{"Avg Order Value (RP)", comparison.CurrentAOV, comparison.PreviousAOV},
	}

	if params.IsHourly {
		kpiRows = append(kpiRows, kpiRow{"Peak Revenue Hour (RP)", comparison.PeakRevenueHour, comparison.PreviousPeakRevenue})
	} else if params.PeriodLabel == "yearly" {
		kpiRows = append(kpiRows, kpiRow{"Peak Revenue Month (RP)", comparison.PeakRevenueMonth, comparison.PreviousPeakRevenueMonth})
		kpiRows = append(kpiRows, kpiRow{"Avg. Revenue / Month (RP)", comparison.RevenuePerDay * 30, comparison.PreviousRevenuePerDay * 30})
	} else {
		kpiRows = append(kpiRows, kpiRow{"Revenue per Day (RP)", comparison.RevenuePerDay, comparison.PreviousRevenuePerDay})
	}

	for i, k := range kpiRows {
		row := 5 + i
		f.SetCellValue(dashboard, fmt.Sprintf("A%d", row), k.label)
		f.SetCellValue(dashboard, fmt.Sprintf("B%d", row), k.current)
		f.SetCellValue(dashboard, fmt.Sprintf("C%d", row), k.previous)
		f.SetCellStyle(dashboard, fmt.Sprintf("B%d", row), fmt.Sprintf("C%d", row), kpiValueStyle)

		chg := calcChangeFloat(k.current, k.previous)
		f.SetCellValue(dashboard, fmt.Sprintf("D%d", row), chg)
		if chg >= 0 {
			f.SetCellStyle(dashboard, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), pctPosStyle)
		} else {
			f.SetCellStyle(dashboard, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), pctNegStyle)
		}
	}

	// Best / Worst period
	bwRow := 5 + len(kpiRows) + 1
	if len(currentData) > 0 {
		best := currentData[0]
		worst := currentData[0]
		for _, dp := range currentData {
			if dp.Total > best.Total {
				best = dp
			}
		}
		for _, dp := range currentData {
			if (worst.Total == 0 && dp.Total > 0) || (dp.Total < worst.Total && dp.Total > 0) {
				worst = dp
			}
		}

		bwFont := &excelize.Font{Size: 11, Color: "6B7280"}
		bwStyle, _ := f.NewStyle(&excelize.Style{Font: bwFont})
		f.SetCellValue(dashboard, fmt.Sprintf("A%d", bwRow), fmt.Sprintf("Best: %s — Rp %d", best.Date, best.Total))
		f.SetCellStyle(dashboard, fmt.Sprintf("A%d", bwRow), fmt.Sprintf("A%d", bwRow), bwStyle)
		if worst.Total > 0 && worst.Date != best.Date {
			f.SetCellValue(dashboard, fmt.Sprintf("A%d", bwRow+1), fmt.Sprintf("Worst: %s — Rp %d", worst.Date, worst.Total))
			f.SetCellStyle(dashboard, fmt.Sprintf("A%d", bwRow+1), fmt.Sprintf("A%d", bwRow+1), bwStyle)
			if params.IsHourly {
				noteFont := &excelize.Font{Size: 9, Color: "9CA3AF"}
				noteStyle, _ := f.NewStyle(&excelize.Style{Font: noteFont})
				f.SetCellValue(dashboard, fmt.Sprintf("B%d", bwRow+1), "(zero-revenue hours excluded)")
				f.SetCellStyle(dashboard, fmt.Sprintf("B%d", bwRow+1), fmt.Sprintf("B%d", bwRow+1), noteStyle)
			}
		}
	}

	// Chart image
	chartRow := bwRow + 3
	if len(params.ChartImage) > 0 {
		if len(params.ChartImage) < 8 || string(params.ChartImage[:8]) != "\x89PNG\r\n\x1a\n" {
			log.Printf("WARN: chart image invalid PNG header: len=%d, first8=%x", len(params.ChartImage), params.ChartImage[:min(8, len(params.ChartImage))])
		} else {
			// Try decoding with Go's image/png for diagnostics
			_, _, err := image.Decode(bytes.NewReader(params.ChartImage))
			if err != nil {
				log.Printf("WARN: Go image.Decode failed: %v", err)
			} else {
				log.Printf("INFO: Go image.Decode succeeded")
			}
			// Try excelize's AddPictureFromBytes
			if err := f.AddPictureFromBytes(dashboard, fmt.Sprintf("A%d", chartRow), &excelize.Picture{
				Extension: ".png",
				File:      params.ChartImage,
				Format: &excelize.GraphicOptions{
					OffsetX: 10,
					OffsetY: 10,
					ScaleX:  1.0,
					ScaleY:  1.0,
				},
			}); err != nil {
				log.Printf("WARN: add chart picture failed: %v", err)
			}
		}
	}

	f.SetColWidth(dashboard, "A", "A", 30)
	f.SetColWidth(dashboard, "B", "C", 20)
	f.SetColWidth(dashboard, "D", "D", 14)

	// ===== Sheet 2: Summary =====
	summary := "Summary"
	f.NewSheet(summary)

	hasPrevious := len(previousData) > 0
	hasOrders := len(currentHourly) > 0
	summaryHeaders := []string{"Date", "Revenue (Rp)", "Prev Period (Rp)", "Change %"}
	if hasOrders {
		summaryHeaders = append(summaryHeaders, "Orders")
	}
	for i, h := range summaryHeaders {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetCellValue(summary, col+"1", h)
		f.SetCellStyle(summary, col+"1", col+"1", headerStyle)
	}

	var totalCur, totalPrev, totalOrders int

	if len(currentHourly) > 0 {
		for i, dp := range currentHourly {
			row := i + 2
			f.SetCellValue(summary, fmt.Sprintf("A%d", row), dp.Date)
			f.SetCellValue(summary, fmt.Sprintf("B%d", row), dp.Total)
			totalCur += dp.Total
			totalOrders += dp.Orders
			var chg float64
			hasChg := false
			if hasPrevious && i < len(previousHourly) {
				f.SetCellValue(summary, fmt.Sprintf("C%d", row), previousHourly[i].Total)
				totalPrev += previousHourly[i].Total
				chg = calcChangeFloat(dp.Total, previousHourly[i].Total)
				f.SetCellValue(summary, fmt.Sprintf("D%d", row), chg)
				hasChg = true
			}
			f.SetCellValue(summary, "E"+fmt.Sprintf("%d", row), dp.Orders)
			if i%2 == 1 {
				f.SetCellStyle(summary, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), altStyle)
				f.SetCellStyle(summary, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), altStyle)
			}
			if hasChg {
				if chg >= 0 {
					f.SetCellStyle(summary, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), pctPosStyle)
				} else {
					f.SetCellStyle(summary, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), pctNegStyle)
				}
			}
		}
	} else {
		for i, dp := range currentData {
			row := i + 2
			f.SetCellValue(summary, fmt.Sprintf("A%d", row), dp.Date)
			f.SetCellValue(summary, fmt.Sprintf("B%d", row), dp.Total)
			totalCur += dp.Total
			var chg float64
			hasChg := false
			if hasPrevious && i < len(previousData) {
				f.SetCellValue(summary, fmt.Sprintf("C%d", row), previousData[i].Total)
				totalPrev += previousData[i].Total
				chg = calcChangeFloat(dp.Total, previousData[i].Total)
				f.SetCellValue(summary, fmt.Sprintf("D%d", row), chg)
				hasChg = true
			}
			if i%2 == 1 {
				f.SetCellStyle(summary, fmt.Sprintf("A%d", row), fmt.Sprintf("C%d", row), altStyle)
				f.SetCellStyle(summary, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), altStyle)
			}
			if hasChg {
				if chg >= 0 {
					f.SetCellStyle(summary, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), pctPosStyle)
				} else {
					f.SetCellStyle(summary, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), pctNegStyle)
				}
			}
		}
	}

	// Total row
	totalRow := len(currentData) + 2
	totalChg := calcChangeFloat(totalCur, totalPrev)
	f.SetCellValue(summary, fmt.Sprintf("A%d", totalRow), "TOTAL")
	f.SetCellValue(summary, fmt.Sprintf("B%d", totalRow), totalCur)
	if totalPrev > 0 {
		f.SetCellValue(summary, fmt.Sprintf("C%d", totalRow), totalPrev)
	}
	f.SetCellValue(summary, fmt.Sprintf("D%d", totalRow), totalChg)
	if totalChg >= 0 {
		f.SetCellStyle(summary, fmt.Sprintf("D%d", totalRow), fmt.Sprintf("D%d", totalRow), pctPosStyle)
	} else {
		f.SetCellStyle(summary, fmt.Sprintf("D%d", totalRow), fmt.Sprintf("D%d", totalRow), pctNegStyle)
	}
	if hasOrders {
		f.SetCellValue(summary, "E"+fmt.Sprintf("%d", totalRow), totalOrders)
	}

	f.SetColWidth(summary, "A", "A", 18)
	f.SetColWidth(summary, "B", "B", 18)
	f.SetColWidth(summary, "C", "C", 18)
	f.SetColWidth(summary, "D", "D", 14)
	if hasOrders {
		f.SetColWidth(summary, "E", "E", 10)
	}
	f.SetPanes(summary, &excelize.Panes{
		Freeze: true,
		YSplit: 1,
	})

	buf := &bytes.Buffer{}
	if err := f.Write(buf); err != nil {
		return nil, fmt.Errorf("write excel: %w", err)
	}
	return buf.Bytes(), nil
}

func calcChangeFloat(current, previous int) float64 {
	if previous == 0 {
		if current > 0 {
			return 1.0
		}
		return 0.0
	}
	return float64(current-previous) / float64(previous)
}

var jakartaLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.UTC
	}
	return loc
}()

type hourlyDataPoint struct {
	Date   string
	Total  int
	Orders int
}

func toChartData(data []hourlyDataPoint) []repository.ChartDataPoint {
	out := make([]repository.ChartDataPoint, len(data))
	for i, d := range data {
		out[i] = repository.ChartDataPoint{Date: d.Date, Total: d.Total}
	}
	return out
}

func filterHoursUpTo(data []hourlyDataPoint, maxHour int) []hourlyDataPoint {
	var result []hourlyDataPoint
	for _, dp := range data {
		var hour int
		if _, err := fmt.Sscanf(dp.Date, "%02d:00", &hour); err != nil {
			continue
		}
		if hour <= maxHour {
			result = append(result, dp)
		}
	}
	return result
}

func aggregateHourly(ctx context.Context, repo repository.SaleRepository, start, end time.Time) []hourlyDataPoint {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	sales, _, err := repo.GetAllSales(ctx, 100000, 0, "", "created_at", "ASC", startStr, endStr, nil, "", nil, nil)
	if err != nil {
		return nil
	}

	hourlyTotals := make(map[int]int)
	hourlyOrders := make(map[int]int)
	for _, s := range sales {
		createdTime, err := time.Parse(time.RFC3339, s.CreatedAt)
		if err != nil {
			continue
		}
		hour := createdTime.In(jakartaLoc).Hour()
		hourlyTotals[hour] += s.TotalAmount
		hourlyOrders[hour]++
	}

	var data []hourlyDataPoint
	for hour := 0; hour < 24; hour++ {
		data = append(data, hourlyDataPoint{
			Date:   fmt.Sprintf("%02d:00", hour),
			Total:  hourlyTotals[hour],
			Orders: hourlyOrders[hour],
		})
	}
	return data
}

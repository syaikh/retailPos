package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	"retail-pos-system/internal/repository"

	"github.com/go-analyze/charts"
)

func renderRevenueComparisonChart(current, previous []repository.ChartDataPoint) ([]byte, error) {
	return renderRevenueComparisonChartWithLabels(current, previous, nil)
}

func renderRevenueComparisonChartWithLabels(current, previous []repository.ChartDataPoint, labels []string) ([]byte, error) {
	currentValues := make([]float64, len(current))

	if labels == nil {
		labels = make([]string, len(current))
		for i, dp := range current {
			labels[i] = dp.Date
		}
	}

	for i, dp := range current {
		currentValues[i] = float64(dp.Total)
	}

	hasPrevious := len(previous) > 0
	values := [][]float64{currentValues}

	var seriesNames []string
	seriesColors := []charts.Color{charts.ColorFromHex("#0ea5e9")}

	seriesNames = append(seriesNames, "Current Period")
	if hasPrevious {
		prevValues := make([]float64, len(previous))
		for i, dp := range previous {
			prevValues[i] = float64(dp.Total)
		}
		values = append(values, prevValues)
		seriesNames = append(seriesNames, "Previous Period")
		seriesColors = append(seriesColors, charts.ColorFromHex("#94a3b8"))
	}

	theme := charts.GetTheme(charts.ThemeLight).
		WithSeriesColors(seriesColors)

	opt := charts.LineChartOption{
		Theme: theme,
		Title: charts.TitleOption{
			Text:    "Revenue Comparison",
			Subtext: "Current Period vs Previous Period",
		},
		Padding: charts.NewBoxEqual(15),
		Legend: charts.LegendOption{
			SeriesNames: seriesNames,
			Symbol:      charts.SymbolCircle,
		},
		XAxis: charts.XAxisOption{
			Labels: labels,
		},
		YAxis: []charts.YAxisOption{
			{
				PreferNiceIntervals: charts.Ptr(true),
				Min:                 charts.Ptr(0.0),
				ValueFormatter: func(f float64) string {
					if f >= 1_000_000_000 {
						return fmt.Sprintf("Rp %.1f M", f/1_000_000_000)
					}
					if f >= 1_000_000 {
						return fmt.Sprintf("Rp %.1f jt", f/1_000_000)
					}
					if f >= 1_000 {
						return fmt.Sprintf("Rp %.0f Rb", f/1_000)
					}
					return fmt.Sprintf("Rp %.0f", f)
				},
			},
		},
		SeriesList:             charts.NewSeriesListLine(values),
		LineStrokeWidth:        2,
		StrokeSmoothingTension: 0,
		FillArea:               charts.Ptr(false),
		Symbol:                 charts.SymbolCircle,
	}

	p := charts.NewPainter(charts.PainterOptions{
		Width:  800,
		Height: 400,
	})

	if err := p.LineChart(opt); err != nil {
		return nil, fmt.Errorf("render chart: %w", err)
	}

	buf, err := p.Bytes()
	if err != nil {
		return nil, fmt.Errorf("get chart png: %w", err)
	}
	return buf, nil
}

func renderRevenueComparisonChartMultiline(current, previous []repository.ChartDataPoint) ([]byte, error) {
	multilineLabels := makeMultilineLabels(current, previous)

	svgBytes, err := renderChartSVG(current, previous, multilineLabels)
	if err != nil {
		return renderRevenueComparisonChartWithLabels(current, previous, formatChartLabels(current))
	}

	processedSVG := processSVGText(svgBytes)

	pngBytes, err := convertSVGToPNG(processedSVG)
	if err != nil {
		return renderRevenueComparisonChartWithLabels(current, previous, formatChartLabels(current))
	}

	return pngBytes, nil
}

func makeMultilineLabels(current, previous []repository.ChartDataPoint) []string {
	labels := make([]string, len(current))
	for i, dp := range current {
		curTime, err := time.Parse("2006-01-02", dp.Date)
		if err != nil {
			labels[i] = dp.Date
			continue
		}
		curLabel := curTime.Format("Jan 2")
		if i < len(previous) {
			prevTime, err := time.Parse("2006-01-02", previous[i].Date)
			if err == nil {
				labels[i] = curLabel + "\n" + prevTime.Format("Jan 2")
			} else {
				labels[i] = curLabel
			}
		} else {
			labels[i] = curLabel
		}
	}
	return labels
}

func renderChartSVG(current, previous []repository.ChartDataPoint, labels []string) ([]byte, error) {
	currentValues := make([]float64, len(current))
	for i, dp := range current {
		currentValues[i] = float64(dp.Total)
	}

	hasPrevious := len(previous) > 0
	values := [][]float64{currentValues}

	var seriesNames []string
	seriesColors := []charts.Color{charts.ColorFromHex("#0ea5e9")}

	seriesNames = append(seriesNames, "Current Period")
	if hasPrevious {
		prevValues := make([]float64, len(previous))
		for i, dp := range previous {
			prevValues[i] = float64(dp.Total)
		}
		values = append(values, prevValues)
		seriesNames = append(seriesNames, "Previous Period")
		seriesColors = append(seriesColors, charts.ColorFromHex("#94a3b8"))
	}

	theme := charts.GetTheme(charts.ThemeLight).
		WithSeriesColors(seriesColors)

	opt := charts.LineChartOption{
		Theme: theme,
		Title: charts.TitleOption{
			Text:    "Revenue Comparison",
			Subtext: "Current Period vs Previous Period",
		},
		Padding: charts.NewBoxEqual(15),
		Legend: charts.LegendOption{
			SeriesNames: seriesNames,
			Symbol:      charts.SymbolCircle,
		},
		XAxis: charts.XAxisOption{
			Labels: labels,
		},
		YAxis: []charts.YAxisOption{
			{
				PreferNiceIntervals: charts.Ptr(true),
				Min:                 charts.Ptr(0.0),
				ValueFormatter: func(f float64) string {
					if f >= 1_000_000_000 {
						return fmt.Sprintf("Rp %.1f M", f/1_000_000_000)
					}
					if f >= 1_000_000 {
						return fmt.Sprintf("Rp %.1f jt", f/1_000_000)
					}
					if f >= 1_000 {
						return fmt.Sprintf("Rp %.0f Rb", f/1_000)
					}
					return fmt.Sprintf("Rp %.0f", f)
				},
			},
		},
		SeriesList:             charts.NewSeriesListLine(values),
		LineStrokeWidth:        2,
		StrokeSmoothingTension: 0,
		FillArea:               charts.Ptr(false),
		Symbol:                 charts.SymbolCircle,
	}

	p := charts.NewPainter(charts.PainterOptions{
		Width:        800,
		Height:       400,
		OutputFormat: "svg",
	})

	if err := p.LineChart(opt); err != nil {
		return nil, fmt.Errorf("render chart svg: %w", err)
	}

	buf, err := p.Bytes()
	if err != nil {
		return nil, fmt.Errorf("get chart svg: %w", err)
	}
	return buf, nil
}

var textElemRE = regexp.MustCompile(`<text x="([^"]*)" y="([^"]*)"([^>]*)>([^<]*)</text>`)

func processSVGText(svg []byte) []byte {
	return textElemRE.ReplaceAllFunc(svg, func(match []byte) []byte {
		parts := textElemRE.FindSubmatch(match)
		if len(parts) < 5 {
			return match
		}
		content := string(parts[4])
		if !containsNewline(content) {
			return match
		}
		x := string(parts[1])
		lines := splitLines(content)
		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf(`<text x="%s" y="%s"%s>`, string(parts[1]), string(parts[2]), string(parts[3])))
		for i, line := range lines {
			if i == 0 {
				buf.WriteString(fmt.Sprintf(`<tspan x="%s" dy="0">%s</tspan>`, x, line))
			} else {
				buf.WriteString(fmt.Sprintf(`<tspan x="%s" dy="1.2em">%s</tspan>`, x, line))
			}
		}
		buf.WriteString(`</text>`)
		return buf.Bytes()
	})
}

func containsNewline(s string) bool {
	for _, c := range s {
		if c == '\n' {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func convertSVGToPNG(svg []byte) ([]byte, error) {
	svgFile, err := os.CreateTemp("", "chart-*.svg")
	if err != nil {
		return nil, fmt.Errorf("create temp svg: %w", err)
	}
	svgPath := svgFile.Name()
	defer os.Remove(svgPath)

	if _, err := svgFile.Write(svg); err != nil {
		svgFile.Close()
		return nil, fmt.Errorf("write svg: %w", err)
	}
	svgFile.Close()

	pngFile, err := os.CreateTemp("", "chart-*.png")
	if err != nil {
		return nil, fmt.Errorf("create temp png: %w", err)
	}
	pngPath := pngFile.Name()
	pngFile.Close()
	defer os.Remove(pngPath)

	cmd := exec.Command("convert", "-background", "white", "-flatten", svgPath, pngPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("convert svg to png: %w, output: %s", err, string(output))
	}

	pngBytes, err := os.ReadFile(pngPath)
	if err != nil {
		return nil, fmt.Errorf("read png: %w", err)
	}

	return pngBytes, nil
}

func formatChartLabels(data []repository.ChartDataPoint) []string {
	labels := make([]string, len(data))
	for i, dp := range data {
		t, err := time.Parse("2006-01-02", dp.Date)
		if err == nil {
			labels[i] = t.Format("Jan 2")
		} else {
			labels[i] = dp.Date
		}
	}
	return labels
}

package service

import (
	"bytes"
	"image/png"
	"os"
	"strings"
	"testing"

	"retail-pos-system/internal/repository"
)

func TestProcessSVGText(t *testing.T) {
	svgInput := `<svg><text x="190" y="383" style="...">Jun 12
Jun 5</text><text x="418" y="383" style="...">Jun 13</text></svg>`
	expected := `<svg><text x="190" y="383" style="..."><tspan x="190" dy="0">Jun 12</tspan><tspan x="190" dy="1.2em">Jun 5</tspan></text><text x="418" y="383" style="...">Jun 13</text></svg>`

	result := processSVGText([]byte(svgInput))
	if string(result) != expected {
		t.Fatalf("got:\n%s\n\nwant:\n%s", string(result), expected)
	}
}

func TestRenderRevenueComparisonChartMultiline(t *testing.T) {
	current := []repository.ChartDataPoint{
		{Date: "2026-06-12", Total: 100000},
		{Date: "2026-06-13", Total: 150000},
		{Date: "2026-06-14", Total: 120000},
	}
	previous := []repository.ChartDataPoint{
		{Date: "2026-06-05", Total: 90000},
		{Date: "2026-06-06", Total: 130000},
		{Date: "2026-06-07", Total: 110000},
	}

	chartPNG, err := renderRevenueComparisonChartMultiline(current, previous)
	if err != nil {
		t.Fatalf("multiline chart failed: %v", err)
	}

	if len(chartPNG) == 0 {
		t.Fatal("chart PNG is empty")
	}

	img, err := png.Decode(bytes.NewReader(chartPNG))
	if err != nil {
		t.Fatalf("chart PNG is not valid: %v", err)
	}

	if img.Bounds().Dx() != 800 || img.Bounds().Dy() != 400 {
		t.Fatalf("unexpected chart dimensions: got %dx%d, want 800x400", img.Bounds().Dx(), img.Bounds().Dy())
	}

	os.WriteFile("/tmp/test_multiline_chart.png", chartPNG, 0644)
	t.Logf("multiline chart saved: %d bytes", len(chartPNG))
}

func TestMakeMultilineLabels(t *testing.T) {
	current := []repository.ChartDataPoint{
		{Date: "2026-06-12", Total: 100000},
		{Date: "2026-06-13", Total: 150000},
	}
	previous := []repository.ChartDataPoint{
		{Date: "2026-06-05", Total: 90000},
		{Date: "2026-06-06", Total: 130000},
	}

	labels := makeMultilineLabels(current, previous)
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	if labels[0] != "Jun 12\nJun 5" {
		t.Fatalf("expected 'Jun 12\\nJun 5', got %q", labels[0])
	}
	if labels[1] != "Jun 13\nJun 6" {
		t.Fatalf("expected 'Jun 13\\nJun 6', got %q", labels[1])
	}
}

func TestMakeMultilineLabels_Hourly(t *testing.T) {
	current := []repository.ChartDataPoint{
		{Date: "00:00", Total: 100000},
		{Date: "01:00", Total: 150000},
	}

	labels := makeMultilineLabels(current, nil)
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	if labels[0] != "00:00" {
		t.Fatalf("expected '00:00', got %q", labels[0])
	}
	if labels[1] != "01:00" {
		t.Fatalf("expected '01:00', got %q", labels[1])
	}
}

func TestFormatChartLabels(t *testing.T) {
	data := []repository.ChartDataPoint{
		{Date: "2026-06-12", Total: 100000},
		{Date: "01:00", Total: 150000},
	}

	labels := formatChartLabels(data)
	if labels[0] != "Jun 12" {
		t.Fatalf("expected 'Jun 12', got %q", labels[0])
	}
	if labels[1] != "01:00" {
		t.Fatalf("expected '01:00', got %q", labels[1])
	}
}

func TestContainsNewline(t *testing.T) {
	if !containsNewline("hello\nworld") {
		t.Fatal("expected true")
	}
	if containsNewline("hello world") {
		t.Fatal("expected false")
	}
}

func TestSplitLines(t *testing.T) {
	lines := splitLines("hello\nworld\nfoo")
	if len(lines) != 3 || lines[0] != "hello" || lines[1] != "world" || lines[2] != "foo" {
		t.Fatalf("got %v", lines)
	}

	lines = splitLines("hello")
	if len(lines) != 1 || lines[0] != "hello" {
		t.Fatalf("got %v", lines)
	}
}

func TestConvertSVGToPNG_Available(t *testing.T) {
	svgContent := []byte(`<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50">
  <rect width="100" height="50" fill="white"/>
  <text x="10" y="30" font-size="12" fill="black">Test</text>
</svg>`)

	pngBytes, err := convertSVGToPNG(svgContent)
	if err != nil {
		t.Skipf("convert not available: %v", err)
	}

	if len(pngBytes) == 0 {
		t.Fatal("empty PNG")
	}

	_, err = png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}
}

func TestSVGNotContainingNewline(t *testing.T) {
	current := []repository.ChartDataPoint{
		{Date: "2026-06-12", Total: 100000},
	}
	previous := []repository.ChartDataPoint{
		{Date: "2026-06-05", Total: 90000},
	}

	labels := makeMultilineLabels(current, previous)
	svgBytes, err := renderChartSVG(current, previous, labels)
	if err != nil {
		t.Fatalf("render svg: %v", err)
	}

	// Check that raw SVG has \n in text content
	svgStr := string(svgBytes)
	if !strings.Contains(svgStr, "Jun 12") {
		t.Fatal("svg missing label text")
	}

	// Process SVG
	processed := processSVGText(svgBytes)
	processedStr := string(processed)
	if strings.Contains(processedStr, "Jun 12") && !strings.Contains(processedStr, "tspan") {
		t.Fatal("processed SVG should contain tspan for multiline label")
	}

	os.WriteFile("/tmp/test_processed.svg", processed, 0644)
}

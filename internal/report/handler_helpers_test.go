package report

import (
	"testing"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/stretchr/testify/assert"
)

var wib = time.FixedZone("WIB", 7*3600)

func TestGetDailyRanges(t *testing.T) {
	tests := []struct {
		name          string
		refDate       time.Time
		completedMode bool
		want          PeriodRange
	}{
		{
			name:          "realtime mode",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: false,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 9, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 16, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2026, 1, 2, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2026, 1, 9, 0, 0, 0, 0, wib),
			},
		},
		{
			name:          "completed mode",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: true,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 16, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2026, 1, 8, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2026, 1, 9, 0, 0, 0, 0, wib),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDailyRanges(tt.refDate, tt.completedMode)
			assert.Equal(t, tt.want.CurrentStart, got.CurrentStart)
			assert.Equal(t, tt.want.CurrentEnd, got.CurrentEnd)
			assert.Equal(t, tt.want.PreviousStart, got.PreviousStart)
			assert.Equal(t, tt.want.PreviousEnd, got.PreviousEnd)
		})
	}
}

func TestGet7DaysRanges(t *testing.T) {
	tests := []struct {
		name          string
		refDate       time.Time
		completedMode bool
		want          PeriodRange
	}{
		{
			name:          "realtime mode",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: false,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 9, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 16, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2026, 1, 2, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2026, 1, 9, 0, 0, 0, 0, wib),
			},
		},
		{
			name:          "completed mode thursday",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: true,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 5, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 12, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2025, 12, 29, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2026, 1, 5, 0, 0, 0, 0, wib),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := get7DaysRanges(tt.refDate, tt.completedMode)
			assert.Equal(t, tt.want.CurrentStart, got.CurrentStart)
			assert.Equal(t, tt.want.CurrentEnd, got.CurrentEnd)
			assert.Equal(t, tt.want.PreviousStart, got.PreviousStart)
			assert.Equal(t, tt.want.PreviousEnd, got.PreviousEnd)
		})
	}
}

func TestGet30DaysRanges(t *testing.T) {
	refDate := time.Date(2026, 1, 15, 0, 0, 0, 0, wib)
	got := get30DaysRanges(refDate)
	want := PeriodRange{
		CurrentStart:  time.Date(2025, 12, 17, 0, 0, 0, 0, wib),
		CurrentEnd:    time.Date(2026, 1, 16, 0, 0, 0, 0, wib),
		PreviousStart: time.Date(2025, 11, 17, 0, 0, 0, 0, wib),
		PreviousEnd:   time.Date(2025, 12, 17, 0, 0, 0, 0, wib),
	}
	assert.Equal(t, want.CurrentStart, got.CurrentStart)
	assert.Equal(t, want.CurrentEnd, got.CurrentEnd)
	assert.Equal(t, want.PreviousStart, got.PreviousStart)
	assert.Equal(t, want.PreviousEnd, got.PreviousEnd)
}

func TestGetWeeklyRanges(t *testing.T) {
	tests := []struct {
		name          string
		refDate       time.Time
		completedMode bool
		want          PeriodRange
	}{
		{
			name:          "realtime mode thursday",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: false,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 12, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 16, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2026, 1, 5, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2026, 1, 9, 0, 0, 0, 0, wib),
			},
		},
		{
			name:          "completed mode thursday",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: true,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 5, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 19, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2025, 12, 29, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2026, 1, 5, 0, 0, 0, 0, wib),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getWeeklyRanges(tt.refDate, tt.completedMode)
			assert.Equal(t, tt.want.CurrentStart, got.CurrentStart)
			assert.Equal(t, tt.want.CurrentEnd, got.CurrentEnd)
			assert.Equal(t, tt.want.PreviousStart, got.PreviousStart)
			assert.Equal(t, tt.want.PreviousEnd, got.PreviousEnd)
		})
	}
}

func TestGetRealtimeRanges(t *testing.T) {
	refDate := time.Date(2026, 1, 15, 10, 30, 0, 0, wib)
	got := getRealtimeRanges(refDate)
	want := PeriodRange{
		CurrentStart:  time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
		CurrentEnd:    time.Date(2026, 1, 15, 11, 0, 0, 0, wib),
		PreviousStart: time.Date(2026, 1, 14, 0, 0, 0, 0, wib),
		PreviousEnd:   time.Date(2026, 1, 14, 11, 0, 0, 0, wib),
	}
	assert.Equal(t, want.CurrentStart, got.CurrentStart)
	assert.Equal(t, want.CurrentEnd, got.CurrentEnd)
	assert.Equal(t, want.PreviousStart, got.PreviousStart)
	assert.Equal(t, want.PreviousEnd, got.PreviousEnd)
}

func TestGetMonthlyRanges(t *testing.T) {
	tests := []struct {
		name          string
		refDate       time.Time
		completedMode bool
		want          PeriodRange
	}{
		{
			name:          "realtime mode",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: false,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 1, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 16, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2025, 12, 1, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2025, 12, 16, 0, 0, 0, 0, wib),
			},
		},
		{
			name:          "completed mode",
			refDate:       time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			completedMode: true,
			want: PeriodRange{
				CurrentStart:  time.Date(2025, 12, 1, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 1, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2025, 11, 1, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2025, 12, 1, 0, 0, 0, 0, wib),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMonthlyRanges(tt.refDate, tt.completedMode)
			assert.Equal(t, tt.want.CurrentStart, got.CurrentStart)
			assert.Equal(t, tt.want.CurrentEnd, got.CurrentEnd)
			assert.Equal(t, tt.want.PreviousStart, got.PreviousStart)
			assert.Equal(t, tt.want.PreviousEnd, got.PreviousEnd)
		})
	}
}

func TestGetYearlyRanges(t *testing.T) {
	tests := []struct {
		name          string
		refDate       time.Time
		completedMode bool
		want          PeriodRange
	}{
		{
			name:          "realtime mode",
			refDate:       time.Date(2026, 7, 13, 0, 0, 0, 0, wib),
			completedMode: false,
			want: PeriodRange{
				CurrentStart:  time.Date(2026, 1, 1, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 7, 14, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2025, 1, 1, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2025, 7, 14, 0, 0, 0, 0, wib),
			},
		},
		{
			name:          "completed mode",
			refDate:       time.Date(2026, 7, 13, 0, 0, 0, 0, wib),
			completedMode: true,
			want: PeriodRange{
				CurrentStart:  time.Date(2025, 1, 1, 0, 0, 0, 0, wib),
				CurrentEnd:    time.Date(2026, 1, 1, 0, 0, 0, 0, wib),
				PreviousStart: time.Date(2024, 1, 1, 0, 0, 0, 0, wib),
				PreviousEnd:   time.Date(2025, 1, 1, 0, 0, 0, 0, wib),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getYearlyRanges(tt.refDate, tt.completedMode)
			assert.Equal(t, tt.want.CurrentStart, got.CurrentStart)
			assert.Equal(t, tt.want.CurrentEnd, got.CurrentEnd)
			assert.Equal(t, tt.want.PreviousStart, got.PreviousStart)
			assert.Equal(t, tt.want.PreviousEnd, got.PreviousEnd)
		})
	}
}

func TestGetComparisonRanges_AllPeriodTypes(t *testing.T) {
	refDate := time.Date(2026, 7, 14, 0, 0, 0, 0, wib)

	tests := []struct {
		name      string
		period    PeriodType
		completed bool
	}{
		{"daily realtime", PeriodDaily, false},
		{"daily completed", PeriodDaily, true},
		{"7days realtime", Period7Days, false},
		{"7days completed", Period7Days, true},
		{"weekly realtime", PeriodWeekly, false},
		{"weekly completed", PeriodWeekly, true},
		{"monthly realtime", PeriodMonthly, false},
		{"monthly completed", PeriodMonthly, true},
		{"yearly realtime", PeriodYearly, false},
		{"yearly completed", PeriodYearly, true},
		{"unknown period defaults to daily", PeriodType("unknown"), false},
		{"unknown period completed defaults to daily", PeriodType("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := getComparisonRanges(tt.period, refDate, tt.completed)
			assert.False(t, pr.CurrentStart.IsZero(), "CurrentStart should not be zero")
			assert.False(t, pr.CurrentEnd.IsZero(), "CurrentEnd should not be zero")
			assert.False(t, pr.PreviousStart.IsZero(), "PreviousStart should not be zero")
			assert.False(t, pr.PreviousEnd.IsZero(), "PreviousEnd should not be zero")
			assert.True(t, pr.CurrentEnd.After(pr.CurrentStart), "CurrentEnd should be after CurrentStart")
			assert.True(t, pr.PreviousEnd.After(pr.PreviousStart), "PreviousEnd should be after PreviousStart")
			assert.Greater(t, pr.DaysInPeriod, 0, "DaysInPeriod should be positive")
		})
	}
}

func TestGetComparisonRanges_DaysInPeriod(t *testing.T) {
	refDate := time.Date(2026, 7, 14, 0, 0, 0, 0, wib)

	pr := getComparisonRanges(PeriodDaily, refDate, false)
	assert.Equal(t, 7, pr.DaysInPeriod, "daily realtime should have 7-day range")

	pr = getComparisonRanges(PeriodDaily, refDate, true)
	assert.Equal(t, 1, pr.DaysInPeriod, "daily completed should have 1-day range")

	pr = getComparisonRanges(PeriodMonthly, refDate, false)
	assert.Greater(t, pr.DaysInPeriod, 0)
}

func TestGetComparisonRanges_IsPartial(t *testing.T) {
	midMonth := time.Date(2026, 7, 14, 0, 0, 0, 0, wib)
	endOfWeek := time.Date(2026, 7, 19, 0, 0, 0, 0, wib) // Sunday

	// Weekly realtime mid-week IS partial because isPeriodIncomplete + periodType == PeriodWeekly
	pr := getComparisonRanges(PeriodWeekly, midMonth, false)
	assert.True(t, pr.IsPartial, "weekly realtime mid-week should be partial")

	// Weekly completed mid-week is also partial
	pr = getComparisonRanges(PeriodWeekly, midMonth, true)
	assert.True(t, pr.IsPartial, "weekly completed mid-week should be partial")

	// Weekly realtime on Sunday (not incomplete) should not be partial
	pr = getComparisonRanges(PeriodWeekly, endOfWeek, false)
	assert.False(t, pr.IsPartial, "weekly realtime on sunday should not be partial")

	// Monthly realtime mid-month IS partial (isPeriodIncomplete + PeriodMonthly)
	pr = getComparisonRanges(PeriodMonthly, midMonth, false)
	assert.True(t, pr.IsPartial, "monthly realtime mid-month should be partial")

	pr = getComparisonRanges(PeriodMonthly, midMonth, true)
	assert.True(t, pr.IsPartial, "monthly completed mid-month should be partial")

	// Daily never partial
	pr = getComparisonRanges(PeriodDaily, midMonth, false)
	assert.False(t, pr.IsPartial, "daily never partial")

	pr = getComparisonRanges(PeriodDaily, midMonth, true)
	assert.False(t, pr.IsPartial, "daily completed never partial")
}

func TestGetComparisonRanges_DefaultCase(t *testing.T) {
	refDate := time.Date(2026, 7, 14, 0, 0, 0, 0, wib)
	pr := getComparisonRanges(PeriodType("bogus"), refDate, false)
	dailyPr := getDailyRanges(time.Date(refDate.Year(), refDate.Month(), refDate.Day(), 0, 0, 0, 0, shared.JakartaLocation()), false)
	assert.Equal(t, dailyPr.CurrentStart, pr.CurrentStart, "unknown period should default to daily")
}

func TestIsPeriodIncomplete(t *testing.T) {
	tests := []struct {
		name           string
		periodType     PeriodType
		refDate        time.Time
		wantIncomplete bool
	}{
		{
			name:           "weekly wednesday",
			periodType:     PeriodWeekly,
			refDate:        time.Date(2026, 1, 14, 0, 0, 0, 0, wib),
			wantIncomplete: true,
		},
		{
			name:           "weekly sunday",
			periodType:     PeriodWeekly,
			refDate:        time.Date(2026, 1, 18, 0, 0, 0, 0, wib),
			wantIncomplete: false,
		},
		{
			name:           "weekly saturday",
			periodType:     PeriodWeekly,
			refDate:        time.Date(2026, 1, 17, 0, 0, 0, 0, wib),
			wantIncomplete: true,
		},
		{
			name:           "monthly mid january",
			periodType:     PeriodMonthly,
			refDate:        time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			wantIncomplete: true,
		},
		{
			name:           "monthly jan 31",
			periodType:     PeriodMonthly,
			refDate:        time.Date(2026, 1, 31, 0, 0, 0, 0, wib),
			wantIncomplete: false,
		},
		{
			name:           "monthly dec 31",
			periodType:     PeriodMonthly,
			refDate:        time.Date(2026, 12, 31, 0, 0, 0, 0, wib),
			wantIncomplete: false,
		},
		{
			name:           "yearly january",
			periodType:     PeriodYearly,
			refDate:        time.Date(2026, 1, 1, 0, 0, 0, 0, wib),
			wantIncomplete: true,
		},
		{
			name:           "yearly november",
			periodType:     PeriodYearly,
			refDate:        time.Date(2026, 11, 15, 0, 0, 0, 0, wib),
			wantIncomplete: true,
		},
		{
			name:           "yearly december 31",
			periodType:     PeriodYearly,
			refDate:        time.Date(2026, 12, 31, 0, 0, 0, 0, wib),
			wantIncomplete: false,
		},
		{
			name:           "daily always false",
			periodType:     PeriodDaily,
			refDate:        time.Date(2026, 1, 15, 0, 0, 0, 0, wib),
			wantIncomplete: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPeriodIncomplete(tt.periodType, tt.refDate)
			assert.Equal(t, tt.wantIncomplete, got)
		})
	}
}

package report

import (
	"testing"
	"time"

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

func TestIsPeriodIncomplete(t *testing.T) {
	tests := []struct {
		name         string
		periodType   PeriodType
		refDate      time.Time
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

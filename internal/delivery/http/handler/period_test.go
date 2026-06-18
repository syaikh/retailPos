package handler

import (
	"testing"
	"time"

	"retail-pos-system/internal/config"

	"github.com/stretchr/testify/assert"
)

// TestDailyRanges_H7Comparison verifies that yesterday view (completedMode)
// compares to same day last week (H-7), not day before yesterday (H-2)
func TestDailyRanges_H7Comparison(t *testing.T) {
	jkt := config.Load().Timezone

	// Reference date is May 31 (already yesterday), simulating yesterday view
	refDate := time.Date(2026, 5, 31, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodDaily, refDate, true) // completedMode = true

	// Current should be the reference date (May 31)
	assert.Equal(t, 2026, ranges.CurrentStart.Year())
	assert.Equal(t, time.May, ranges.CurrentStart.Month())
	assert.Equal(t, 31, ranges.CurrentStart.Day())

	// Previous should be H-7 (May 24), NOT H-2 (May 30)
	assert.Equal(t, 2026, ranges.PreviousStart.Year())
	assert.Equal(t, time.May, ranges.PreviousStart.Month())
	assert.Equal(t, 24, ranges.PreviousStart.Day(), "Previous should be H-7 (May 24), not H-2")

	// Verify the day offset is exactly 7 days
	dayDiff := int(ranges.CurrentStart.Sub(ranges.PreviousStart).Hours() / 24)
	assert.Equal(t, 7, dayDiff, "Should be exactly 7 days difference")
}

// TestWeeklyRanges_PartialDetection verifies partial week detection for all days
// Only Sunday is complete (last day of week), matching monthly pattern
func TestWeeklyRanges_PartialDetection(t *testing.T) {
	jkt := config.Load().Timezone

	// Test case 1: Monday (dayOfWeek = 1) - partial (not last day of week)
	monday := time.Date(2026, 6, 1, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodWeekly, monday, false) // todate mode
	assert.True(t, ranges.IsPartial, "Monday week should be partial (not last day of week)")

	// Test case 2: Tuesday (dayOfWeek = 2) - partial
	tuesday := time.Date(2026, 6, 2, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, tuesday, false)
	assert.True(t, ranges.IsPartial, "Tuesday week should be partial")

	// Test case 3: Wednesday (dayOfWeek = 3) - partial
	wednesday := time.Date(2026, 6, 3, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, wednesday, false)
	assert.True(t, ranges.IsPartial, "Wednesday week should be partial")

	// Test case 4: Thursday (dayOfWeek = 4) - partial (not last day)
	thursday := time.Date(2026, 6, 4, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, thursday, false)
	assert.True(t, ranges.IsPartial, "Thursday week should be partial")

	// Test case 5: Friday (dayOfWeek = 5) - partial
	friday := time.Date(2026, 6, 5, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, friday, false)
	assert.True(t, ranges.IsPartial, "Friday week should be partial")

	// Test case 6: Saturday (dayOfWeek = 6) - partial
	saturday := time.Date(2026, 6, 6, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, saturday, false)
	assert.True(t, ranges.IsPartial, "Saturday week should be partial")

	// Test case 7: Sunday (dayOfWeek = 0) - complete (last day of week)
	sunday := time.Date(2026, 6, 7, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, sunday, false)
	assert.False(t, ranges.IsPartial, "Sunday week should be complete (last day of week)")
}

// TestWeeklyRanges_LikeForLikeComparison verifies that partial weeks use same-day comparison
func TestWeeklyRanges_LikeForLikeComparison(t *testing.T) {
	jkt := config.Load().Timezone

	// Thursday: 4 days completed (Mon-Thu), should compare 4 days vs previous
	thursday := time.Date(2026, 6, 4, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodWeekly, thursday, false)

	// Current: Mon June 1 - Fri June 5 (4 days)
	assert.Equal(t, 2026, ranges.CurrentStart.Year())
	assert.Equal(t, time.June, ranges.CurrentStart.Month())
	assert.Equal(t, 1, ranges.CurrentStart.Day())
	assert.Equal(t, 2026, ranges.CurrentEnd.Year())
	assert.Equal(t, time.June, ranges.CurrentEnd.Month())
	assert.Equal(t, 5, ranges.CurrentEnd.Day()) // CurrentEnd is exclusive, so day 5 means up to June 4

	// Previous: Mon May 25 - Fri May 29 (same 4 days)
	assert.Equal(t, 2026, ranges.PreviousStart.Year())
	assert.Equal(t, time.May, ranges.PreviousStart.Month())
	assert.Equal(t, 25, ranges.PreviousStart.Day())
	assert.Equal(t, 2026, ranges.PreviousEnd.Year())
	assert.Equal(t, time.May, ranges.PreviousEnd.Month())
	assert.Equal(t, 29, ranges.PreviousEnd.Day()) // PreviousEnd is exclusive
}

// TestMonthlyRanges_LikeForLikeComparison verifies that partial months use month-to-date comparison
func TestMonthlyRanges_LikeForLikeComparison(t *testing.T) {
	jkt := config.Load().Timezone

	// June 2: 2 days elapsed, should compare Jun 1-2 vs May 1-2
	june2 := time.Date(2026, 6, 2, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodMonthly, june2, false) // todate mode

	// Current: Jun 1 - Jun 3 (exclusive, so Jun 1-2)
	assert.Equal(t, 2026, ranges.CurrentStart.Year())
	assert.Equal(t, time.June, ranges.CurrentStart.Month())
	assert.Equal(t, 1, ranges.CurrentStart.Day())
	assert.Equal(t, 2026, ranges.CurrentEnd.Year())
	assert.Equal(t, time.June, ranges.CurrentEnd.Month())
	assert.Equal(t, 3, ranges.CurrentEnd.Day()) // Exclusive, so includes Jun 1-2

	// Previous: May 1 - May 3 (same 2 days)
	assert.Equal(t, 2026, ranges.PreviousStart.Year())
	assert.Equal(t, time.May, ranges.PreviousStart.Month())
	assert.Equal(t, 1, ranges.PreviousStart.Day())
	assert.Equal(t, 2026, ranges.PreviousEnd.Year())
	assert.Equal(t, time.May, ranges.PreviousEnd.Month())
	assert.Equal(t, 3, ranges.PreviousEnd.Day()) // Exclusive, so includes May 1-2
}

// TestMonthlyRanges_CompletedMode verifies completed mode compares full last month
func TestMonthlyRanges_CompletedMode(t *testing.T) {
	jkt := config.Load().Timezone

	// June 2 in completed mode should compare full May vs full April
	june2 := time.Date(2026, 6, 2, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodMonthly, june2, true) // completed mode

	// Current: Full May 2026 (last full month)
	assert.Equal(t, 2026, ranges.CurrentStart.Year())
	assert.Equal(t, time.May, ranges.CurrentStart.Month())
	assert.Equal(t, 1, ranges.CurrentStart.Day())
	assert.Equal(t, 2026, ranges.CurrentEnd.Year())
	assert.Equal(t, time.June, ranges.CurrentEnd.Month())
	assert.Equal(t, 1, ranges.CurrentEnd.Day()) // June 1 exclusive

	// Previous: Full April 2026
	assert.Equal(t, 2026, ranges.PreviousStart.Year())
	assert.Equal(t, time.April, ranges.PreviousStart.Month())
	assert.Equal(t, 1, ranges.PreviousStart.Day())
	assert.Equal(t, 2026, ranges.PreviousEnd.Year())
	assert.Equal(t, time.May, ranges.PreviousEnd.Month())
	assert.Equal(t, 1, ranges.PreviousEnd.Day()) // May 1 exclusive
}

// TestYearlyRanges_WithDec31ComparisonDate verifies that yearly mode with Dec 31 date
// compares full year vs full previous year (e.g., 2024 vs 2023, not 2024 vs 2022)
func TestYearlyRanges_WithDec31ComparisonDate(t *testing.T) {
	jkt := config.Load().Timezone

	// Dec 31, 2024 as reference date (end of year)
	dec31_2024 := time.Date(2024, 12, 31, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodYearly, dec31_2024, false) // todate mode

	// Current: Full 2024 (Jan 1 2024 - Jan 1 2025 exclusive)
	assert.Equal(t, 2024, ranges.CurrentStart.Year())
	assert.Equal(t, time.January, ranges.CurrentStart.Month())
	assert.Equal(t, 1, ranges.CurrentStart.Day())
	assert.Equal(t, 2025, ranges.CurrentEnd.Year())
	assert.Equal(t, time.January, ranges.CurrentEnd.Month())
	assert.Equal(t, 1, ranges.CurrentEnd.Day()) // Jan 1 2025 exclusive

	// Previous: Full 2023 (Jan 1 2023 - Jan 1 2024 exclusive)
	assert.Equal(t, 2023, ranges.PreviousStart.Year())
	assert.Equal(t, time.January, ranges.PreviousStart.Month())
	assert.Equal(t, 1, ranges.PreviousStart.Day())
	assert.Equal(t, 2024, ranges.PreviousEnd.Year())
	assert.Equal(t, time.January, ranges.PreviousEnd.Month())
	assert.Equal(t, 1, ranges.PreviousEnd.Day()) // Jan 1 2024 exclusive
}

// TestYearlyRanges_CompletedMode verifies completed mode compares last full year
func TestYearlyRanges_CompletedMode(t *testing.T) {
	jkt := config.Load().Timezone

	// Any date in 2026, completed mode should compare full 2025 vs full 2024
	june4_2026 := time.Date(2026, 6, 4, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodYearly, june4_2026, true) // completed mode

	// Current: Full 2025
	assert.Equal(t, 2025, ranges.CurrentStart.Year())
	assert.Equal(t, time.January, ranges.CurrentStart.Month())
	assert.Equal(t, 1, ranges.CurrentStart.Day())
	assert.Equal(t, 2026, ranges.CurrentEnd.Year())
	assert.Equal(t, time.January, ranges.CurrentEnd.Month())
	assert.Equal(t, 1, ranges.CurrentEnd.Day()) // Jan 1 2026 exclusive

	// Previous: Full 2024
	assert.Equal(t, 2024, ranges.PreviousStart.Year())
	assert.Equal(t, time.January, ranges.PreviousStart.Month())
	assert.Equal(t, 1, ranges.PreviousStart.Day())
	assert.Equal(t, 2025, ranges.PreviousEnd.Year())
	assert.Equal(t, time.January, ranges.PreviousEnd.Month())
	assert.Equal(t, 1, ranges.PreviousEnd.Day()) // Jan 1 2025 exclusive
}

// TestRealtimeRanges_IncludesCurrentHour verifies that getRealtimeRanges includes
// the current (partial) hour in both current and previous periods.
func TestRealtimeRanges_IncludesCurrentHour(t *testing.T) {
	jkt := config.Load().Timezone

	// Simulate 05:39 Jakarta on June 16, 2026
	now := time.Date(2026, 6, 16, 5, 39, 0, 0, jkt)
	ranges := getRealtimeRanges(now)

	// Current period: today midnight to 06:00 (exclusive) → covers hours 0-5
	assert.Equal(t, 2026, ranges.CurrentStart.Year())
	assert.Equal(t, time.June, ranges.CurrentStart.Month())
	assert.Equal(t, 16, ranges.CurrentStart.Day())
	assert.Equal(t, 0, ranges.CurrentStart.Hour())
	assert.Equal(t, 0, ranges.CurrentStart.Minute())

	assert.Equal(t, 2026, ranges.CurrentEnd.Year())
	assert.Equal(t, time.June, ranges.CurrentEnd.Month())
	assert.Equal(t, 16, ranges.CurrentEnd.Day())
	assert.Equal(t, 6, ranges.CurrentEnd.Hour()) // 06:00 exclusive
	assert.Equal(t, 0, ranges.CurrentEnd.Minute())

	// Previous period: yesterday midnight to 06:00 (exclusive) → covers yesterday hours 0-5
	assert.Equal(t, 2026, ranges.PreviousStart.Year())
	assert.Equal(t, time.June, ranges.PreviousStart.Month())
	assert.Equal(t, 15, ranges.PreviousStart.Day())
	assert.Equal(t, 0, ranges.PreviousStart.Hour())

	assert.Equal(t, 2026, ranges.PreviousEnd.Year())
	assert.Equal(t, time.June, ranges.PreviousEnd.Month())
	assert.Equal(t, 15, ranges.PreviousEnd.Day())
	assert.Equal(t, 6, ranges.PreviousEnd.Hour()) // 06:00 exclusive
}

// TestRealtimeRanges_MidnightBoundary verifies correct behavior at exact midnight
func TestRealtimeRanges_MidnightBoundary(t *testing.T) {
	jkt := config.Load().Timezone

	// At 00:15 Jakarta on June 16 (just after midnight)
	now := time.Date(2026, 6, 16, 0, 15, 0, 0, jkt)
	ranges := getRealtimeRanges(now)

	// CurrentHour = 0, current period end = 01:00 (exclusive)
	assert.Equal(t, 0, ranges.CurrentStart.Hour())
	assert.Equal(t, 1, ranges.CurrentEnd.Hour())

	// Previous: yesterday at same hours
	assert.Equal(t, 15, ranges.PreviousStart.Day())
	assert.Equal(t, 1, ranges.PreviousEnd.Hour())
}

// TestRealtimeRanges_EndOfDay verifies coverage up to hour 23
func TestRealtimeRanges_EndOfDay(t *testing.T) {
	jkt := config.Load().Timezone

	// At 23:45 Jakarta on June 16 (1 hour before midnight)
	now := time.Date(2026, 6, 16, 23, 45, 0, 0, jkt)
	ranges := getRealtimeRanges(now)

	// Current hour = 23, current period end = 00:00 next day (exclusive)
	assert.Equal(t, 16, ranges.CurrentStart.Day())
	assert.Equal(t, 0, ranges.CurrentStart.Hour())

	assert.Equal(t, 17, ranges.CurrentEnd.Day()) // midnight next day (June 17)
	assert.Equal(t, 0, ranges.CurrentEnd.Hour())  // 00:00

	// Previous period: yesterday 00:00 to today 00:00
	assert.Equal(t, 15, ranges.PreviousStart.Day())
	assert.Equal(t, 0, ranges.PreviousStart.Hour())
	assert.Equal(t, 16, ranges.PreviousEnd.Day())
	assert.Equal(t, 0, ranges.PreviousEnd.Hour())
}

// TestRealtimeRanges_TimezonePreservation verifies the ranges preserve Asia/Jakarta timezone
func TestRealtimeRanges_TimezonePreservation(t *testing.T) {
	jkt := config.Load().Timezone

	now := time.Date(2026, 6, 16, 10, 30, 0, 0, jkt)
	ranges := getRealtimeRanges(now)

	// All times should be in Asia/Jakarta timezone
	assert.Equal(t, jkt.String(), ranges.CurrentStart.Location().String())
	assert.Equal(t, jkt.String(), ranges.CurrentEnd.Location().String())
	assert.Equal(t, jkt.String(), ranges.PreviousStart.Location().String())
	assert.Equal(t, jkt.String(), ranges.PreviousEnd.Location().String())
}

// TestMonthlyRanges_IsPeriodIncomplete verifies monthly incomplete detection
func TestMonthlyRanges_IsPeriodIncomplete(t *testing.T) {
	jkt := config.Load().Timezone

	// Jan 1 → next day is Jan 2 (same month) → incomplete
	jan1 := time.Date(2024, 1, 1, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodMonthly, jan1), "Jan 1 should be incomplete")

	// Jan 30 → next day is Jan 31 (same month) → incomplete
	jan30 := time.Date(2024, 1, 30, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodMonthly, jan30), "Jan 30 should be incomplete")

	// Jan 31 → next day is Feb 1 (different month) → complete
	jan31 := time.Date(2024, 1, 31, 0, 0, 0, 0, jkt)
	assert.False(t, isPeriodIncomplete(PeriodMonthly, jan31), "Jan 31 should be complete")

	// Feb 28 non-leap → next day is Mar 1 (different month) → complete
	feb28 := time.Date(2023, 2, 28, 0, 0, 0, 0, jkt)
	assert.False(t, isPeriodIncomplete(PeriodMonthly, feb28), "Feb 28 non-leap should be complete")

	// Feb 28 leap → next day is Feb 29 (same month) → incomplete
	feb28Leap := time.Date(2024, 2, 28, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodMonthly, feb28Leap), "Feb 28 leap should be incomplete")

	// Feb 29 leap → next day is Mar 1 (different month) → complete
	feb29Leap := time.Date(2024, 2, 29, 0, 0, 0, 0, jkt)
	assert.False(t, isPeriodIncomplete(PeriodMonthly, feb29Leap), "Feb 29 leap should be complete")

	// Dec 31 → next day is Jan 1 next year (different month) → complete
	dec31 := time.Date(2024, 12, 31, 0, 0, 0, 0, jkt)
	assert.False(t, isPeriodIncomplete(PeriodMonthly, dec31), "Dec 31 should be complete")
}

// TestWeeklyRanges_IsPeriodIncomplete verifies weekly incomplete detection
// Only Sunday is complete (last day of week)
func TestWeeklyRanges_IsPeriodIncomplete(t *testing.T) {
	jkt := config.Load().Timezone

	// Monday → incomplete (not last day)
	monday := time.Date(2024, 1, 1, 0, 0, 0, 0, jkt) // Monday
	assert.True(t, isPeriodIncomplete(PeriodWeekly, monday), "Monday should be incomplete")

	// Tuesday → incomplete
	tuesday := time.Date(2024, 1, 2, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodWeekly, tuesday), "Tuesday should be incomplete")

	// Wednesday → incomplete
	wednesday := time.Date(2024, 1, 3, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodWeekly, wednesday), "Wednesday should be incomplete")

	// Thursday → incomplete
	thursday := time.Date(2024, 1, 4, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodWeekly, thursday), "Thursday should be incomplete")

	// Friday → incomplete
	friday := time.Date(2024, 1, 5, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodWeekly, friday), "Friday should be incomplete")

	// Saturday → incomplete
	saturday := time.Date(2024, 1, 6, 0, 0, 0, 0, jkt)
	assert.True(t, isPeriodIncomplete(PeriodWeekly, saturday), "Saturday should be incomplete")

	// Sunday → complete (next day is Monday, new week)
	sunday := time.Date(2024, 1, 7, 0, 0, 0, 0, jkt)
	assert.False(t, isPeriodIncomplete(PeriodWeekly, sunday), "Sunday should be complete")
}

// TestYearlyRanges_YearIncomplete verifies that current year is marked as incomplete
func TestYearlyRanges_YearIncomplete(t *testing.T) {
	jkt := config.Load().Timezone

	// June 4 (not Dec 31) should be incomplete for yearly
	june4_2024 := time.Date(2024, 6, 4, 0, 0, 0, 0, jkt)
	incomplete := isPeriodIncomplete(PeriodYearly, june4_2024)
	assert.True(t, incomplete, "Year should be incomplete before Dec 31")

	// Dec 31 should be complete
	dec31_2024 := time.Date(2024, 12, 31, 0, 0, 0, 0, jkt)
	incomplete = isPeriodIncomplete(PeriodYearly, dec31_2024)
	assert.False(t, incomplete, "Year should be complete on Dec 31")
}
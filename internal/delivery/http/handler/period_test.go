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

// TestWeeklyRanges_ThreeDayThreshold verifies that weeks starting Mon-Wed are disabled
// and weeks starting Thu-Sat are selectable for to-date mode
func TestWeeklyRanges_ThreeDayThreshold(t *testing.T) {
	jkt := config.Load().Timezone

	// Test case 1: Monday (dayOfWeek = 1) - should be incomplete, before threshold
	monday := time.Date(2026, 6, 1, 0, 0, 0, 0, jkt)
	ranges := GetComparisonRanges(PeriodWeekly, monday, false) // todate mode
	assert.True(t, ranges.IsPartial, "Monday week should be partial (before Thursday threshold)")

	// Test case 2: Tuesday (dayOfWeek = 2) - should be incomplete, before threshold
	tuesday := time.Date(2026, 6, 2, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, tuesday, false)
	assert.True(t, ranges.IsPartial, "Tuesday week should be partial (before Thursday threshold)")

	// Test case 3: Wednesday (dayOfWeek = 3) - should be incomplete, before threshold
	wednesday := time.Date(2026, 6, 3, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, wednesday, false)
	assert.True(t, ranges.IsPartial, "Wednesday week should be partial (before Thursday threshold)")

	// Test case 4: Thursday (dayOfWeek = 4) - should be complete, threshold met
	thursday := time.Date(2026, 6, 4, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, thursday, false)
	assert.False(t, ranges.IsPartial, "Thursday week should be complete (3+ days elapsed)")

	// Test case 5: Friday (dayOfWeek = 5) - should be complete
	friday := time.Date(2026, 6, 5, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, friday, false)
	assert.False(t, ranges.IsPartial, "Friday week should be complete")

	// Test case 6: Saturday (dayOfWeek = 6) - should be complete (full week done)
	saturday := time.Date(2026, 6, 6, 0, 0, 0, 0, jkt)
	ranges = GetComparisonRanges(PeriodWeekly, saturday, false)
	assert.False(t, ranges.IsPartial, "Saturday week should be complete")
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
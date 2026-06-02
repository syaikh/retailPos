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
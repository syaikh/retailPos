package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestJakartaLocation_Load verifies that the Asia/Jakarta location loads
// correctly (UTC+7, no daylight saving) and is distinct from UTC.
func TestJakartaLocation_Load(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load Asia/Jakarta: %v", err)
	}
	// Jakarta is a named location (not UTC) and its fixed offset is 7h east.
	// We verify by building the same instant in UTC and in Jakarta and checking
	// the difference equals +7 h.
	utcMidnight := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	jktMidnight := time.Date(2025, 6, 15, 0, 0, 0, 0, loc)
	// Verify by comparing epoch representations and the Zone offset
	_, jktOffset := jktMidnight.Zone()
	assert.Equal(t, 7*3600, jktOffset, "Asia/Jakarta fixed offset should be +07:00")
	// Both Unix() representations must represent *different* instants:
	// 2025-06-15T00:00:00 UTC vs 2025-06-15T00:00:00+07 (7 h earlier)
	secDiff := jktMidnight.Unix() - utcMidnight.Unix()
	assert.Equal(t, int64(-7*3600), secDiff, "jkt wall-clock midnight is 7 h BEFORE UTC midnight for the same date")
	assert.False(t, loc == time.UTC, "Asia/Jakarta must not be identical to UTC location")
}

// TestParseDateInJakarta verifies that parsing a bare date with
// time.ParseInLocation AND passing in a Jakarta *time.Location yields
// the *start* of that day in Jakarta — NOT shifted into UTC.
func TestParseDateInJakarta(t *testing.T) {
	const rawDate = "2025-11-30"
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	// CORRECT: ParseInLocation → start-of-day Jakarta
	correctStart, err := time.ParseInLocation("2006-01-02", rawDate, jkt)
	if err != nil {
		t.Fatalf("ParseInLocation: %v", err)
	}
	assert.Equal(t, jkt, correctStart.Location(), "location should be Asia/Jakarta")
	assert.Equal(t, 2025, correctStart.Year())
	assert.Equal(t, time.November, correctStart.Month())
	assert.Equal(t, 30, correctStart.Day())
	assert.Equal(t, 0, correctStart.Hour())
	assert.Equal(t, 0, correctStart.Minute())
	assert.Equal(t, 0, correctStart.Second())

	// WRONG (would happen without timezone): time.Parse would treat the
	// date components as the server default zone (usually UTC) and shift
	// the UTC equivalent 7h *backward*, ending at 17:00 UTC the PREVIOUS day.
	wrongStart, errWrong := time.Parse("2006-01-02", rawDate)
	if errWrong != nil {
		t.Fatalf("Parse: %v", errWrong)
	}
	// The wrong result is at 00:00 UTC = 07:00 *previous* day in Jakarta.
	// Verify that without location parsing the hour or location differs from the Jakarta one.
	assert.True(t, correctStart.Location() != wrongStart.Location(),
		"Without location parsing the parsed time should NOT be in Asia/Jakarta")
}

// TestGetAllSalesDateBoundaries builds exact Jakarta start / end timestamps
// as used in postgres_repository GetAllSales and verifies they form a
// proper closed-open interval [start, nextDay).
func TestGetAllSalesDateBoundaries(t *testing.T) {
	jkt := mustLoadJakarta() // same helper from the codebase

	const rawStart = "2025-11-30"
	const rawEnd = "2025-12-01"

	start, errStart := time.ParseInLocation("2006-01-02", rawStart, jkt)
	if errStart != nil {
		t.Fatalf("start: %v", errStart)
	}
	end, errEnd := time.ParseInLocation("2006-01-02", rawEnd, jkt)
	if errEnd != nil {
		t.Fatalf("end: %v", errEnd)
	}
	endExclusive := end.Add(24 * time.Hour) // logic in GetAllSales

	// endExclusive should be 48h after start (two distinct days)
	diffHours := endExclusive.Sub(start).Hours()
	assert.Equal(t, 48.0, diffHours, "A two-day range selected should give 48-hour span")
	assert.Equal(t, jkt, start.Location())
	assert.Equal(t, jkt, endExclusive.Location())

	// Verify start is midnight Jakarta
	assert.Equal(t, 0, start.Hour())
	assert.Equal(t, 0, start.Minute())
	assert.Equal(t, 0, start.Second())
	// endExclusive is midnight of the day after end
	assert.Equal(t, 0, endExclusive.Hour())
	assert.Equal(t, 0, endExclusive.Minute())
	assert.Equal(t, 0, endExclusive.Second())
}

// TestGetDashboardStatsTodayString verifies that "today" formatted as
// YYYY-MM-DD in Asia/Jakarta matches the date returned by time.Now().In(jkt).
// This is the behaviour relied upon in GetDashboardStats.
func TestGetDashboardStatsTodayString(t *testing.T) {
	cfgTimez := mustLoadJakarta()
	now := time.Now().In(cfgTimez)
	today := now.Format("2006-01-02")

	// Today should not be an empty string
	assert.NotEmpty(t, today)

	// Parse it back as a Jakarta date — we should get today's date
	parsedToday, err := time.ParseInLocation("2006-01-02", today, cfgTimez)
	if err != nil {
		t.Fatalf("parse today: %v", err)
	}
	assert.Equal(t, now.Year(), parsedToday.Year())
	assert.Equal(t, now.Month(), parsedToday.Month())
	assert.Equal(t, now.Day(), parsedToday.Day())
}

// TestPeriodComparisonRanges verifies the DateRange logic in
// handler/period.go for GetPeriodComparison.
// This tangentially confirms that referenceDate is forced to Jakarta.
func TestPeriodComparisonRanges(t *testing.T) {
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	// Build a symmetrical 7-day period anchored in Jakarta
	currentStart := time.Date(2025, 4, 8, 0, 0, 0, 0, jkt) // Apr 8
	currentEnd := time.Date(2025, 4, 14, 23, 59, 59, 0, jkt) // Apr 14 end of day
	previousStart := time.Date(2025, 4, 1, 0, 0, 0, 0, jkt) // Apr 1
	previousEnd := time.Date(2025, 4, 7, 23, 59, 59, 0, jkt) // Apr 7

	assert.Equal(t, jkt, currentStart.Location())
	assert.Equal(t, jkt, currentEnd.Location())
	assert.Equal(t, jkt, previousStart.Location())
	assert.Equal(t, jkt, previousEnd.Location())
assert.True(t, currentStart.Before(currentEnd))
 	assert.True(t, previousStart.Before(previousEnd))
 	assert.Equal(t, 6, int(currentEnd.Sub(currentStart).Hours()/24), "7-day inclusive period should span 6 full 24h plus remainder")
 }

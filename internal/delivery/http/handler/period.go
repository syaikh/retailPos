package handler

import (
	"time"

	"retail-pos-system/internal/config"
)

// PeriodType defines the comparison granularity
type PeriodType string

const (
  PeriodDaily   PeriodType = "daily"
  PeriodWeekly  PeriodType = "weekly"
  PeriodMonthly PeriodType = "monthly"
  PeriodYearly  PeriodType = "yearly"
)

// PeriodRange defines a time range for comparison
type PeriodRange struct {
	CurrentStart  time.Time
	CurrentEnd    time.Time
	PreviousStart time.Time
	PreviousEnd   time.Time
	IsPartial     bool
	DaysInPeriod  int
}

// GetComparisonRanges calculates aligned period ranges
//
// For To-Date mode (default):
// - Compares same number of days (e.g., May 1-6 vs Apr 1-6)
//
// For Completed mode:
// - Compares full previous period (e.g., Last week vs previous week)
func GetComparisonRanges(
	periodType PeriodType,
	referenceDate time.Time,
	completedMode bool,
) PeriodRange {

	cfg := config.Load()
	// Normalize referenceDate to the configured timezone (default Asia/Jakarta)
	refDate := time.Date(referenceDate.Year(), referenceDate.Month(),
		referenceDate.Day(), 0, 0, 0, 0, cfg.Timezone)

	var pr PeriodRange

	switch periodType {
	case PeriodDaily:
		pr = getDailyRanges(refDate, completedMode)
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
	// For weekly: isPartial if not last day of week (Mon-Sat), matching monthly pattern
	// For other periods in completed mode: use existing logic
	pr.IsPartial = isPeriodIncomplete(periodType, refDate) && (periodType == PeriodWeekly || completedMode)

	return pr
}

func getDailyRanges(refDate time.Time, completedMode bool) PeriodRange {
	if completedMode {
		// For yesterday view: compare yesterday full day vs same day last week (H-7) full day
		// refDate represents yesterday's date
		sevenDaysAgo := refDate.AddDate(0, 0, -7)
		return PeriodRange{
			CurrentStart:  refDate,
			CurrentEnd:    refDate.AddDate(0, 0, 1),
			PreviousStart: sevenDaysAgo,
			PreviousEnd:   sevenDaysAgo.AddDate(0, 0, 1),
		}
	}

	// To-date: 7 days ending at refDate (yesterday) vs previous 7 days
	// For refDate = Jun 1 (yesterday), returns May 26 - Jun 1 vs May 19 - May 25
	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -6), // 7 days including refDate
		CurrentEnd:    refDate.AddDate(0, 0, 1),  // refDate + 1 day (exclusive)
		PreviousStart: refDate.AddDate(0, 0, -13), // 13 days before refDate
		PreviousEnd:   refDate.AddDate(0, 0, -6),   // 7 days before refDate (exclusive)
	}
}

// get30DaysRanges calculates 30-day comparison period
// refDate represents the end date (yesterday)
func get30DaysRanges(refDate time.Time) PeriodRange {
	// 30 days ending at refDate (exclusive) vs previous 30 days
	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -29), // 30 days ending at refDate
		CurrentEnd:    refDate.AddDate(0, 0, 1),   // refDate + 1 day (exclusive)
		PreviousStart: refDate.AddDate(0, 0, -59),   // 59 days before refDate
		PreviousEnd:   refDate.AddDate(0, 0, -29),   // 30 days before refDate (exclusive)
	}
}

func getWeeklyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfWeek := refDate.AddDate(0, 0, -int(refDate.Weekday()-time.Monday))
	if refDate.Weekday() == time.Sunday {
		startOfWeek = refDate.AddDate(0, 0, -6)
	}

	if completedMode {
		// Last full week (Mon-Sun): compare previous week to week before
		// For refDate = Jun 1 (Monday), current week = May 25 - May 31, previous = May 18-24
		endOfWeek := startOfWeek.AddDate(0, 0, 7) // Sunday of current week

		return PeriodRange{
			CurrentStart:  startOfWeek.AddDate(0, 0, -7), // Monday of previous week (May 18)
			CurrentEnd:    endOfWeek,                    // Sunday of current week (exclusive) (Jun 8)
			PreviousStart: startOfWeek.AddDate(0, 0, -14), // Monday of previous-previous week (May 11)
			PreviousEnd:   startOfWeek.AddDate(0, 0, -7),  // Monday of previous week (exclusive) (May 25)
		}
	}

	// To-date: same number of days
	daysElapsed := int(refDate.Sub(startOfWeek).Hours()/24) + 1
	previousStart := startOfWeek.AddDate(0, 0, -7)

	// Like-for-like comparison for all days (matching monthly MTD pattern)
	// Example: Tuesday (daysElapsed=2) -> prevEnd = previousStart + 2 days
	var currentStart, currentEnd, prevStart, prevEnd time.Time
	currentStart = startOfWeek
	currentEnd = refDate.AddDate(0, 0, 1) // Include today
	prevStart = previousStart
	prevEnd = previousStart.AddDate(0, 0, daysElapsed) // Same number of days as current period

	return PeriodRange{
		CurrentStart:  currentStart,
		CurrentEnd:    currentEnd,
		PreviousStart: prevStart,
		PreviousEnd:   prevEnd,
	}
}

// getRealtimeRanges compares today (00:00 through current hour, including partial data) vs yesterday (same hours)
// Example: refDate=07:39, returns today [00:00, 08:00) vs yesterday [00:00, 08:00) (includes partial hour 7 data)
func getRealtimeRanges(refDate time.Time) PeriodRange {
	// refDate is the current time (e.g., 07:39 Jakarta)
	// Include the current (partial) hour — end at currentHour+1 (exclusive)
	// At 07:39: currentHour=7, end=08:00 → covers [00:00, 08:00) = hours 0-7
	currentHour := refDate.Hour()
	currentPeriodEnd := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), currentHour+1, 0, 0, 0, refDate.Location())

	// Yesterday at same hours
	yesterdayStart := refDate.AddDate(0, 0, -1)
	yesterdaySamePeriodEnd := time.Date(yesterdayStart.Year(), yesterdayStart.Month(), yesterdayStart.Day(), currentHour+1, 0, 0, 0, yesterdayStart.Location())

	return PeriodRange{
		CurrentStart:  time.Date(refDate.Year(), refDate.Month(), refDate.Day(), 0, 0, 0, 0, refDate.Location()),    // today 00:00
		CurrentEnd:    currentPeriodEnd,                                                                            // currentHour+1:00 (exclusive) — includes partial hour
		PreviousStart: time.Date(yesterdayStart.Year(), yesterdayStart.Month(), yesterdayStart.Day(), 0, 0, 0, 0, yesterdayStart.Location()), // yesterday 00:00
		PreviousEnd:   yesterdaySamePeriodEnd,                                                                      // same hour+1 yesterday (exclusive)
	}
}

func getMonthlyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfMonth := time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())

	if completedMode {
		// Last full month
		lastMonthEnd := startOfMonth
		lastMonthStart := lastMonthEnd.AddDate(0, -1, 0)

		return PeriodRange{
			CurrentStart:  lastMonthStart,
			CurrentEnd:    lastMonthEnd,
			PreviousStart: lastMonthStart.AddDate(0, -1, 0),
			PreviousEnd:   lastMonthStart,
		}
	}

	// To-date: same number of days
	daysElapsed := refDate.Day()
	previousStart := startOfMonth.AddDate(0, -1, 0)
	previousEnd := previousStart.AddDate(0, 0, daysElapsed)
	// Cap previousEnd to not exceed startOfMonth — prevents overflow when
	// previous month has fewer days (e.g., May 31 → Apr 1 + 31 = May 2)
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
		// Last full year (december of previous year)
		lastYearEnd := startOfYear
		lastYearStart := lastYearEnd.AddDate(-1, 0, 0)

		return PeriodRange{
			CurrentStart:  lastYearStart,
			CurrentEnd:    lastYearEnd,
			PreviousStart: lastYearStart.AddDate(-1, 0, 0),
			PreviousEnd:   lastYearStart, // End at Jan 1 of lastYearStart (exclusive), so full year before that
		}
	}

	// To-date: year-to-date vs same period last year
	// For refDate = Dec 31, 2024: compare Jan 1-Dec 31 2024 vs Jan 1-Dec 31 2023
	// For refDate = May 31, 2026: compare Jan 1-May 31 2026 vs Jan 1-May 31 2025
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
		// Week is incomplete if not last day (matching monthly pattern)
		// Mon-Sat -> incomplete (next day still same week)
		// Sunday -> complete (next day is Monday = new week)
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Weekday() != time.Monday
	case PeriodMonthly:
		// Month is incomplete if not last day
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Month() == refDate.Month()
	case PeriodYearly:
		// Year is incomplete if not December 31
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Year() == refDate.Year() || refDate.Month() != time.December
	default:
		return false
	}
}
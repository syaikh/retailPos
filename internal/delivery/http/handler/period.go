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
	// For weekly: isPartial if before Thursday (Mon-Wed: 0-2 days completed)
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

	// For partial weeks (not full week completed), use like-for-like comparison
	// Compare Mon-Fri of this week vs Mon-Fri of last week
	// This applies when we're in the middle of a week (Tue-Sat)
	weekday := refDate.Weekday()
	var currentStart, currentEnd, prevStart, prevEnd time.Time
	
	if weekday == time.Saturday {
		// Saturday - full week completed, compare full weeks
		currentStart = startOfWeek
		currentEnd = refDate.AddDate(0, 0, 1) // Sunday + 1 = next day
		prevStart = previousStart
		prevEnd = previousStart.AddDate(0, 0, 7) // Full previous week
	} else {
		// Partial week - use like-for-like (same days vs last week)
		// Example: Thursday (weekday=4) -> daysElapsed=4 -> prevEnd = previousStart + 4 days
		currentStart = startOfWeek
		currentEnd = refDate.AddDate(0, 0, 1) // Include today
		prevStart = previousStart
		prevEnd = previousStart.AddDate(0, 0, daysElapsed) // Same number of days as current period
	}

	return PeriodRange{
		CurrentStart:  currentStart,
		CurrentEnd:    currentEnd,
		PreviousStart: prevStart,
		PreviousEnd:   prevEnd,
	}
}

// getRealtimeRanges compares today (00:00 to last full hour inclusive) vs yesterday (same hours)
// Example: at 07:39, compares 00:00-07:00 today (inclusive) vs 00:00-07:00 yesterday (inclusive)
func getRealtimeRanges(refDate time.Time) PeriodRange {
	// refDate is today at 00:00, get current time
	now := time.Now().In(refDate.Location())
	
	// Only use full hours - add 1 hour to include the last full hour
	currentHour := now.Hour()
	// To include hour 05 (inclusive), end should be 06:00 (exclusive)
	lastFullHourExclusive := time.Date(now.Year(), now.Month(), now.Day(), currentHour+1, 0, 0, 0, now.Location())
	
	yesterday := refDate.AddDate(0, 0, -1) // yesterday at 00:00
	yesterdaySameHourExclusive := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), currentHour+1, 0, 0, 0, yesterday.Location())

	return PeriodRange{
		CurrentStart:  refDate,      // today 00:00
		CurrentEnd:    lastFullHourExclusive,  // currentHour+1 00:00 (exclusive)
		PreviousStart: yesterday,   // yesterday 00:00
		PreviousEnd:   yesterdaySameHourExclusive, // same hour+1 yesterday (exclusive)
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

	return PeriodRange{
		CurrentStart:  startOfMonth,
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: previousStart,
		PreviousEnd:   previousStart.AddDate(0, 0, daysElapsed),
	}
}

func getYearlyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfYear := time.Date(refDate.Year(), 1, 1, 0, 0, 0, 0, refDate.Location())

	if completedMode {
		// Last full year
		lastYearEnd := startOfYear
		lastYearStart := lastYearEnd.AddDate(-1, 0, 0)

		return PeriodRange{
			CurrentStart:  lastYearStart,
			CurrentEnd:    lastYearEnd,
			PreviousStart: lastYearStart.AddDate(-1, 0, 0),
			PreviousEnd:   lastYearStart,
		}
	}

	// To-date: same number of days from start of year
	daysElapsed := refDate.YearDay()
	previousStart := startOfYear.AddDate(-1, 0, 0)

	return PeriodRange{
		CurrentStart:  startOfYear,
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: previousStart,
		PreviousEnd:   previousStart.AddDate(0, 0, daysElapsed),
	}
}

func isPeriodIncomplete(periodType PeriodType, refDate time.Time) bool {
	switch periodType {
	case PeriodWeekly:
		// Week is incomplete if before Thursday (weekday < 4)
		// Mon=1, Tue=2, Wed=3 -> incomplete (less than 3 days completed)
		// Thu=4, Fri=5, Sat=6, Sun=0 -> complete (3+ days)
		weekday := refDate.Weekday()
		// Sunday = 0, but we treat it as complete (6 days done)
		// Saturday = 6, complete (full week)
		return weekday >= time.Monday && weekday <= time.Wednesday
	case PeriodMonthly:
		// Month is incomplete if not last day
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Month() == refDate.Month()
	default:
		return false
	}
}
package handler

import (
	"time"
)

// PeriodType defines the comparison granularity
type PeriodType string

const (
	PeriodDaily   PeriodType = "daily"
	PeriodWeekly  PeriodType = "weekly"
	PeriodMonthly PeriodType = "monthly"
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

	// Normalize to start of day
	refDate := time.Date(referenceDate.Year(), referenceDate.Month(),
		referenceDate.Day(), 0, 0, 0, 0, referenceDate.Location())

	var pr PeriodRange

	switch periodType {
	case PeriodDaily:
		pr = getDailyRanges(refDate, completedMode)
	case PeriodWeekly:
		pr = getWeeklyRanges(refDate, completedMode)
	case PeriodMonthly:
		pr = getMonthlyRanges(refDate, completedMode)
	default:
		pr = getDailyRanges(refDate, completedMode)
	}

	pr.DaysInPeriod = int(pr.CurrentEnd.Sub(pr.CurrentStart).Hours() / 24)
	pr.IsPartial = completedMode && isPeriodIncomplete(periodType, refDate)

	return pr
}

func getDailyRanges(refDate time.Time, completedMode bool) PeriodRange {
	if completedMode {
		// Yesterday vs day before
		return PeriodRange{
			CurrentStart:  refDate.AddDate(0, 0, -1),
			CurrentEnd:    refDate,
			PreviousStart: refDate.AddDate(0, 0, -2),
			PreviousEnd:   refDate.AddDate(0, 0, -1),
		}
	}

	// Today vs yesterday (same day offset)
	return PeriodRange{
		CurrentStart:  refDate,
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -1),
		PreviousEnd:   refDate,
	}
}

func getWeeklyRanges(refDate time.Time, completedMode bool) PeriodRange {
	// Start of week (Monday)
	startOfWeek := refDate.AddDate(0, 0, -int(refDate.Weekday()-time.Monday))
	if refDate.Weekday() == time.Sunday {
		startOfWeek = refDate.AddDate(0, 0, -6)
	}

	if completedMode {
		// Last full week (Sun-Sat or Mon-Sun)
		lastWeekEnd := startOfWeek
		lastWeekStart := lastWeekEnd.AddDate(0, 0, -7)

		return PeriodRange{
			CurrentStart:  lastWeekStart,
			CurrentEnd:    lastWeekEnd,
			PreviousStart: lastWeekStart.AddDate(0, 0, -7),
			PreviousEnd:   lastWeekStart,
		}
	}

	// To-date: same number of days
	daysElapsed := int(refDate.Sub(startOfWeek).Hours()/24) + 1
	previousStart := startOfWeek.AddDate(0, 0, -7)

	return PeriodRange{
		CurrentStart:  startOfWeek,
		CurrentEnd:    refDate.AddDate(0, 0, 1), // inclusive of today
		PreviousStart: previousStart,
		PreviousEnd:   previousStart.AddDate(0, 0, daysElapsed),
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

func isPeriodIncomplete(periodType PeriodType, refDate time.Time) bool {
	switch periodType {
	case PeriodWeekly:
		// Week is incomplete if not Saturday
		return refDate.Weekday() != time.Saturday
	case PeriodMonthly:
		// Month is incomplete if not last day
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Month() == refDate.Month()
	default:
		return false
	}
}
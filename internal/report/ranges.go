package report

import (
	"net/http"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type PeriodType string

const (
	PeriodDaily   PeriodType = "daily"
	Period7Days   PeriodType = "7days"
	PeriodWeekly  PeriodType = "weekly"
	PeriodMonthly PeriodType = "monthly"
	PeriodYearly  PeriodType = "yearly"
)

type PeriodRange struct {
	CurrentStart  time.Time
	CurrentEnd    time.Time
	PreviousStart time.Time
	PreviousEnd   time.Time
	IsPartial     bool
	DaysInPeriod  int
}

type dateRange struct {
	Start time.Time
	End   time.Time
}

func getComparisonRanges(
	periodType PeriodType,
	referenceDate time.Time,
	completedMode bool,
) PeriodRange {
	refDate := time.Date(referenceDate.Year(), referenceDate.Month(),
		referenceDate.Day(), 0, 0, 0, 0, shared.JakartaLocation())

	var pr PeriodRange

	switch periodType {
	case PeriodDaily:
		pr = getDailyRanges(refDate, completedMode)
	case Period7Days:
		pr = get7DaysRanges(refDate, completedMode)
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
	pr.IsPartial = isPeriodIncomplete(periodType, refDate) && (periodType == PeriodWeekly || periodType == PeriodMonthly || completedMode)

	return pr
}

func getDailyRanges(refDate time.Time, completedMode bool) PeriodRange {
	if completedMode {
		sevenDaysAgo := refDate.AddDate(0, 0, -7)
		return PeriodRange{
			CurrentStart:  refDate,
			CurrentEnd:    refDate.AddDate(0, 0, 1),
			PreviousStart: sevenDaysAgo,
			PreviousEnd:   sevenDaysAgo.AddDate(0, 0, 1),
		}
	}

	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -6),
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -13),
		PreviousEnd:   refDate.AddDate(0, 0, -6),
	}
}

func get7DaysRanges(refDate time.Time, completedMode bool) PeriodRange {
	if completedMode {
		weekday := refDate.Weekday()
		daysSinceMonday := int(weekday - time.Monday)
		if weekday == time.Sunday {
			daysSinceMonday = 6
		}
		weekStart := refDate.AddDate(0, 0, -daysSinceMonday)
		return PeriodRange{
			CurrentStart:  weekStart.AddDate(0, 0, -7),
			CurrentEnd:    weekStart,
			PreviousStart: weekStart.AddDate(0, 0, -14),
			PreviousEnd:   weekStart.AddDate(0, 0, -7),
		}
	}
	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -6),
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -13),
		PreviousEnd:   refDate.AddDate(0, 0, -6),
	}
}

func get30DaysRanges(refDate time.Time) PeriodRange {
	return PeriodRange{
		CurrentStart:  refDate.AddDate(0, 0, -29),
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: refDate.AddDate(0, 0, -59),
		PreviousEnd:   refDate.AddDate(0, 0, -29),
	}
}

func getWeeklyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfWeek := refDate.AddDate(0, 0, -int(refDate.Weekday()-time.Monday))
	if refDate.Weekday() == time.Sunday {
		startOfWeek = refDate.AddDate(0, 0, -6)
	}

	if completedMode {
		endOfWeek := startOfWeek.AddDate(0, 0, 7)
		return PeriodRange{
			CurrentStart:  startOfWeek.AddDate(0, 0, -7),
			CurrentEnd:    endOfWeek,
			PreviousStart: startOfWeek.AddDate(0, 0, -14),
			PreviousEnd:   startOfWeek.AddDate(0, 0, -7),
		}
	}

	daysElapsed := int(refDate.Sub(startOfWeek).Hours()/24) + 1
	previousStart := startOfWeek.AddDate(0, 0, -7)

	return PeriodRange{
		CurrentStart:  startOfWeek,
		CurrentEnd:    refDate.AddDate(0, 0, 1),
		PreviousStart: previousStart,
		PreviousEnd:   previousStart.AddDate(0, 0, daysElapsed),
	}
}

// getRealtimeRanges bounds both periods to the start of the in-progress hour
// (exclusive), so only completed hour buckets are compared. Example: at 11:20
// the current period is [00:00, 11:00) — hours 00..10 — matching yesterday's
// [00:00, 11:00). Ending at currentHour+1 would include the partial 11:00
// bucket that a mid-hour MV refresh (startup, retry, manual) may have created.
func getRealtimeRanges(refDate time.Time) PeriodRange {
	currentHour := refDate.Hour()
	currentPeriodEnd := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), currentHour, 0, 0, 0, refDate.Location())

	yesterdayStart := refDate.AddDate(0, 0, -1)
	yesterdaySamePeriodEnd := time.Date(yesterdayStart.Year(), yesterdayStart.Month(), yesterdayStart.Day(), currentHour, 0, 0, 0, yesterdayStart.Location())

	return PeriodRange{
		CurrentStart:  time.Date(refDate.Year(), refDate.Month(), refDate.Day(), 0, 0, 0, 0, refDate.Location()),
		CurrentEnd:    currentPeriodEnd,
		PreviousStart: time.Date(yesterdayStart.Year(), yesterdayStart.Month(), yesterdayStart.Day(), 0, 0, 0, 0, yesterdayStart.Location()),
		PreviousEnd:   yesterdaySamePeriodEnd,
	}
}

func getMonthlyRanges(refDate time.Time, completedMode bool) PeriodRange {
	startOfMonth := time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())

	if completedMode {
		lastMonthEnd := startOfMonth
		lastMonthStart := lastMonthEnd.AddDate(0, -1, 0)
		return PeriodRange{
			CurrentStart:  lastMonthStart,
			CurrentEnd:    lastMonthEnd,
			PreviousStart: lastMonthStart.AddDate(0, -1, 0),
			PreviousEnd:   lastMonthStart,
		}
	}

	daysElapsed := refDate.Day()
	previousStart := startOfMonth.AddDate(0, -1, 0)
	previousEnd := previousStart.AddDate(0, 0, daysElapsed)
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
		lastYearEnd := startOfYear
		lastYearStart := lastYearEnd.AddDate(-1, 0, 0)
		return PeriodRange{
			CurrentStart:  lastYearStart,
			CurrentEnd:    lastYearEnd,
			PreviousStart: lastYearStart.AddDate(-1, 0, 0),
			PreviousEnd:   lastYearStart,
		}
	}

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
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Weekday() != time.Monday
	case PeriodMonthly:
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Month() == refDate.Month()
	case PeriodYearly:
		nextDay := refDate.AddDate(0, 0, 1)
		return nextDay.Year() == refDate.Year() || refDate.Month() != time.December
	default:
		return false
	}
}

func parseDateRange(c *gin.Context, defaultStartDaysAgo int) (dateRange, bool) {
	jakartaLoc := shared.JakartaLocation()
	now := time.Now().In(jakartaLoc)
	startDateStr := c.DefaultQuery("startDate", now.AddDate(0, 0, -defaultStartDaysAgo).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("endDate", now.Format("2006-01-02"))

	startDate, err := time.ParseInLocation("2006-01-02", startDateStr, jakartaLoc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate"})
		return dateRange{}, false
	}
	endDate, err := time.ParseInLocation("2006-01-02", endDateStr, jakartaLoc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate"})
		return dateRange{}, false
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endDate must not be before startDate"})
		return dateRange{}, false
	}
	if startDate.AddDate(0, 0, 366).Before(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date range must not exceed 366 days"})
		return dateRange{}, false
	}
	return dateRange{Start: startDate, End: endDate}, true
}

func parseDateParam(c *gin.Context, paramName string) (time.Time, bool) {
	val := c.Query(paramName)
	if val == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", val, shared.JakartaLocation())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + paramName})
		return time.Time{}, false
	}
	return t, true
}

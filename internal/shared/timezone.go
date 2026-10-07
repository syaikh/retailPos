package shared

import (
	"errors"
	"time"
)

var (
	jakartaLoc   *time.Location
	loadLocation = time.LoadLocation
)

func init() {
	loadJakartaLocation()
}

func loadJakartaLocation() {
	var err error
	jakartaLoc, err = loadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
}

func JakartaLocation() *time.Location {
	return jakartaLoc
}

// ParseJakartaFilterDate parses a date filter value from a query parameter. It
// accepts both a full RFC3339 timestamp and a plain YYYY-MM-DD date; the date
// form is interpreted in the Asia/Jakarta location. Empty inputs and any other
// value are rejected with an error, so every caller surfaces an invalid filter
// instead of silently dropping it and returning unfiltered data (backend
// review X2).
func ParseJakartaFilterDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("date filter is empty")
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(jakartaLoc), nil
	}
	return time.ParseInLocation("2006-01-02", s, jakartaLoc)
}

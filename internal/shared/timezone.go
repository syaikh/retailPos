package shared

import "time"

var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
}

func JakartaLocation() *time.Location {
	return jakartaLoc
}

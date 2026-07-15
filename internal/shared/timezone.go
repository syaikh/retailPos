package shared

import "time"

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

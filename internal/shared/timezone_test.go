package shared

import (
	"fmt"
	"testing"
	"time"
)

func TestJakartaLocation(t *testing.T) {
	loc := JakartaLocation()
	if loc == nil {
		t.Fatal("expected non-nil location")
	}
	if loc.String() != "Asia/Jakarta" {
		t.Errorf("expected Asia/Jakarta, got %s", loc.String())
	}
}

func TestJakartaLocation_ReturnsSameInstance(t *testing.T) {
	loc1 := JakartaLocation()
	loc2 := JakartaLocation()
	if loc1 != loc2 {
		t.Error("expected same pointer on repeated calls")
	}
}

func TestJakartaLocation_CanConvertTime(t *testing.T) {
	loc := JakartaLocation()
	now := time.Now().In(loc)
	if now.Location() != loc {
		t.Error("expected time in Jakarta location")
	}
}

func TestInitLocation_FallbackToUTC(t *testing.T) {
	orig := loadLocation
	origLoc := jakartaLoc
	defer func() {
		loadLocation = orig
		jakartaLoc = origLoc
	}()

	loadLocation = func(name string) (*time.Location, error) {
		return nil, fmt.Errorf("tzdata missing")
	}

	loadJakartaLocation()

	if jakartaLoc != time.UTC {
		t.Errorf("expected UTC fallback, got %s", jakartaLoc)
	}
}

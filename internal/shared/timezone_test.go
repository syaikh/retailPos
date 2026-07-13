package shared

import (
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

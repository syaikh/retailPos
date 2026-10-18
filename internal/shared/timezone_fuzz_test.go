package shared

import (
	"testing"
	"time"
)

// FuzzParseJakartaFilterDate exercises the report date-filter parser, which is
// the entry point for every dashboard/report range. It accepts a full RFC3339
// timestamp or a plain YYYY-MM-DD date interpreted in Asia/Jakarta, and rejects
// empty or unrecognised input so a malformed filter surfaces instead of
// silently returning unfiltered data.
func FuzzParseJakartaFilterDate(f *testing.F) {
	f.Add("")
	f.Add("2026-10-18")
	f.Add("2026-10-18T15:04:05Z")
	f.Add("2026-10-18T15:04:05+07:00")
	f.Add("2026-10-18 15:04:05")
	f.Add("2026-13-45")
	f.Add("not-a-date")

	f.Fuzz(func(t *testing.T, s string) {
		got, err := ParseJakartaFilterDate(s)

		_, rfcErr := time.Parse(time.RFC3339, s)
		_, dateErr := time.Parse("2006-01-02", s)
		parseable := rfcErr == nil || dateErr == nil

		switch {
		case err != nil && parseable:
			t.Fatalf("ParseJakartaFilterDate(%q) failed on an accepted layout: %v", s, err)
		case err == nil && !parseable:
			t.Fatalf("ParseJakartaFilterDate(%q) accepted input matching neither accepted layout", s)
		case err != nil:
			return
		}

		if got.Location().String() != "Asia/Jakarta" {
			t.Fatalf("ParseJakartaFilterDate(%q) returned location %q, want Asia/Jakarta", s, got.Location())
		}
		if again, err2 := ParseJakartaFilterDate(s); err2 != nil || !again.Equal(got) {
			t.Fatalf("ParseJakartaFilterDate(%q) not deterministic: %v/%v vs %v/%v", s, got, err, again, err2)
		}
		// The date form must round-trip in Jakarta time; a value that shifts a
		// day would move report boundaries.
		if dateErr == nil && got.Format("2006-01-02") != s {
			t.Fatalf("ParseJakartaFilterDate(%q) round-trip = %q", s, got.Format("2006-01-02"))
		}
	})
}

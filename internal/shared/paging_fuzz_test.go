package shared

import (
	"strconv"
	"testing"
)

// FuzzParsePaginationParams exercises the query-parameter parser every list
// endpoint funnels through. The contract is: malformed or non-positive limits
// fall back to 20, limits clamp at DefaultMaxPageLimit, negative offsets clamp
// to 0, and an offset past MaxPageOffset is rejected with (0, 0, error) so the
// handler never forwards an unbounded scan.
func FuzzParsePaginationParams(f *testing.F) {
	f.Add("", "")
	f.Add("20", "0")
	f.Add("50", "100")
	f.Add("100000", "999999")
	f.Add("-5", "-1")
	f.Add("abc", "12x")
	f.Add("0", "10000")

	f.Fuzz(func(t *testing.T, limitStr, offsetStr string) {
		limit, offset, err := ParsePaginationParams(limitStr, offsetStr)

		// Reference: independent re-derivation of the documented contract.
		wantLimit, _ := strconv.Atoi(limitStr)
		switch {
		case wantLimit <= 0:
			wantLimit = 20
		case wantLimit > DefaultMaxPageLimit:
			wantLimit = DefaultMaxPageLimit
		}
		wantOffset, _ := strconv.Atoi(offsetStr)
		if wantOffset < 0 {
			wantOffset = 0
		}
		wantErr := wantOffset > MaxPageOffset

		if wantErr {
			if err == nil {
				t.Fatalf("ParsePaginationParams(%q, %q) accepted offset %d above MaxPageOffset %d",
					limitStr, offsetStr, wantOffset, MaxPageOffset)
			}
			return
		}
		if err != nil {
			t.Fatalf("ParsePaginationParams(%q, %q) returned unexpected error: %v", limitStr, offsetStr, err)
		}
		if limit != wantLimit {
			t.Fatalf("ParsePaginationParams(%q, %q) limit = %d, want %d", limitStr, offsetStr, limit, wantLimit)
		}
		if offset != wantOffset {
			t.Fatalf("ParsePaginationParams(%q, %q) offset = %d, want %d", limitStr, offsetStr, offset, wantOffset)
		}
		if limit < 1 || limit > DefaultMaxPageLimit || offset < 0 || offset > MaxPageOffset {
			t.Fatalf("ParsePaginationParams(%q, %q) returned out-of-range (%d, %d)", limitStr, offsetStr, limit, offset)
		}
	})
}

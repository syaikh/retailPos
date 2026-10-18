package product

import (
	"strconv"
	"strings"
	"testing"
)

// FuzzParseIDs exercises the comma-separated id-list parser used by query
// filters. It must never panic on arbitrary strings, must drop every
// non-positive or non-numeric token, and must de-duplicate while preserving
// first-seen order so generated SQL filters stay stable and bounded.
func FuzzParseIDs(f *testing.F) {
	f.Add("")
	f.Add("1")
	f.Add("1,2,3")
	f.Add("1, 2, 1, 3")
	f.Add("0,-1,abc,,2")
	f.Add("2147483648")
	f.Add(",")

	f.Fuzz(func(t *testing.T, raw string) {
		got := parseIDs(raw)

		// Reference: independent re-derivation of the documented contract.
		var want []int
		seen := make(map[int]bool)
		for _, token := range strings.Split(raw, ",") {
			v, err := strconv.Atoi(strings.TrimSpace(token))
			if err != nil || v <= 0 || seen[v] {
				continue
			}
			seen[v] = true
			want = append(want, v)
		}

		if len(got) != len(want) {
			t.Fatalf("parseIDs(%q) = %v, want %v", raw, got, want)
		}
		for i, id := range got {
			if id != want[i] {
				t.Fatalf("parseIDs(%q) = %v, want %v", raw, got, want)
			}
			if id <= 0 {
				t.Fatalf("parseIDs(%q) returned non-positive id %d", raw, id)
			}
		}
	})
}

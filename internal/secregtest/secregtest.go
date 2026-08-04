// Package secregtest is a data-driven security regression harness. It asserts
// how fields must appear in JSON API responses for a given caller (role /
// permission set), so that a sensitive field accidentally re-added to a
// response fails CI immediately.
//
// A field expectation has one of three states — visible, null, or absent —
// because sanitizer implementations differ (some omit the field, some leave it
// null). See docs/audits/sensitive-data-audit.md §Regression Test Strategy.
package secregtest

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// State describes how a field must appear in a JSON response.
type State string

const (
	// StateVisible requires the key to be present with a non-null value.
	StateVisible State = "visible"
	// StateNull requires the key to be present with a null value.
	StateNull State = "null"
	// StateAbsent requires the key to be absent.
	StateAbsent State = "absent"
)

// Field binds a JSON path to an expected state. A path is dot-separated:
// object keys and array indexes interleaved, e.g. "data.items.0.cost".
type Field struct {
	Path  string
	State State
}

// Visible returns an expectation that path must hold a non-null value.
func Visible(path string) Field { return Field{Path: path, State: StateVisible} }

// Null returns an expectation that path must be present and null.
func Null(path string) Field { return Field{Path: path, State: StateNull} }

// Absent returns an expectation that path must not appear in the response.
func Absent(path string) Field { return Field{Path: path, State: StateAbsent} }

// Check asserts that body satisfies every field expectation. Body must be a
// valid JSON document.
func Check(t *testing.T, body []byte, fields ...Field) {
	t.Helper()
	for _, err := range checkErr(body, fields...) {
		t.Errorf("%s", err)
	}
}

// checkErr returns every expectation violation as an error. It is separated
// from Check so the harness can be unit-tested without intercepting *testing.T.
func checkErr(body []byte, fields ...Field) []error {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return []error{fmt.Errorf("secregtest: invalid JSON body: %v", err)}
	}

	var errs []error
	for _, f := range fields {
		if err := checkField(root, f); err != nil {
			errs = append(errs, fmt.Errorf("secregtest: field %q: %s", f.Path, err))
		}
	}
	return errs
}

// checkField validates a single field expectation against the parsed document.
func checkField(root any, f Field) error {
	value, found := lookup(root, strings.Split(f.Path, "."))
	switch f.State {
	case StateVisible:
		switch {
		case !found:
			return fmt.Errorf("expected visible, got absent")
		case value == nil:
			return fmt.Errorf("expected visible, got null")
		}
	case StateNull:
		switch {
		case !found:
			return fmt.Errorf("expected null, got absent")
		case value != nil:
			return fmt.Errorf("expected null, got %v", value)
		}
	case StateAbsent:
		if found {
			return fmt.Errorf("expected absent, got %v", value)
		}
	default:
		return fmt.Errorf("unknown state %q", f.State)
	}
	return nil
}

// lookup navigates node using segments, returning the final value and whether
// every segment resolved. A numeric segment indexes into an array.
func lookup(node any, segments []string) (any, bool) {
	if len(segments) == 0 {
		return node, true
	}

	seg := segments[0]
	switch n := node.(type) {
	case map[string]any:
		v, ok := n[seg]
		if !ok {
			return nil, false
		}
		return lookup(v, segments[1:])
	case []any:
		idx, err := strconv.Atoi(seg)
		if err != nil || idx < 0 || idx >= len(n) {
			return nil, false
		}
		return lookup(n[idx], segments[1:])
	default:
		return nil, false
	}
}

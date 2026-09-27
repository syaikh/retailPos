package shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name       string
		limitStr   string
		offsetStr  string
		expectLim  int
		expectOffs int
		wantErr    bool
	}{
		{"valid limit and offset", "10", "0", 10, 0, false},
		{"zero limit defaults", "0", "0", 20, 0, false},
		{"empty strings default", "", "", 20, 0, false},
		{"negative limit defaults", "-1", "0", 20, 0, false},
		{"over max limit clamps", "150", "0", 100, 0, false},
		{"negative offset clamped", "10", "-5", 10, 0, false},
		{"non-numeric strings", "abc", "xyz", 20, 0, false},
		{"max valid limit", "100", "50", 100, 50, false},
		{"offset at max boundary kept", "20", "10000", 20, 10000, false},
		{"offset beyond max rejected", "20", "10001", 0, 0, true},
		{"offset far beyond max rejected", "20", "999999", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset, err := ParsePaginationParams(tt.limitStr, tt.offsetStr)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expectLim, limit)
			assert.Equal(t, tt.expectOffs, offset)
		})
	}
}

func TestKeysetCursorRoundTrip(t *testing.T) {
	parsed, err := ParseKeysetCursor("2026-01-02T03:04:05.123456Z", "42")
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, 42, parsed.ID)

	// String() must round-trip through ParseKeysetCursor with no precision
	// loss, otherwise the seek predicate drops or repeats boundary rows.
	parts := strings.SplitN(parsed.String(), "|", 2)
	require.Len(t, parts, 2)
	assert.Equal(t, "42", parts[1])
	roundTripped, err := ParseKeysetCursor(parts[0], parts[1])
	require.NoError(t, err)
	require.NotNil(t, roundTripped)
	assert.True(t, parsed.CreatedAt.Equal(roundTripped.CreatedAt),
		"timestamp changed: %s vs %s", parsed.CreatedAt, roundTripped.CreatedAt)
	assert.Equal(t, parsed.ID, roundTripped.ID)
}

func TestParseKeysetCursorErrors(t *testing.T) {
	tests := []struct {
		name      string
		createdAt string
		id        string
	}{
		{"id without timestamp", "", "42"},
		{"timestamp without id", "2026-01-02T03:04:05Z", ""},
		{"malformed timestamp", "not-a-timestamp", "42"},
		{"non-numeric id", "2026-01-02T03:04:05Z", "abc"},
		{"negative id", "2026-01-02T03:04:05Z", "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, err := ParseKeysetCursor(tt.createdAt, tt.id)
			assert.Error(t, err)
			assert.Nil(t, cursor)
		})
	}
}

func TestParseKeysetCursorOffsetMode(t *testing.T) {
	cursor, err := ParseKeysetCursor("", "")
	assert.NoError(t, err)
	assert.Nil(t, cursor)
}

func TestParseIntParam(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"empty string", "", 0, false},
		{"positive", "42", 42, false},
		{"zero", "0", 0, false},
		{"negative", "-3", -3, false},
		{"whitespace", "  7 ", 0, true},
		{"non-numeric", "abc", 0, true},
		{"partial numeric", "12abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIntParam(tt.input)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	data := "test"

	tests := []struct {
		name          string
		total         int
		limit         int
		offset        int
		expectedPages int
	}{
		{"evenly divisible", 100, 10, 0, 10},
		{"zero total", 0, 10, 0, 0},
		{"exact fit", 1, 10, 0, 1},
		{"partial page", 11, 10, 0, 2},
		{"zero limit", 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewPaginatedResponse(data, tt.total, tt.limit, tt.offset)
			assert.Equal(t, data, resp.Data)
			assert.Equal(t, tt.total, resp.Total)
			assert.Equal(t, tt.limit, resp.Limit)
			assert.Equal(t, tt.offset, resp.Offset)
			assert.Equal(t, tt.expectedPages, resp.TotalPages)
		})
	}
}

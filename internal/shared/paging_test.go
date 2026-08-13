package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name       string
		limitStr   string
		offsetStr  string
		expectLim  int
		expectOffs int
	}{
		{"valid limit and offset", "10", "0", 10, 0},
		{"zero limit defaults", "0", "0", 20, 0},
		{"empty strings default", "", "", 20, 0},
		{"negative limit defaults", "-1", "0", 20, 0},
		{"over max limit defaults", "150", "0", 20, 0},
		{"negative offset clamped", "10", "-5", 10, 0},
		{"non-numeric strings", "abc", "xyz", 20, 0},
		{"max valid limit", "100", "50", 100, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := ParsePaginationParams(tt.limitStr, tt.offsetStr)
			assert.Equal(t, tt.expectLim, limit)
			assert.Equal(t, tt.expectOffs, offset)
		})
	}
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

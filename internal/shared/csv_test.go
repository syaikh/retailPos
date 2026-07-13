package shared

import (
	"bytes"
	"encoding/csv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeCSVField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"formula injection", "=SUM(A1:A10)", "'=SUM(A1:A10)"},
		{"command injection", "+cmd|'/C calc'!A0", "'+cmd|'/C calc'!A0"},
		{"minus prefix", "-1+1", "'-1+1"},
		{"at prefix", "@SUM(A1)", "'@SUM(A1)"},
		{"tab prefix", "\tcmd", "'\tcmd"},
		{"carriage return prefix", "\r\n", "'\r\n"},
		{"normal text", "normal text", "normal text"},
		{"empty string", "", ""},
		{"equals test", "=test", "'=test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeCSVField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteCSVRow(t *testing.T) {
	t.Run("clean fields round trip", func(t *testing.T) {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)

		record := []string{"hello", "world", "123"}
		err := WriteCSVRow(w, record)
		w.Flush()
		assert.NoError(t, err)

		r := csv.NewReader(&buf)
		rows, err := r.ReadAll()
		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, record, rows[0])
	})

	t.Run("dangerous fields sanitized", func(t *testing.T) {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)

		record := []string{"=SUM(A1)", "+cmd", "-1", "clean"}
		err := WriteCSVRow(w, record)
		w.Flush()
		assert.NoError(t, err)

		r := csv.NewReader(&buf)
		rows, err := r.ReadAll()
		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, []string{"'=SUM(A1)", "'+cmd", "'-1", "clean"}, rows[0])
	})

	t.Run("no write error", func(t *testing.T) {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)

		err := WriteCSVRow(w, []string{"a", "b"})
		w.Flush()
		assert.NoError(t, err)
		assert.True(t, buf.Len() > 0)
	})
}

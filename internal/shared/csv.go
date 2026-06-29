package shared

import (
	"encoding/csv"
	"strings"
)

var csvDangerousPrefixes = []string{"=", "+", "-", "@", "\t", "\r"}

func SanitizeCSVField(s string) string {
	for _, p := range csvDangerousPrefixes {
		if strings.HasPrefix(s, p) {
			return "'" + s
		}
	}
	return s
}

func WriteCSVRow(w *csv.Writer, record []string) error {
	sanitized := make([]string, len(record))
	for i, v := range record {
		sanitized[i] = SanitizeCSVField(v)
	}
	return w.Write(sanitized)
}

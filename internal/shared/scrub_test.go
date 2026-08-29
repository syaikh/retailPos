package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrubPII(t *testing.T) {
	m := map[string]interface{}{
		"customer_name": "Budi",
		"customer_id":   float64(7),
		"total":         float64(1000),
		"items": []interface{}{
			map[string]interface{}{"name": "x", "phone": "0812"},
		},
		"nested": map[string]interface{}{"email": "a@b.com", "ok": 1},
	}

	ScrubPII(m)

	assert.NotContains(t, m, "customer_name", "customer PII must be removed")
	assert.Contains(t, m, "customer_id", "non-PII id should be kept")
	assert.Contains(t, m, "total")

	items := m["items"].([]interface{})
	item := items[0].(map[string]interface{})
	assert.NotContains(t, item, "phone", "nested PII must be removed")

	nested := m["nested"].(map[string]interface{})
	assert.NotContains(t, nested, "email", "nested PII must be removed")
	assert.Contains(t, nested, "ok")
}

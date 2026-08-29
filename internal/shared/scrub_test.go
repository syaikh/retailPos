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

func TestDiffChanges(t *testing.T) {
	oldMap := map[string]interface{}{
		"name":  "Widget",
		"price": float64(1000),
		"sku":   "A1",
	}
	newMap := map[string]interface{}{
		"name":  "Widget",
		"price": float64(2000),
		"color": "red",
	}

	oldVals, newVals := DiffChanges(oldMap, newMap)

	// Unchanged key omitted from both sides.
	assert.NotContains(t, oldVals, "name")
	assert.NotContains(t, newVals, "name")
	// Changed key present on both sides.
	assert.Equal(t, float64(1000), oldVals["price"])
	assert.Equal(t, float64(2000), newVals["price"])
	// Added key: present in new, absent in old.
	assert.NotContains(t, oldVals, "color")
	assert.Equal(t, "red", newVals["color"])

	// Removed key: present in old, nil in new.
	removedOld, removedNew := DiffChanges(
		map[string]interface{}{"a": "x"},
		map[string]interface{}{"b": "y"},
	)
	assert.Equal(t, "x", removedOld["a"])
	assert.Nil(t, removedNew["a"])
	assert.Equal(t, "y", removedNew["b"])
	assert.NotContains(t, removedOld, "b")
}

package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"retail-pos-system/internal/domain"
)

func TestGenerateAuditDescription(t *testing.T) {
	makeLog := func(username, action, entityType string, entityID *int, newVals, oldVals interface{}) domain.AuditLog {
		return domain.AuditLog{
			Username:   username,
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			NewValues:  newVals,
			OldValues:  oldVals,
		}
	}
	makeID := func(id int) *int {
		return &id
	}
	productName := func(n string) map[string]interface{} {
		return map[string]interface{}{"name": n}
	}

	tests := []struct {
		name     string
		log      domain.AuditLog
		expected string
	}{
		{
			name:     "auth login with username should include username",
			log:      makeLog("Superadmin", "login", "auth", makeID(1), nil, nil),
			expected: "Logged in Superadmin",
		},
		{
			name:     "auth logout with username should include username",
			log:      makeLog("Cashier", "logout", "auth", makeID(3), nil, nil),
			expected: "Logged out Cashier",
		},
		{
			name:     "auth login without username falls back to action only",
			log:      makeLog("", "login", "auth", makeID(1), nil, nil),
			expected: "Logged in",
		},
		{
			name:     "create product with identifier",
			log:      makeLog("Admin", "create", "product", makeID(10), productName("Kopi"), nil),
			expected: "Created product: Kopi",
		},
		{
			name:     "update sale with invoice identifier",
			log:      makeLog("Manager", "update", "sale", makeID(5), map[string]interface{}{"invoice_number": "INV-001"}, nil),
			expected: "Updated sale: INV-001",
		},
		{
			name:     "delete user with entity id fallback",
			log:      makeLog("Superadmin", "delete", "user", makeID(7), nil, nil),
			expected: "Deleted user #7",
		},
		{
			name:     "create role with identifier",
			log:      makeLog("Admin", "create", "role", makeID(2), map[string]interface{}{"name": "Manager"}, nil),
			expected: "Created role: Manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{}
			result := h.generateAuditDescription(&tt.log)
			assert.Equal(t, tt.expected, result)
		})
	}
}

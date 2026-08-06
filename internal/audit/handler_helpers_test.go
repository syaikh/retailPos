package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAuditDescription(t *testing.T) {
	eid := 5
	tests := []struct {
		name string
		log  *Log
		want string
	}{
		{
			name: "create product",
			log: &Log{
				Action:     "CREATE",
				EntityType: "product",
				NewValues:  map[string]interface{}{"name": "Widget"},
			},
			want: "Created product: Widget",
		},
		{
			name: "update user",
			log: &Log{
				Action:     "UPDATE",
				EntityType: "user",
				NewValues:  map[string]interface{}{"username": "john"},
			},
			want: "Updated user: john",
		},
		{
			name: "delete category",
			log: &Log{
				Action:     "DELETE",
				EntityType: "category",
				NewValues:  map[string]interface{}{"name": "Electronics"},
			},
			want: "Deleted category: Electronics",
		},
		{
			name: "login with username",
			log: &Log{
				Action:     "LOGIN",
				EntityType: "auth",
				Username:   "admin",
			},
			want: "Logged in admin",
		},
		{
			name: "logout with username",
			log: &Log{
				Action:     "LOGOUT",
				EntityType: "auth",
				Username:   "admin",
			},
			want: "Logged out admin",
		},
		{
			name: "create with entity id no identifier",
			log: &Log{
				Action:     "CREATE",
				EntityType: "entity",
				EntityID:   &eid,
			},
			want: "Created entity #5",
		},
		{
			name: "custom action",
			log: &Log{
				Action:     "CUSTOM",
				EntityType: "entity",
				NewValues:  map[string]interface{}{"name": "something"},
			},
			want: "Custom entity: something",
		},
		{
			name: "empty action",
			log: &Log{
				Action:     "",
				EntityType: "product",
			},
			want: "",
		},
		{
			name: "invoice number identifier",
			log: &Log{
				Action:     "CREATE",
				EntityType: "sale",
				NewValues:  map[string]interface{}{"invoice_number": "INV-001"},
			},
			want: "Created sale: INV-001",
		},
		{
			name: "login without username",
			log: &Log{
				Action:     "LOGIN",
				EntityType: "auth",
			},
			want: "Logged in",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateAuditDescription(tt.log)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseDateParam(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "empty string",
			s:    "",
			want: "",
		},
		{
			name: "valid YYYY-MM-DD",
			s:    "2026-01-15",
			want: "2026-01-15",
		},
		{
			name: "valid RFC3339",
			s:    "2026-01-15T10:30:00Z",
			want: "2026-01-15T10:30:00Z",
		},
		{
			name: "invalid format",
			s:    "not-a-date",
			want: "",
		},
		{
			name: "wrong separator",
			s:    "2026/01/15",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDateParam(tt.s)
			assert.Equal(t, tt.want, got)
		})
	}
}

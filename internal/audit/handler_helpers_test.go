package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
				Username:   "manager",
			},
			want: "Logged in manager",
		},
		{
			name: "logout with username",
			log: &Log{
				Action:     "LOGOUT",
				EntityType: "auth",
				Username:   "manager",
			},
			want: "Logged out manager",
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

func TestQueryDateParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		query    string
		want     string
		wantOK   bool
		wantCode int
	}{
		{
			name:     "parameter absent",
			query:    "",
			want:     "",
			wantOK:   true,
			wantCode: http.StatusOK,
		},
		{
			name:     "valid YYYY-MM-DD",
			query:    "?start_date=2026-01-15",
			want:     "2026-01-15",
			wantOK:   true,
			wantCode: http.StatusOK,
		},
		{
			name:     "valid RFC3339",
			query:    "?start_date=2026-01-15T10:30:00Z",
			want:     "2026-01-15T10:30:00Z",
			wantOK:   true,
			wantCode: http.StatusOK,
		},
		{
			name:     "invalid format",
			query:    "?start_date=not-a-date",
			want:     "",
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "wrong separator",
			query:    "?start_date=2026/01/15",
			want:     "",
			wantOK:   false,
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/audit-logs"+tt.query, nil)

			got, ok := queryDateParam(c, "start_date")
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

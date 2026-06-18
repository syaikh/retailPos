package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func ptr(s string) *string {
	return &s
}

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "john_doe", "johndoe"},
		{"uppercase", "JohnDoe", "johndoe"},
		{"leading/trailing spaces", "  john  ", "john"},
		{"numbers allowed", "user123", "user123"},
		{"special chars removed", "hello@world!", "helloworld"},
		{"dash removed", "john-doe", "johndoe"},
		{"empty string", "", ""},
		{"only special chars", "@#$%", ""},
		{"mixed", "  User_123!  ", "user123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeUsername(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestParseIntPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *int
	}{
		{"positive", "123", intPtr(123)},
		{"zero", "0", intPtr(0)},
		{"negative", "-1", intPtr(-1)},
		{"empty string", "", nil},
		{"invalid", "abc", nil},
		{"mixed invalid", "123abc", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseIntPtr(tc.input)
			if tc.expected == nil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got)
				assert.Equal(t, *tc.expected, *got)
			}
		})
	}
}

func intPtr(v int) *int {
	return &v
}

func TestValidateCustomerRequest(t *testing.T) {
	validName := "John Doe"
	validEmail := ptr("john@example.com")
	validPhone := ptr("08123456789")

	t.Run("valid request", func(t *testing.T) {
		err := validateCustomerRequest(validName, validEmail, validPhone)
		assert.NoError(t, err)
	})

	t.Run("empty name", func(t *testing.T) {
		err := validateCustomerRequest("", validEmail, validPhone)
		assert.EqualError(t, err, "name is required")
	})

	t.Run("whitespace name", func(t *testing.T) {
		err := validateCustomerRequest("  ", validEmail, validPhone)
		assert.EqualError(t, err, "name is required")
	})

	t.Run("name too long", func(t *testing.T) {
		longName := ""
		for i := 0; i < 201; i++ {
			longName += "a"
		}
		err := validateCustomerRequest(longName, validEmail, validPhone)
		assert.EqualError(t, err, "name must be at most 200 characters")
	})

	t.Run("nil phone", func(t *testing.T) {
		err := validateCustomerRequest(validName, validEmail, nil)
		assert.EqualError(t, err, "phone is required")
	})

	t.Run("empty phone", func(t *testing.T) {
		err := validateCustomerRequest(validName, validEmail, ptr(""))
		assert.EqualError(t, err, "phone is required")
	})

	t.Run("whitespace phone", func(t *testing.T) {
		err := validateCustomerRequest(validName, validEmail, ptr("  "))
		assert.EqualError(t, err, "phone is required")
	})

	t.Run("invalid phone format", func(t *testing.T) {
		err := validateCustomerRequest(validName, validEmail, ptr("abc"))
		assert.EqualError(t, err, "invalid phone format")
	})

	t.Run("nil email", func(t *testing.T) {
		err := validateCustomerRequest(validName, nil, validPhone)
		assert.EqualError(t, err, "email is required")
	})

	t.Run("empty email", func(t *testing.T) {
		err := validateCustomerRequest(validName, ptr(""), validPhone)
		assert.EqualError(t, err, "email is required")
	})

	t.Run("invalid email format", func(t *testing.T) {
		err := validateCustomerRequest(validName, ptr("not-an-email"), validPhone)
		assert.EqualError(t, err, "invalid email format")
	})

	t.Run("email missing domain", func(t *testing.T) {
		err := validateCustomerRequest(validName, ptr("user@"), validPhone)
		assert.EqualError(t, err, "invalid email format")
	})
}

func TestNormalizeProductBarcode(t *testing.T) {
	t.Run("nil barcode stays nil", func(t *testing.T) {
		p := &domain.Product{Barcode: nil}
		(&Handler{}).normalizeProductBarcode(p)
		assert.Nil(t, p.Barcode)
	})

	t.Run("trimmed barcode kept", func(t *testing.T) {
		barcode := "  8991234567890  "
		p := &domain.Product{Barcode: &barcode}
		(&Handler{}).normalizeProductBarcode(p)
		assert.NotNil(t, p.Barcode)
		assert.Equal(t, "8991234567890", *p.Barcode)
	})

	t.Run("empty barcode becomes nil", func(t *testing.T) {
		barcode := ""
		p := &domain.Product{Barcode: &barcode}
		(&Handler{}).normalizeProductBarcode(p)
		assert.Nil(t, p.Barcode)
	})

	t.Run("whitespace barcode becomes nil", func(t *testing.T) {
		barcode := "   "
		p := &domain.Product{Barcode: &barcode}
		(&Handler{}).normalizeProductBarcode(p)
		assert.Nil(t, p.Barcode)
	})
}

func TestValidateProductPayload(t *testing.T) {
	h := &Handler{}

	t.Run("valid product", func(t *testing.T) {
		name := "Test Product"
		sku := "TST-001"
		catName := "Test Category"
		err := h.validateProductPayload(&domain.Product{
			Name:         name,
			SKU:          sku,
			CategoryName: &catName,
			Price:        10000,
			Stock:        10,
		})
		assert.NoError(t, err)
	})

	t.Run("valid product with category id", func(t *testing.T) {
		catID := 1
		err := h.validateProductPayload(&domain.Product{
			Name:       "Test",
			SKU:        "TST-002",
			CategoryID: &catID,
			Price:      5000,
			Stock:      5,
		})
		assert.NoError(t, err)
	})

	t.Run("empty name", func(t *testing.T) {
		err := h.validateProductPayload(&domain.Product{
			Name: "",
			SKU:  "TST-003",
		})
		assert.EqualError(t, err, "name is required")
	})

	t.Run("whitespace name", func(t *testing.T) {
		err := h.validateProductPayload(&domain.Product{
			Name: "   ",
			SKU:  "TST-004",
		})
		assert.EqualError(t, err, "name is required")
	})

	t.Run("empty sku", func(t *testing.T) {
		err := h.validateProductPayload(&domain.Product{
			Name: "Test",
			SKU:  "",
		})
		assert.EqualError(t, err, "sku is required")
	})

	t.Run("no category", func(t *testing.T) {
		err := h.validateProductPayload(&domain.Product{
			Name:  "Test",
			SKU:   "TST-005",
			Price: 1000,
			Stock: 1,
		})
		assert.EqualError(t, err, "category is required")
	})

	t.Run("price zero", func(t *testing.T) {
		catName := "Test"
		err := h.validateProductPayload(&domain.Product{
			Name:         "Test",
			SKU:          "TST-006",
			CategoryName: &catName,
			Price:        0,
			Stock:        1,
		})
		assert.EqualError(t, err, "price must be greater than 0")
	})

	t.Run("price negative", func(t *testing.T) {
		catName := "Test"
		err := h.validateProductPayload(&domain.Product{
			Name:         "Test",
			SKU:          "TST-007",
			CategoryName: &catName,
			Price:        -1,
			Stock:        1,
		})
		assert.EqualError(t, err, "price must be greater than 0")
	})

	t.Run("stock negative", func(t *testing.T) {
		catName := "Test"
		err := h.validateProductPayload(&domain.Product{
			Name:         "Test",
			SKU:          "TST-008",
			CategoryName: &catName,
			Price:        1000,
			Stock:        -1,
		})
		assert.EqualError(t, err, "stock must not be negative")
	})
}

func TestUserRole(t *testing.T) {
	h := &Handler{}

	t.Run("role exists and is string", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "superadmin")
		assert.Equal(t, "superadmin", h.userRole(c))
	})

	t.Run("role is lowercase trimmed", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "  ADMIN  ")
		assert.Equal(t, "admin", h.userRole(c))
	})

	t.Run("role does not exist", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		assert.Equal(t, "", h.userRole(c))
	})

	t.Run("role is not a string", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", 42)
		assert.Equal(t, "", h.userRole(c))
	})
}

func TestHasPermission(t *testing.T) {
	h := &Handler{}

	t.Run("permission exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("permissions", []string{"read:products", "write:products"})
		assert.True(t, h.hasPermission(c, "read:products"))
	})

	t.Run("permission does not exist", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("permissions", []string{"read:products"})
		assert.False(t, h.hasPermission(c, "delete:products"))
	})

	t.Run("no permissions key", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		assert.False(t, h.hasPermission(c, "anything"))
	})

	t.Run("permissions is wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("permissions", "not-a-slice")
		assert.False(t, h.hasPermission(c, "anything"))
	})
}

func TestCanManageProduct(t *testing.T) {
	h := &Handler{}

	t.Run("superadmin bypasses permission check", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "superadmin")
		assert.True(t, h.canManageProduct(c, "any:perm"))
	})

	t.Run("admin bypasses permission check", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "admin")
		assert.True(t, h.canManageProduct(c, "any:perm"))
	})

	t.Run("staff bypasses permission check", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "staff")
		assert.True(t, h.canManageProduct(c, "any:perm"))
	})

	t.Run("cashier with permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "cashier")
		c.Set("permissions", []string{"manage:products"})
		assert.True(t, h.canManageProduct(c, "manage:products"))
	})

	t.Run("cashier without permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "cashier")
		c.Set("permissions", []string{"read:products"})
		assert.False(t, h.canManageProduct(c, "manage:products"))
	})
}

func TestCanManageCategory(t *testing.T) {
	h := &Handler{}

	t.Run("superadmin bypasses permission check", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "superadmin")
		assert.True(t, h.canManageCategory(c, "any:perm"))
	})

	t.Run("admin bypasses permission check", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "admin")
		assert.True(t, h.canManageCategory(c, "any:perm"))
	})

	t.Run("staff does NOT bypass for category", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "staff")
		c.Set("permissions", []string{"manage:categories"})
		assert.True(t, h.canManageCategory(c, "manage:categories"))
	})

	t.Run("staff without permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "staff")
		c.Set("permissions", []string{"read:categories"})
		assert.False(t, h.canManageCategory(c, "manage:categories"))
	})

	t.Run("cashier with permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "cashier")
		c.Set("permissions", []string{"manage:categories"})
		assert.True(t, h.canManageCategory(c, "manage:categories"))
	})

	t.Run("cashier without permission", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("role", "cashier")
		c.Set("permissions", []string{"read:categories"})
		assert.False(t, h.canManageCategory(c, "manage:categories"))
	})
}

func TestExportFilenameTimezone(t *testing.T) {
	// Verify that time.Now().In(Asia/Jakarta) produces a valid YYYY-MM-DD filename
	now := time.Now().In(config.Load().Timezone)
	filename := "audit-logs-" + now.Format("2006-01-02")
	assert.Regexp(t, `^audit-logs-\d{4}-\d{2}-\d{2}$`, filename)

	// Verify Jakarta timezone is used (offset +7h)
	_, offset := now.Zone()
	assert.Equal(t, 7*3600, offset, "should use +07:00 offset")
}

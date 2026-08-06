package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
)

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if dbPool == nil {
		t.Skip("no database connection")
	}
}

func testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", 1)
		c.Set("username", "testuser")
		c.Set("roleID", 1)
		c.Set("role", "superadmin")
		c.Set("permissions", []string{"inventory.adjust"})
		c.Set("storeID", nil)
		c.Next()
	}
}

func testPermMiddleware(perm permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func setupInventoryRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	repo := NewRepository(dbPool)
	bus := eventbus.New()
	go bus.Run()

	svc := NewService(repo, bus)
	h := NewHandler(svc, nil)

	r := gin.New()
	h.RegisterRoutes(r.Group("/"), testAuthMiddleware(), testPermMiddleware)
	return r
}

func TestHandler_AdjustStock(t *testing.T) {
	skipIfNoDB(t)
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	insertTestUser(ctx, t, 1)
	r := setupInventoryRouter()

	t.Run("success increase", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "HDL-ADJ-INC-001")
		insertTestStock(ctx, t, productID, 10)

		body := `{"product_id":` + strconv.Itoa(productID) + `,"quantity_change":5,"notes":"restock"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp["status"])
	})

	t.Run("success decrease", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "HDL-ADJ-DEC-001")
		insertTestStock(ctx, t, productID, 20)

		body := `{"product_id":` + strconv.Itoa(productID) + `,"quantity_change":-5,"notes":"sale"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp["status"])
	})

	t.Run("invalid json", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/inventory/adjust", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("zero quantity change", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "HDL-ADJ-ZERO-001")
		insertTestStock(ctx, t, productID, 10)

		body := `{"product_id":` + strconv.Itoa(productID) + `,"quantity_change":0,"notes":"no change"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing notes", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "HDL-ADJ-NOTES-001")
		insertTestStock(ctx, t, productID, 10)

		body := `{"product_id":` + strconv.Itoa(productID) + `,"quantity_change":5,"notes":""}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("insufficient stock", func(t *testing.T) {
		productID := insertTestProduct(ctx, t, "HDL-ADJ-INSF-001")
		insertTestStock(ctx, t, productID, 3)

		body := `{"product_id":` + strconv.Itoa(productID) + `,"quantity_change":-10,"notes":"overdraft"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/inventory/adjust", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

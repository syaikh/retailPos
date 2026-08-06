package supplier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockAuditCreator struct {
	createAuditLogFn func(ctx context.Context, log *audit.Log) error
	lastLog          *audit.Log
}

func (m *mockAuditCreator) CreateAuditLog(ctx context.Context, log *audit.Log) error {
	m.lastLog = log
	if m.createAuditLogFn != nil {
		return m.createAuditLogFn(ctx, log)
	}
	return nil
}

type mockSupplierServiceForAudit struct {
	getByIDFn                 func(ctx context.Context, id int) (*Supplier, error)
	getAllFn                  func(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error)
	createFn                  func(ctx context.Context, supplier *Supplier) error
	updateFn                  func(ctx context.Context, supplier *Supplier) error
	deleteFn                  func(ctx context.Context, id int) error
	linkProductFn             func(ctx context.Context, ps *ProductSupplier) error
	unlinkProductFn           func(ctx context.Context, productID, supplierID int) error
	getProductSupplierFn      func(ctx context.Context, productID, supplierID int) (*ProductSupplier, error)
	setPreferredSupplierFn    func(ctx context.Context, productID, supplierID int) error
	updateProductSupplierFn   func(ctx context.Context, ps *ProductSupplier) error
	bulkUpdateFn              func(ctx context.Context, ids []int, isActive bool) (int, error)
	bulkDeleteFn              func(ctx context.Context, ids []int) (int, error)
	getProductsBySupplierIDFn func(ctx context.Context, supplierID int) ([]ProductSupplier, error)
}

func (m *mockSupplierServiceForAudit) GetByID(ctx context.Context, id int) (*Supplier, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockSupplierServiceForAudit) GetByCode(ctx context.Context, code string) (*Supplier, error) {
	return nil, nil
}

func (m *mockSupplierServiceForAudit) GetAll(ctx context.Context, limit, offset int, search string, isActive *bool) ([]Supplier, int, error) {
	if m.getAllFn != nil {
		return m.getAllFn(ctx, limit, offset, search, isActive)
	}
	return []Supplier{}, 0, nil
}

func (m *mockSupplierServiceForAudit) Create(ctx context.Context, supplier *Supplier) error {
	if m.createFn != nil {
		return m.createFn(ctx, supplier)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) Update(ctx context.Context, supplier *Supplier) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, supplier)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) Delete(ctx context.Context, id int) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) LinkProduct(ctx context.Context, ps *ProductSupplier) error {
	if m.linkProductFn != nil {
		return m.linkProductFn(ctx, ps)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) UnlinkProduct(ctx context.Context, productID, supplierID int) error {
	if m.unlinkProductFn != nil {
		return m.unlinkProductFn(ctx, productID, supplierID)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) GetProductSupplier(ctx context.Context, productID, supplierID int) (*ProductSupplier, error) {
	if m.getProductSupplierFn != nil {
		return m.getProductSupplierFn(ctx, productID, supplierID)
	}
	return nil, nil
}

func (m *mockSupplierServiceForAudit) GetPreferredSupplier(ctx context.Context, productID int) (*ProductSupplier, error) {
	return nil, nil
}

func (m *mockSupplierServiceForAudit) SetPreferredSupplier(ctx context.Context, productID, supplierID int) error {
	if m.setPreferredSupplierFn != nil {
		return m.setPreferredSupplierFn(ctx, productID, supplierID)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) UpdateProductSupplier(ctx context.Context, ps *ProductSupplier) error {
	if m.updateProductSupplierFn != nil {
		return m.updateProductSupplierFn(ctx, ps)
	}
	return nil
}

func (m *mockSupplierServiceForAudit) GetSuppliersByProductID(ctx context.Context, productID int) ([]ProductSupplier, error) {
	return nil, nil
}

func (m *mockSupplierServiceForAudit) GetProductsBySupplierID(ctx context.Context, supplierID int) ([]ProductSupplier, error) {
	if m.getProductsBySupplierIDFn != nil {
		return m.getProductsBySupplierIDFn(ctx, supplierID)
	}
	return nil, nil
}

func (m *mockSupplierServiceForAudit) BulkUpdate(ctx context.Context, ids []int, isActive bool) (int, error) {
	if m.bulkUpdateFn != nil {
		return m.bulkUpdateFn(ctx, ids, isActive)
	}
	return 0, nil
}

func (m *mockSupplierServiceForAudit) BulkDelete(ctx context.Context, ids []int) (int, error) {
	if m.bulkDeleteFn != nil {
		return m.bulkDeleteFn(ctx, ids)
	}
	return 0, nil
}

func requireAuditLog(t *testing.T, auditSvc *mockAuditCreator) *audit.Log {
	t.Helper()
	assert.NotNil(t, auditSvc.lastLog, "expected audit log to be created")
	return auditSvc.lastLog
}

func TestAuditHandler_CreateSupplier(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		createFn: func(ctx context.Context, supplier *Supplier) error {
			supplier.ID = 100
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"name":"Audit Supplier","code":"AUDIT-1","contact_name":"Jane","is_active":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/suppliers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp struct {
		Data Supplier `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Audit Supplier", resp.Data.Name)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "create", log.Action)
	assert.Equal(t, "supplier", log.EntityType)
	assert.Equal(t, "testuser", log.Username)
	assert.Equal(t, "superadmin", log.Role)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Created supplier Audit Supplier", log.Description)
}

func TestAuditHandler_UpdateSupplier(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		getByIDFn: func(ctx context.Context, id int) (*Supplier, error) {
			return &Supplier{ID: id, Name: "Old Name", Code: "OLD-1"}, nil
		},
		updateFn: func(ctx context.Context, supplier *Supplier) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"name":"Updated Supplier"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/suppliers/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "update", log.Action)
	assert.Equal(t, "supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Contains(t, log.Description, "Updated Supplier")
	assert.Equal(t, "Updated supplier Updated Supplier", log.Description)
}

func TestAuditHandler_DeleteSupplier(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		getByIDFn: func(ctx context.Context, id int) (*Supplier, error) {
			return &Supplier{ID: id, Name: "ToDelete"}, nil
		},
		deleteFn: func(ctx context.Context, id int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/suppliers/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "delete", log.Action)
	assert.Equal(t, "supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Deleted supplier ToDelete", log.Description)
}

func TestAuditHandler_LinkProduct(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		linkProductFn: func(ctx context.Context, ps *ProductSupplier) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"product_id":1,"unit_cost":5000}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/suppliers/1/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "create", log.Action)
	assert.Equal(t, "product_supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Linked product #1 to supplier #1", log.Description)
}

func TestAuditHandler_UnlinkProduct(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		unlinkProductFn: func(ctx context.Context, productID, supplierID int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/suppliers/1/products/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "delete", log.Action)
	assert.Equal(t, "product_supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Unlinked product #1 from supplier #1", log.Description)
}

func TestAuditHandler_UpdateProductSupplier(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		getProductSupplierFn: func(ctx context.Context, productID, supplierID int) (*ProductSupplier, error) {
			return &ProductSupplier{ProductID: productID, SupplierID: supplierID, UnitCost: 5000}, nil
		},
		updateProductSupplierFn: func(ctx context.Context, ps *ProductSupplier) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"unit_cost":7000}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/suppliers/1/products/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "update", log.Action)
	assert.Equal(t, "product_supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Updated product-supplier link for product #1 supplier #1", log.Description)
}

func TestAuditHandler_SetPreferredSupplier(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		setPreferredSupplierFn: func(ctx context.Context, productID, supplierID int) error {
			return nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/suppliers/1/products/1/preferred", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "update", log.Action)
	assert.Equal(t, "product_supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Set supplier #1 as preferred for product #1", log.Description)
}

func TestAuditHandler_BulkUpdate(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		bulkUpdateFn: func(ctx context.Context, ids []int, isActive bool) (int, error) {
			return len(ids), nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"ids":[1,2,3],"is_active":false}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/suppliers/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "bulk_update", log.Action)
	assert.Equal(t, "supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Bulk updated 3 suppliers", log.Description)
}

func TestAuditHandler_BulkDelete(t *testing.T) {
	svc := &mockSupplierServiceForAudit{
		bulkDeleteFn: func(ctx context.Context, ids []int) (int, error) {
			return len(ids), nil
		},
	}
	auditSvc := &mockAuditCreator{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, middleware.CtxKeyUserID, 1)
		ctx = context.WithValue(ctx, middleware.CtxKeyUsername, "testuser")
		ctx = context.WithValue(ctx, middleware.CtxKeyRole, "superadmin")
		ctx = context.WithValue(ctx, middleware.CtxKeyStoreID, (*int)(nil))
		ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, "127.0.0.1")
		ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, "test-agent")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := NewHandler(svc, auditSvc)
	h.RegisterRoutes(r.Group("/"), func(c *gin.Context) { c.Next() }, func(perm permissions.Code) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	body := `{"ids":[1,2,3]}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/suppliers/bulk", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	log := requireAuditLog(t, auditSvc)
	assert.Equal(t, "bulk_delete", log.Action)
	assert.Equal(t, "supplier", log.EntityType)
	assert.Equal(t, 1, *log.UserID)
	assert.Equal(t, "Bulk deleted 3 suppliers", log.Description)
}

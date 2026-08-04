package customer

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type CustomerService interface {
	GetAllCustomers(ctx context.Context, limit, offset int, search string, isActive *bool, storeID *int, customerGroupID *int) ([]Customer, int, error)
	GetCustomerByID(ctx context.Context, id int, storeID *int) (*Customer, error)
	CreateCustomer(ctx context.Context, customer *Customer, storeID *int) error
	UpdateCustomer(ctx context.Context, customer *Customer, id int, storeID *int) error
	DeleteCustomer(ctx context.Context, id int, storeID *int) error
	BulkUpdateCustomersStatus(ctx context.Context, ids []int, isActive bool, storeID *int) error
	BulkDeleteCustomers(ctx context.Context, ids []int, storeID *int) error
}

type Handler struct {
	svc      CustomerService
	auditSvc audit.AuditCreator
}

func NewHandler(svc CustomerService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.GET("/customers", auth, perm(permissions.CustomerView), h.GetCustomers)
	r.GET("/customers/:id", auth, perm(permissions.CustomerView), h.GetCustomerByID)
	r.POST("/customers", auth, perm(permissions.CustomerCreate), h.CreateCustomer)
	r.PUT("/customers/:id", auth, perm(permissions.CustomerUpdate), h.UpdateCustomer)
	r.DELETE("/customers/:id", auth, perm(permissions.CustomerDelete), h.DeleteCustomer)
	r.POST("/customers/bulk/status", auth, perm(permissions.CustomerUpdate), h.BulkUpdateCustomerStatus)
	r.POST("/customers/bulk/delete", auth, perm(permissions.CustomerDelete), h.BulkDeleteCustomers)
}

var phoneRegex = regexp.MustCompile(`^[0-9+\-() ]{7,20}$`)

func validateCustomerName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 200 {
		return fmt.Errorf("name must be at most 200 characters")
	}
	return nil
}

func validateCustomerEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validateCustomerPhone(phone string) error {
	if strings.TrimSpace(phone) == "" {
		return fmt.Errorf("phone is required")
	}
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("invalid phone format")
	}
	return nil
}

func (h *Handler) GetCustomers(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := strings.TrimSpace(c.Query("search"))

	var isActive *bool
	v := c.Query("is_active")
	if v == "" {
		v = c.Query("isActive")
	}
	if v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			isActive = &b
		}
	}

	storeIDPtr := shared.GetStoreID(c)

	var customerGroupID *int
	if v := c.Query("customer_group_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			customerGroupID = &id
		}
	}

	customers, total, err := h.svc.GetAllCustomers(c.Request.Context(), limit, offset, search, isActive, storeIDPtr, customerGroupID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if customers == nil {
		customers = []Customer{}
	}
	shared.JSONPaginated(c, customers, total, limit, offset)
}

func (h *Handler) GetCustomerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	customer, err := h.svc.GetCustomerByID(c.Request.Context(), id, storeIDPtr)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	var req struct {
		Name            string  `json:"name"`
		Phone           string  `json:"phone"`
		Email           string  `json:"email"`
		Address         *string `json:"address"`
		Note            *string `json:"note"`
		IsActive        *bool   `json:"is_active"`
		CustomerGroupID *int    `json:"customer_group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateCustomerName(req.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateCustomerPhone(req.Phone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateCustomerEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	customer := &Customer{
		Name:            req.Name,
		Phone:           &req.Phone,
		Email:           &req.Email,
		Address:         req.Address,
		Note:            req.Note,
		IsActive:        isActive,
		CustomerGroupID: req.CustomerGroupID,
	}

	if err := h.svc.CreateCustomer(c.Request.Context(), customer, storeIDPtr); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "create",
			EntityType:  "customer",
			EntityID:    &customer.ID,
			NewValues:   shared.ToJSONMap(customer),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Created customer %s", customer.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": customer})
}

func (h *Handler) UpdateCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Name            *string `json:"name"`
		Phone           *string `json:"phone"`
		Email           *string `json:"email"`
		Address         *string `json:"address"`
		Note            *string `json:"note"`
		IsActive        *bool   `json:"is_active"`
		CustomerGroupID *int    `json:"customer_group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		if err := validateCustomerName(*req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Email != nil {
		if err := validateCustomerEmail(*req.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if req.Phone != nil {
		if err := validateCustomerPhone(*req.Phone); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	storeIDPtr := shared.GetStoreID(c)

	existing, err := h.svc.GetCustomerByID(c.Request.Context(), id, storeIDPtr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if existing.IsWalkIn {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify walk-in customer"})
		return
	}

	var oldCustomer *Customer
	if h.auditSvc != nil {
		oldCustomer = existing
	}

	customer := &Customer{
		ID:              id,
		Phone:           req.Phone,
		Email:           req.Email,
		Address:         req.Address,
		Note:            req.Note,
		CustomerGroupID: req.CustomerGroupID,
	}
	if req.Name != nil {
		customer.Name = *req.Name
	}
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	if err := h.svc.UpdateCustomer(c.Request.Context(), customer, id, storeIDPtr); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "update",
			EntityType:  "customer",
			EntityID:    &id,
			OldValues:   shared.ToJSONMap(oldCustomer),
			NewValues:   shared.ToJSONMap(customer),
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: fmt.Sprintf("Updated customer %s", customer.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *Handler) DeleteCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	existing, err := h.svc.GetCustomerByID(c.Request.Context(), id, storeIDPtr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "customer not found"})
		return
	}
	if existing.IsWalkIn {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify walk-in customer"})
		return
	}

	var oldCustomerName string
	if h.auditSvc != nil {
		oldCustomerName = existing.Name
	}

	if err := h.svc.DeleteCustomer(c.Request.Context(), id, storeIDPtr); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		userID := middleware.UserIDFromContext(c.Request.Context())
		var description string
		if oldCustomerName != "" {
			description = fmt.Sprintf("Deleted customer %s", oldCustomerName)
		} else {
			description = fmt.Sprintf("Deleted customer #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      userID,
			Username:    middleware.UsernameFromContext(c.Request.Context()),
			Role:        middleware.RoleFromContext(c.Request.Context()),
			Action:      "delete",
			EntityType:  "customer",
			EntityID:    &id,
			IPAddress:   middleware.IPAddressFromContext(c.Request.Context()),
			UserAgent:   middleware.UserAgentFromContext(c.Request.Context()),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) BulkUpdateCustomerStatus(c *gin.Context) {
	var req struct {
		IDs      []int `json:"ids"`
		IsActive bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no customer IDs provided"})
		return
	}
	if len(req.IDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many customer IDs (max 200)"})
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	if err := h.svc.BulkUpdateCustomersStatus(c.Request.Context(), req.IDs, req.IsActive, storeIDPtr); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) BulkDeleteCustomers(c *gin.Context) {
	var req struct {
		IDs []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no customer IDs provided"})
		return
	}
	if len(req.IDs) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many customer IDs (max 200)"})
		return
	}

	storeIDPtr := shared.GetStoreID(c)

	if err := h.svc.BulkDeleteCustomers(c.Request.Context(), req.IDs, storeIDPtr); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

package customer

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/customers", auth, perm("customer:read"), h.GetCustomers)
	r.GET("/customers/:id", auth, perm("customer:read"), h.GetCustomerByID)
	r.POST("/customers", auth, perm("customer:create"), h.CreateCustomer)
	r.PUT("/customers/:id", auth, perm("customer:update"), h.UpdateCustomer)
	r.DELETE("/customers/:id", auth, perm("customer:delete"), h.DeleteCustomer)
	r.POST("/customers/bulk/status", auth, perm("customer:update"), h.BulkUpdateCustomerStatus)
	r.POST("/customers/bulk/delete", auth, perm("customer:delete"), h.BulkDeleteCustomers)
}

func validateCustomerEmail(email string) error {
	if email != "" && !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func validateCustomerPhone(phone string) error {
	if phone == "" {
		return fmt.Errorf("phone is required")
	}
	return nil
}

func (h *Handler) GetCustomers(c *gin.Context) {
	limit := 50
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && l > 0 {
		limit = l
	}
	if limit > 200 {
		limit = 200
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	search := strings.TrimSpace(c.Query("search"))

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			isActive = &b
		}
	}

	customers, total, err := h.svc.GetAllCustomers(c.Request.Context(), limit, offset, search, isActive)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if customers == nil {
		customers = []Customer{}
	}
	c.JSON(http.StatusOK, gin.H{"data": customers, "total": total})
}

func (h *Handler) GetCustomerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	customer, err := h.svc.GetCustomerByID(c.Request.Context(), id)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *Handler) CreateCustomer(c *gin.Context) {
	var req struct {
		Name     string  `json:"name"`
		Phone    string  `json:"phone"`
		Email    string  `json:"email"`
		Address  *string `json:"address"`
		Note     *string `json:"note"`
		IsActive bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateCustomerEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateCustomerPhone(req.Phone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := &Customer{
		Name:     req.Name,
		Phone:    &req.Phone,
		Email:    &req.Email,
		Address:  req.Address,
		Note:     req.Note,
		IsActive: req.IsActive,
	}

	if err := h.svc.CreateCustomer(c.Request.Context(), customer); err != nil {
		shared.InternalError(c, err)
		return
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
		Name     string  `json:"name"`
		Phone    *string `json:"phone"`
		Email    *string `json:"email"`
		Address  *string `json:"address"`
		Note     *string `json:"note"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

	customer := &Customer{
		ID:       id,
		Name:     req.Name,
		Phone:    req.Phone,
		Email:    req.Email,
		Address:  req.Address,
		Note:     req.Note,
	}
	if req.IsActive != nil {
		customer.IsActive = *req.IsActive
	}

	if err := h.svc.UpdateCustomer(c.Request.Context(), customer, id); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *Handler) DeleteCustomer(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteCustomer(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
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

	if err := h.svc.BulkUpdateCustomersStatus(c.Request.Context(), req.IDs, req.IsActive); err != nil {
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

	if err := h.svc.BulkDeleteCustomers(c.Request.Context(), req.IDs); err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"retail-pos-system/internal/auth"
	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"
	"retail-pos-system/pkg/websocket"
)

func getCtx(c *gin.Context) context.Context {
	return c.Request.Context()
}

type Handler struct {
	authRepo       repository.UserRepository
	roleRepo       repository.RoleRepository
	productRepo    repository.ProductRepository
	saleRepo       repository.SaleRepository
	authService    *auth.AuthService
	hub            *websocket.Hub
	auditRepo      repository.AuditLogRepository
}

func NewHandler(
	authRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	productRepo repository.ProductRepository,
	saleRepo repository.SaleRepository,
	authService *auth.AuthService,
	hub *websocket.Hub,
	auditRepo repository.AuditLogRepository,
) *Handler {
	return &Handler{
		authRepo:    authRepo,
		roleRepo:    roleRepo,
		productRepo: productRepo,
		saleRepo:    saleRepo,
		authService: authService,
		hub:         hub,
		auditRepo:   auditRepo,
	}
}

// Auth Handlers
func (h *Handler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.authService.Login(getCtx(c), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("refresh_token", resp.RefreshToken, 7*24*3600, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"access_token": resp.AccessToken, "refresh_token": resp.RefreshToken, "user": resp.User})
}

func (h *Handler) Logout(c *gin.Context) {
	userID := getUserID(c)
	token, _ := c.Cookie("refresh_token")
	if token != "" && userID > 0 {
		h.authService.Logout(getCtx(c), userID, token)
	}
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		refreshToken, _ = c.Cookie("refresh_token")
	}
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
		return
	}
	newAccessToken, err := h.authService.RefreshToken(getCtx(c), refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}

func (h *Handler) ValidateSession(c *gin.Context) {
	userID := getUserID(c)
	role, _ := c.Get("role")
	permissions, _ := c.Get("permissions")
	storeID, _ := c.Get("storeID")

	user, err := h.authRepo.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	user.Password = ""

	var perms []string
	if p, ok := permissions.([]string); ok {
		perms = p
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "user": user, "role": role, "permissions": perms, "store_id": storeID})
}

// Product Handlers
func (h *Handler) GetProducts(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val > 0 {
			offset = val
		}
	}
	products, total, err := h.productRepo.GetAllProducts(getCtx(c), limit, offset, c.Query("search"), nil, "created_at", "DESC", nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch products"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": products, "total": total})
}

func (h *Handler) GetProductByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	product, err := h.productRepo.GetProductByID(getCtx(c), id, nil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var product domain.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.productRepo.CreateProduct(getCtx(c), &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}
	h.logAudit(c, "create", "product", product.ID, nil, product)
	// Broadcast product update
	if h.hub != nil {
		websocket.BroadcastProductUpdate(h.hub, &product)
	}
	c.JSON(http.StatusCreated, gin.H{"data": product})
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	old, _ := h.productRepo.GetProductByID(getCtx(c), id, nil)
	var product domain.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	product.ID = id
	if err := h.productRepo.UpdateProduct(getCtx(c), &product, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
		return
	}
	h.logAudit(c, "update", "product", product.ID, old, product)
	// Broadcast product update
	if h.hub != nil {
		websocket.BroadcastProductUpdate(h.hub, &product)
	}
	c.JSON(http.StatusOK, gin.H{"data": product})
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	old, _ := h.productRepo.GetProductByID(getCtx(c), id, nil)
	if err := h.productRepo.DeleteProduct(getCtx(c), id, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}
	h.logAudit(c, "delete", "product", id, old, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Sale Handlers
func (h *Handler) CreateSale(c *gin.Context) {
	var req domain.SaleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	sale := &domain.Sale{
		InvoiceNumber: req.InvoiceNumber,
		CashierID:     req.CashierID,
		StoreID:       req.StoreID,
		Subtotal:      req.Subtotal,
		Discount:      req.Discount,
		Tax:           req.Tax,
		TotalAmount:   req.TotalAmount,
		PaymentMethod: req.PaymentMethod,
		Status:        "completed",
	}
	if err := h.saleRepo.CreateSale(getCtx(c), nil, sale, req.Items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sale: " + err.Error()})
		return
	}
	newSale, _ := h.saleRepo.GetSaleByID(getCtx(c), sale.ID)
	h.logAudit(c, "create", "sale", sale.ID, nil, newSale)
	// Broadcast sale created event
	if h.hub != nil {
		websocket.BroadcastSaleCreated(h.hub, newSale)
		// Also broadcast stock updates for sold items
		for _, item := range newSale.Items {
			if product, err := h.productRepo.GetProductByID(getCtx(c), item.ProductID, newSale.StoreID); err == nil {
				websocket.BroadcastStockUpdate(h.hub, product)
				// Check for low stock
				if product.Stock <= product.StockMin {
					websocket.BroadcastLowStockAlert(h.hub, product)
				}
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": newSale})
}

func (h *Handler) GetSalesHistory(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val > 0 {
			offset = val
		}
	}
	sales, total, err := h.saleRepo.GetAllSales(getCtx(c), limit, offset, c.Query("search"), "created_at", "DESC", c.Query("startDate"), c.Query("endDate"), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sales"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sales, "total": total})
}

func (h *Handler) GetSaleByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	sale, err := h.saleRepo.GetSaleByID(getCtx(c), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sale not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sale})
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	// Temporarily return simple response without any repo calls
	c.JSON(http.StatusOK, gin.H{"data": map[string]interface{}{
		"total_sales": 0, "total_revenue": 0, "total_products": 0,
		"low_stock_count": 0, "todays_sales": 0, "todays_revenue": 0, "active_customers": 0,
	}})
}

func (h *Handler) GetSalesChartData(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
}

// Admin Handlers
func (h *Handler) ListUsers(c *gin.Context) {
	users, total, err := h.authRepo.GetAllUsers(getCtx(c), 50, 0, c.Query("search"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.roleRepo.GetAllRoles(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch roles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func (h *Handler) CreateRole(c *gin.Context) {
	var role domain.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.roleRepo.CreateRole(getCtx(c), &role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create role"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": role})
}

func (h *Handler) UpdateRolePermissions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req struct{ PermissionIDs []int `json:"permission_ids"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.roleRepo.UpdateRolePermissions(getCtx(c), id, req.PermissionIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.roleRepo.DeleteRole(getCtx(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.roleRepo.GetAllPermissions(getCtx(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": perms})
}

// Audit Logs
func (h *Handler) ListAuditLogs(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val > 0 {
			offset = val
		}
	}
	var userID *int
	if uid := c.Query("user_id"); uid != "" {
		if val, err := strconv.Atoi(uid); err == nil {
			userID = &val
		}
	}
	logs, total, err := h.auditRepo.GetAll(getCtx(c), limit, offset, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}

// User CRUD (Admin)
func (h *Handler) CreateUser(c *gin.Context) {
	var input struct {
		Username  string `json:"username" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=6"`
		RoleID    int    `json:"role_id" binding:"required"`
		StoreID   *int   `json:"store_id"`
		IsActive  bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hashedPassword, err := h.authService.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	user := &domain.User{
		Username: input.Username,
		Email:    input.Email,
		Password: hashedPassword,
		RoleID:   input.RoleID,
		StoreID:  input.StoreID,
		IsActive: input.IsActive,
	}
	if err := h.authRepo.CreateUser(getCtx(c), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	// Fetch created user with role
	created, err := h.authRepo.GetByID(user.ID)
	if err == nil {
		created.Password = ""
		h.logAudit(c, "create", "user", created.ID, nil, created)
		c.JSON(http.StatusCreated, gin.H{"data": created})
		return
	}
	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input struct {
		Username   string  `json:"username"`
		Email      string  `json:"email"`
		Password   string  `json:"password"` // optional
		RoleID     int     `json:"role_id"`
		StoreID    *int    `json:"store_id"`
		IsActive   bool    `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Fetch existing
	existing, err := h.authRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// Update fields
	existing.Username = input.Username
	existing.Email = input.Email
	existing.RoleID = input.RoleID
	existing.StoreID = input.StoreID
	existing.IsActive = input.IsActive
	if input.Password != "" {
		hashed, err := h.authService.HashPassword(input.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		existing.Password = hashed
	}
	// Update in repository
	if err := h.authRepo.UpdateUser(getCtx(c), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	existing.Password = ""
	h.logAudit(c, "update", "user", existing.ID, nil, existing)
	c.JSON(http.StatusOK, gin.H{"data": existing})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// Check existence before delete
	_, err := h.authRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := h.authRepo.DeleteUser(getCtx(c), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}
	h.logAudit(c, "delete", "user", id, nil, nil)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ExportInventory(c *gin.Context) {
	products, _, _ := h.productRepo.GetAllProducts(getCtx(c), 10000, 0, "", nil, "name", "ASC", nil, nil)
	c.JSON(http.StatusOK, gin.H{"data": products})
}

func (h *Handler) logAudit(c *gin.Context, action, entityType string, entityID int, oldValues, newValues interface{}) {
	userIDVal, _ := c.Get("userID")
	usernameVal, _ := c.Get("username")
	roleVal, _ := c.Get("role")
	ip := c.ClientIP()

	var userID int
	var username string
	var role string

	switch v := userIDVal.(type) {
	case int:
		userID = v
	case float64: // sometimes numbers come as float64
		userID = int(v)
	}
	if s, ok := usernameVal.(string); ok {
		username = s
	}
	if s, ok := roleVal.(string); ok {
		role = s
	}

	log := &domain.AuditLog{
		UserID:     &userID,
		Username:   username,
		Role:       role,
		Action:     action,
		EntityType: entityType,
		EntityID:   &entityID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  ip,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	// Fire and forget
	go h.auditRepo.Create(context.Background(), log)
}

func getUserID(c *gin.Context) int {
	if uid, exists := c.Get("userID"); exists {
		return uid.(int)
	}
	return 0
}

func getRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}

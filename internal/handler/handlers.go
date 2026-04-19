package handler

import (
	"fmt"
	"net/http"
	"os"
	"retailPos/internal/auth"
	model "retailPos/internal/model"
	"retailPos/internal/repo"
	"retailPos/internal/service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authService      auth.AuthService
	userRepo         *repo.UserRepo
	roleRepo         *repo.RoleRepo
	productRepo      *repo.ProductRepo
	productGroupRepo *repo.ProductGroupRepo
	statsRepo        *repo.StatsRepo
	salesRepo        *repo.SalesRepo
	salesService     *service.SalesService
}

func NewHandler(
	authService auth.AuthService,
	userRepo *repo.UserRepo,
	roleRepo *repo.RoleRepo,
	productRepo *repo.ProductRepo,
	productGroupRepo *repo.ProductGroupRepo,
	statsRepo *repo.StatsRepo,
	salesRepo *repo.SalesRepo,
	salesService *service.SalesService,
) *Handler {
	return &Handler{
		authService:      authService,
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		productRepo:      productRepo,
		productGroupRepo: productGroupRepo,
		statsRepo:        statsRepo,
		salesRepo:        salesRepo,
		salesService:     salesService,
	}
}

// Old AuthMiddleware has been removed and replaced by auth.AuthMiddleware

func (h *Handler) parseCommonParams(c *gin.Context) (limit, offset int, search, sortBy, sortDir, startDate, endDate string) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, _ = strconv.Atoi(limitStr)
	offset, _ = strconv.Atoi(offsetStr)
	search = c.Query("search")
	sortBy = c.Query("sortBy")
	sortDir = c.DefaultQuery("sortDir", "asc")
	startDate = c.Query("start_date")
	endDate = c.Query("end_date")
	return
}

// hasPermission checks if the current user (from context) has a given permission
func (h *Handler) hasPermission(c *gin.Context, permCode string) bool {
	permsInterface, exists := c.Get("permissions")
	if !exists {
		return false
	}
	perms, ok := permsInterface.([]string)
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permCode {
			return true
		}
	}
	return false
}

// Auth Handlers
func (h *Handler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()
	user, tokenPair, err := h.authService.Login(c.Request.Context(), input.Username, input.Password, ip)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Determine if we're in production (HTTPS required)
	isProd := os.Getenv("ENV") == "production" || os.Getenv("NODE_ENV") == "production"
	secureFlag := isProd

	// Set HTTP-only cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secureFlag,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// Token tidak lagi di-response body
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) Logout(c *gin.Context) {
	// Hapus cookie
	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		refreshToken := c.GetHeader("X-Refresh-Token")
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token required"})
			return
		}
		input.RefreshToken = refreshToken
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Use same secure flag as login
	isProd := os.Getenv("ENV") == "production" || os.Getenv("NODE_ENV") == "production"
	secureFlag := isProd

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session_token",
		Value:    tokenPair.AccessToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secureFlag,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	c.JSON(http.StatusOK, gin.H{
		"message":       "Token refreshed",
		"refresh_token": tokenPair.RefreshToken,
	})
}

func (h *Handler) ValidateSession(c *gin.Context) {
	userID := c.GetInt("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No valid session"})
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Load user's role and permissions
	role, err := h.userRepo.GetUserRole(userID)
	if err != nil || role == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load role"})
		return
	}

	permissions, err := h.userRepo.ListUserPermissions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"role":        role.Name,
			"role_id":     user.RoleID,
			"permissions": permissions,
		},
	})
}

// Admin endpoints: List all permissions (reference)
func (h *Handler) ListPermissions(c *gin.Context) {
	// Only admin can list all permissions
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	perms, err := h.roleRepo.GetAllPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

// ListRoles returns all roles with their permission codes
func (h *Handler) ListRoles(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	roles, err := h.roleRepo.GetAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]gin.H, 0, len(roles))
	for _, r := range roles {
		permCodes, _ := h.roleRepo.GetRolePermissions(r.ID)
		result = append(result, gin.H{
			"id":          r.ID,
			"name":        r.Name,
			"description": r.Description,
			"is_system":   r.IsSystem,
			"permissions": permCodes,
		})
	}
	c.JSON(http.StatusOK, gin.H{"roles": result})
}

// CreateRole creates a new role
func (h *Handler) CreateRole(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	var input struct {
		Name        string `json:"name" binding:"required,min=3,max=50"`
		Description string `json:"description"`
		Permissions []int  `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role, err := h.roleRepo.CreateRole(input.Name, input.Description, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, permID := range input.Permissions {
		h.roleRepo.AssignPermissionToRole(role.ID, permID)
	}
	c.JSON(http.StatusCreated, gin.H{"role": role})
}

// UpdateRolePermissions updates permissions for a role
func (h *Handler) UpdateRolePermissions(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}
	role, err := h.roleRepo.GetRoleByID(roleID)
	if err != nil || role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system role"})
		return
	}
	var input struct {
		PermissionIDs []int `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Remove existing permissions
	if err := h.roleRepo.ClearRolePermissions(roleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Add new permissions
	for _, permID := range input.PermissionIDs {
		h.roleRepo.AssignPermissionToRole(roleID, permID)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated"})
}

// DeleteRole deletes a non-system role
func (h *Handler) DeleteRole(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}
	role, err := h.roleRepo.GetRoleByID(roleID)
	if err != nil || role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system role"})
		return
	}
	if err := h.roleRepo.DeleteRole(roleID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Role is assigned to users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}

// ListUsers returns all users with their roles
func (h *Handler) ListUsers(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	users, err := h.userRepo.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type userResp struct {
		ID           int    `json:"id"`
		Username     string `json:"username"`
		RoleID       int    `json:"role_id"`
		RoleName     string `json:"role_name"`
		IsSystemRole bool   `json:"is_system_role"`
	}
	result := make([]userResp, 0, len(users))
	for _, u := range users {
		role, _ := h.roleRepo.GetRoleByID(u.RoleID)
		isSystem := false
		roleName := ""
		if role != nil {
			roleName = role.Name
			isSystem = role.IsSystem
		}
		result = append(result, userResp{
			ID:           u.ID,
			Username:     u.Username,
			RoleID:       u.RoleID,
			RoleName:     roleName,
			IsSystemRole: isSystem,
		})
	}
	c.JSON(http.StatusOK, gin.H{"users": result})
}

// UpdateUserRole updates a user's role
func (h *Handler) UpdateUserRole(c *gin.Context) {
	if c.GetString("role") != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin only"})
		return
	}
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	targetUser, err := h.userRepo.GetByID(userID)
	if err != nil || targetUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	targetRole, _ := h.roleRepo.GetRoleByID(targetUser.RoleID)
	if targetRole != nil && targetRole.Name == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify admin users"})
		return
	}
	var input struct {
		RoleID int `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newRole, _ := h.roleRepo.GetRoleByID(input.RoleID)
	if newRole == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if err := h.roleRepo.UpdateUserRole(userID, input.RoleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}

// Product Handlers
func (h *Handler) GetProducts(c *gin.Context) {
	barcode := c.Query("barcode")
	if barcode != "" {
		// First try by barcode
		p, err := h.productRepo.GetByBarcode(barcode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// If not found by barcode, try by SKU (legacy support)
		if p == nil {
			p, err = h.productRepo.GetBySKU(barcode)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if p == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusOK, p)
		return
	}

	limit, offset, search, sortBy, sortDir, _, _ := h.parseCommonParams(c)

	var groupID *int
	if g := c.Query("group_id"); g != "" {
		id, err := strconv.Atoi(g)
		if err == nil {
			groupID = &id
		}
	}

	var maxStock *int
	if ms := c.Query("maxStock"); ms != "" {
		val, err := strconv.Atoi(ms)
		if err == nil {
			maxStock = &val
		}
	}

	products, total, err := h.productRepo.GetAll(limit, offset, search, groupID, sortBy, sortDir, maxStock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   products,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	if !h.hasPermission(c, "product:create") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		return
	}

	var p model.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize empty barcode to nil for database NULL
	if p.Barcode != nil && *p.Barcode == "" {
		p.Barcode = nil
	}

	// 1. Check SKU Uniqueness (including deleted)
	existingBySKU, err := h.productRepo.GetBySKUWithDeleted(p.SKU)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check SKU existence"})
		return
	}

	if existingBySKU != nil {
		if existingBySKU.DeletedAt == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Product with this SKU already exists"})
			return
		}
		// Restore the deleted product
		p.ID = existingBySKU.ID
		if err := h.productRepo.Restore(&p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore product"})
			return
		}
		c.JSON(http.StatusCreated, p)
		return
	}

	// 2. Check Barcode Uniqueness if provided
	if p.Barcode != nil && *p.Barcode != "" {
		existingByBarcode, err := h.productRepo.GetByBarcodeWithDeleted(*p.Barcode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Barcode existence"})
			return
		}
		if existingByBarcode != nil {
			status := "exists"
			if existingByBarcode.DeletedAt != nil {
				status = "exists in deleted records"
			}
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Product with this Barcode already %s (SKU: %s)", status, existingByBarcode.SKU)})
			return
		}
	}

	// 3. Create new product
	if err := h.productRepo.Create(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	if !h.hasPermission(c, "product:update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var p model.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = idInt

	// Normalize empty barcode to nil for database NULL
	if p.Barcode != nil && *p.Barcode == "" {
		p.Barcode = nil
	}

	if err := h.productRepo.Update(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *Handler) DeleteProduct(c *gin.Context) {
	if !h.hasPermission(c, "product:delete") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	existingP, err := h.productRepo.GetByID(idInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check product"})
		return
	}

	if existingP == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if existingP.Stock > 0 {
		c.JSON(http.StatusFailedDependency, gin.H{"error": "Product can only be deleted if stock is 0"})
		return
	}

	if err := h.productRepo.Delete(idInt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// Sales Handlers
func (h *Handler) CreateSale(c *gin.Context) {
	var sale model.Sale
	if err := c.ShouldBindJSON(&sale); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sale.CashierID = c.GetInt("user_id")
	if err := h.salesService.CreateSale(c.Request.Context(), &sale); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sale)
}

// Product Group Handlers
func (h *Handler) GetProductGroups(c *gin.Context) {
	limit, offset, search, sortBy, sortDir, _, _ := h.parseCommonParams(c)
	groups, total, err := h.productGroupRepo.GetAll(limit, offset, search, sortBy, sortDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   groups,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) CreateProductGroup(c *gin.Context) {
	if !h.hasPermission(c, "group:create") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		return
	}

	var g model.ProductGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.productGroupRepo.Create(&g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, g)
}

func (h *Handler) UpdateProductGroup(c *gin.Context) {
	if !h.hasPermission(c, "group:update") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var g model.ProductGroup
	if err := c.ShouldBindJSON(&g); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	g.ID = idInt

	if err := h.productGroupRepo.Update(&g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, g)
}

func (h *Handler) DeleteProductGroup(c *gin.Context) {
	if !h.hasPermission(c, "group:delete") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.productGroupRepo.Delete(idInt); err != nil {
		// Provide a friendly error if it's a foreign key constraint violation
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Kategori tidak dapat dihapus karena masih digunakan oleh satu atau lebih produk."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product Group deleted successfully"})
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	stats, err := h.statsRepo.GetDashboardStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) GetSalesHistory(c *gin.Context) {
	limit, offset, search, sortBy, sortDir, startDate, endDate := h.parseCommonParams(c)
	fmt.Printf("GetSalesHistory: startDate=%s, endDate=%s, limit=%d, offset=%d\n", startDate, endDate, limit, offset)
	sales, total, err := h.salesRepo.GetAll(limit, offset, search, sortBy, sortDir, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   sales,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetSalesChartData(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))
	groupBy := c.DefaultQuery("group_by", "day")

	fmt.Printf("CHART HANDLER: start=%s, end=%s, group=%s\n", startDate, endDate, groupBy)

	data, err := h.statsRepo.GetSalesChartData(startDate, endDate, groupBy)
	if err != nil {
		fmt.Printf("CHART ERROR: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fmt.Printf("CHART DATA: %+v\n", data)
	c.JSON(http.StatusOK, data)
}

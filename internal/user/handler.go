package user

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func auditContextFromGin(c *gin.Context) (userID *int, username, role string) {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(int); ok {
			userID = &id
		}
	}
	if v, ok := c.Get("username"); ok {
		username, _ = v.(string)
	}
	if v, ok := c.Get("role"); ok {
		role, _ = v.(string)
	}
	return
}

type UserService interface {
	GetUserByID(ctx context.Context, id int) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetAllUsers(ctx context.Context, limit, offset int, search, sortBy, sortDir string, roleID *int, isActive *bool) ([]User, int, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id int) error
	GetAllRoles(ctx context.Context) ([]Role, error)
	GetRoleByID(ctx context.Context, id int) (*Role, error)
	CreateRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id int) error
	CountUsersByRole(ctx context.Context, roleID int) (int, error)
	GetAllPermissions(ctx context.Context) ([]Permission, error)
	UpdateRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error
}

type Handler struct {
	svc      UserService
	auditSvc *audit.Service
}

func NewHandler(svc UserService, auditSvc *audit.Service) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.GET("/admin/users", auth, perm("user:read"), h.ListUsers)
	r.POST("/admin/users", auth, perm("user:create"), h.CreateUser)
	r.PUT("/admin/users/:id", auth, perm("user:update"), h.UpdateUser)
	r.DELETE("/admin/users/:id", auth, perm("user:delete"), h.DeleteUser)
	r.GET("/admin/roles", auth, perm("role:read"), h.ListRoles)
	r.POST("/admin/roles", auth, perm("role:create"), h.CreateRole)
	r.PUT("/admin/roles/:id", auth, perm("role:update"), h.UpdateRole)
	r.PUT("/admin/roles/:id/permissions", auth, perm("role:update"), h.UpdateRolePermissions)
	r.DELETE("/admin/roles/:id", auth, perm("role:delete"), h.DeleteRole)
	r.GET("/admin/permissions", auth, perm("role:read"), h.ListPermissions)
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	RoleID   int    `json:"role_id" binding:"required"`
	StoreID  *int   `json:"store_id"`
	IsActive *bool  `json:"is_active"`
}

type UpdateUserRequest struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	RoleID   *int    `json:"role_id"`
	StoreID  *int    `json:"store_id"`
	IsActive *bool   `json:"is_active"`
}

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type UpdateRolePermissionsRequest struct {
	PermissionIDs []int `json:"permission_ids" binding:"required"`
}

var usernameRegex = regexp.MustCompile(`^[a-z0-9]+$`)

func (h *Handler) ListUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "id")
	sortDir := c.DefaultQuery("sort_dir", "desc")

	var roleID *int
	if v := c.Query("role_id"); v != "" {
		id, err := strconv.Atoi(v)
		if err == nil {
			roleID = &id
		}
	}

	var isActive *bool
	if v := c.Query("is_active"); v != "" {
		b := v == "true"
		isActive = &b
	}

	users, total, err := h.svc.GetAllUsers(c.Request.Context(), limit, offset, search, sortBy, sortDir, roleID, isActive)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Username = strings.ToLower(strings.TrimSpace(req.Username))
	if !usernameRegex.MatchString(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username must be lowercase alphanumeric"})
		return
	}

	existing, err := h.svc.GetUserByUsername(c.Request.Context(), req.Username)
	if err == nil && existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 14)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		RoleID:   req.RoleID,
		StoreID:  req.StoreID,
		IsActive: isActive,
	}

	if err := h.svc.CreateUser(c.Request.Context(), user); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID, actorUsername, actorRole := auditContextFromGin(c)
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    actorUsername,
			Role:        actorRole,
			Action:      "create",
			EntityType:  "user",
			EntityID:    &user.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"username": user.Username, "email": user.Email, "role_id": user.RoleID, "is_active": user.IsActive}),
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: fmt.Sprintf("Created user %s", user.Username),
		})
	}
	user.Password = ""
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *Handler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RoleID != nil || req.StoreID != nil {
		if currentUserID, ok := c.Get("userID"); ok {
			if uid, ok := currentUserID.(int); ok && uid == id {
				c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify your own role or store"})
				return
			}
		}
	}

	existing, err := h.svc.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	var oldValues map[string]interface{}
	if h.auditSvc != nil {
		oldValues = shared.ToJSONMap(map[string]interface{}{
			"username":  existing.Username,
			"email":     existing.Email,
			"role_id":   existing.RoleID,
			"is_active": existing.IsActive,
		})
	}

	if req.Username != nil {
		newUsername := strings.ToLower(strings.TrimSpace(*req.Username))
		if !usernameRegex.MatchString(newUsername) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username must be lowercase alphanumeric"})
			return
		}
		if newUsername != existing.Username {
			dup, err := h.svc.GetUserByUsername(c.Request.Context(), newUsername)
			if err == nil && dup != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
				return
			}
		}
		existing.Username = newUsername
	}
	if req.Email != nil {
		existing.Email = *req.Email
	}
	if req.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), 14)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		existing.Password = string(hashedPassword)
	}
	if req.RoleID != nil {
		existing.RoleID = *req.RoleID
	}
	if req.StoreID != nil {
		existing.StoreID = req.StoreID
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.svc.UpdateUser(c.Request.Context(), existing); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID, actorUsername, actorRole := auditContextFromGin(c)
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    actorUsername,
			Role:        actorRole,
			Action:      "update",
			EntityType:  "user",
			EntityID:    &existing.ID,
			OldValues:   oldValues,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"username": existing.Username, "email": existing.Email, "role_id": existing.RoleID, "is_active": existing.IsActive}),
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: fmt.Sprintf("Updated user %s", existing.Username),
		})
	}
	existing.Password = ""
	c.JSON(http.StatusOK, gin.H{"data": existing})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var oldUsername string
	if h.auditSvc != nil {
		if u, err := h.svc.GetUserByID(c.Request.Context(), id); err == nil {
			oldUsername = u.Username
		}
	}

	if err := h.svc.DeleteUser(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID, actorUsername, actorRole := auditContextFromGin(c)
		var description string
		if oldUsername != "" {
			description = fmt.Sprintf("Deleted user %s", oldUsername)
		} else {
			description = fmt.Sprintf("Deleted user #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    actorUsername,
			Role:        actorRole,
			Action:      "delete",
			EntityType:  "user",
			EntityID:    &id,
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ListRoles(c *gin.Context) {
	roles, err := h.svc.GetAllRoles(c.Request.Context())
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

func (h *Handler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := &Role{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := h.svc.CreateRole(c.Request.Context(), role); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID, actorUsername, actorRole := auditContextFromGin(c)
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    actorUsername,
			Role:        actorRole,
			Action:      "create",
			EntityType:  "role",
			EntityID:    &role.ID,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"name": role.Name, "description": role.Description}),
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: fmt.Sprintf("Created role %s", role.Name),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"data": role})
}

func (h *Handler) UpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing, err := h.svc.GetRoleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	oldValues := shared.ToJSONMap(map[string]interface{}{"name": existing.Name, "description": existing.Description})
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if err := h.svc.UpdateRole(c.Request.Context(), existing); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID, actorUsername, actorRole := auditContextFromGin(c)
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    actorUsername,
			Role:        actorRole,
			Action:      "update",
			EntityType:  "role",
			EntityID:    &existing.ID,
			OldValues:   oldValues,
			NewValues:   shared.ToJSONMap(map[string]interface{}{"name": existing.Name, "description": existing.Description}),
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: fmt.Sprintf("Updated role %s", existing.Name),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": existing})
}

func (h *Handler) UpdateRolePermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	var req UpdateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateRolePermissions(c.Request.Context(), id, req.PermissionIDs); err != nil {
		shared.InternalError(c, err)
		return
	}
	role, err := h.svc.GetRoleByID(c.Request.Context(), id)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": role})
}

func (h *Handler) DeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role id"})
		return
	}
	count, err := h.svc.CountUsersByRole(c.Request.Context(), id)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete role: users are assigned to this role"})
		return
	}
	var oldRoleName string
	if h.auditSvc != nil {
		if r, err := h.svc.GetRoleByID(c.Request.Context(), id); err == nil {
			oldRoleName = r.Name
		}
	}

	if err := h.svc.DeleteRole(c.Request.Context(), id); err != nil {
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		actorID, actorUsername, actorRole := auditContextFromGin(c)
		var description string
		if oldRoleName != "" {
			description = fmt.Sprintf("Deleted role %s", oldRoleName)
		} else {
			description = fmt.Sprintf("Deleted role #%d", id)
		}
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.AuditLog{
			UserID:      actorID,
			Username:    actorUsername,
			Role:        actorRole,
			Action:      "delete",
			EntityType:  "role",
			EntityID:    &id,
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *Handler) ListPermissions(c *gin.Context) {
	permissions, err := h.svc.GetAllPermissions(c.Request.Context())
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": permissions})
}

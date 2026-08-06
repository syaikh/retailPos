package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type AuthLoginService interface {
	Login(ctx context.Context, username, password string) (*LoginResponse, error)
	RefreshToken(ctx context.Context, oldRefreshToken string) (string, string, error)
	ValidateToken(tokenString string) (*AuthClaims, error)
	ChangePassword(ctx context.Context, userID int, currentPassword, newPassword string) error
	Logout(ctx context.Context, userID int, refreshToken string) error
	HashPassword(password string) (string, error)
}

type AuthHandler struct {
	svc      AuthLoginService
	auditSvc audit.Creator
}

func NewAuthHandler(svc AuthLoginService, auditSvc audit.Creator) *AuthHandler {
	return &AuthHandler{svc: svc, auditSvc: auditSvc}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup, auth, csrf gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/validate", auth, csrf, h.ValidateSession)
	r.POST("/logout", auth, csrf, h.Logout)
}

func (h *AuthHandler) RegisterRefreshRoute(r *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	handlers := append(middlewares, h.RefreshToken)
	r.POST("/refresh", handlers...)
}

func (h *AuthHandler) RegisterChangePasswordRoute(r *gin.RouterGroup, middlewares ...gin.HandlerFunc) {
	handlers := append(middlewares, h.ChangePassword)
	r.POST("/change-password", handlers...)
}

func (h *AuthHandler) RegisterLoginRoute(r *gin.RouterGroup, loginRateLimit gin.HandlerFunc) {
	r.POST("/login", loginRateLimit, h.Login)
}

// Login godoc
// @Summary Login
// @Description Authenticate user and return access token with refresh token cookie
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, shared.CtxKeyIPAddress, shared.GetIPAddress(c))
	ctx = context.WithValue(ctx, shared.CtxKeyUserAgent, shared.GetUserAgent(c))
	resp, err := h.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		shared.LogWarn(c.Request.Context(), "failed login attempt",
			"username", req.Username,
			"ip", shared.GetIPAddress(c),
			"user_agent", shared.GetUserAgent(c),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	domain := os.Getenv("COOKIE_DOMAIN")
	secure := os.Getenv("COOKIE_SECURE") == "true"

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", resp.RefreshToken, int(7*24*time.Hour/time.Second), "/", domain, secure, true)
	c.JSON(http.StatusOK, gin.H{
		"access_token": resp.AccessToken,
		"user":         resp.User,
	})

	if h.auditSvc != nil {
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      &resp.User.ID,
			Username:    resp.User.Username,
			Role:        resp.User.Role.Name,
			Action:      "login",
			EntityType:  "auth",
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: fmt.Sprintf("User %s logged in", req.Username),
		})
	}
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Exchange a refresh token for a new access token
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	token := c.GetHeader("X-Refresh-Token")
	if token == "" {
		token, _ = c.Cookie("refresh_token")
	}
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token is required"})
		return
	}

	accessToken, newRefreshToken, err := h.svc.RefreshToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	domain := os.Getenv("COOKIE_DOMAIN")
	secure := os.Getenv("COOKIE_SECURE") == "true"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", newRefreshToken, int(7*24*time.Hour/time.Second), "/", domain, secure, true)

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
}

func (h *AuthHandler) ValidateSession(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	permissions, _ := c.Get("permissions")
	storeID, _ := c.Get("storeID")

	resp := WithPermissions{
		Permissions: []string{},
	}
	if v, ok := userID.(int); ok {
		resp.ID = v
	}
	if v, ok := username.(string); ok {
		resp.Username = v
	}
	if v, ok := role.(string); ok {
		resp.Role = v
	}
	if v, ok := permissions.([]string); ok {
		resp.Permissions = v
	}
	if v, ok := storeID.(*int); ok {
		resp.StoreID = v
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       resp.ID,
			"username": resp.Username,
			"role":     resp.Role,
			"store_id": resp.StoreID,
		},
		"permissions": resp.Permissions,
	})
}

// ChangePassword godoc
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object true "Password change payload"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current_password and new_password (min 8 chars) are required"})
		return
	}

	userID, _ := c.Get("userID")
	id, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user session"})
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), id, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		shared.InternalError(c, err)
		return
	}

	if h.auditSvc != nil {
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		usernameStr, _ := username.(string)
		roleStr, _ := role.(string)
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      &id,
			Username:    usernameStr,
			Role:        roleStr,
			Action:      "update",
			EntityType:  "user",
			EntityID:    &id,
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: "Changed password",
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "password changed"})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := c.Get("userID")
	refreshToken, _ := c.Cookie("refresh_token")

	id, ok := userID.(int)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user session"})
		return
	}

	if refreshToken != "" {
		if err := h.svc.Logout(c.Request.Context(), id, refreshToken); err != nil {
			shared.InternalError(c, err)
			return
		}
	}

	domain := os.Getenv("COOKIE_DOMAIN")
	secure := os.Getenv("COOKIE_SECURE") == "true"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", "", -1, "/", domain, secure, true)

	if h.auditSvc != nil {
		userID, _ := c.Get("userID")
		uid, _ := userID.(int)
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		usernameStr, _ := username.(string)
		roleStr, _ := role.(string)
		_ = h.auditSvc.CreateAuditLog(c.Request.Context(), &audit.Log{
			UserID:      &uid,
			Username:    usernameStr,
			Role:        roleStr,
			Action:      "logout",
			EntityType:  "auth",
			IPAddress:   shared.GetIPAddress(c),
			UserAgent:   shared.GetUserAgent(c),
			Description: "User logged out",
		})
	}

	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

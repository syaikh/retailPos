package user

import (
	"context"
	"net/http"
	"os"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *AuthService
}

func NewAuthHandler(svc *AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(string) gin.HandlerFunc) {
	r.POST("/login", h.Login)
	r.POST("/refresh", h.RefreshToken)
	r.POST("/validate", auth, h.ValidateSession)
	r.POST("/logout", auth, h.Logout)
}

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
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	domain := os.Getenv("COOKIE_DOMAIN")
	secure := os.Getenv("COOKIE_SECURE") == "true"

	c.SetCookie("refresh_token", resp.RefreshToken, int(7*24*time.Hour/time.Second), "/", domain, secure, true)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  resp.AccessToken,
		"refresh_token": resp.RefreshToken,
		"user":          resp.User,
	})
}

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
	c.SetCookie("refresh_token", newRefreshToken, int(7*24*time.Hour/time.Second), "/", domain, secure, true)

	c.JSON(http.StatusOK, gin.H{"access_token": accessToken, "refresh_token": newRefreshToken})
}

func (h *AuthHandler) ValidateSession(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	permissions, _ := c.Get("permissions")
	storeID, _ := c.Get("storeID")

	resp := UserWithPermissions{
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	domain := os.Getenv("COOKIE_DOMAIN")
	secure := os.Getenv("COOKIE_SECURE") == "true"
	c.SetCookie("refresh_token", "", -1, "/", domain, secure, true)

	c.JSON(http.StatusOK, gin.H{"status": "logged out"})
}

package auth

import (
	"net/http"
	"strings"

	"retailPos/internal/repo"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(tokenService TokenService, roleRepo *repo.RoleRepo, userRepo *repo.UserRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""

		// Read from cookie first
		cookie, err := c.Cookie("session_token")
		if err == nil && cookie != "" {
			tokenStr = cookie
		} else {
			// Fallback to Authorization header
			authHeader := c.GetHeader("Authorization")
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		userID, roleID, roleName, err := tokenService.ValidateAccessToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Store basic user info
		c.Set("user_id", userID)
		c.Set("role_id", roleID)
		c.Set("role", roleName)

		// Fetch and store user to get store_id
		user, err := userRepo.GetByID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
			c.Abort()
			return
		}
		if user != nil && user.StoreID != nil {
			c.Set("store_id", *user.StoreID)
		}

		// Fetch and store user permissions from DB
		permissions, err := roleRepo.ListUserPermissions(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load permissions"})
			c.Abort()
			return
		}
		c.Set("permissions", permissions)

		c.Next()
	}
}

func RoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		roleStr, ok := role.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role"})
			c.Abort()
			return
		}

		hasAccess := false
		for _, reqRole := range requiredRoles {
			if roleStr == reqRole {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

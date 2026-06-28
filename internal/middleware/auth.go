package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"retail-pos-system/internal/user"

	"github.com/gin-gonic/gin"
)

func setCtxValue(ctx context.Context, key, val interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, key, val)
}

func NewModularAuthMiddleware(authService *user.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
			return
		}

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("userID", claims.ID)
		c.Set("username", claims.Username)
		c.Set("roleID", claims.RoleID)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Set("storeID", claims.StoreID)

		// Store user info in request context so event listeners can access it
		ctx := c.Request.Context()
		ctx = setCtxValue(ctx, CtxKeyUserID, claims.ID)
		ctx = setCtxValue(ctx, CtxKeyUsername, claims.Username)
		ctx = setCtxValue(ctx, CtxKeyRole, claims.Role)
		c.Request = c.Request.WithContext(ctx)

		c.Header("X-User-ID", fmt.Sprintf("%d", claims.ID))
		c.Header("X-User-Role", claims.Role)

		c.Next()
	}
}

func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user role not found in context"})
			return
		}

		if role.(string) != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role permission"})
			return
		}

		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms, exists := c.Get("permissions")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "permissions not found in context"})
			return
		}

		permissions := perms.([]string)
		if !hasPermission(permissions, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permission", "required": permission})
			return
		}

		c.Next()
	}
}

func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPerms, exists := c.Get("permissions")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "permissions not found in context"})
			return
		}

		userPermissions := userPerms.([]string)
		for _, perm := range permissions {
			if hasPermission(userPermissions, perm) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions", "required_any": permissions})
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user role not found"})
			return
		}

		roleStr, ok := role.(string)
		if !ok || roleStr != "superadmin" && roleStr != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && (parts[0] == "Bearer" || parts[0] == "Token") {
			return parts[1]
		}
	}

	token := c.Query("token")
	if token != "" {
		return token
	}

	cookie, err := c.Cookie("access_token")
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
}

func hasPermission(userPerms []string, requiredPerm string) bool {
	for _, perm := range userPerms {
		if perm == requiredPerm {
			return true
		}
	}
	return false
}

func GetUserID(c *gin.Context) int {
	if userID, exists := c.Get("userID"); exists {
		return userID.(int)
	}
	return 0
}

func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get("role"); exists {
		return role.(string)
	}
	return ""
}

func GetPermissions(c *gin.Context) []string {
	if perms, exists := c.Get("permissions"); exists {
		return perms.([]string)
	}
	return []string{}
}

func GetStoreID(c *gin.Context) *int {
	if storeID, exists := c.Get("storeID"); exists {
		if sid, ok := storeID.(*int); ok {
			return sid
		}
	}
	return nil
}

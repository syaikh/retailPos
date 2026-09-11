package middleware

import (
	"context"
	"net/http"
	"strings"

	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "authorization token required"))
			return
		}

		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "invalid or expired token"))
			return
		}

		c.Set("userID", claims.ID)
		c.Set("username", claims.Username)
		c.Set("roleID", claims.RoleID)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Set("storeID", claims.StoreID)
		c.Set("reportsToID", claims.ReportsToID)

		// Store user info in request context so event listeners can access it
		ctx := c.Request.Context()
		ctx = setCtxValue(ctx, CtxKeyUserID, claims.ID)
		ctx = setCtxValue(ctx, CtxKeyUsername, claims.Username)
		ctx = setCtxValue(ctx, CtxKeyRole, claims.Role)
		ctx = setCtxValue(ctx, CtxKeyStoreID, claims.StoreID)
		ctx = setCtxValue(ctx, CtxKeyPermissions, claims.Permissions)
		ctx = setCtxValue(ctx, CtxKeyReportsToID, claims.ReportsToID)
		ctx = setCtxValue(ctx, shared.CtxKeyIPAddress, shared.GetIPAddress(c))
		ctx = setCtxValue(ctx, shared.CtxKeyUserAgent, shared.GetUserAgent(c))
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func RoleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "user role not found in context"))
			return
		}

		if role.(string) != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, shared.NewError(shared.ErrForbidden, "insufficient role permission"))
			return
		}

		c.Next()
	}
}

func RequirePermission(permission permissions.Code) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms, exists := c.Get("permissions")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "permissions not found in context"))
			return
		}

		userPermissions := perms.([]string)
		if !hasPermission(userPermissions, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":    shared.ErrorDetail{Code: shared.ErrForbidden, Message: "insufficient permission"},
				"required": permission,
			})
			return
		}

		c.Next()
	}
}

func RequireAnyPermission(requiredPerms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPerms, exists := c.Get("permissions")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "permissions not found in context"))
			return
		}

		userPermissions := userPerms.([]string)
		for _, perm := range requiredPerms {
			if hasPermission(userPermissions, permissions.Code(perm)) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":        shared.ErrorDetail{Code: shared.ErrForbidden, Message: "insufficient permissions"},
			"required_any": requiredPerms,
		})
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, shared.NewError(shared.ErrUnauthorized, "user role not found"))
			return
		}

		roleStr, ok := role.(string)
		if !ok || roleStr != permissions.RoleSuperadmin && roleStr != permissions.RoleManager {
			c.AbortWithStatusJSON(http.StatusForbidden, shared.NewError(shared.ErrForbidden, "admin access required"))
			return
		}

		c.Next()
	}
}

// RequireStoreID rejects requests from operational roles (cashier, supervisor,
// finance, inventory_staff) when the JWT carries no store_id. Superadmin and
// manager bypass this check because they can operate across stores.
func RequireStoreID() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleStr, _ := role.(string)

		// Superadmin and manager bypass — they can operate across stores.
		if roleStr == permissions.RoleSuperadmin || roleStr == permissions.RoleManager {
			c.Next()
			return
		}

		// Operational roles must have a store_id.
		if permissions.OperationalRoles[roleStr] {
			storeID, _ := c.Get("storeID")
			if storeID == nil {
				c.AbortWithStatusJSON(http.StatusForbidden, shared.NewError(shared.ErrForbidden, "store not configured for this user"))
				return
			}
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

	cookie, err := c.Cookie("access_token")
	if err == nil && cookie != "" {
		return cookie
	}

	return ""
}

func hasPermission(userPerms []string, requiredPerm permissions.Code) bool {
	for _, perm := range userPerms {
		if perm == string(requiredPerm) {
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

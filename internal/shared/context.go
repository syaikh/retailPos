package shared

import (
	"context"

	"github.com/gin-gonic/gin"
)

type CtxKey string

const (
	CtxKeyIPAddress   CtxKey = "ipAddress"
	CtxKeyUserAgent   CtxKey = "userAgent"
	CtxKeyRequestID   CtxKey = "requestID"
	CtxKeyRequestPath CtxKey = "requestPath"
)

func GetUserID(c *gin.Context) int {
	userID, exists := c.Get("userID")
	if !exists {
		return 0
	}
	id, ok := userID.(int)
	if !ok {
		return 0
	}
	return id
}

func GetUserIDWithOK(c *gin.Context) (int, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	id, ok := userID.(int)
	if !ok {
		return 0, false
	}
	return id, true
}

func GetUsername(c *gin.Context) string {
	username, exists := c.Get("username")
	if !exists {
		return ""
	}
	name, ok := username.(string)
	if !ok {
		return ""
	}
	return name
}

func GetRole(c *gin.Context) string {
	role, exists := c.Get("role")
	if !exists {
		return ""
	}
	r, ok := role.(string)
	if !ok {
		return ""
	}
	return r
}

func GetStoreID(c *gin.Context) *int {
	storeID, exists := c.Get("storeID")
	if !exists {
		return nil
	}
	sid, ok := storeID.(*int)
	if !ok {
		return nil
	}
	return sid
}

func GetStoreIDInt(c *gin.Context) int {
	sid := GetStoreID(c)
	if sid == nil {
		return 0
	}
	return *sid
}

// GetIPAddress returns the client IP as resolved by gin: X-Forwarded-For is
// honored only when the TCP peer is a trusted proxy per router.SetTrustedProxies
// (see cmd/server/main.go). This value keys login-failure counters and lands in
// audit logs, so it must stay consistent with the rate limiter's view of the
// client — using RemoteAddr directly would record the reverse proxy's address
// for every user behind nginx.
func GetIPAddress(c *gin.Context) string {
	return c.ClientIP()
}

func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestID, id)
}

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(CtxKeyRequestID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func SetRequestPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestPath, path)
}

func GetRequestPath(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(CtxKeyRequestPath); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func GetUserAgent(c *gin.Context) string {
	return c.GetHeader("User-Agent")
}

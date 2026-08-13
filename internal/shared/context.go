package shared

import (
	"context"
	"net"
	"strings"

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

func GetIPAddress(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(c.Request.RemoteAddr)
	}
	return host
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

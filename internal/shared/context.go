package shared

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type CtxKey string

const (
	CtxKeyIPAddress CtxKey = "ipAddress"
	CtxKeyUserAgent CtxKey = "userAgent"
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

func GetIPAddress(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		log.Printf("warning: X-Forwarded-For header detected (%s); using RemoteAddr instead", xff)
	}

	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(c.Request.RemoteAddr)
	}
	return host
}

func GetUserAgent(c *gin.Context) string {
	return c.GetHeader("User-Agent")
}

func AbortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message})
}

func AbortForbidden(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": message})
}

func AbortInternalError(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": message})
}

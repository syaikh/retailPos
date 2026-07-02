package shared

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func JSONSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

func JSONError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"error": message,
	})
}

func JSONPaginated(c *gin.Context, data interface{}, total, limit, offset int) {
	c.JSON(http.StatusOK, NewPaginatedResponse(data, total, limit, offset))
}

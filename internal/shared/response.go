package shared

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ToJSONMap(v interface{}) map[string]interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	return m
}

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

func InternalError(c *gin.Context, err error) {
	log.Printf("Internal server error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}

func JSONPaginated(c *gin.Context, data interface{}, total, limit, offset int) {
	c.JSON(http.StatusOK, NewPaginatedResponse(data, total, limit, offset))
}

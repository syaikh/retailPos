package shared

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Standard error codes for structured API error responses.
const (
	ErrBadRequest   = "BAD_REQUEST"
	ErrNotFound     = "NOT_FOUND"
	ErrUnauthorized = "UNAUTHORIZED"
	ErrForbidden    = "FORBIDDEN"
	ErrConflict     = "CONFLICT"
	ErrInternal     = "INTERNAL_ERROR"
	ErrValidation   = "VALIDATION_ERROR"
	ErrRateLimited  = "RATE_LIMITED"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

func NewError(code, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
}

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

func JSONError(c *gin.Context, status int, code, message string) {
	c.JSON(status, NewError(code, message))
}

func InternalError(c *gin.Context, err error) {
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}
	LogError(ctx, "internal server error", err)
	c.JSON(http.StatusInternalServerError, NewError(ErrInternal, "internal server error"))
}

func JSONPaginated(c *gin.Context, data interface{}, total, limit, offset int) {
	c.JSON(http.StatusOK, NewPaginatedResponse(data, total, limit, offset))
}

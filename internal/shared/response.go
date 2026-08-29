package shared

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"

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

// piiScrubKeys are lowercase substrings of map keys that must never be persisted
// in audit payloads. They target directly identifiable customer data.
var piiScrubKeys = []string{"phone", "email", "customer_name"}

// ScrubPII removes known PII keys from a map recursively (including nested
// maps and slices) and returns it. It is intended for audit new_values so the
// immutable trail does not accumulate customer-identifying data (e.g.
// customer_name on sale records).
func ScrubPII(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return m
	}
	for k, v := range m {
		lk := strings.ToLower(k)
		scrubbed := false
		for _, bad := range piiScrubKeys {
			if strings.Contains(lk, bad) {
				delete(m, k)
				scrubbed = true
				break
			}
		}
		if scrubbed {
			continue
		}
		switch val := v.(type) {
		case map[string]interface{}:
			ScrubPII(val)
		case []interface{}:
			scrubSlice(val)
		}
	}
	return m
}

func scrubSlice(s []interface{}) {
	for _, item := range s {
		switch v := item.(type) {
		case map[string]interface{}:
			ScrubPII(v)
		case []interface{}:
			scrubSlice(v)
		}
	}
}

// DiffChanges compares an old and new representation of an entity and returns
// focused old/new maps containing only the keys whose values actually changed.
// Keys present in new but not old (or vice versa) are included; unchanged keys
// are omitted. This keeps audit payloads minimal (smaller, less PII) while still
// recording enough to reconstruct what changed.
func DiffChanges(oldMap, newMap map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	if oldMap == nil {
		oldMap = map[string]interface{}{}
	}
	if newMap == nil {
		newMap = map[string]interface{}{}
	}
	oldVals := map[string]interface{}{}
	newVals := map[string]interface{}{}
	for k, nv := range newMap {
		ov, present := oldMap[k]
		if !present || !valuesEqual(ov, nv) {
			if present {
				oldVals[k] = ov
			}
			newVals[k] = nv
		}
	}
	for k, ov := range oldMap {
		if _, present := newMap[k]; !present {
			oldVals[k] = ov
			newVals[k] = nil
		}
	}
	return oldVals, newVals
}

// valuesEqual reports whether two JSON-derived values are semantically equal.
func valuesEqual(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if reflect.DeepEqual(a, b) {
		return true
	}
	ab, errA := json.Marshal(a)
	bb, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return bytes.Equal(ab, bb)
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

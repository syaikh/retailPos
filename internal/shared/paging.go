package shared

import (
	"math"
	"strconv"
)

const DefaultMaxPageLimit = 100

func ParsePaginationParams(limitStr, offsetStr string) (int, int) {
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > DefaultMaxPageLimit {
		limit = 20
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ParseIntParam parses an optional integer query parameter, returning 0 when
// absent or malformed.
func ParseIntParam(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	TotalPages int         `json:"total_pages"`
}

func NewPaginatedResponse(data interface{}, total, limit, offset int) PaginatedResponse {
	totalPages := 0
	if limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}
	return PaginatedResponse{
		Data:       data,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		TotalPages: totalPages,
	}
}

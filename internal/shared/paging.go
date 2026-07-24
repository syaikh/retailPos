package shared

import (
	"math"
	"strconv"
	"strings"
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

var allowedSortColumns = map[string]map[string]bool{
	"products": {"name": true, "sku": true, "price": true, "stock": true, "created_at": true, "updated_at": true},
	"sales":    {"created_at": true, "total_amount": true, "invoice_number": true},
	"users":    {"username": true, "email": true, "created_at": true, "updated_at": true},
}

func SanitizeSortBy(sortBy string, context string) string {
	if cols, ok := allowedSortColumns[context]; ok {
		if cols[sortBy] {
			return sortBy
		}
	}
	return "created_at"
}

func SanitizeSortDir(sortDir string) string {
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		return "DESC"
	}
	return sortDir
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

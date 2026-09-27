package shared

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

const DefaultMaxPageLimit = 100

// MaxPageOffset bounds how deep any list endpoint can page. Beyond this the
// OFFSET scan cost grows linearly while the result is of little practical
// use; requests with a larger offset are rejected with 400 rather than
// silently clamped, because serving a different page than the one requested
// would make clients render row ranges that do not match the data.
const MaxPageOffset = 10000

// ParsePaginationParams parses limit/offset query params. limit is clamped to
// [1, DefaultMaxPageLimit] (defaulting to 20 when absent or invalid) and
// offset is clamped to [0, MaxPageOffset] when negative. An offset beyond
// MaxPageOffset returns an error so handlers can reject the request instead
// of quietly answering with the wrong page.
func ParsePaginationParams(limitStr, offsetStr string) (int, int, error) {
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	} else if limit > DefaultMaxPageLimit {
		limit = DefaultMaxPageLimit
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	} else if offset > MaxPageOffset {
		return 0, 0, fmt.Errorf("offset must not exceed %d", MaxPageOffset)
	}
	return limit, offset, nil
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
	// HasMore is true when at least one row exists beyond this page. For
	// cursor responses it is set from the authoritative limit+1 probe; for
	// plain offset responses it is derived from total/limit/offset.
	HasMore bool `json:"has_more"`
	// NextCursor is the opaque keyset seek position after the last row of
	// this page, present only when HasMore is true.
	NextCursor string `json:"next_cursor,omitempty"`
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
		HasMore:    limit > 0 && offset+limit < total,
	}
}

// KeysetCursor is the seek position for keyset (cursor) pagination over
// lists ordered by (created_at, id).
type KeysetCursor struct {
	CreatedAt time.Time
	ID        int
}

// ParseKeysetCursor parses the optional after_created_at (RFC3339) and
// after_id query parameters. It returns (nil, nil) when both are absent
// (offset mode), and an error when only one is present or either is
// malformed. The pair is always provided together so a partial cursor can
// never silently degrade into a wrong page.
func ParseKeysetCursor(createdAtStr, idStr string) (*KeysetCursor, error) {
	if createdAtStr == "" && idStr == "" {
		return nil, nil
	}
	if createdAtStr == "" {
		return nil, errors.New("after_id requires after_created_at")
	}
	if idStr == "" {
		return nil, errors.New("after_created_at requires after_id")
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, errors.New("after_created_at must be an RFC3339 timestamp")
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 {
		return nil, errors.New("after_id must be a non-negative integer")
	}
	return &KeysetCursor{CreatedAt: createdAt, ID: id}, nil
}

// String renders the cursor as the opaque next_cursor value sent to clients:
// "<RFC3339Nano timestamp>|<id>". RFC3339Nano keeps the full sub-second
// precision of the stored timestamp, so the seek predicate stays exact and
// page boundaries never duplicate or drop rows.
func (c *KeysetCursor) String() string {
	return c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + strconv.Itoa(c.ID)
}

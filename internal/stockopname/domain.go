package stockopname

import "errors"

var (
	ErrNotFound            = errors.New("stock opname session not found")
	ErrInvalidState        = errors.New("invalid session state for this operation")
	ErrSessionLocked       = errors.New("session is locked")
	ErrAlreadySubmitted    = errors.New("session already submitted")
	ErrAlreadyApproved     = errors.New("session already approved")
	ErrNotAllItemsCounted  = errors.New("all items must be counted before submit")
	ErrNotAssigned         = errors.New("user is not assigned to this session")
	ErrSeparationOfDuties  = errors.New("counter cannot approve or verify the same session")
	ErrItemNotFound        = errors.New("stock opname item not found")
	ErrAssignmentNotFound  = errors.New("assignment not found")
	ErrInvalidQuantity     = errors.New("invalid quantity")
	ErrApprovalCommentReq  = errors.New("approval/rejection comment is required")
	ErrUnsupportedScope    = errors.New("unsupported scope type")
	ErrNoItems             = errors.New("session has no items to count")
	ErrNoPermission        = errors.New("user lacks permission")
	ErrAdjustmentFailed    = errors.New("inventory adjustment failed")
	ErrProductNotFound     = errors.New("product not found")
	ErrInvalidAssigneeRole = errors.New("user role is not valid for this assignment role")
	ErrAssigneeNotFound    = errors.New("assignee user not found or inactive")
	ErrNoScopes            = errors.New("at least one scope is required")
	ErrScopeIDRequired     = errors.New("scope_id is required for this scope type")
	ErrScopeOverlap        = errors.New("scope overlaps an active session")
	ErrAlreadyVerified     = errors.New("session already verified")
	ErrAlreadyPosted       = errors.New("session already posted")
	ErrAlreadyClosed       = errors.New("session already closed")
	ErrAdjustmentNotFound  = errors.New("inventory adjustment not found")
	ErrOpenCommentReq      = errors.New("comment is required to open the session")
	ErrLocationScopeSingle = errors.New("location scope must be used alone")
	ErrLocationNotFound    = errors.New("storage location not found")
	ErrLocationInactive    = errors.New("storage location is inactive")
	ErrStoreForbidden      = errors.New("stock opname session is not in your store")
)

const (
	StatusDraft        = "draft"
	StatusOpen         = "open"
	StatusCounting     = "counting"
	StatusVerification = "verification"
	StatusNeedsRecount = "needs_recount"
	StatusApproved     = "approved"
	StatusPosted       = "posted"
	StatusClosed       = "closed"
	StatusCancelled    = "cancelled"
	// StatusPendingApproval is retained as an alias for external consumers of
	// the archived v1 state; the active workflow uses StatusVerification.
	StatusPendingApproval = StatusVerification
)

const (
	AssignmentRoleCounter    = "counter"
	AssignmentRoleSupervisor = "supervisor"
)

const (
	MovementTypeStockOpname = "stock_opname"
	ItemStatusPending       = "pending"
	ItemStatusCounted       = "counted"
)

var validScopes = map[string]bool{
	"store":     true,
	"warehouse": true,
	"category":  true,
	"brand":     true,
	"supplier":  true,
	"product":   true,
	"manual":    true,
	"location":  true,
}

// SessionScope is one lookup scope of a session. A session may span several
// scopes (e.g. "category: 12" plus "supplier: 5"); the product universe is the
// union of all scopes. The stored scope_name displays nicely in UI and reports.
type SessionScope struct {
	ID            int    `json:"id"`
	StockOpnameID int    `json:"stock_opname_id"`
	ScopeType     string `json:"scope_type"`
	ScopeID       int64  `json:"scope_id"`
	ScopeName     string `json:"scope_name,omitempty"`
}

type Session struct {
	ID              int             `json:"id"`
	SessionNumber   string          `json:"session_number"`
	Title           string          `json:"title,omitempty"`
	ScopeType       string          `json:"scope_type"`
	ScopeID         int64           `json:"scope_id"`
	ScopeName       string          `json:"scope_name"`
	Scopes          []SessionScope  `json:"scopes,omitempty"`
	WarehouseID     *int            `json:"warehouse_id,omitempty"`
	StoreID         *int            `json:"store_id,omitempty"`
	LocationID      *int            `json:"location_id,omitempty"`
	BlindCount      bool            `json:"blind_count"`
	Notes           string          `json:"notes,omitempty"`
	Status          string          `json:"status"`
	CreatedBy       int             `json:"created_by"`
	OpenedBy        *int            `json:"opened_by,omitempty"`
	OpenedAt        string          `json:"opened_at,omitempty"`
	VerifiedBy      *int            `json:"verified_by,omitempty"`
	VerifiedAt      string          `json:"verified_at,omitempty"`
	ApprovedBy      *int            `json:"approved_by,omitempty"`
	ApprovedAt      string          `json:"approved_at,omitempty"`
	PostedBy        *int            `json:"posted_by,omitempty"`
	PostedAt        string          `json:"posted_at,omitempty"`
	ClosedBy        *int            `json:"closed_by,omitempty"`
	ClosedAt        string          `json:"closed_at,omitempty"`
	TotalDifference float64         `json:"total_difference"`
	TotalAdjustment float64         `json:"total_adjustment"`
	CancelledAt     string          `json:"cancelled_at,omitempty"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
	Items           []SessionItem   `json:"items,omitempty"`
	Assignments     []Assignment    `json:"assignments,omitempty"`
	Summary         *SessionSummary `json:"summary,omitempty"`
}

type SessionItem struct {
	ID            int     `json:"id"`
	StockOpnameID int     `json:"stock_opname_id"`
	ProductID     int     `json:"product_id"`
	ProductName   string  `json:"product_name"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	UOMName       string  `json:"uom_name"`
	OpeningQty    float64 `json:"opening_qty"`
	ExpectedQty   float64 `json:"expected_qty"`
	PhysicalQty   float64 `json:"physical_qty"`
	DifferenceQty float64 `json:"difference_qty"`
	AdjustmentQty float64 `json:"adjustment_qty"`
	Status        string  `json:"status"`
	Reason        string  `json:"reason,omitempty"`
	CountSequence int     `json:"count_sequence"`
	LastCountedBy *int    `json:"last_counted_by,omitempty"`
	LastCountedAt string  `json:"last_counted_at,omitempty"`
}

type Assignment struct {
	ID            int    `json:"id"`
	StockOpnameID int    `json:"stock_opname_id"`
	UserID        int    `json:"user_id"`
	Username      string `json:"username"`
	Role          string `json:"role"`
	AssignedAt    string `json:"assigned_at"`
}

type AssignableUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	RoleID   int    `json:"role_id"`
	RoleName string `json:"role_name"`
}

type SessionSummary struct {
	TotalItems      int     `json:"total_items"`
	CountedItems    int     `json:"counted_items"`
	PendingItems    int     `json:"pending_items"`
	TotalDifference float64 `json:"total_difference"`
	TotalAdjustment float64 `json:"total_adjustment"`
}

type CountRecord struct {
	ID                int     `json:"id"`
	StockOpnameItemID int     `json:"stock_opname_item_id"`
	CountSequence     int     `json:"count_sequence"`
	PhysicalQty       float64 `json:"physical_qty"`
	CountedBy         int     `json:"counted_by"`
	CountedByUser     string  `json:"counted_by_user"`
	CountedAt         string  `json:"counted_at"`
	Remarks           string  `json:"remarks,omitempty"`
}

// Adjustment is the source-of-truth ledger document emitted when differences
// are actually applied to stock. Takes its own document number (ia_seq), so
// posting can be approved separately from the physical count.
type Adjustment struct {
	ID               int              `json:"id"`
	AdjustmentNumber string           `json:"adjustment_number"`
	SessionID        int              `json:"session_id"`
	SessionNumber    string           `json:"session_number,omitempty"`
	Status           string           `json:"status"`
	Notes            string           `json:"notes,omitempty"`
	CreatedBy        int              `json:"created_by"`
	CreatedByName    string           `json:"created_by_name,omitempty"`
	CreatedAt        string           `json:"created_at"`
	Items            []AdjustmentItem `json:"items,omitempty"`
	TotalDifference  float64          `json:"total_difference"`
	TotalAdjustment  float64          `json:"total_adjustment"`
}

type AdjustmentItem struct {
	ID            int     `json:"id"`
	AdjustmentID  int     `json:"adjustment_id"`
	ProductID     int     `json:"product_id"`
	ProductName   string  `json:"product_name,omitempty"`
	SKU           string  `json:"sku,omitempty"`
	WarehouseID   *int    `json:"warehouse_id,omitempty"`
	StoreID       *int    `json:"store_id,omitempty"`
	ExpectedQty   float64 `json:"expected_qty"`
	PhysicalQty   float64 `json:"physical_qty"`
	DifferenceQty float64 `json:"difference_qty"`
	AdjustmentQty float64 `json:"adjustment_qty"`
	UnitCost      float64 `json:"unit_cost"`
	LineTotal     float64 `json:"line_total"`
	Reason        string  `json:"reason,omitempty"`
}

// request payloads

// Scope describes a single lookup scope on session creation. A convenience
// single primary scope is derived from the first scope for legacy consumers.
type Scope struct {
	ScopeType string `json:"scope_type"`
	ScopeID   int64  `json:"scope_id"`
	ScopeName string `json:"scope_name,omitempty"`
}

type CreateSessionRequest struct {
	Title       string  `json:"title"`
	Scopes      []Scope `json:"scopes"`
	ScopeType   string  `json:"scope_type"`
	ScopeID     int64   `json:"scope_id"`
	WarehouseID *int    `json:"warehouse_id,omitempty"`
	StoreID     *int    `json:"store_id,omitempty"`
	BlindCount  bool    `json:"blind_count"`
	Notes       string  `json:"notes"`
}

type OpenRequest struct {
	Comment string `json:"comment"`
}

type AssignRequest struct {
	UserID int    `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required"`
}

type ReassignRequest struct {
	Role string `json:"role" binding:"required"`
}

type SaveCountRequest struct {
	PhysicalQty float64 `json:"physical_qty"`
	Remarks     string  `json:"remarks,omitempty"`
}

type VerifyRequest struct {
	Comment string `json:"comment"`
}

type RejectRequest struct {
	Comment string `json:"comment"`
}

type RecountRequest struct {
	Comment string `json:"comment"`
}

type PostAdjustmentRequest struct {
	Comment string `json:"comment"`
	Notes   string `json:"notes,omitempty"`
}

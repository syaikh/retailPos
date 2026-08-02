package stockopname

import "errors"

var (
	ErrNotFound             = errors.New("stock opname session not found")
	ErrActiveSessionExists  = errors.New("an active stock opname session already exists")
	ErrInvalidState         = errors.New("invalid session state for this operation")
	ErrSessionLocked        = errors.New("session is locked")
	ErrAlreadySubmitted     = errors.New("session already submitted")
	ErrAlreadyApproved      = errors.New("session already approved")
	ErrNotAllItemsCounted   = errors.New("all items must be counted before submit")
	ErrNotAssigned          = errors.New("user is not assigned to this session")
	ErrSeparationOfDuties   = errors.New("counter cannot approve the same session")
	ErrItemNotFound         = errors.New("stock opname item not found")
	ErrAssignmentNotFound   = errors.New("assignment not found")
	ErrInvalidQuantity      = errors.New("invalid quantity")
	ErrApprovalCommentReq   = errors.New("approval/rejection comment is required")
	ErrUnsupportedScope     = errors.New("unsupported scope type")
	ErrNoItems              = errors.New("session has no items to count")
	ErrNoPermission         = errors.New("user lacks permission")
	ErrAdjustmentFailed     = errors.New("inventory adjustment failed")
	ErrProductNotFound      = errors.New("product not found")
	ErrInvalidAssigneeRole  = errors.New("user role is not valid for this assignment role")
	ErrAssigneeNotFound     = errors.New("assignee user not found or inactive")
)

const (
	StatusDraft           = "draft"
	StatusCounting        = "counting"
	StatusPendingApproval = "pending_approval"
	StatusNeedsRecount    = "needs_recount"
	StatusApproved        = "approved"
	StatusCancelled       = "cancelled"
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
	"store":    true,
	"warehouse": true,
	"category": true,
	"product":  true,
}

type Session struct {
	ID            int             `json:"id"`
	SessionNumber string          `json:"session_number"`
	ScopeType     string          `json:"scope_type"`
	ScopeID       int64           `json:"scope_id"`
	WarehouseID   *int            `json:"warehouse_id,omitempty"`
	BlindCount    bool            `json:"blind_count"`
	Status        string          `json:"status"`
	CreatedBy     int             `json:"created_by"`
	ApprovedBy    *int            `json:"approved_by,omitempty"`
	ApprovedAt    string          `json:"approved_at,omitempty"`
	CancelledAt   string          `json:"cancelled_at,omitempty"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
	Items         []SessionItem   `json:"items,omitempty"`
	Assignments   []Assignment    `json:"assignments,omitempty"`
	Summary       *SessionSummary `json:"summary,omitempty"`
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
	ID            int     `json:"id"`
	StockOpnameItemID int `json:"stock_opname_item_id"`
	CountSequence int     `json:"count_sequence"`
	PhysicalQty   float64 `json:"physical_qty"`
	CountedBy     int     `json:"counted_by"`
	CountedByUser string  `json:"counted_by_user"`
	CountedAt     string  `json:"counted_at"`
	Remarks       string  `json:"remarks,omitempty"`
}

// request payloads

type CreateSessionRequest struct {
	ScopeType   string `json:"scope_type" binding:"required"`
	ScopeID     int64  `json:"scope_id" binding:"required"`
	WarehouseID *int   `json:"warehouse_id,omitempty"`
	BlindCount  bool   `json:"blind_count"`
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

type ApproveRequest struct {
	Comment string `json:"comment"`
}

type RejectRequest struct {
	Comment string `json:"comment"`
}

type RecountRequest struct {
	Comment string `json:"comment"`
}

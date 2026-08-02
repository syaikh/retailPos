package stockopname

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"retail-pos-system/internal/shared"
)

type Service struct {
	repo     *Repository
	eventBus shared.EventBus
}

func NewService(repo *Repository, eventBus shared.EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

// publishStatusEvent broadcasts a stock opname status transition to the
// event bus. Only session-level status changes are published; item-level
// counts are excluded to preserve blind counting (BR-008).
func (s *Service) publishStatusEvent(ctx context.Context, topic string, sessionID int, status string) {
	if s.eventBus == nil {
		return
	}
	sessionNumber, storeID, err := s.repo.GetSessionBroadcastMeta(ctx, sessionID)
	if err != nil {
		slog.Warn("[stock-opname] failed to load session broadcast meta for status event", "session_id", sessionID, "error", err)
		return
	}
	_ = s.eventBus.Publish(ctx, topic, map[string]interface{}{
		"session_id":     sessionID,
		"session_number": sessionNumber,
		"status":         status,
		"store_id":       storeIDOrZero(storeID),
	})
}

// storeIDOrZero normalizes a store id for event payloads: nil or non-positive
// values are sent as 0 so the websocket layer treats the event as global.
func storeIDOrZero(storeID *int) int {
	if storeID == nil || *storeID <= 0 {
		return 0
	}
	return *storeID
}

// Status event topics published to the event bus.
const (
	EventStockOpnameCreated     = "stock_opname.created"
	EventStockOpnameSubmitted   = "stock_opname.submitted"
	EventStockOpnameApproved    = "stock_opname.approved"
	EventStockOpnameRejected    = "stock_opname.rejected"
	EventStockOpnameRecount     = "stock_opname.needs_recount"
	EventStockOpnameCancelled   = "stock_opname.cancelled"
)

func (s *Service) CreateSession(ctx context.Context, req *CreateSessionRequest, userID int) (*Session, error) {
	if !validScopes[req.ScopeType] {
		return nil, ErrUnsupportedScope
	}
	if req.ScopeID <= 0 {
		return nil, fmt.Errorf("scope_id is required")
	}

	active, err := s.repo.GetActiveSessionByScope(ctx, req.ScopeType, req.ScopeID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, ErrActiveSessionExists
	}

	items, err := s.repo.LoadSnapshotProducts(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNoItems
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	number, err := s.repo.GetNextSessionNumber(ctx)
	if err != nil {
		return nil, err
	}

	var storeID *int
	switch req.ScopeType {
	case "store":
		sid := int(req.ScopeID)
		storeID = &sid
	case "warehouse":
		whID := req.WarehouseID
		if whID == nil {
			id := int(req.ScopeID)
			whID = &id
		}
		storeID, err = s.repo.GetWarehouseStoreID(ctx, *whID)
		if err != nil {
			return nil, err
		}
	}

	session := &Session{
		SessionNumber: number,
		ScopeType:     req.ScopeType,
		ScopeID:       req.ScopeID,
		WarehouseID:   req.WarehouseID,
		StoreID:       storeID,
		BlindCount:    req.BlindCount,
		Status:        StatusDraft,
		CreatedBy:     userID,
	}
	if err := s.repo.CreateSession(ctx, tx, session); err != nil {
		return nil, err
	}
	if err := s.repo.InsertSessionItems(ctx, tx, session.ID, items); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	s.publishStatusEvent(ctx, EventStockOpnameCreated, session.ID, session.Status)
	return session, nil
}

// GetSessionForUser returns the session, masking system quantities for
// counters when blind count is enabled (BR-008).
func (s *Service) GetSessionForUser(ctx context.Context, id, userID int) (*Session, error) {
	session, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	blind, err := s.isCounterOnBlindSession(ctx, session, userID)
	if err != nil {
		return nil, err
	}
	if blind {
		for i := range session.Items {
			session.Items[i].OpeningQty = 0
			session.Items[i].ExpectedQty = 0
			session.Items[i].DifferenceQty = 0
		}
	}
	return session, nil
}

func (s *Service) isCounterOnBlindSession(ctx context.Context, session *Session, userID int) (bool, error) {
	if !session.BlindCount || userID == 0 {
		return false, nil
	}
	assigned, err := s.repo.IsCounterAssigned(ctx, session.ID, userID)
	if err != nil {
		return false, err
	}
	return assigned, nil
}

func (s *Service) ListSessions(ctx context.Context, limit, offset int, status, search string) ([]Session, int, error) {
	return s.repo.ListSessions(ctx, limit, offset, status, search)
}

// ListAssignableUsers returns active users eligible for stock opname
// assignment, optionally filtered by username/email search.
func (s *Service) ListAssignableUsers(ctx context.Context, search string) ([]AssignableUser, error) {
	return s.repo.ListAssignableUsers(ctx, search)
}

// allowedAssigneeRoles maps an assignment role to the system roles permitted
// to hold it. Counters are drawn from floor staff; supervisors must be
// manager-level or above.
var allowedAssigneeRoles = map[string]map[string]bool{
	AssignmentRoleCounter:    {"cashier": true, "staff": true},
	AssignmentRoleSupervisor: {"manager": true, "admin": true},
}

// validateAssigneeRole ensures the user holding an assignment is compatible
// with the requested assignment role (e.g. a staff member cannot be
// supervisor). Returns ErrInvalidAssigneeRole otherwise.
func (s *Service) validateAssigneeRole(ctx context.Context, userID int, role string) error {
	userRole, err := s.repo.GetUserRoleName(ctx, userID)
	if err != nil {
		return err
	}
	if !allowedAssigneeRoles[role][userRole] {
		return ErrInvalidAssigneeRole
	}
	return nil
}

func (s *Service) CancelSession(ctx context.Context, id, userID int) error {
	if err := s.repo.CancelSession(ctx, id, userID); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameCancelled, id, StatusCancelled)
	return nil
}

func (s *Service) AssignCounter(ctx context.Context, sessionID, userID int, role string) error {
	if role != AssignmentRoleCounter && role != AssignmentRoleSupervisor {
		return fmt.Errorf("invalid assignment role %q", role)
	}
	if err := s.validateAssigneeRole(ctx, userID, role); err != nil {
		return err
	}
	status, err := s.repo.GetSessionStatus(ctx, sessionID)
	if err != nil {
		return err
	}
	if !isEditableStatus(status) {
		return ErrSessionLocked
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.repo.InsertAssignment(ctx, tx, sessionID, userID, role); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) ReassignCounter(ctx context.Context, sessionID, assignmentID int, role string) error {
	if role != AssignmentRoleCounter && role != AssignmentRoleSupervisor {
		return fmt.Errorf("invalid assignment role %q", role)
	}
	userID, err := s.repo.GetAssignmentUserID(ctx, sessionID, assignmentID)
	if err != nil {
		return err
	}
	if err := s.validateAssigneeRole(ctx, userID, role); err != nil {
		return err
	}
	status, err := s.repo.GetSessionStatus(ctx, sessionID)
	if err != nil {
		return err
	}
	if !isEditableStatus(status) {
		return ErrSessionLocked
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.repo.UpdateAssignmentRole(ctx, tx, sessionID, assignmentID, role); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) GetAssignments(ctx context.Context, sessionID int) ([]Assignment, error) {
	if _, err := s.repo.GetSessionStatus(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.repo.ListAssignments(ctx, sessionID)
}

func (s *Service) SaveCount(ctx context.Context, itemID, userID int, qty float64, remarks string) error {
	if qty < 0 {
		return ErrInvalidQuantity
	}
	_, session, err := s.repo.GetItemForCount(ctx, itemID)
	if err != nil {
		return err
	}
	if !isEditableStatus(session.Status) {
		return ErrSessionLocked
	}
	assigned, err := s.repo.IsCounterAssigned(ctx, session.ID, userID)
	if err != nil {
		return err
	}
	if !assigned {
		return ErrNotAssigned
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.repo.LockItemForCount(ctx, tx, itemID); err != nil {
		return err
	}
	seq, err := s.repo.NextCountSequence(ctx, tx, itemID)
	if err != nil {
		return err
	}
	if err := s.repo.SaveCount(ctx, tx, itemID, seq, qty, userID, remarks); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) GetCountHistory(ctx context.Context, itemID int) ([]CountRecord, error) {
	return s.repo.GetCountHistory(ctx, itemID)
}

func (s *Service) SubmitSession(ctx context.Context, id, userID int) error {
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusCounting {
		return ErrInvalidState
	}
	assigned, err := s.repo.IsCounterAssigned(ctx, id, userID)
	if err != nil {
		return err
	}
	if !assigned {
		return ErrNotAssigned
	}
	pending, err := s.repo.CountPendingItems(ctx, id)
	if err != nil {
		return err
	}
	if pending > 0 {
		return ErrNotAllItemsCounted
	}
	if err := s.repo.UpdateStatus(ctx, id, StatusCounting, StatusPendingApproval); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameSubmitted, id, StatusPendingApproval)
	return nil
}

// StartCounting transitions a draft session into the counting state. Only an
// assigned counter can start counting (state machine: Draft --Start Counting--> Counting).
func (s *Service) StartCounting(ctx context.Context, id, userID int) error {
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusDraft {
		return ErrInvalidState
	}
	counter, err := s.repo.IsCounterAssigned(ctx, id, userID)
	if err != nil {
		return err
	}
	if !counter {
		return ErrNotAssigned
	}
	return s.repo.UpdateStatus(ctx, id, StatusDraft, StatusCounting)
}

func (s *Service) ApproveSession(ctx context.Context, id, userID int, comment string) error {
	if strings.TrimSpace(comment) == "" {
		return ErrApprovalCommentReq
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := s.repo.LockSessionForApproval(ctx, tx, id)
	if err != nil {
		return err
	}
	if session.Status != StatusPendingApproval {
		return ErrInvalidState
	}
	counter, err := s.repo.IsCounterAssigned(ctx, id, userID)
	if err != nil {
		return err
	}
	if counter {
		return ErrSeparationOfDuties
	}

	items, err := s.repo.LoadApprovalItems(ctx, tx, id)
	if err != nil {
		return err
	}
	productIDs := make([]int, len(items))
	for i, it := range items {
		productIDs[i] = it.ProductID
	}
	stock, err := s.repo.LockStockForProducts(ctx, tx, productIDs)
	if err != nil {
		return err
	}

	movements := make([]movementRow, 0, len(items))
	for _, it := range items {
		expected := float64(stock[it.ProductID])
		diff := it.PhysicalQy - expected
		adj := diff
		if err := s.repo.UpdateItemAdjustment(ctx, tx, it.ID, expected, diff, adj); err != nil {
			return err
		}
		if diff == 0 {
			continue
		}
		newQty := int(math.Round(expected + diff))
		if err := s.repo.UpdateProductStock(ctx, tx, it.ProductID, newQty); err != nil {
			return fmt.Errorf("%w: %v", ErrAdjustmentFailed, err)
		}
		notes := fmt.Sprintf("Stock opname %s: physical %.2f vs expected %.2f", session.SessionNumber, it.PhysicalQy, expected)
		movements = append(movements, movementRow{ProductID: it.ProductID, QuantityChange: int(math.Round(diff)), Notes: notes})
	}

	if err := s.repo.InsertMovements(ctx, tx, id, userID, movements); err != nil {
		return err
	}
	if err := s.repo.ApproveSessionStatus(ctx, tx, id, userID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameApproved, id, StatusApproved)
	return nil
}

func (s *Service) RejectSession(ctx context.Context, id, userID int, comment string) error {
	if strings.TrimSpace(comment) == "" {
		return ErrApprovalCommentReq
	}
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusPendingApproval {
		return ErrInvalidState
	}
	if err := s.repo.UpdateStatus(ctx, id, StatusPendingApproval, StatusNeedsRecount); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameRejected, id, StatusNeedsRecount)
	return nil
}

func (s *Service) RequestRecount(ctx context.Context, id, userID int, comment string) error {
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusPendingApproval {
		return ErrInvalidState
	}
	if err := s.repo.UpdateStatus(ctx, id, StatusPendingApproval, StatusNeedsRecount); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameRecount, id, StatusNeedsRecount)
	return nil
}

func (s *Service) ResumeCounting(ctx context.Context, id, userID int) error {
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusNeedsRecount {
		return ErrInvalidState
	}
	return s.repo.UpdateStatus(ctx, id, StatusNeedsRecount, StatusCounting)
}

func (s *Service) Summary(ctx context.Context, id, userID int) (*SessionSummary, error) {
	session, err := s.repo.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	blind, err := s.isCounterOnBlindSession(ctx, session, userID)
	if err != nil {
		return nil, err
	}
	sum := &SessionSummary{}
	for _, it := range session.Items {
		sum.TotalItems++
		if it.Status == ItemStatusCounted {
			sum.CountedItems++
		} else {
			sum.PendingItems++
		}
		if !blind {
			sum.TotalDifference += it.DifferenceQty
			sum.TotalAdjustment += it.AdjustmentQty
		}
	}
	return sum, nil
}

func (s *Service) DifferenceReport(ctx context.Context, id, userID int) (*Session, error) {
	return s.GetSessionForUser(ctx, id, userID)
}

func isEditableStatus(status string) bool {
	switch status {
	case StatusDraft, StatusCounting, StatusNeedsRecount:
		return true
	}
	return false
}

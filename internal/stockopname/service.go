package stockopname

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"retail-pos-system/internal/shared"
)

type Service struct {
	repo         *Repository
	eventBus     shared.EventBus
	stockApplier StockApplier
}

func NewService(repo *Repository, eventBus shared.EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

// SetStockApplier wires the inventory-owned implementation of the StockApplier
// port. The composition root MUST call this before any posting path runs; an
// unwired service fails fast at runtime (see ports.go).
func (s *Service) SetStockApplier(applier StockApplier) {
	s.stockApplier = applier
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
	EventStockOpnameCreated   = "stock_opname.created"
	EventStockOpnameOpened    = "stock_opname.opened"
	EventStockOpnameSubmitted = "stock_opname.submitted"
	EventStockOpnameApproved  = "stock_opname.approved"
	EventStockOpnamePosted    = "stock_opname.posted"
	EventStockOpnameClosed    = "stock_opname.closed"
	EventStockOpnameRejected  = "stock_opname.rejected"
	EventStockOpnameRecount   = "stock_opname.needs_recount"
	EventStockOpnameCancelled = "stock_opname.cancelled"
)

// CreateSession creates a stock opname session that spans one or more scopes.
// Overlapping active sessions are rejected per-SKU (parallel sessions may run
// as long as they never both count the same SKU). Creation is serialised with
// an advisory lock so the overlap check is race-free.
func (s *Service) CreateSession(ctx context.Context, req *CreateSessionRequest, userID int) (*Session, error) {
	scopes := normalizeScopes(req)
	if len(scopes) == 0 {
		return nil, ErrNoScopes
	}
	hasLocationScope := false
	for _, sc := range scopes {
		if !validScopes[sc.ScopeType] {
			return nil, ErrUnsupportedScope
		}
		if sc.ScopeID <= 0 && sc.ScopeType != "manual" {
			return nil, ErrScopeIDRequired
		}
		if sc.ScopeType == "location" {
			hasLocationScope = true
		}
	}
	if hasLocationScope && len(scopes) != 1 {
		return nil, ErrLocationScopeSingle
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.repo.AcquireCreateLock(ctx, tx); err != nil {
		return nil, err
	}

	// Validate location scopes up front so an unknown or inactive location
	// yields a clear 4xx instead of being masked by an empty candidate
	// universe (ErrNoItems) when the rack carries no products. Warehouse/store
	// resolution below reuses these lookups instead of re-querying the scope.
	type locationScope struct {
		warehouseID *int
		storeID     *int
	}
	locationScopes := make(map[int64]locationScope)
	for _, sc := range scopes {
		if sc.ScopeType != "location" {
			continue
		}
		wid, sid, err := s.repo.GetLocationScope(ctx, tx, int(sc.ScopeID))
		if err != nil {
			return nil, err
		}
		locationScopes[sc.ScopeID] = locationScope{warehouseID: wid, storeID: sid}
	}

	// Resolve scope display names and build the candidate product universe.
	sessionScopes := make([]SessionScope, 0, len(scopes))
	candidate := make(map[int]bool)
	for _, sc := range scopes {
		name := sc.ScopeName
		if name == "" {
			n, err := s.repo.ResolveScopeName(ctx, tx, sc.ScopeType, sc.ScopeID)
			if err != nil {
				return nil, err
			}
			name = n
		}
		sessionScopes = append(sessionScopes, SessionScope{ScopeType: sc.ScopeType, ScopeID: sc.ScopeID, ScopeName: name})
		ids, err := s.repo.ScopeProductIDs(ctx, tx, sc)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			candidate[id] = true
		}
	}
	if len(candidate) == 0 {
		return nil, ErrNoItems
	}

	// Enforce the per-SKU overlap rule against other in-progress sessions.
	active, err := s.repo.ListActiveSessions(ctx, tx)
	if err != nil {
		return nil, err
	}
	var conflictIDs []int
	seen := make(map[int]bool)
	for _, sess := range active {
		otherScopes, err := s.repo.LoadSessionScopes(ctx, tx, sess.ID)
		if err != nil {
			return nil, err
		}
		for _, osc := range otherScopes {
			ids, err := s.repo.ScopeProductIDs(ctx, tx, Scope{ScopeType: osc.ScopeType, ScopeID: osc.ScopeID})
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				if candidate[id] && !seen[id] {
					seen[id] = true
					conflictIDs = append(conflictIDs, id)
				}
			}
		}
	}
	if len(conflictIDs) > 0 {
		skus, err := s.repo.GetProductSKUs(ctx, conflictIDs)
		if err != nil {
			return nil, err
		}
		return nil, &ScopeOverlapError{SKUs: skuValues(skus, conflictIDs)}
	}

	var storeID *int
	if req.StoreID != nil {
		storeID = req.StoreID
	}
	var warehouseID = req.WarehouseID
	var locationID *int
	for _, sc := range scopes {
		switch sc.ScopeType {
		case "store":
			sid := int(sc.ScopeID)
			storeID = &sid
		case "warehouse":
			if warehouseID == nil {
				wid := int(sc.ScopeID)
				warehouseID = &wid
			}
			if storeID == nil {
				sid, err := s.repo.GetWarehouseStoreID(ctx, int(sc.ScopeID))
				if err != nil {
					return nil, err
				}
				storeID = sid
			}
		case "location":
			lid := int(sc.ScopeID)
			locationID = &lid
			if loc, ok := locationScopes[sc.ScopeID]; ok {
				warehouseID = loc.warehouseID
				storeID = loc.storeID
			} else {
				wid, sid, err := s.repo.GetLocationScope(ctx, tx, lid)
				if err != nil {
					return nil, err
				}
				warehouseID = wid
				storeID = sid
			}
		}
	}

	var items []SessionItem
	if locationID != nil {
		items, err = s.repo.LoadSnapshotProductsByLocation(ctx, tx, *locationID, productIDList(candidate))
	} else {
		items, err = s.repo.LoadSnapshotProductsByIDs(ctx, tx, productIDList(candidate))
	}
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNoItems
	}

	// Generate the session number only after every validation above has passed
	// so a failed scope/snapshot does not burn an so_seq value.
	number, err := s.repo.GetNextSessionNumber(ctx)
	if err != nil {
		return nil, err
	}

	primary := sessionScopes[0]
	session := &Session{
		SessionNumber: number,
		Title:         req.Title,
		ScopeType:     primary.ScopeType,
		ScopeID:       primary.ScopeID,
		ScopeName:     primary.ScopeName,
		Scopes:        sessionScopes,
		WarehouseID:   warehouseID,
		StoreID:       storeID,
		LocationID:    locationID,
		BlindCount:    req.BlindCount,
		Notes:         req.Notes,
		Status:        StatusDraft,
		CreatedBy:     userID,
	}
	if err := s.repo.CreateSession(ctx, tx, session); err != nil {
		return nil, err
	}
	if err := s.repo.InsertSessionScopes(ctx, tx, session.ID, sessionScopes); err != nil {
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

func normalizeScopes(req *CreateSessionRequest) []Scope {
	if len(req.Scopes) > 0 {
		return req.Scopes
	}
	if req.ScopeType == "" {
		return nil
	}
	return []Scope{{ScopeType: req.ScopeType, ScopeID: req.ScopeID, ScopeName: ""}}
}

func productIDList(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out
}

func skuValues(skus map[int]string, ids []int) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if sku := skus[id]; sku != "" {
			out = append(out, sku)
		}
	}
	return out
}

// ScopeOverlapError reports which SKUs cannot be counted because they are
// already part of another in-progress session.
type ScopeOverlapError struct {
	SKUs []string
}

func (e *ScopeOverlapError) Error() string {
	return fmt.Sprintf("%v: %s", ErrScopeOverlap, strings.Join(e.SKUs, ", "))
}

func (e *ScopeOverlapError) Unwrap() error { return ErrScopeOverlap }

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

// OpenSession moves a draft session into 'open', signalling the cycle count is
// ready for counting (Draft --Open--> Open).
func (s *Service) OpenSession(ctx context.Context, id, userID int, comment string) error {
	if strings.TrimSpace(comment) == "" {
		return ErrOpenCommentReq
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.repo.MarkSessionOpened(ctx, tx, id, userID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameOpened, id, StatusOpen)
	return nil
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
	if err := s.repo.UpdateStatus(ctx, id, StatusCounting, StatusVerification); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameSubmitted, id, StatusVerification)
	return nil
}

// StartCounting transitions a draft or open session into the counting state.
// Only an assigned counter can start counting.
func (s *Service) StartCounting(ctx context.Context, id, userID int) error {
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusDraft && status != StatusOpen {
		return ErrInvalidState
	}
	counter, err := s.repo.IsCounterAssigned(ctx, id, userID)
	if err != nil {
		return err
	}
	if !counter {
		return ErrNotAssigned
	}
	return s.repo.UpdateStatus(ctx, id, status, StatusCounting)
}

// VerifySession approves a submitted count without applying it to stock: the
// live-stock snapshot is computed (expected/difference preview persisted) and
// the session moves to 'approved'. Posting is a separate, later step.
func (s *Service) VerifySession(ctx context.Context, id, userID int, comment string) error {
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
	if session.Status != StatusVerification {
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
	var stock map[int]int
	if session.LocationID != nil {
		stock, err = s.repo.LockStockForLocation(ctx, tx, productIDs, *session.LocationID)
	} else {
		stock, err = s.repo.LockStockForProducts(ctx, tx, productIDs)
	}
	if err != nil {
		return err
	}

	var totalDiff, totalAdj float64
	for _, it := range items {
		expected := float64(stock[it.ProductID])
		diff := it.PhysicalQy - expected
		adj := diff
		reason := fmt.Sprintf("Stock opname %s: physical %.2f vs expected %.2f", session.SessionNumber, it.PhysicalQy, expected)
		if err := s.repo.UpdateItemAdjustment(ctx, tx, it.ID, expected, diff, adj, reason); err != nil {
			return err
		}
		totalDiff += diff
		totalAdj += adj
	}
	if err := s.repo.MarkSessionTotals(ctx, tx, id, totalDiff, totalAdj); err != nil {
		return err
	}
	if err := s.repo.MarkSessionVerified(ctx, tx, id, userID); err != nil {
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
	if status != StatusVerification {
		return ErrInvalidState
	}
	if err := s.repo.UpdateStatus(ctx, id, StatusVerification, StatusNeedsRecount); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameRejected, id, StatusNeedsRecount)
	return nil
}

func (s *Service) RequestRecount(ctx context.Context, id, userID int, comment string) error {
	if strings.TrimSpace(comment) == "" {
		return ErrApprovalCommentReq
	}
	status, err := s.repo.GetSessionStatus(ctx, id)
	if err != nil {
		return err
	}
	if status != StatusVerification {
		return ErrInvalidState
	}
	if err := s.repo.InsertRecountRequest(ctx, id, userID, comment); err != nil {
		return err
	}
	if err := s.repo.UpdateStatus(ctx, id, StatusVerification, StatusNeedsRecount); err != nil {
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

// PostAdjustment applies an approved session's differences to live stock,
// records an inventory adjustment document (own number from ia_seq) and emits
// inventory movements. Expected quantities are recomputed from live stock at
// posting time so the ledger always reflects reality (BR: expected = live at
// posting).
func (s *Service) PostAdjustment(ctx context.Context, id, userID int, req *PostAdjustmentRequest) (*Adjustment, error) {
	if s.stockApplier == nil {
		return nil, errors.New("stock opname service: stock applier not wired; call SetStockApplier")
	}
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := s.repo.LockSessionForApproval(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if session.Status != StatusApproved {
		return nil, ErrInvalidState
	}
	counter, err := s.repo.IsCounterAssigned(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if counter {
		return nil, ErrSeparationOfDuties
	}

	items, err := s.repo.LoadApprovalItems(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	productIDs := make([]int, len(items))
	for i, it := range items {
		productIDs[i] = it.ProductID
	}
	var stock map[int]int
	if session.LocationID != nil {
		stock, err = s.repo.LockStockForLocation(ctx, tx, productIDs, *session.LocationID)
	} else {
		stock, err = s.repo.LockStockForProducts(ctx, tx, productIDs)
	}
	if err != nil {
		return nil, err
	}

	adjItems := make([]AdjustmentItem, 0, len(items))
	movements := make([]movementRow, 0, len(items))
	var totalDiff, totalAdj float64
	for _, it := range items {
		expected := float64(stock[it.ProductID])
		diff := it.PhysicalQy - expected
		adj := diff
		reason := fmt.Sprintf("Stock opname %s: physical %.2f vs expected %.2f", session.SessionNumber, it.PhysicalQy, expected)
		if err := s.repo.UpdateItemAdjustment(ctx, tx, it.ID, expected, diff, adj, reason); err != nil {
			return nil, err
		}
		totalDiff += diff
		totalAdj += adj
		adjItems = append(adjItems, AdjustmentItem{
			ProductID:     it.ProductID,
			ExpectedQty:   expected,
			PhysicalQty:   it.PhysicalQy,
			DifferenceQty: diff,
			AdjustmentQty: adj,
			UnitCost:      it.UnitCost,
			LineTotal:     adj * it.UnitCost,
			Reason:        reason,
		})
		if diff == 0 {
			continue
		}
		delta := int(math.Round(diff))
		if session.LocationID != nil {
			err = s.stockApplier.ReconcileLocationStock(ctx, tx, shared.LocationStockReconcile{
				ProductID:   it.ProductID,
				LocationID:  *session.LocationID,
				WarehouseID: session.WarehouseID,
				StoreID:     session.StoreID,
				Delta:       delta,
			})
		} else {
			err = s.stockApplier.SetProductStock(ctx, tx, shared.StockSetItem{
				ProductID: it.ProductID,
				Quantity:  int(math.Round(expected + diff)),
			})
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrAdjustmentFailed, err)
		}
		movements = append(movements, movementRow{ProductID: it.ProductID, QuantityChange: delta, Notes: reason})
	}

	// A zero-discrepancy count still posts an adjustment document recording the
	// verified quantities (all differences zero, no stock movements) so the
	// session advances to 'posted' and can be closed.
	if err := s.repo.InsertMovements(ctx, tx, id, userID, movements); err != nil {
		return nil, err
	}

	number, err := s.repo.GetNextAdjustmentNumber(ctx)
	if err != nil {
		return nil, err
	}
	adjustment := &Adjustment{
		AdjustmentNumber: number,
		SessionID:        id,
		Notes:            req.Notes,
		CreatedBy:        userID,
	}
	if err := s.repo.InsertAdjustment(ctx, tx, adjustment); err != nil {
		return nil, err
	}
	if err := s.repo.InsertAdjustmentItems(ctx, tx, adjustment.ID, adjItems); err != nil {
		return nil, err
	}
	if err := s.repo.MarkSessionTotals(ctx, tx, id, totalDiff, totalAdj); err != nil {
		return nil, err
	}
	if err := s.repo.MarkSessionPosted(ctx, tx, id, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.publishStatusEvent(ctx, EventStockOpnamePosted, id, StatusPosted)

	posted, err := s.repo.GetAdjustmentBySession(ctx, id)
	if err != nil {
		return nil, err
	}
	return posted, nil
}

// CloseSession closes a posted session. Deviations are already applied to
// stock during posting; closing only finalises the record.
func (s *Service) CloseSession(ctx context.Context, id, userID int) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := s.repo.MarkSessionClosed(ctx, tx, id, userID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.publishStatusEvent(ctx, EventStockOpnameClosed, id, StatusClosed)
	return nil
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

// ListAdjustments returns posted adjustment documents (paginated).
func (s *Service) ListAdjustments(ctx context.Context, limit, offset int, status, search string) ([]Adjustment, int, error) {
	return s.repo.ListAdjustments(ctx, limit, offset, status, search)
}

// GetAdjustment returns a single adjustment document by id.
func (s *Service) GetAdjustment(ctx context.Context, id int) (*Adjustment, error) {
	return s.repo.GetAdjustment(ctx, id)
}

func isEditableStatus(status string) bool {
	switch status {
	case StatusDraft, StatusOpen, StatusCounting, StatusNeedsRecount:
		return true
	}
	return false
}

package consignment

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

var (
	ErrStoreForbidden = errors.New("store access forbidden")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Repo exposes the repository for the checkout provider (same package).
func (s *Service) Repo() *Repository {
	return s.repo
}

func (s *Service) repoForQuery(ctx context.Context, q queryer) queryer {
	if q != nil {
		return q
	}
	return s.repo.db
}

// --- Store scoping ---

// requireStore returns the effective store scope for an operation. A nil
// claimsStore (superadmin/admin) may pass storeID nil to mean "all stores";
// otherwise the caller's store is enforced.
func resolveStore(claimsStore *int, requested *int) (*int, error) {
	if claimsStore == nil {
		return requested, nil
	}
	if requested != nil && *requested != *claimsStore {
		return nil, ErrStoreForbidden
	}
	return claimsStore, nil
}

// checkArrangementStore verifies an arrangement belongs to the caller's store.
func checkArrangementStore(a *Arrangement, claimsStore *int) error {
	if claimsStore == nil {
		return nil
	}
	if a.StoreID != *claimsStore {
		return ErrStoreForbidden
	}
	return nil
}

// applyLazyEnded derives the lazy Ended status on a loaded arrangement, mirroring
// the read path (GetArrangement). Write paths call it so an arrangement whose
// visit is stale rejects writes exactly like reads report (BR-47/BR-48).
func applyLazyEnded(a *Arrangement) {
	if a.Status == StatusActive && isStaleVisit(a.LastVisitAt) {
		a.Status = StatusEnded
	}
}

// --- Arrangements ---

// CreateArrangement opens a consignment partnership for a consignment-flagged
// supplier. Only one active arrangement may exist per supplier+store.
func (s *Service) CreateArrangement(ctx context.Context, req *CreateArrangementRequest, userID int, claimsStore *int) (*Arrangement, error) {
	storeID, err := resolveStore(claimsStore, &req.StoreID)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	isConsignment, err := s.repo.supplierStoreOrPanic().IsConsignmentSupplier(ctx, s.repo.db, req.SupplierID)
	if err != nil {
		return nil, err
	}
	if !isConsignment {
		return nil, ErrNotConsignmentSupplier
	}

	existing, err := s.repo.GetActiveArrangement(ctx, tx, req.SupplierID, *storeID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrActiveArrangementExists
	}

	a := &Arrangement{
		SupplierID: req.SupplierID,
		StoreID:    *storeID,
		Status:     StatusActive,
		CreatedBy:  userID,
	}
	if err := s.repo.InsertArrangement(ctx, tx, a); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) ListArrangements(ctx context.Context, claimsStore *int, limit, offset int, search, status string) ([]Arrangement, int, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListArrangements(ctx, s.repo.db, storeID, limit, offset, search, status)
}

func (s *Service) GetArrangement(ctx context.Context, id int, claimsStore *int) (*Arrangement, error) {
	a, err := s.repo.GetArrangementByID(ctx, s.repo.db, id)
	if err != nil {
		return nil, err
	}
	if err := checkArrangementStore(a, claimsStore); err != nil {
		return nil, err
	}
	terms, err := s.repo.ListTerms(ctx, s.repo.db, id)
	if err != nil {
		return nil, err
	}
	a.Terms = terms
	if a.Status == StatusActive && isStaleVisit(a.LastVisitAt) {
		a.Status = StatusEnded
	}
	return a, nil
}

// SetTerms replaces the pricing/commission terms of an arrangement. All
// products must be owned by the arrangement's supplier (BR-02/03 guard).
func (s *Service) SetTerms(ctx context.Context, arrangementID int, reqs []SetTermsRequest, userID int, claimsStore *int) ([]Term, error) {
	a, err := s.repo.GetArrangementByID(ctx, s.repo.db, arrangementID)
	if err != nil {
		return nil, err
	}
	if err := checkArrangementStore(a, claimsStore); err != nil {
		return nil, err
	}
	applyLazyEnded(a)
	if a.Status == StatusEnded {
		return nil, ErrArrangementEnded
	}

	terms := make([]Term, 0, len(reqs))
	for _, r := range reqs {
		if err := validateShare(r.StoreShareType, r.StoreShareValue); err != nil {
			return nil, err
		}
		terms = append(terms, Term{
			ProductID:       r.ProductID,
			Price:           r.Price,
			StoreShareType:  r.StoreShareType,
			StoreShareValue: r.StoreShareValue,
			CreatedBy:       userID,
		})
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.repo.ReplaceTerms(ctx, tx, arrangementID, terms); err != nil {
		return nil, err
	}
	if err := s.repo.TouchVisit(ctx, tx, arrangementID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListTerms(ctx, s.repo.db, arrangementID)
}

// --- Receipts ---

// CreateReceipt records accepted consignment goods after inspection and adds
// them to the consignment ownership ledger plus the global product_stock
// (through the inventory port). BR-02/BR-03/BR-05b ownership conflicts are
// rejected atomically inside the transaction.
func (s *Service) CreateReceipt(ctx context.Context, req *ReceiptRequest, userID int, claimsStore *int) (*Receipt, error) {
	if len(req.Items) == 0 {
		return nil, ErrInvalidQty
	}
	a, err := s.repo.GetArrangementByID(ctx, s.repo.db, req.ArrangementID)
	if err != nil {
		return nil, err
	}
	if err := checkArrangementStore(a, claimsStore); err != nil {
		return nil, err
	}
	if a.Status == StatusEnded {
		return nil, ErrArrangementEnded
	}

	productIDs := make([]int, 0, len(req.Items))
	for _, it := range req.Items {
		if it.AcceptedQty <= 0 {
			return nil, ErrInvalidQty
		}
		productIDs = append(productIDs, it.ProductID)
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ownership conflict resolution: each product must be free to be consigned
	// under this supplier (BR-02/03/05b). Locks the ledger row so concurrent
	// receipts serialize.
	for _, pid := range productIDs {
		if err := s.resolveOwnership(ctx, tx, a, pid); err != nil {
			return nil, err
		}
	}

	receiptNumber, err := s.repo.NextReceiptNumber(ctx, tx)
	if err != nil {
		return nil, err
	}
	rec := &Receipt{
		ReceiptNumber: receiptNumber,
		SupplierID:    a.SupplierID,
		StoreID:       a.StoreID,
		ArrangementID: a.ID,
		ReceivedBy:    userID,
		Notes:         req.Notes,
	}
	if err := s.repo.InsertReceipt(ctx, tx, rec); err != nil {
		return nil, err
	}

	for _, r := range req.Items {
		term, err := s.repo.GetTermByProduct(ctx, tx, a.ID, r.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %d has no consignment term: %w", r.ProductID, err)
		}
		item := &ReceiptItem{
			ConsignmentReceiptID: rec.ID,
			ProductID:            r.ProductID,
			AcceptedQty:          r.AcceptedQty,
			Price:                term.Price,
			StoreShareType:       term.StoreShareType,
			StoreShareValue:      term.StoreShareValue,
			Notes:                r.Notes,
		}
		if err := s.repo.InsertReceiptItem(ctx, tx, item); err != nil {
			return nil, err
		}

		// Ownership ledger + global product_stock (through inventory port).
		if _, err := s.repo.UpsertConsignmentStock(ctx, tx, r.ProductID, a.SupplierID, a.ID, a.StoreID, r.AcceptedQty); err != nil {
			return nil, err
		}
		if err := s.repo.stockAdjusterOrPanic().ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
			ProductID:      r.ProductID,
			Delta:          r.AcceptedQty,
			MovementType:   MovementTypeConsignmentReceipt,
			ReferenceID:    rec.ID,
			ReferenceTable: "consignment_receipts",
			UserID:         userID,
			Notes:          "consignment receipt " + receiptNumber,
		}); err != nil {
			return nil, err
		}
	}

	if err := s.repo.TouchVisit(ctx, tx, a.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetReceiptByID(ctx, s.repo.db, rec.ID)
}

// resolveOwnership enforces the SKU ownership rules against the ledger.
//   - no ledger row: reject if the product still has store-owned stock
//     (BR-02), else free to consign under this supplier.
//   - row owned by another supplier: reject (BR-03); a fully released row
//     (available=0, pending_return=0) is re-ownable by a new supplier.
//   - row owned by this supplier: allowed (top-up, BR-05).
func (s *Service) resolveOwnership(ctx context.Context, tx pgx.Tx, a *Arrangement, productID int) error {
	row, err := s.repo.LockConsignmentStock(ctx, tx, productID)
	if err != nil {
		return err
	}
	if row == nil {
		// No ledger row: reject when the product still carries store-owned
		// stock in the global product_stock bucket.
		hasStoreStock, err := s.hasStoreOwnedStock(ctx, productID)
		if err != nil {
			return err
		}
		if hasStoreStock {
			return ErrConflictStoreStock
		}
		return nil
	}
	if row.SupplierID != a.SupplierID {
		if row.AvailableQty == 0 && row.PendingReturnQty == 0 {
			// Ownership released; the new supplier takes over the row.
			return nil
		}
		if row.PendingReturnQty > 0 {
			return ErrPendingReturnBlocksTransfer
		}
		return ErrConflictOtherSupplier
	}
	return nil
}

// hasStoreOwnedStock reports whether a product still has store-owned stock in
// the global product_stock bucket. Consignment receipts also write the global
// bucket, but a ledger row already exists in that case, so a non-nil row here
// with no ledger row is store-owned.
func (s *Service) hasStoreOwnedStock(ctx context.Context, productID int) (bool, error) {
	var qty int
	err := s.repo.db.QueryRow(ctx, `
		SELECT COALESCE(quantity, 0)
		FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
	`, productID).Scan(&qty)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return qty > 0, nil
}

func (s *Service) GetReceipt(ctx context.Context, id int, claimsStore *int) (*Receipt, error) {
	rec, err := s.repo.GetReceiptByID(ctx, s.repo.db, id)
	if err != nil {
		return nil, err
	}
	if claimsStore != nil && rec.StoreID != *claimsStore {
		return nil, ErrStoreForbidden
	}
	return rec, nil
}

func (s *Service) ListReceipts(ctx context.Context, supplierID int, claimsStore *int) ([]Receipt, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	if storeID == nil {
		return nil, ErrStoreForbidden
	}
	return s.repo.ListReceipts(ctx, s.repo.db, supplierID, *storeID)
}

// --- Consignment stock ---

func (s *Service) ListStock(ctx context.Context, supplierID int, claimsStore *int) ([]StockRow, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	var sid *int
	if supplierID > 0 {
		sid = &supplierID
	}
	return s.repo.ListConsignmentStock(ctx, s.repo.db, sid, storeID)
}

// ListSuppliers returns all suppliers flagged as consignment.
func (s *Service) ListSuppliers(ctx context.Context) ([]shared.SupplierRef, error) {
	return s.repo.supplierStoreOrPanic().ListConsignmentSuppliers(ctx, s.repo.db)
}

// --- Pending returns ---

// CreatePendingReturn pulls qty out of the sellable (available) pool and marks
// it for return. The goods leave product_stock so they are no longer sellable
// (BR-26/AC-C20).
func (s *Service) CreatePendingReturn(ctx context.Context, req *CreatePendingReturnRequest, userID int, claimsStore *int) (*PendingReturn, error) {
	if req.Qty <= 0 {
		return nil, ErrInvalidQty
	}
	if !validPendingReturnReason(req.Reason) {
		return nil, ErrInvalidReason
	}

	row, err := s.repo.GetConsignmentStock(ctx, s.repo.db, req.ProductID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrConsignmentNotFound
	}
	if claimsStore != nil && row.StoreID != *claimsStore {
		return nil, ErrStoreForbidden
	}
	a, err := s.repo.GetArrangementByID(ctx, s.repo.db, row.ArrangementID)
	if err != nil {
		return nil, err
	}
	if a.Status == StatusEnded {
		return nil, ErrArrangementEnded
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	locked, err := s.repo.LockConsignmentStock(ctx, tx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if locked == nil {
		return nil, ErrConsignmentNotFound
	}
	if locked.AvailableQty < req.Qty {
		return nil, ErrInsufficientConsignmentStock
	}
	if err := s.repo.MoveToPendingReturn(ctx, tx, req.ProductID, req.Qty); err != nil {
		return nil, err
	}

	pr := &PendingReturn{
		SupplierID:    row.SupplierID,
		ProductID:     req.ProductID,
		ArrangementID: row.ArrangementID,
		StoreID:       row.StoreID,
		Qty:           req.Qty,
		Reason:        req.Reason,
		Notes:         req.Notes,
		CreatedBy:     userID,
	}
	if err := s.repo.InsertPendingReturn(ctx, tx, pr); err != nil {
		return nil, err
	}

	// Remove from the sellable product_stock bucket (BR-26).
	if err := s.repo.stockAdjusterOrPanic().ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
		ProductID:      req.ProductID,
		Delta:          -req.Qty,
		MovementType:   MovementTypeConsignmentPendingReturn,
		ReferenceID:    pr.ID,
		ReferenceTable: "consignment_pending_returns",
		UserID:         userID,
		Notes:          "pending return " + pr.Reason,
	}); err != nil {
		return nil, err
	}

	if err := s.repo.TouchVisit(ctx, tx, row.ArrangementID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return pr, nil
}

func (s *Service) ListPendingReturns(ctx context.Context, supplierID int, claimsStore *int) ([]PendingReturn, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	if storeID == nil {
		return nil, ErrStoreForbidden
	}
	return s.repo.ListOpenPendingReturns(ctx, s.repo.db, supplierID, *storeID)
}

// --- Returns ---

// CreateReturn formally hands goods back to the supplier. Items referencing an
// open pending return reduce pending_return; free items reduce available. Both
// paths also reduce the global product_stock (AC-C23). Ownership is released
// once available and pending return both reach zero.
func (s *Service) CreateReturn(ctx context.Context, req *ReturnRequest, userID int, claimsStore *int) (*ConsignmentReturn, error) {
	if len(req.Items) == 0 {
		return nil, ErrInvalidQty
	}
	a, err := s.repo.GetArrangementByID(ctx, s.repo.db, req.ArrangementID)
	if err != nil {
		return nil, err
	}
	if err := checkArrangementStore(a, claimsStore); err != nil {
		return nil, err
	}
	if a.Status == StatusEnded {
		return nil, ErrArrangementEnded
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	returnNumber, err := s.repo.NextReturnNumber(ctx, tx)
	if err != nil {
		return nil, err
	}
	ret := &ConsignmentReturn{
		ReturnNumber:  returnNumber,
		SupplierID:    a.SupplierID,
		StoreID:       a.StoreID,
		ArrangementID: a.ID,
		ReturnedBy:    userID,
		Notes:         req.Notes,
	}
	if err := s.repo.InsertReturn(ctx, tx, ret); err != nil {
		return nil, err
	}

	for _, r := range req.Items {
		if r.Qty <= 0 {
			return nil, ErrInvalidQty
		}
		if !validPendingReturnReason(r.Reason) {
			return nil, ErrInvalidReason
		}
		row, err := s.repo.LockConsignmentStock(ctx, tx, r.ProductID)
		if err != nil {
			return nil, err
		}
		if row == nil || row.SupplierID != a.SupplierID {
			return nil, ErrConflictOtherSupplier
		}
		item := &ReturnItem{
			ConsignmentReturnID: ret.ID,
			ProductID:           r.ProductID,
			Qty:                 r.Qty,
			Reason:              r.Reason,
			PendingReturnID:     r.PendingReturnID,
			Notes:               r.Notes,
		}
		if err := s.repo.InsertReturnItem(ctx, tx, item); err != nil {
			return nil, err
		}

		if r.PendingReturnID != nil {
			// Return an already-pulled pending return: pending_return - qty.
			// The goods already left the sellable product_stock when the
			// pending return was created (BR-26/AC-C20), so the formal return
			// must NOT subtract the global bucket again.
			pr, err := s.repo.GetPendingReturnByID(ctx, tx, *r.PendingReturnID)
			if err != nil {
				return nil, err
			}
			if pr.Status != PendingReturnOpen || pr.ProductID != r.ProductID || pr.Qty < r.Qty {
				return nil, ErrPendingReturnNotFound
			}
			if err := s.repo.ResolvePendingReturn(ctx, tx, r.ProductID, r.Qty); err != nil {
				return nil, err
			}
			if r.Qty == pr.Qty {
				if err := s.repo.MarkPendingReturnReturned(ctx, tx, *r.PendingReturnID); err != nil {
					return nil, err
				}
			} else {
				// Partial resolution (AC-C25): keep the pending return open with
				// the leftover qty so it can be returned later, instead of
				// orphaning the remainder behind a premature 'returned' status.
				if err := s.repo.ReducePendingReturnQty(ctx, tx, *r.PendingReturnID, r.Qty); err != nil {
					return nil, err
				}
			}
		} else {
			if err := s.repo.ReduceAvailable(ctx, tx, r.ProductID, r.Qty); err != nil {
				return nil, err
			}

			// Remove freely returned goods from the sellable product_stock
			// (AC-C23). Pending-return-linked items are skipped because their
			// stock already left the global bucket at pending-return time.
			movementType := MovementTypeConsignmentReturn
			if r.Reason == ReasonCustomerReturn {
				movementType = MovementTypeConsignmentCustomerReturn
			}
			if err := s.repo.stockAdjusterOrPanic().ApplyConsignmentDelta(ctx, tx, shared.ConsignmentStockDelta{
				ProductID:      r.ProductID,
				Delta:          -r.Qty,
				MovementType:   movementType,
				ReferenceID:    ret.ID,
				ReferenceTable: "consignment_returns",
				UserID:         userID,
				Notes:          "consignment return " + returnNumber,
			}); err != nil {
				return nil, err
			}
		}

		if err := s.repo.ReleaseOwnership(ctx, tx, r.ProductID); err != nil {
			return nil, err
		}
	}

	if err := s.repo.TouchVisit(ctx, tx, a.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetReturnByID(ctx, s.repo.db, ret.ID)
}

func (s *Service) GetReturn(ctx context.Context, id int, claimsStore *int) (*ConsignmentReturn, error) {
	ret, err := s.repo.GetReturnByID(ctx, s.repo.db, id)
	if err != nil {
		return nil, err
	}
	if claimsStore != nil && ret.StoreID != *claimsStore {
		return nil, ErrStoreForbidden
	}
	return ret, nil
}

func (s *Service) ListReturns(ctx context.Context, supplierID int, claimsStore *int) ([]ConsignmentReturn, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	if storeID == nil {
		return nil, ErrStoreForbidden
	}
	return s.repo.ListReturns(ctx, s.repo.db, supplierID, *storeID)
}

// --- Settlements ---

// GetSettlementPreview computes the full settlement of all unsettled
// consignment sales for a supplier/store (BR-41, partial settlement never
// allowed).
func (s *Service) GetSettlementPreview(ctx context.Context, supplierID int, claimsStore *int) (*Settlement, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	if storeID == nil {
		return nil, ErrStoreForbidden
	}
	a, err := s.repo.GetActiveArrangement(ctx, s.repo.db, supplierID, *storeID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrConsignmentNotFound
	}

	items, err := s.repo.ListUnsettledSaleItems(ctx, s.repo.db, supplierID, *storeID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrEmptySettlement
	}
	return buildSettlementPreview(supplierID, *storeID, items), nil
}

// CreateSettlement finalizes a full settlement for the supplier's unsettled
// consignment sales, linking every sale item to the new settlement.
func (s *Service) CreateSettlement(ctx context.Context, req *CreateSettlementRequest, userID int, claimsStore *int) (*Settlement, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	if storeID == nil {
		return nil, ErrStoreForbidden
	}
	a, err := s.repo.GetActiveArrangement(ctx, s.repo.db, req.SupplierID, *storeID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrConsignmentNotFound
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	items, err := s.repo.ListUnsettledSaleItems(ctx, tx, req.SupplierID, *storeID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrEmptySettlement
	}
	preview := buildSettlementPreview(req.SupplierID, *storeID, items)

	number, err := s.repo.NextSettlementNumber(ctx, tx)
	if err != nil {
		return nil, err
	}
	settlement := &Settlement{
		SettlementNumber: number,
		SupplierID:       req.SupplierID,
		StoreID:          *storeID,
		TotalSaleValue:   preview.TotalSaleValue,
		TotalStoreShare:  preview.TotalStoreShare,
		TotalPayable:     preview.TotalPayable,
		Status:           SettlementPendingPayment,
		CreatedBy:        userID,
	}
	if err := s.repo.InsertSettlement(ctx, tx, settlement); err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
		if err := s.repo.InsertSettlementItem(ctx, tx, &SettlementItem{
			ConsignmentSettlementID: settlement.ID,
			ConsignmentSaleItemID:   it.ID,
			ProductID:               it.ProductID,
			Quantity:                it.Quantity,
			UnitPrice:               it.UnitPrice,
			Subtotal:                it.Subtotal,
			StoreShare:              it.StoreShareAmount,
		}); err != nil {
			return nil, err
		}
	}
	if err := s.repo.MarkSaleItemsSettled(ctx, tx, ids, settlement.ID); err != nil {
		return nil, err
	}

	if err := s.repo.TouchVisit(ctx, tx, a.ID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetSettlementByID(ctx, s.repo.db, settlement.ID)
}

func (s *Service) GetSettlement(ctx context.Context, id int, claimsStore *int) (*Settlement, error) {
	st, err := s.repo.GetSettlementByID(ctx, s.repo.db, id)
	if err != nil {
		return nil, err
	}
	if claimsStore != nil && st.StoreID != *claimsStore {
		return nil, ErrStoreForbidden
	}
	return st, nil
}

func (s *Service) ListSettlements(ctx context.Context, supplierID int, claimsStore *int) ([]Settlement, error) {
	storeID, err := resolveStore(claimsStore, nil)
	if err != nil {
		return nil, err
	}
	if storeID == nil {
		return nil, ErrStoreForbidden
	}
	return s.repo.ListSettlements(ctx, s.repo.db, supplierID, *storeID)
}

// ListPaymentMethods returns the active payment methods for the payout picker.
func (s *Service) ListPaymentMethods(ctx context.Context) ([]PaymentMethod, error) {
	return s.repo.paymentMethodsOrPanic().ActivePaymentMethods(ctx)
}

// CreatePayout records a money-out to the supplier and marks the settlement
// paid once the full payable is covered. A settlement may receive multiple
// payouts but only while pending_payment.
func (s *Service) CreatePayout(ctx context.Context, settlementID int, req *CreatePayoutRequest, userID int, claimsStore *int) (*Payout, error) {
	st, err := s.repo.GetSettlementByID(ctx, s.repo.db, settlementID)
	if err != nil {
		return nil, err
	}
	if claimsStore != nil && st.StoreID != *claimsStore {
		return nil, ErrStoreForbidden
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := s.repo.GetSettlementByIDQuery(ctx, tx, settlementID)
	if err != nil {
		return nil, err
	}
	if current.Status != SettlementPendingPayment {
		return nil, ErrSettlementAlreadyPaid
	}
	paidSoFar, err := s.repo.SumPayoutsBySettlement(ctx, tx, settlementID)
	if err != nil {
		return nil, err
	}
	if req.Amount <= 0 || req.Amount > current.TotalPayable-paidSoFar {
		return nil, ErrInvalidPayoutAmount
	}

	pm, err := s.repo.paymentMethodsOrPanic().PaymentMethodByID(ctx, req.PaymentMethodID)
	if err != nil || pm == nil {
		return nil, ErrPaymentMethodNotFound
	}

	number, err := s.repo.NextPayoutNumber(ctx, tx)
	if err != nil {
		return nil, err
	}
	payout := &Payout{
		PayoutNumber:    number,
		SettlementID:    settlementID,
		PaymentMethodID: req.PaymentMethodID,
		Amount:          req.Amount,
		ReferenceNumber: req.ReferenceNumber,
		PaidBy:          userID,
		Notes:           req.Notes,
	}
	if err := s.repo.InsertPayout(ctx, tx, payout); err != nil {
		return nil, err
	}

	// If the settlement is now fully paid, close it.
	if paidSoFar+req.Amount >= current.TotalPayable {
		if err := s.repo.MarkSettlementPaid(ctx, tx, settlementID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return payout, nil
}

func buildSettlementPreview(supplierID, storeID int, items []SaleItemRecord) *Settlement {
	settlement := &Settlement{
		SupplierID: supplierID,
		StoreID:    storeID,
		Status:     SettlementPendingPayment,
	}
	settlement.Items = make([]SettlementItem, 0, len(items))
	for _, it := range items {
		settlement.TotalSaleValue += it.Subtotal
		settlement.TotalStoreShare += it.StoreShareAmount
		settlement.Items = append(settlement.Items, SettlementItem{
			ConsignmentSaleItemID: it.ID,
			ProductID:             it.ProductID,
			ProductName:           it.ProductName,
			Quantity:              it.Quantity,
			UnitPrice:             it.UnitPrice,
			Subtotal:              it.Subtotal,
			StoreShare:            it.StoreShareAmount,
		})
	}
	settlement.TotalPayable = settlement.TotalSaleValue - settlement.TotalStoreShare
	return settlement
}

func validateShare(shareType string, shareValue float64) error {
	if shareType != ShareTypePercentage && shareType != ShareTypeFixedAmount {
		return ErrInvalidShareType
	}
	if shareValue <= 0 {
		return ErrInvalidShareValue
	}
	if shareType == ShareTypePercentage && shareValue >= 100 {
		return ErrInvalidShareValueForType
	}
	return nil
}

func validPendingReturnReason(reason string) bool {
	switch reason {
	case ReasonDamaged, ReasonExpired, ReasonCustomerReturn, ReasonOther:
		return true
	}
	return false
}
package purchase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
)

const maxBatchSize = 1000

type Repo interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
	CancelPurchaseOrder(ctx context.Context, tx pgx.Tx, id, userID int, cancelledAt string) error
	ConfirmPurchaseOrder(ctx context.Context, tx pgx.Tx, id, userID int, confirmedAt string) error
	CreateGoodsReceipt(ctx context.Context, tx pgx.Tx, gr *GoodsReceipt, items []GoodsReceiptItem) error
	CreatePurchaseOrder(ctx context.Context, tx pgx.Tx, po *Order, items []OrderItem) error
	DeletePurchaseOrder(ctx context.Context, tx pgx.Tx, id int) error
	GetAllPurchaseOrders(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int, supplierIDs []int) ([]Order, int, error)
	GetNextDONumber(ctx context.Context) (string, error)
	GetNextGRNumber(ctx context.Context) (string, error)
	GetNextPONumber(ctx context.Context) (string, error)
	GetPurchaseOrderByID(ctx context.Context, id int, storeID *int) (*Order, error)
	GetReceiptsByPOID(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error)
	LockPurchaseOrderForUpdate(ctx context.Context, tx pgx.Tx, id int) error
	RecalculatePOStatus(ctx context.Context, tx pgx.Tx, poID int) error
	UpdatePOItemQtyReceived(ctx context.Context, tx pgx.Tx, poItemID, qtyReceived int) error
	UpdatePurchaseOrder(ctx context.Context, tx pgx.Tx, po *Order, items []OrderItem) error
}

type service struct {
	repo           Repo
	eventBus       shared.EventBus
	productLookup  ProductLookup
	supplierLookup SupplierLookup
}

func NewService(repo Repo, eventBus shared.EventBus) Service {
	return &service{repo: repo, eventBus: eventBus}
}

func (s *service) SetProductLookup(l ProductLookup) {
	s.productLookup = l
}

func (s *service) SetSupplierLookup(l SupplierLookup) {
	s.supplierLookup = l
}

func (s *service) lookupProductNames(ctx context.Context, ids []int) (map[int]ProductInfo, error) {
	if s.productLookup == nil {
		return nil, fmt.Errorf("product lookup port is not wired")
	}
	return s.productLookup.GetProductNamesByIDs(ctx, ids)
}

func (s *service) lookupSupplierNames(ctx context.Context, ids []int) (map[int]SupplierInfo, error) {
	if s.supplierLookup == nil {
		return nil, fmt.Errorf("supplier lookup port is not wired")
	}
	return s.supplierLookup.GetSupplierNamesByIDs(ctx, ids)
}

func (s *service) lookupSupplierIDs(ctx context.Context, name string) ([]int, error) {
	if s.supplierLookup == nil {
		return nil, fmt.Errorf("supplier lookup port is not wired")
	}
	return s.supplierLookup.GetSupplierIDsByName(ctx, name)
}

func (s *service) applySupplierNames(po *Order, names map[int]SupplierInfo) {
	if info, ok := names[po.SupplierID]; ok {
		po.SupplierName = info.Name
	}
}

func (s *service) CreateDraft(ctx context.Context, po *Order, items []OrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: items cannot be empty", ErrInvalidInput)
	}
	if po.SupplierID == 0 {
		return fmt.Errorf("%w: supplier_id is required", ErrInvalidInput)
	}

	storeID := po.StoreID
	if storeID == 0 {
		return fmt.Errorf("%w: store_id is required", ErrInvalidInput)
	}

	po.Status = StatusDraft
	po.Subtotal = 0
	po.GrandTotal = 0

	productIDs := make([]int, len(items))
	seen := make(map[int]bool)
	for i, item := range items {
		if item.ProductID == 0 {
			return fmt.Errorf("%w: product_id is required for item %d", ErrInvalidInput, i)
		}
		if item.QtyOrdered <= 0 {
			return fmt.Errorf("%w: qty_ordered must be greater than 0 for product %d", ErrInvalidInput, item.ProductID)
		}
		if item.UnitCost < 0 {
			return fmt.Errorf("%w: unit_cost cannot be negative for product %d", ErrInvalidInput, item.ProductID)
		}
		if seen[item.ProductID] {
			return ErrDuplicatePOItem
		}
		seen[item.ProductID] = true
		productIDs[i] = item.ProductID
		items[i].Subtotal = item.QtyOrdered*item.UnitCost - item.DiscountAmount
		if items[i].Subtotal < 0 {
			items[i].Subtotal = 0
		}
		po.Subtotal += items[i].Subtotal
	}

	productMap, err := s.lookupProductNames(ctx, productIDs)
	if err != nil {
		return fmt.Errorf("lookup products: %w", err)
	}
	for i, item := range items {
		if info, ok := productMap[item.ProductID]; ok {
			items[i].ProductName = info.Name
			items[i].SKU = info.SKU
		}
	}

	po.GrandTotal = po.Subtotal - po.DiscountAmount - po.TaxAmount
	if po.GrandTotal < 0 {
		po.GrandTotal = 0
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	poNumber, err := s.repo.GetNextPONumber(ctx)
	if err != nil {
		return err
	}
	po.PONumber = poNumber

	if err := s.repo.CreatePurchaseOrder(ctx, tx, po, items); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	_ = s.eventBus.Publish(ctx, events.TopicPOCreated, &events.PurchaseOrderEvent{
		POID:     po.ID,
		PONumber: po.PONumber,
		StoreID:  po.StoreID,
	})

	return nil
}

func (s *service) UpdateDraft(ctx context.Context, id int, po *Order, items []OrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: items cannot be empty", ErrInvalidInput)
	}
	if po.SupplierID == 0 {
		return fmt.Errorf("%w: supplier_id is required", ErrInvalidInput)
	}
	if po.UpdatedBy == 0 {
		return fmt.Errorf("%w: updated_by is required", ErrInvalidInput)
	}

	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return ErrPurchaseOrderNotDraft
	}

	po.Subtotal = 0
	po.GrandTotal = 0
	seen := make(map[int]bool)
	for i, item := range items {
		if item.ProductID == 0 {
			return fmt.Errorf("%w: product_id is required for item %d", ErrInvalidInput, i)
		}
		if item.QtyOrdered <= 0 {
			return fmt.Errorf("%w: qty_ordered must be greater than 0 for product %d", ErrInvalidInput, item.ProductID)
		}
		if item.UnitCost < 0 {
			return fmt.Errorf("%w: unit_cost cannot be negative for product %d", ErrInvalidInput, item.ProductID)
		}
		if seen[item.ProductID] {
			return ErrDuplicatePOItem
		}
		seen[item.ProductID] = true
		items[i].Subtotal = item.QtyOrdered*item.UnitCost - item.DiscountAmount
		if items[i].Subtotal < 0 {
			items[i].Subtotal = 0
		}
		po.Subtotal += items[i].Subtotal
	}
	po.GrandTotal = po.Subtotal - po.DiscountAmount - po.TaxAmount
	if po.GrandTotal < 0 {
		po.GrandTotal = 0
	}

	productIDs := make([]int, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}
	productMap, err := s.lookupProductNames(ctx, productIDs)
	if err != nil {
		return fmt.Errorf("lookup products: %w", err)
	}
	for i, item := range items {
		if info, ok := productMap[item.ProductID]; ok {
			items[i].ProductName = info.Name
			items[i].SKU = info.SKU
		}
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	po.ID = id
	if err := s.repo.UpdatePurchaseOrder(ctx, tx, po, items); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (s *service) DeleteDraft(ctx context.Context, id int) error {
	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return ErrPurchaseOrderNotDraft
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.repo.DeletePurchaseOrder(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// InTx runs fn inside a single transaction on the purchase database, committing
// on success and rolling back on error. Used to make a PO mutation and its audit
// log atomic.
func (s *service) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ConfirmTx confirms a purchase order within an existing transaction.
func (s *service) ConfirmTx(ctx context.Context, tx pgx.Tx, id, userID int) error {
	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return ErrPurchaseOrderNotDraft
	}

	now := time.Now().In(shared.JakartaLocation()).Format(time.RFC3339)

	if err := s.repo.LockPurchaseOrderForUpdate(ctx, tx, id); err != nil {
		return err
	}

	return s.repo.ConfirmPurchaseOrder(ctx, tx, id, userID, now)
}

// NotifyPOConfirmed publishes the PO-confirmed event after a successful commit.
func (s *service) NotifyPOConfirmed(ctx context.Context, id int) {
	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		slog.Warn("failed to reload purchase order for event publish", "po_id", id, "error", err)
		return
	}
	_ = s.eventBus.Publish(ctx, events.TopicPOConfirmed, &events.PurchaseOrderEvent{
		POID:     id,
		PONumber: existing.PONumber,
		StoreID:  existing.StoreID,
	})
}

func (s *service) Confirm(ctx context.Context, id, userID int) error {
	if err := s.InTx(ctx, func(tx pgx.Tx) error {
		return s.ConfirmTx(ctx, tx, id, userID)
	}); err != nil {
		return err
	}
	s.NotifyPOConfirmed(ctx, id)
	return nil
}

// CancelTx cancels a purchase order within an existing transaction.
func (s *service) CancelTx(ctx context.Context, tx pgx.Tx, id, userID int) error {
	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		return err
	}

	for _, item := range existing.Items {
		if item.QtyReceived > 0 {
			return ErrPurchaseOrderHasReceipts
		}
	}

	if existing.Status != StatusDraft && existing.Status != StatusConfirmed {
		return ErrPurchaseOrderCancelled
	}

	now := time.Now().In(shared.JakartaLocation()).Format(time.RFC3339)

	if err := s.repo.LockPurchaseOrderForUpdate(ctx, tx, id); err != nil {
		return err
	}

	return s.repo.CancelPurchaseOrder(ctx, tx, id, userID, now)
}

// NotifyPOCancelled publishes the PO-cancelled event after a successful commit.
func (s *service) NotifyPOCancelled(ctx context.Context, id int) {
	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		slog.Warn("failed to reload purchase order for event publish", "po_id", id, "error", err)
		return
	}
	_ = s.eventBus.Publish(ctx, events.TopicPOCancelled, &events.PurchaseOrderEvent{
		POID:     id,
		PONumber: existing.PONumber,
		StoreID:  existing.StoreID,
	})
}

func (s *service) Cancel(ctx context.Context, id, userID int) error {
	if err := s.InTx(ctx, func(tx pgx.Tx) error {
		return s.CancelTx(ctx, tx, id, userID)
	}); err != nil {
		return err
	}
	s.NotifyPOCancelled(ctx, id)
	return nil
}

func (s *service) GetDetail(ctx context.Context, id int, storeID *int) (*Order, error) {
	po, err := s.repo.GetPurchaseOrderByID(ctx, id, storeID)
	if err != nil {
		return nil, err
	}
	names, err := s.lookupSupplierNames(ctx, []int{po.SupplierID})
	if err != nil {
		return nil, err
	}
	s.applySupplierNames(po, names)
	return po, nil
}

func (s *service) List(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int) ([]Order, int, error) {
	var supplierIDs []int
	if search != "" {
		ids, err := s.lookupSupplierIDs(ctx, search)
		if err != nil {
			return nil, 0, err
		}
		supplierIDs = ids
	}

	pos, total, err := s.repo.GetAllPurchaseOrders(ctx, limit, offset, search, sortBy, sortDir, status, supplierID, startDate, endDate, storeID, supplierIDs)
	if err != nil {
		return nil, 0, err
	}

	if len(pos) > 0 {
		ids := make([]int, len(pos))
		for i, po := range pos {
			ids[i] = po.SupplierID
		}
		names, err := s.lookupSupplierNames(ctx, ids)
		if err != nil {
			return nil, 0, err
		}
		for i := range pos {
			s.applySupplierNames(&pos[i], names)
		}
	}

	return pos, total, nil
}

func (s *service) GetReceipts(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error) {
	return s.repo.GetReceiptsByPOID(ctx, poID, storeID)
}

func (s *service) CreateGoodsReceipt(ctx context.Context, poID, userID, storeID int, reqItems []CreateGRItemInput) (*GoodsReceipt, error) {
	po, err := s.repo.GetPurchaseOrderByID(ctx, poID, nil)
	if err != nil {
		return nil, err
	}
	if po.Status != StatusConfirmed && po.Status != StatusPartialReceived {
		return nil, ErrInvalidPOStatusForReceiving
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.repo.LockPurchaseOrderForUpdate(ctx, tx, poID); err != nil {
		return nil, err
	}

	po, err = s.repo.GetPurchaseOrderByID(ctx, poID, nil)
	if err != nil {
		return nil, err
	}

	if po.Status != StatusConfirmed && po.Status != StatusPartialReceived {
		return nil, ErrInvalidPOStatusForReceiving
	}

	itemMap := make(map[int]*OrderItem)
	for i := range po.Items {
		itemMap[po.Items[i].ID] = &po.Items[i]
	}

	grNumber, err := s.repo.GetNextGRNumber(ctx)
	if err != nil {
		return nil, err
	}

	doNumber, err := s.repo.GetNextDONumber(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(shared.JakartaLocation()).Format(time.RFC3339)
	gr := &GoodsReceipt{
		GRNumber:            grNumber,
		DeliveryOrderNumber: doNumber,
		PurchaseOrderID:     poID,
		StoreID:             storeID,
		ReceivedBy:          userID,
		ReceivedAt:          now,
	}

	var grItems []GoodsReceiptItem
	var receiptItems []events.PurchaseReceiptItem

	for _, reqItem := range reqItems {
		poItem, ok := itemMap[reqItem.PurchaseOrderItemID]
		if !ok {
			return nil, ErrPOItemNotFound
		}

		remaining := poItem.QtyOrdered - poItem.QtyReceived
		if reqItem.QtyGood < 0 || reqItem.QtyDamaged < 0 {
			return nil, ErrInvalidReceivingQty
		}
		if reqItem.QtyGood+reqItem.QtyDamaged > remaining {
			return nil, fmt.Errorf("%w: received quantity exceeds remaining for product %d: ordered=%d, remaining=%d, requested=%d", ErrOverReceiving,
				poItem.ProductID, poItem.QtyOrdered, remaining, reqItem.QtyGood+reqItem.QtyDamaged)
		}

		grItem := GoodsReceiptItem{
			PurchaseOrderItemID: reqItem.PurchaseOrderItemID,
			ProductID:           poItem.ProductID,
			QtyGood:             reqItem.QtyGood,
			QtyDamaged:          reqItem.QtyDamaged,
			UnitCost:            poItem.UnitCost,
			ProductName:         poItem.ProductName,
		}
		if reqItem.Notes != nil {
			grItem.Notes = *reqItem.Notes
		}
		grItems = append(grItems, grItem)

		if reqItem.QtyGood > 0 {
			receiptItems = append(receiptItems, events.PurchaseReceiptItem{
				ProductID: poItem.ProductID,
				QtyGood:   reqItem.QtyGood,
				UnitCost:  poItem.UnitCost,
			})
		}
	}

	if len(grItems) == 0 {
		return nil, ErrNoItemsToReceive
	}

	if err := s.repo.CreateGoodsReceipt(ctx, tx, gr, grItems); err != nil {
		return nil, err
	}
	gr.Items = grItems

	for start := 0; start < len(reqItems); start += maxBatchSize {
		end := start + maxBatchSize
		if end > len(reqItems) {
			end = len(reqItems)
		}
		chunk := reqItems[start:end]

		chunkValueStrings := make([]string, 0, len(chunk))
		chunkValueArgs := make([]interface{}, 0, len(chunk)*2)
		for i, reqItem := range chunk {
			chunkValueStrings = append(chunkValueStrings, fmt.Sprintf("($%d::int,$%d::int)", i*2+1, i*2+2))
			chunkValueArgs = append(chunkValueArgs, reqItem.PurchaseOrderItemID, reqItem.QtyGood+reqItem.QtyDamaged)
		}

		chunkQuery := fmt.Sprintf(`
			UPDATE purchase_order_items poi
			SET qty_received = poi.qty_received + v.qty_received, updated_at = NOW()
			FROM (VALUES %s) AS v(id, qty_received)
			WHERE poi.id = v.id
		`, strings.Join(chunkValueStrings, ","))

		if _, err := tx.Exec(ctx, chunkQuery, chunkValueArgs...); err != nil {
			return nil, fmt.Errorf("bulk update po item qty received: %w", err)
		}
	}

	if err := s.repo.RecalculatePOStatus(ctx, tx, poID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	bgCtx := context.Background()
	_ = s.eventBus.Publish(bgCtx, events.TopicGoodsReceiptCreated, &events.GoodsReceiptCreated{
		GRID:                gr.ID,
		GRNumber:            gr.GRNumber,
		DeliveryOrderNumber: gr.DeliveryOrderNumber,
		POID:                poID,
		PONumber:            po.PONumber,
		StoreID:             storeID,
	})

	if len(receiptItems) > 0 {
		_ = s.eventBus.Publish(bgCtx, events.TopicPurchaseReceiptCompleted, &events.PurchaseReceiptCompleted{
			POID:    poID,
			GRID:    gr.ID,
			StoreID: storeID,
			UserID:  userID,
			Items:   receiptItems,
		})
	}

	return gr, nil
}

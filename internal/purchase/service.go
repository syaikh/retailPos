package purchase

import (
	"context"
	"fmt"
	"time"

	"retail-pos-system/internal/shared"
)

type Service struct {
	repo     *Repository
	eventBus shared.EventBus
}

func NewService(repo *Repository, eventBus shared.EventBus) *Service {
	return &Service{repo: repo, eventBus: eventBus}
}

func (s *Service) CreateDraft(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("items cannot be empty")
	}
	if po.SupplierID == 0 {
		return fmt.Errorf("supplier_id is required")
	}

	storeID := po.StoreID
	if storeID == 0 {
		return fmt.Errorf("store_id is required")
	}

	po.Status = StatusDraft
	po.Subtotal = 0
	po.GrandTotal = 0

	productIDs := make([]int, len(items))
	seen := make(map[int]bool)
	for i, item := range items {
		if item.ProductID == 0 {
			return fmt.Errorf("product_id is required for item %d", i)
		}
		if item.QtyOrdered <= 0 {
			return fmt.Errorf("qty_ordered must be greater than 0 for product %d", item.ProductID)
		}
		if item.UnitCost < 0 {
			return fmt.Errorf("unit_cost cannot be negative for product %d", item.ProductID)
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

	productMap, err := s.repo.GetProductNamesByIDs(ctx, productIDs)
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

	_ = s.eventBus.Publish(ctx, "purchase_order.created", map[string]interface{}{
		"po_id":     po.ID,
		"po_number": po.PONumber,
	})

	return nil
}

func (s *Service) UpdateDraft(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("items cannot be empty")
	}
	if po.SupplierID == 0 {
		return fmt.Errorf("supplier_id is required")
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
			return fmt.Errorf("product_id is required for item %d", i)
		}
		if item.QtyOrdered <= 0 {
			return fmt.Errorf("qty_ordered must be greater than 0 for product %d", item.ProductID)
		}
		if item.UnitCost < 0 {
			return fmt.Errorf("unit_cost cannot be negative for product %d", item.ProductID)
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

func (s *Service) DeleteDraft(ctx context.Context, id int) error {
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

func (s *Service) Confirm(ctx context.Context, id, userID int) error {
	existing, err := s.repo.GetPurchaseOrderByID(ctx, id, nil)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return ErrPurchaseOrderNotDraft
	}

	now := time.Now().In(shared.JakartaLocation()).Format(time.RFC3339)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.repo.LockPurchaseOrderForUpdate(ctx, tx, id); err != nil {
		return err
	}

	if err := s.repo.ConfirmPurchaseOrder(ctx, tx, id, userID, now); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	_ = s.eventBus.Publish(ctx, "purchase_order.confirmed", map[string]interface{}{
		"po_id":     id,
		"po_number": existing.PONumber,
	})

	return nil
}

func (s *Service) Cancel(ctx context.Context, id, userID int) error {
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

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.repo.LockPurchaseOrderForUpdate(ctx, tx, id); err != nil {
		return err
	}

	if err := s.repo.CancelPurchaseOrder(ctx, tx, id, userID, now); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	_ = s.eventBus.Publish(ctx, "purchase_order.cancelled", map[string]interface{}{
		"po_id":     id,
		"po_number": existing.PONumber,
	})

	return nil
}

func (s *Service) GetDetail(ctx context.Context, id int, storeID *int) (*PurchaseOrder, error) {
	return s.repo.GetPurchaseOrderByID(ctx, id, storeID)
}

func (s *Service) List(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int) ([]PurchaseOrder, int, error) {
	return s.repo.GetAllPurchaseOrders(ctx, limit, offset, search, sortBy, sortDir, status, supplierID, startDate, endDate, storeID)
}

func (s *Service) GetReceipts(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error) {
	return s.repo.GetReceiptsByPOID(ctx, poID, storeID)
}

func (s *Service) CreateGoodsReceipt(ctx context.Context, poID, userID, storeID int, reqItems []CreateGRItemInput) (*GoodsReceipt, error) {
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

	itemMap := make(map[int]*PurchaseOrderItem)
	for i := range po.Items {
		itemMap[po.Items[i].ID] = &po.Items[i]
	}

	grNumber, err := s.repo.GetNextGRNumber(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().In(shared.JakartaLocation()).Format(time.RFC3339)
	gr := &GoodsReceipt{
		GRNumber:        grNumber,
		PurchaseOrderID: poID,
		StoreID:         storeID,
		ReceivedBy:      userID,
		ReceivedAt:      now,
	}

	var grItems []GoodsReceiptItem
	var receiptItems []PurchaseReceiptItem

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
			receiptItems = append(receiptItems, PurchaseReceiptItem{
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

	for _, reqItem := range reqItems {
		if err := s.repo.UpdatePOItemQtyReceived(ctx, tx, reqItem.PurchaseOrderItemID, reqItem.QtyGood+reqItem.QtyDamaged); err != nil {
			return nil, err
		}
	}

	if err := s.repo.RecalculatePOStatus(ctx, tx, poID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	_ = s.eventBus.Publish(ctx, "goods_receipt.created", map[string]interface{}{
		"gr_id":     gr.ID,
		"gr_number": gr.GRNumber,
		"po_id":     poID,
	})

	if len(receiptItems) > 0 {
		_ = s.eventBus.Publish(ctx, "PurchaseReceiptCompleted", PurchaseReceiptPayload{
			POID:    poID,
			GRID:    gr.ID,
			StoreID: storeID,
			UserID:  userID,
			Items:   receiptItems,
		})
	}

	return gr, nil
}

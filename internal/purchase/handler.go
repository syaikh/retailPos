package purchase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
)

type PurchaseService interface {
	CreateDraft(ctx context.Context, po *PurchaseOrder, items []PurchaseOrderItem) error
	UpdateDraft(ctx context.Context, id int, po *PurchaseOrder, items []PurchaseOrderItem) error
	DeleteDraft(ctx context.Context, id int) error
	Confirm(ctx context.Context, id, userID int) error
	Cancel(ctx context.Context, id, userID int) error
	GetDetail(ctx context.Context, id int, storeID *int) (*PurchaseOrder, error)
	List(ctx context.Context, limit, offset int, search, sortBy, sortDir, status, supplierID, startDate, endDate string, storeID *int) ([]PurchaseOrder, int, error)
	GetReceipts(ctx context.Context, poID int, storeID *int) ([]GoodsReceipt, error)
	CreateGoodsReceipt(ctx context.Context, poID, userID, storeID int, items []CreateGRItemInput) (*GoodsReceipt, error)
}

type Handler struct {
	svc      PurchaseService
	auditSvc audit.AuditCreator
}

func NewHandler(svc PurchaseService, auditSvc audit.AuditCreator) *Handler {
	return &Handler{svc: svc, auditSvc: auditSvc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, auth gin.HandlerFunc, perm func(permissions.Code) gin.HandlerFunc) {
	r.POST("/purchase-orders", auth, perm(permissions.PurchaseOrderCreate), h.CreateDraft)
	r.GET("/purchase-orders", auth, perm(permissions.PurchaseOrderView), h.ListPOs)
	r.GET("/purchase-orders/:id", auth, perm(permissions.PurchaseOrderView), h.GetPO)
	r.PUT("/purchase-orders/:id", auth, perm(permissions.PurchaseOrderUpdate), h.UpdateDraft)
	r.DELETE("/purchase-orders/:id", auth, perm(permissions.PurchaseOrderDelete), h.DeleteDraft)
	r.POST("/purchase-orders/:id/confirm", auth, perm(permissions.PurchaseOrderConfirm), h.ConfirmPO)
	r.POST("/purchase-orders/:id/cancel", auth, perm(permissions.PurchaseOrderCancel), h.CancelPO)
	r.GET("/purchase-orders/:id/receipts", auth, perm(permissions.PurchaseOrderView), h.GetReceipts)
	r.POST("/goods-receipts", auth, perm(permissions.PurchaseOrderReceive), h.CreateGoodsReceipt)
}

func toDomainItems(reqItems []CreatePOItemRequest) []PurchaseOrderItem {
	items := make([]PurchaseOrderItem, len(reqItems))
	for i, it := range reqItems {
		items[i] = PurchaseOrderItem{
			ProductID:      it.ProductID,
			QtyOrdered:     it.QtyOrdered,
			UnitCost:       it.UnitCost,
			DiscountAmount: it.DiscountAmount,
			Notes:          stringValue(it.Notes),
		}
	}
	return items
}

func toDomainUpdateItems(reqItems []UpdatePOItemRequest) []PurchaseOrderItem {
	items := make([]PurchaseOrderItem, len(reqItems))
	for i, it := range reqItems {
		items[i] = PurchaseOrderItem{
			ID:             it.ID,
			ProductID:      it.ProductID,
			QtyOrdered:     it.QtyOrdered,
			UnitCost:       it.UnitCost,
			DiscountAmount: it.DiscountAmount,
			Notes:          stringValue(it.Notes),
		}
	}
	return items
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *Handler) CreateDraft(c *gin.Context) {
	var req struct {
		SupplierID              int                   `json:"supplier_id" binding:"required"`
		StoreID                 *int                  `json:"store_id,omitempty"`
		WarehouseID             *int                  `json:"warehouse_id,omitempty"`
		ExpectedDate            string                `json:"expected_date,omitempty"`
		PaymentTerm             string                `json:"payment_term,omitempty"`
		DeliveryAddress         string                `json:"delivery_address,omitempty"`
		SupplierReferenceNumber string                `json:"supplier_reference_number,omitempty"`
		Notes                   string                `json:"notes,omitempty"`
		Items                   []CreatePOItemRequest `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	uid, ok := shared.GetUserIDWithOK(c)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	storeID := shared.GetStoreID(c)
	if storeID == nil {
		storeID = req.StoreID
	}
	if storeID == nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "store_id is required")
		return
	}

	po := &PurchaseOrder{
		SupplierID:              req.SupplierID,
		StoreID:                 *storeID,
		ExpectedDate:            req.ExpectedDate,
		PaymentTerm:             req.PaymentTerm,
		DeliveryAddress:         req.DeliveryAddress,
		SupplierReferenceNumber: req.SupplierReferenceNumber,
		Notes:                   req.Notes,
		CreatedBy:               uid,
		UpdatedBy:               uid,
	}
	if req.WarehouseID != nil {
		po.WarehouseID = req.WarehouseID
	}

	if err := h.svc.CreateDraft(ctx, po, toDomainItems(req.Items)); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrDuplicatePOItem):
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		default:
			shared.InternalError(c, err)
		}
		return
	}

	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      &uid,
		Username:    middleware.UsernameFromContext(c),
		Role:        middleware.RoleFromContext(c),
		Action:      "create",
		EntityType:  "purchase_order",
		EntityID:    &po.ID,
		Description: fmt.Sprintf("Created purchase order %s", po.PONumber),
	})

	c.JSON(http.StatusCreated, gin.H{"data": po})
}

func (h *Handler) UpdateDraft(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid id")
		return
	}

	var req struct {
		SupplierID              int                   `json:"supplier_id" binding:"required"`
		ExpectedDate            string                `json:"expected_date,omitempty"`
		PaymentTerm             string                `json:"payment_term,omitempty"`
		DeliveryAddress         string                `json:"delivery_address,omitempty"`
		SupplierReferenceNumber string                `json:"supplier_reference_number,omitempty"`
		Notes                   string                `json:"notes,omitempty"`
		Items                   []UpdatePOItemRequest `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	uid, ok := shared.GetUserIDWithOK(c)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}

	po := &PurchaseOrder{
		ID:                      id,
		SupplierID:              req.SupplierID,
		ExpectedDate:            req.ExpectedDate,
		PaymentTerm:             req.PaymentTerm,
		DeliveryAddress:         req.DeliveryAddress,
		SupplierReferenceNumber: req.SupplierReferenceNumber,
		Notes:                   req.Notes,
		UpdatedBy:               uid,
	}

	if err := h.svc.UpdateDraft(ctx, id, po, toDomainUpdateItems(req.Items)); err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrDuplicatePOItem):
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		case errors.Is(err, ErrPurchaseOrderNotFound):
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, err.Error())
		case errors.Is(err, ErrPurchaseOrderNotDraft):
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
		default:
			shared.InternalError(c, err)
		}
		return
	}

	po, _ = h.svc.GetDetail(ctx, id, shared.GetStoreID(c))
	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      &uid,
		Username:    middleware.UsernameFromContext(c),
		Role:        middleware.RoleFromContext(c),
		Action:      "update",
		EntityType:  "purchase_order",
		EntityID:    &id,
		Description: fmt.Sprintf("Updated purchase order %s", po.PONumber),
	})

	shared.JSONSuccess(c, po)
}

func (h *Handler) DeleteDraft(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid id")
		return
	}

	ctx := c.Request.Context()
	uid, ok := shared.GetUserIDWithOK(c)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}

	if err := h.svc.DeleteDraft(ctx, id); err != nil {
		if err == ErrPurchaseOrderNotDraft {
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
		} else {
			shared.InternalError(c, err)
		}
		return
	}

	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      &uid,
		Username:    middleware.UsernameFromContext(c),
		Role:        middleware.RoleFromContext(c),
		Action:      "delete",
		EntityType:  "purchase_order",
		EntityID:    &id,
		Description: fmt.Sprintf("Deleted purchase order id=%d", id),
	})

	shared.JSONSuccess(c, gin.H{"deleted": true})
}

func (h *Handler) ConfirmPO(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid id")
		return
	}

	ctx := c.Request.Context()
	uid, ok := shared.GetUserIDWithOK(c)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}

	po, err := h.svc.GetDetail(ctx, id, shared.GetStoreID(c))
	if err != nil {
		if err == ErrPurchaseOrderNotFound {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, err.Error())
		} else {
			shared.InternalError(c, err)
		}
		return
	}

	if err := h.svc.Confirm(ctx, id, uid); err != nil {
		switch err {
		case ErrPurchaseOrderNotDraft:
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
		case ErrPurchaseOrderAlreadyConfirmed:
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
		default:
			shared.InternalError(c, err)
		}
		return
	}

	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      &uid,
		Username:    middleware.UsernameFromContext(c),
		Role:        middleware.RoleFromContext(c),
		Action:      "update",
		EntityType:  "purchase_order",
		EntityID:    &id,
		Description: fmt.Sprintf("Confirmed purchase order %s", po.PONumber),
	})

	shared.JSONSuccess(c, gin.H{"status": StatusConfirmed})
}

func (h *Handler) CancelPO(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid id")
		return
	}

	ctx := c.Request.Context()
	uid, ok := shared.GetUserIDWithOK(c)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}

	po, err := h.svc.GetDetail(ctx, id, shared.GetStoreID(c))
	if err != nil {
		if err == ErrPurchaseOrderNotFound {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, err.Error())
		} else {
			shared.InternalError(c, err)
		}
		return
	}

	if err := h.svc.Cancel(ctx, id, uid); err != nil {
		switch err {
		case ErrPurchaseOrderCancelled:
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
		case ErrPurchaseOrderHasReceipts:
			shared.JSONError(c, http.StatusConflict, shared.ErrConflict, err.Error())
		default:
			shared.InternalError(c, err)
		}
		return
	}

	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      &uid,
		Username:    middleware.UsernameFromContext(c),
		Role:        middleware.RoleFromContext(c),
		Action:      "update",
		EntityType:  "purchase_order",
		EntityID:    &id,
		Description: fmt.Sprintf("Cancelled purchase order %s", po.PONumber),
	})

	shared.JSONSuccess(c, gin.H{"status": StatusCancelled})
}

func (h *Handler) GetPO(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid id")
		return
	}

	po, err := h.svc.GetDetail(c.Request.Context(), id, shared.GetStoreID(c))
	if err != nil {
		if err == ErrPurchaseOrderNotFound {
			shared.JSONError(c, http.StatusNotFound, shared.ErrNotFound, "purchase order not found")
			return
		}
		shared.InternalError(c, err)
		return
	}

	shared.JSONSuccess(c, po)
}

func (h *Handler) ListPOs(c *gin.Context) {
	limit, offset := shared.ParsePaginationParams(c.Query("limit"), c.Query("offset"))
	search := c.Query("search")
	sortBy := c.Query("sort_by")
	sortDir := c.Query("sort_dir")
	status := c.Query("status")
	supplierID := c.Query("supplier_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	storeID := shared.GetStoreID(c)

	pos, total, err := h.svc.List(c.Request.Context(), limit, offset, search, sortBy, sortDir, status, supplierID, startDate, endDate, storeID)
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if pos == nil {
		pos = []PurchaseOrder{}
	}
	shared.JSONPaginated(c, pos, total, limit, offset)
}

func (h *Handler) GetReceipts(c *gin.Context) {
	poID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "invalid id")
		return
	}

	receipts, err := h.svc.GetReceipts(c.Request.Context(), poID, shared.GetStoreID(c))
	if err != nil {
		shared.InternalError(c, err)
		return
	}
	if receipts == nil {
		receipts = []GoodsReceipt{}
	}
	shared.JSONSuccess(c, receipts)
}

func (h *Handler) CreateGoodsReceipt(c *gin.Context) {
	var req CreateGoodsReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()
	uid, ok := shared.GetUserIDWithOK(c)
	if !ok {
		shared.JSONError(c, http.StatusUnauthorized, shared.ErrUnauthorized, "user not authenticated")
		return
	}
	storeID := shared.GetStoreID(c)
	if storeID == nil {
		shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, "store_id is required")
		return
	}

	grItems := make([]CreateGRItemInput, len(req.Items))
	for i, item := range req.Items {
		grItems[i] = CreateGRItemInput(item)
	}

	gr, err := h.svc.CreateGoodsReceipt(ctx, req.PurchaseOrderID, uid, *storeID, grItems)
	if err != nil {
		switch {
		case err == ErrPOItemNotFound:
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		case err == ErrInvalidPOStatusForReceiving:
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		case errors.Is(err, ErrOverReceiving):
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		case err == ErrInvalidReceivingQty:
			shared.JSONError(c, http.StatusBadRequest, shared.ErrBadRequest, err.Error())
		default:
			shared.InternalError(c, err)
		}
		return
	}

	_ = h.auditSvc.CreateAuditLog(ctx, &audit.AuditLog{
		UserID:      &uid,
		Username:    middleware.UsernameFromContext(c),
		Role:        middleware.RoleFromContext(c),
		Action:      "create",
		EntityType:  "goods_receipt",
		EntityID:    &gr.ID,
		Description: fmt.Sprintf("Created goods receipt %s for PO id=%d", gr.GRNumber, req.PurchaseOrderID),
	})

	c.JSON(http.StatusCreated, gin.H{"data": gr})
}

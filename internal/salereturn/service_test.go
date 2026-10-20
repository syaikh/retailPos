package salereturn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
)

// ==================== PORT STUBS ====================
// The service's decision logic (validation, planning, outcome selection,
// figure computation, shift adjustment mapping, event publication) is pure
// once the ports are stubbed — no database needed. The transactional flows
// (ProcessReturn / ApproveReturn end to end) are covered by the DB-backed
// tests in repository_test.go's package via CI's postgres service.

type stubSettings struct {
	values map[string]string
	err    error
}

func (m stubSettings) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	return m.values, m.err
}

type stubStock struct {
	stock map[int]int
	err   error
}

func (m stubStock) AvailableStock(_ context.Context, _ shared.DBPool, productID int) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.stock[productID], nil
}

type stubDeduct struct {
	calls   int
	product int
	err     error
}

func (m *stubDeduct) DeductStock(_ context.Context, _ pgx.Tx, items []shared.StockDeductItem) error {
	m.calls++
	if len(items) > 0 {
		m.product = items[0].ProductID
	}
	return m.err
}

type stubConsignment struct {
	ids []int
	err error
}

func (m stubConsignment) ConsignmentProductIDs(_ context.Context, _ pgx.Tx, _ int) ([]int, error) {
	return m.ids, m.err
}

type stubAdjuster struct {
	last  *shared.ShiftReturnAdjustment
	calls int
	err   error
}

func (m *stubAdjuster) ApplyReturnAdjustment(_ context.Context, _ pgx.Tx, adj shared.ShiftReturnAdjustment) error {
	m.calls++
	m.last = &adj
	return m.err
}

type stubBus struct {
	topics []string
	sents  []interface{}
}

func (m *stubBus) Publish(_ context.Context, topic string, event interface{}) error {
	m.topics = append(m.topics, topic)
	m.sents = append(m.sents, event)
	return nil
}

// ==================== POLICY ====================

func TestLoadPolicy_WiredDefaultsAndOutage(t *testing.T) {
	ctx := context.Background()

	t.Run("unset settings port fails fast", func(t *testing.T) {
		s := NewService(nil, nil)
		_, err := s.loadPolicy(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SettingsProvider")
	})

	t.Run("valid values are parsed", func(t *testing.T) {
		s := NewService(nil, nil)
		s.SetSettingsProvider(stubSettings{values: map[string]string{
			settingWindowDays:        "7",
			settingApprovalThreshold: "1000000",
			settingApprovalTimeout:   "30",
		}})
		pol, err := s.loadPolicy(ctx)
		require.NoError(t, err)
		assert.Equal(t, 7, pol.windowDays)
		assert.Equal(t, 1000000, pol.approvalThreshold)
		assert.Equal(t, 30*time.Minute, pol.approvalTimeoutDuration)
	})

	t.Run("settings outage falls back to defaults", func(t *testing.T) {
		s := NewService(nil, nil)
		s.SetSettingsProvider(stubSettings{err: errors.New("db down")})
		pol, err := s.loadPolicy(ctx)
		require.NoError(t, err)
		assert.Equal(t, defaultWindowDays, pol.windowDays)
		assert.Equal(t, defaultApprovalThreshold, pol.approvalThreshold)
		assert.Equal(t, defaultTimeout(), pol.approvalTimeoutDuration)
	})

	t.Run("garbage values keep defaults", func(t *testing.T) {
		s := NewService(nil, nil)
		s.SetSettingsProvider(stubSettings{values: map[string]string{
			settingWindowDays:        "abc",
			settingApprovalThreshold: "-5",
			settingApprovalTimeout:   "0",
		}})
		pol, err := s.loadPolicy(ctx)
		require.NoError(t, err)
		assert.Equal(t, defaultWindowDays, pol.windowDays)
		assert.Equal(t, defaultApprovalThreshold, pol.approvalThreshold)
		assert.Equal(t, defaultTimeout(), pol.approvalTimeoutDuration)
	})
}

func defaultTimeout() time.Duration { return defaultApprovalTimeout * time.Minute }

// ==================== VALIDATION ====================

func TestValidateRequest(t *testing.T) {
	s := NewService(nil, nil)
	valid := ProcessRequest{
		Reason:    "barang rusak",
		Condition: ConditionDamaged,
		Items:     []ItemRequest{{SaleItemID: 1, Quantity: 1}},
	}

	require.NoError(t, s.validateRequest(valid))

	noReason := valid
	noReason.Reason = ""
	assert.ErrorIs(t, s.validateRequest(noReason), ErrInvalidReturn)

	functional := valid
	functional.Condition = ConditionFunctional
	assert.ErrorIs(t, s.validateRequest(functional), ErrConditionUnsupported)

	noItems := valid
	noItems.Items = nil
	assert.ErrorIs(t, s.validateRequest(noItems), ErrInvalidReturn)
}

func TestValidateSaleForReturn(t *testing.T) {
	store1 := 1
	now := time.Now()
	fresh := &shared.ReturnSale{Status: "completed", StoreID: 1, CreatedAt: now.Add(-time.Hour)}

	t.Run("eligible sale passes", func(t *testing.T) {
		require.NoError(t, validateSaleForReturn(fresh, &store1, policy{windowDays: 1}))
	})

	t.Run("nil store scope skips the store check", func(t *testing.T) {
		other := &shared.ReturnSale{Status: "completed", StoreID: 9, CreatedAt: now}
		require.NoError(t, validateSaleForReturn(other, nil, policy{windowDays: 1}))
	})

	t.Run("sale from another store is hidden", func(t *testing.T) {
		other := &shared.ReturnSale{Status: "completed", StoreID: 2, CreatedAt: now}
		assert.ErrorIs(t, validateSaleForReturn(other, &store1, policy{windowDays: 1}), ErrStoreMismatch)
	})

	t.Run("non-completed sale rejected", func(t *testing.T) {
		held := &shared.ReturnSale{Status: "held", StoreID: 1, CreatedAt: now}
		assert.ErrorIs(t, validateSaleForReturn(held, &store1, policy{windowDays: 1}), ErrSaleNotCompleted)
	})

	t.Run("outside the return window", func(t *testing.T) {
		old := &shared.ReturnSale{Status: "completed", StoreID: 1, CreatedAt: now.Add(-48 * time.Hour)}
		assert.ErrorIs(t, validateSaleForReturn(old, &store1, policy{windowDays: 1}), ErrReturnWindowExpired)
	})
}

// ==================== PLANNING ====================

func saleFixture() *shared.ReturnSale {
	return &shared.ReturnSale{
		ID:        1,
		Status:    "completed",
		StoreID:   1,
		ChangeDue: 500,
		Items: []shared.ReturnSaleItem{
			{ID: 10, ProductID: 7, ProductName: "Kabel HDMI", Quantity: 3, UnitPrice: 5000, Subtotal: 15000, TaxRate: 11},
			{ID: 11, ProductID: 8, ProductName: "Mouse", Quantity: 1, UnitPrice: 100000, Subtotal: 100000, TaxRate: 0},
		},
		Payments: []shared.ReturnSalePayment{
			{PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 60500},
		},
	}
}

func TestBuildPlan(t *testing.T) {
	sale := saleFixture()

	t.Run("happy path pre-computes the refund per line", func(t *testing.T) {
		req := ProcessRequest{
			Reason:    "r",
			Condition: ConditionDamaged,
			Items:     []ItemRequest{{SaleItemID: 10, Quantity: 2}},
		}
		lines, err := buildPlan(sale, req, nil)
		require.NoError(t, err)
		require.Len(t, lines, 1)
		assert.Equal(t, 2, lines[0].qty)
		assert.Equal(t, 10000, lines[0].subtotal, "2 * 5000, tax-inclusive")
		assert.Equal(t, 9009, lines[0].dpp, "round(10000*100/111)")
		assert.Equal(t, 991, lines[0].tax)
		assert.True(t, lines[0].isRefund(), "outcomes are decided later; the plan starts as refund")
	})

	t.Run("zero quantity rejected", func(t *testing.T) {
		req := ProcessRequest{Condition: ConditionDamaged, Items: []ItemRequest{{SaleItemID: 10, Quantity: 0}}}
		_, err := buildPlan(sale, req, nil)
		assert.ErrorIs(t, err, ErrInvalidReturn)
	})

	t.Run("duplicate line rejected", func(t *testing.T) {
		req := ProcessRequest{Condition: ConditionDamaged, Items: []ItemRequest{
			{SaleItemID: 10, Quantity: 1},
			{SaleItemID: 10, Quantity: 1},
		}}
		_, err := buildPlan(sale, req, nil)
		assert.ErrorIs(t, err, ErrInvalidReturn)
	})

	t.Run("item of a different sale rejected", func(t *testing.T) {
		req := ProcessRequest{Condition: ConditionDamaged, Items: []ItemRequest{{SaleItemID: 99, Quantity: 1}}}
		_, err := buildPlan(sale, req, nil)
		assert.ErrorIs(t, err, ErrSaleItemNotFound)
	})

	t.Run("already-returned quantity counts against the guard", func(t *testing.T) {
		req := ProcessRequest{Condition: ConditionDamaged, Items: []ItemRequest{{SaleItemID: 10, Quantity: 2}}}
		_, err := buildPlan(sale, req, map[int]int{10: 2})
		assert.ErrorIs(t, err, ErrQtyExceedsSold, "2 sold-returned + 2 requested exceeds the 3 sold")
	})
}

// ==================== OUTCOME DECISION ====================

func TestProposeOutcomes(t *testing.T) {
	s := NewService(nil, nil)
	s.SetStockChecker(stubStock{stock: map[int]int{7: 5, 8: 0}})

	lines := []plannedLine{
		{saleItem: shared.ReturnSaleItem{ProductID: 7}, qty: 2},
		{saleItem: shared.ReturnSaleItem{ProductID: 8}, qty: 1},
	}
	require.NoError(t, s.proposeOutcomes(context.Background(), nil, lines))
	assert.Equal(t, 2, lines[0].replaceQty, "stock available -> replacement candidate")
	assert.Equal(t, 0, lines[1].replaceQty, "no stock -> refund")
}

func TestProposeOutcomes_StockErrorPropagates(t *testing.T) {
	s := NewService(nil, nil)
	s.SetStockChecker(stubStock{err: errors.New("connection lost")})
	lines := []plannedLine{{saleItem: shared.ReturnSaleItem{ProductID: 7}, qty: 1}}
	require.ErrorContains(t, s.proposeOutcomes(context.Background(), nil, lines), "connection lost")
}

// A concurrent sale can take the last unit between proposal and deduction;
// the line must degrade to a refund, not fail the return.
func TestDeduceReplacementStock_DemotesOnInsufficientStock(t *testing.T) {
	s := NewService(nil, nil)
	deduct := &stubDeduct{err: shared.ErrInsufficientStock}
	s.SetStockDeducer(deduct)

	lines := []plannedLine{
		{saleItem: shared.ReturnSaleItem{ProductID: 7}, qty: 2, replaceQty: 2},
		{saleItem: shared.ReturnSaleItem{ProductID: 9}, qty: 1, replaceQty: 0},
	}
	require.NoError(t, s.deduceReplacementStock(context.Background(), nil, lines))
	assert.Equal(t, 0, lines[0].replaceQty, "insufficient stock demotes the line to a refund")
	assert.Equal(t, 1, deduct.calls, "refund lines never reach the stock deducter")
	assert.Equal(t, 7, deduct.product)
}

func TestDeduceReplacementStock_HardErrorFails(t *testing.T) {
	s := NewService(nil, nil)
	s.SetStockDeducer(&stubDeduct{err: errors.New("write failed")})
	lines := []plannedLine{{saleItem: shared.ReturnSaleItem{ProductID: 7}, qty: 1, replaceQty: 1}}
	require.ErrorContains(t, s.deduceReplacementStock(context.Background(), nil, lines), "write failed")
}

func TestRejectConsignment(t *testing.T) {
	s := NewService(nil, nil)
	sale := saleFixture()
	lines := []plannedLine{{saleItem: shared.ReturnSaleItem{ProductID: 7}}}

	s.SetConsignmentGuard(stubConsignment{})
	require.NoError(t, s.rejectConsignment(context.Background(), nil, sale, lines), "no consignment lines")

	s.SetConsignmentGuard(stubConsignment{ids: []int{8}})
	require.NoError(t, s.rejectConsignment(context.Background(), nil, sale, lines), "a different product is fine")

	s.SetConsignmentGuard(stubConsignment{ids: []int{7}})
	err := s.rejectConsignment(context.Background(), nil, sale, lines)
	assert.ErrorIs(t, err, ErrConsignmentNotSupported)
}

// ==================== FIGURES, SHIFT, EVENT ====================

func TestApplyLineFigures(t *testing.T) {
	s := NewService(nil, nil)

	t.Run("all replacement moves no money", func(t *testing.T) {
		ret := &Return{}
		s.applyLineFigures(ret, []plannedLine{
			{qty: 2, subtotal: 10000, tax: 991, replaceQty: 2},
		})
		assert.Equal(t, ReturnTypeReplacement, ret.ReturnType)
		assert.Equal(t, 0, ret.TotalRefundAmount)
		assert.Equal(t, 0, ret.Subtotal)
		assert.Equal(t, 0, ret.TaxReversal)
	})

	t.Run("mixed return counts refund lines only", func(t *testing.T) {
		ret := &Return{}
		s.applyLineFigures(ret, []plannedLine{
			{qty: 2, subtotal: 10000, tax: 991, replaceQty: 0},  // refunded
			{qty: 1, subtotal: 50000, tax: 4505, replaceQty: 1}, // replaced
		})
		assert.Equal(t, ReturnTypeRefund, ret.ReturnType)
		assert.Equal(t, 10000, ret.Subtotal)
		assert.Equal(t, 991, ret.TaxReversal)
		assert.Equal(t, 10000, ret.TotalRefundAmount)
		assert.Equal(t, 0, ret.Discount, "item discounts are baked into unit_price")
		assert.Equal(t, 0, ret.RestockingFee, "no restocking fee in Phase 1")
	})
}

func TestApplyShiftAdjustment(t *testing.T) {
	ctx := context.Background()

	t.Run("pure replacement never touches the shift", func(t *testing.T) {
		adjuster := &stubAdjuster{}
		s := NewService(nil, nil)
		s.SetShiftTotalAdjuster(adjuster)
		require.NoError(t, s.applyShiftAdjustment(ctx, nil, &Return{ShiftID: 9}, nil))
		assert.Equal(t, 0, adjuster.calls)
	})

	t.Run("cash and non-cash split reaches the adjuster", func(t *testing.T) {
		adjuster := &stubAdjuster{}
		s := NewService(nil, nil)
		s.SetShiftTotalAdjuster(adjuster)
		alloc := []refundAllocation{
			{PaymentMethodID: 1, PaymentMethodCode: "CASH", Amount: 7000},
			{PaymentMethodID: 2, PaymentMethodCode: "CARD", Amount: 3000},
		}
		ret := &Return{ShiftID: 9, TotalRefundAmount: 10000}
		require.NoError(t, s.applyShiftAdjustment(ctx, nil, ret, alloc))
		require.NotNil(t, adjuster.last)
		assert.Equal(t, 9, adjuster.last.ShiftID)
		assert.Equal(t, 7000, adjuster.last.CashRefund)
		assert.Equal(t, 3000, adjuster.last.NonCashRefund)
		assert.Equal(t, 10000, adjuster.last.TotalRefund)
	})

	t.Run("a closed shift surfaces the PRD message", func(t *testing.T) {
		adjuster := &stubAdjuster{err: shared.ErrShiftNotOpen}
		s := NewService(nil, nil)
		s.SetShiftTotalAdjuster(adjuster)
		ret := &Return{ShiftID: 9, TotalRefundAmount: 10000}
		err := s.applyShiftAdjustment(ctx, nil, ret, nil)
		require.ErrorIs(t, err, ErrShiftNotOpen)
		assert.NotErrorIs(t, err, shared.ErrShiftNotOpen, "the salereturn sentinel replaces the shared one")
	})

	t.Run("other adjuster errors are wrapped", func(t *testing.T) {
		adjuster := &stubAdjuster{err: errors.New("deadlock")}
		s := NewService(nil, nil)
		s.SetShiftTotalAdjuster(adjuster)
		err := s.applyShiftAdjustment(ctx, nil, &Return{ShiftID: 9, TotalRefundAmount: 1000}, nil)
		require.ErrorContains(t, err, "failed to adjust shift totals")
	})
}

func TestPublishReturned(t *testing.T) {
	ctx := context.Background()

	t.Run("completed return publishes after commit", func(t *testing.T) {
		bus := &stubBus{}
		s := NewService(nil, bus)
		s.publishReturned(ctx, &Return{
			ID: 3, ReturnNumber: "RET-2026-000003", OriginalSaleID: 7,
			StoreID: 1, ReturnType: ReturnTypeRefund, TotalRefundAmount: 12000,
			Status: ReturnStatusCompleted,
		})
		require.Len(t, bus.topics, 1)
		assert.Equal(t, events.TopicSaleReturned, bus.topics[0])
		sent, ok := bus.sents[0].(*events.SaleReturned)
		require.True(t, ok)
		assert.Equal(t, "RET-2026-000003", sent.ReturnNumber)
		assert.Equal(t, 12000, sent.TotalRefundAmount)
	})

	t.Run("pending return publishes nothing", func(t *testing.T) {
		bus := &stubBus{}
		s := NewService(nil, bus)
		s.publishReturned(ctx, &Return{Status: ReturnStatusPendingApproval})
		assert.Empty(t, bus.topics)
	})
}

func TestPublishApprovalRequested(t *testing.T) {
	ctx := context.Background()
	expires := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)

	t.Run("pending return announces the request after commit", func(t *testing.T) {
		bus := &stubBus{}
		s := NewService(nil, bus)
		s.publishApprovalRequested(ctx, &Return{
			ID: 12, ReturnNumber: "RET-2026-000012", OriginalSaleID: 7,
			StoreID: 3, TotalRefundAmount: 750_000, ExpiresAt: &expires,
			Status: ReturnStatusPendingApproval,
		}, Actor{ID: 4, Username: "siti"}, "INV-2026-000042")

		require.Len(t, bus.topics, 1)
		assert.Equal(t, events.TopicReturnApprovalRequested, bus.topics[0])
		sent, ok := bus.sents[0].(*events.ReturnApprovalRequested)
		require.True(t, ok)
		assert.Equal(t, 12, sent.ReturnID)
		assert.Equal(t, "INV-2026-000042", sent.InvoiceNumber)
		assert.Equal(t, "siti", sent.CashierName)
		assert.Equal(t, 750_000, sent.TotalRefundAmount)
		assert.Equal(t, expires.Format(time.RFC3339), sent.ExpiresAt)
	})

	t.Run("completed return announces nothing", func(t *testing.T) {
		bus := &stubBus{}
		s := NewService(nil, bus)
		s.publishApprovalRequested(ctx, &Return{Status: ReturnStatusCompleted}, Actor{}, "")
		assert.Empty(t, bus.topics)
	})
}

func TestPublishApprovalResolved(t *testing.T) {
	ctx := context.Background()

	for _, status := range []string{
		ReturnStatusCompleted, ReturnStatusCancelled, ReturnStatusExpired,
	} {
		t.Run(status+" announces the resolution", func(t *testing.T) {
			bus := &stubBus{}
			s := NewService(nil, bus)
			s.publishApprovalResolved(ctx, &Return{
				ID: 12, ReturnNumber: "RET-2026-000012", StoreID: 3, Status: status,
			})
			require.Len(t, bus.topics, 1)
			assert.Equal(t, events.TopicReturnApprovalResolved, bus.topics[0])
			sent, ok := bus.sents[0].(*events.ReturnApprovalResolved)
			require.True(t, ok)
			assert.Equal(t, status, sent.Status)
		})
	}

	t.Run("pending return announces nothing", func(t *testing.T) {
		bus := &stubBus{}
		s := NewService(nil, bus)
		s.publishApprovalResolved(ctx, &Return{Status: ReturnStatusPendingApproval})
		assert.Empty(t, bus.topics)
	})
}

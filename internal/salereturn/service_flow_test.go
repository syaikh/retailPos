package salereturn

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"retail-pos-system/internal/events"
	"retail-pos-system/internal/shared"
)

// ==================== END-TO-END FLOW STUBS ====================
// The transactional flows (ProcessReturn / ApproveReturn against the real
// repository and transaction) run on the shared test database; only the
// cross-module ports are stubbed. This is what makes the write-path wiring
// (response items, event ItemCount, threshold-after-demotion) observable.

type stubSaleReader struct {
	sale *shared.ReturnSale
	err  error
}

func (m stubSaleReader) SaleForReturn(_ context.Context, _ pgx.Tx, _ int) (*shared.ReturnSale, error) {
	return m.sale, m.err
}

type stubShiftFinder struct {
	shiftID int
	err     error
}

func (m stubShiftFinder) ActiveShiftID(_ context.Context, _ int) (int, error) {
	return m.shiftID, m.err
}

// returnSaleFromSeed mirrors the sale rows seedSaleFixture inserted.
func returnSaleFromSeed(f *seeded) *shared.ReturnSale {
	return &shared.ReturnSale{
		ID:        f.saleID,
		Status:    "completed",
		StoreID:   1,
		ChangeDue: 500,
		CreatedAt: time.Now(),
		Items: []shared.ReturnSaleItem{
			{ID: f.itemID, ProductID: f.productID, ProductName: "Kabel HDMI",
				Quantity: 3, UnitPrice: 5000, Subtotal: 15000, TaxRate: 11},
		},
		Payments: []shared.ReturnSalePayment{
			{PaymentMethodID: f.cashPM, PaymentMethodCode: "CASH", Amount: 15500},
		},
	}
}

func newFlowService(f *seeded, stock map[int]int, ded *stubDeduct, threshold string) (*Service, *stubBus, *stubAdjuster) {
	bus := &stubBus{}
	adjuster := &stubAdjuster{}
	s := NewService(NewRepository(dbPool), bus)
	s.SetSaleReader(stubSaleReader{sale: returnSaleFromSeed(f)})
	s.SetActiveShiftFinder(stubShiftFinder{shiftID: f.shiftID})
	s.SetShiftTotalAdjuster(adjuster)
	s.SetStockChecker(stubStock{stock: stock})
	s.SetStockDeducer(ded)
	s.SetConsignmentGuard(stubConsignment{})
	s.SetSettingsProvider(stubSettings{values: map[string]string{
		settingWindowDays:        "1",
		settingApprovalThreshold: threshold,
		settingApprovalTimeout:   "5",
	}})
	return s, bus, adjuster
}

func flowActor(f *seeded) Actor {
	return Actor{ID: f.userID, Username: "salereturn_flow", Role: "cashier"}
}

// ==================== PROCESS RETURN ====================

func TestProcessReturn_ImmediateReplacement_PopulatesItems(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	ded := &stubDeduct{}
	s, bus, adjuster := newFlowService(f, map[int]int{f.productID: 5}, ded, "500000")

	ret, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusCompleted, ret.Status)
	assert.Equal(t, ReturnTypeReplacement, ret.ReturnType)
	assert.Zero(t, ret.TotalRefundAmount, "a replacement moves goods, not money")
	assert.Equal(t, 1, ded.calls)

	// The write path must surface the inserted lines (not an empty list) and
	// their generated row ids.
	require.Len(t, ret.Items, 1)
	assert.Equal(t, f.itemID, ret.Items[0].SaleItemID)
	assert.Positive(t, ret.Items[0].ID)
	require.Len(t, ret.Items[0].Replacements, 1)
	assert.Equal(t, 1, ret.Items[0].Replacements[0].Quantity)

	assert.Equal(t, 0, adjuster.calls, "no money moved, no shift adjustment")
	require.Len(t, bus.sents, 1, "sale.returned.v1 published once after commit")
	ev, ok := bus.sents[0].(*events.SaleReturned)
	require.True(t, ok)
	assert.Equal(t, 1, ev.ItemCount, "the event must count the committed lines")
}

func TestProcessReturn_ImmediateRefund_PopulatesItemsAndPayments(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	s, bus, adjuster := newFlowService(f, map[int]int{f.productID: 0}, &stubDeduct{}, "500000")

	ret, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusCompleted, ret.Status)
	assert.Equal(t, ReturnTypeRefund, ret.ReturnType)
	assert.Equal(t, 5000, ret.TotalRefundAmount, "qty 1 × unit 5000, tax-inclusive")

	require.Len(t, ret.Items, 1)
	assert.Equal(t, 5000, ret.Items[0].Subtotal)
	require.Len(t, ret.Payments, 1, "the response must carry the refund split")
	assert.Equal(t, 5000, ret.Payments[0].Amount)

	require.Equal(t, 1, adjuster.calls)
	require.NotNil(t, adjuster.last)
	assert.Equal(t, 5000, adjuster.last.CashRefund, "cash net 15500−500 covers the whole refund")

	require.Len(t, bus.sents, 1)
	ev, ok := bus.sents[0].(*events.SaleReturned)
	require.True(t, ok)
	assert.Equal(t, 1, ev.ItemCount)
}

func TestProcessReturn_DemotionPastThreshold_AbortsForRetry(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	// Stock is visible at planning time (replacement proposed → refund 0,
	// under the 1000 threshold), but the atomic deduction loses the race, so
	// the line demotes to a 5000 refund — past the threshold. The return
	// must abort and roll back instead of completing unapproved.
	ded := &stubDeduct{err: shared.ErrInsufficientStock}
	s, bus, adjuster := newFlowService(f, map[int]int{f.productID: 5}, ded, "1000")

	_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	assert.ErrorIs(t, err, ErrStockChanged)
	assert.Equal(t, 0, adjuster.calls, "no shift adjustment on the aborted attempt")
	assert.Empty(t, bus.topics, "nothing published for a rolled-back attempt")

	var count int
	require.NoError(t, dbPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM sale_returns WHERE original_sale_id = $1`, f.saleID).Scan(&count))
	assert.Zero(t, count, "the whole transaction must roll back")

	// The retry (stock now truly gone) plans the line as a refund from the
	// start and correctly parks it for manager approval.
	s2, _, _ := newFlowService(f, map[int]int{f.productID: 0}, &stubDeduct{}, "1000")
	ret, err := s2.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusPendingApproval, ret.Status)
	require.NotNil(t, ret.ExpiresAt)
}

// ==================== APPROVE RETURN ====================

func seedPendingReturn(t *testing.T, f *seeded, expiresAt time.Time) *Return {
	t.Helper()
	ctx := context.Background()
	repo := NewRepository(dbPool)
	ret := insertReturnWithItem(ctx, t, repo, f, ReturnStatusPendingApproval, &expiresAt)
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	require.NoError(t, repo.InsertApproval(ctx, tx, ret.ID, f.userID, nil, expiresAt))
	require.NoError(t, tx.Commit(ctx))
	return ret
}

func TestApproveReturn_CompletesPending_ReplacesProposalRows(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))

	// Stock is gone at approval time: the re-run decision refunds.
	s, bus, adjuster := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
	approved, err := s.ApproveReturn(ctx, flowActor(f), nil, pending.ID, nil)
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusCompleted, approved.Status)
	assert.Equal(t, ReturnTypeRefund, approved.ReturnType)

	// The proposal rows loaded at lock time carried subtotal 10000; approval
	// must re-plan from the sale (qty 1 × 5000) and expose the REPLACED rows,
	// not the stale proposal.
	require.Len(t, approved.Items, 1)
	assert.Equal(t, 5000, approved.Items[0].Subtotal)
	assert.Equal(t, 5000, approved.TotalRefundAmount)

	require.Equal(t, 1, adjuster.calls)
	require.Len(t, bus.sents, 1)
	ev, ok := bus.sents[0].(*events.SaleReturned)
	require.True(t, ok)
	assert.Equal(t, 1, ev.ItemCount)
}

func TestApproveReturn_ElapsedWindow_FlipsAndFails(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	pending := seedPendingReturn(t, f, time.Now().Add(-time.Minute))

	s, bus, adjuster := newFlowService(f, map[int]int{f.productID: 5}, &stubDeduct{}, "500000")
	_, err := s.ApproveReturn(ctx, flowActor(f), nil, pending.ID, nil)
	assert.ErrorIs(t, err, ErrApprovalExpired)
	assert.Equal(t, 0, adjuster.calls)
	assert.Empty(t, bus.topics)

	// The flip commits (the expiry path returns ErrApprovalExpired from a
	// SUCCESSFUL transaction), freeing the sale's returnable quantity.
	repo := NewRepository(dbPool)
	loaded, err := repo.GetReturn(ctx, nil, pending.ID)
	require.NoError(t, err)
	assert.Equal(t, ReturnStatusExpired, loaded.Status)

	returned, err := repo.ReturnedQtyBySale(ctx, dbPool, f.saleID, 0)
	require.NoError(t, err)
	assert.Zero(t, returned[f.itemID], "an expired return must not consume the returnable quantity")
}

// ==================== WIRING & GUARD FAILURES ====================
// These cover the fail-fast loop, the port error paths and the reject guards.
// They hand-build the fixture (no rows) because every assertion fails before
// — or without — writing a return.

func TestProcessReturn_UnwiredPortFailsFast(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := &seeded{userID: 1, shiftID: 1, productID: 1, saleID: 1, itemID: 1, cashPM: 1}
	req := ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	}

	cases := []struct {
		port  string
		unset func(s *Service)
	}{
		{"SaleReader", func(s *Service) { s.SetSaleReader(nil) }},
		{"ActiveShiftFinder", func(s *Service) { s.SetActiveShiftFinder(nil) }},
		{"ShiftTotalAdjuster", func(s *Service) { s.SetShiftTotalAdjuster(nil) }},
		{"StockChecker", func(s *Service) { s.SetStockChecker(nil) }},
		{"StockDeducer", func(s *Service) { s.SetStockDeducer(nil) }},
		{"ConsignmentGuard", func(s *Service) { s.SetConsignmentGuard(nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.port, func(t *testing.T) {
			s, _, _ := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
			tc.unset(s)
			_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, req)
			require.Error(t, err)
			assert.Equal(t, fmt.Sprintf("salereturn: %s port is not wired", tc.port), err.Error())
		})
	}
	assert.Zero(t, countSaleReturns(t), "fail-fast must not open a transaction")
}

func TestApproveReturn_UnwiredPortFailsFast(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := &seeded{userID: 1, shiftID: 1, productID: 1, saleID: 1, itemID: 1, cashPM: 1}

	cases := []struct {
		port  string
		unset func(s *Service)
	}{
		{"SaleReader", func(s *Service) { s.SetSaleReader(nil) }},
		{"ShiftTotalAdjuster", func(s *Service) { s.SetShiftTotalAdjuster(nil) }},
		{"StockChecker", func(s *Service) { s.SetStockChecker(nil) }},
		{"StockDeducer", func(s *Service) { s.SetStockDeducer(nil) }},
		{"ConsignmentGuard", func(s *Service) { s.SetConsignmentGuard(nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.port, func(t *testing.T) {
			s, _, _ := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
			tc.unset(s)
			_, err := s.ApproveReturn(ctx, flowActor(f), nil, 1, nil)
			require.Error(t, err)
			assert.Equal(t, fmt.Sprintf("salereturn: %s port is not wired", tc.port), err.Error())
		})
	}

	// Asymmetry guard: approval does not need the APPROVER's shift (only the
	// processing cashier's shift is adjusted), so an unset ActiveShiftFinder
	// must let the flow reach the repository instead of tripping the guard.
	t.Run("ActiveShiftFinder is not required by approval", func(t *testing.T) {
		s, _, _ := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
		s.SetActiveShiftFinder(nil)
		_, err := s.ApproveReturn(ctx, flowActor(f), nil, 1, nil)
		assert.ErrorIs(t, err, ErrReturnNotFound, "the flow must reach GetReturnForUpdate")
	})

	assert.Zero(t, countSaleReturns(t), "fail-fast must not open a transaction")
}

func TestProcessReturn_NoOpenShiftRejectsReturn(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := &seeded{userID: 1, shiftID: 1, productID: 1, saleID: 1, itemID: 1, cashPM: 1}

	s, bus, adjuster := newFlowService(f, map[int]int{f.productID: 0}, &stubDeduct{}, "500000")
	s.SetActiveShiftFinder(stubShiftFinder{shiftID: 0})

	_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	// A refund pays out of a drawer: with no open shift there is nowhere to
	// take the money from, so the whole return is rejected (PRD §12.1 #1).
	assert.ErrorIs(t, err, ErrShiftNotOpen)
	assert.Zero(t, adjuster.calls, "no shift adjustment without a shift")
	assert.Empty(t, bus.topics, "nothing published for a rejected return")
	assert.Zero(t, countSaleReturns(t))
}

func TestProcessReturn_ShiftFinderErrorPropagates(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := &seeded{userID: 1, shiftID: 1, productID: 1, saleID: 1, itemID: 1, cashPM: 1}

	s, _, _ := newFlowService(f, map[int]int{f.productID: 0}, &stubDeduct{}, "500000")
	s.SetActiveShiftFinder(stubShiftFinder{err: errors.New("db down")})

	_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find active shift: db down",
		"an outage must not be mistaken for a closed shift")
	assert.Zero(t, countSaleReturns(t))
}

func TestProcessReturn_SaleReaderPaths(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := &seeded{userID: 1, shiftID: 1, productID: 1, saleID: 1, itemID: 1, cashPM: 1}
	req := ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	}

	t.Run("nil sale is sale not found", func(t *testing.T) {
		s, _, _ := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
		s.SetSaleReader(stubSaleReader{sale: nil})
		_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, req)
		assert.ErrorIs(t, err, ErrSaleNotFound)
		assert.Zero(t, countSaleReturns(t))
	})

	t.Run("reader error propagates unwrapped", func(t *testing.T) {
		boom := errors.New("sale read failed")
		s, _, _ := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
		s.SetSaleReader(stubSaleReader{err: boom})
		_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, req)
		assert.ErrorIs(t, err, boom, "the provider's error must reach the caller as-is")
		assert.Zero(t, countSaleReturns(t))
	})
}

func TestProcessReturn_ConsignmentGuardErrorPropagates(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := &seeded{userID: 1, shiftID: 1, productID: 1, saleID: 1, itemID: 1, cashPM: 1}

	s, _, _ := newFlowService(f, map[int]int{f.productID: 0}, &stubDeduct{}, "500000")
	s.SetConsignmentGuard(stubConsignment{err: errors.New("consignment down")})

	_, err := s.ProcessReturn(ctx, flowActor(f), nil, f.saleID, ProcessRequest{
		Reason: "barang rusak", Condition: ConditionDamaged,
		Items: []ItemRequest{{SaleItemID: f.itemID, Quantity: 1}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check consignment items: consignment down",
		"an outage must not be read as 'no consignment lines'")
	assert.Zero(t, countSaleReturns(t), "the whole transaction must roll back")
}

func TestRejectReturn_CancelsPendingAndFreesQuantity(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
	repo := NewRepository(dbPool)

	returned, err := repo.ReturnedQtyBySale(ctx, dbPool, f.saleID, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, returned[f.itemID], "the pending proposal consumes the sale's returnable quantity")

	s, bus, adjuster := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")
	out, err := s.RejectReturn(ctx, flowActor(f), nil, pending.ID, "di luar kebijakan")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, ReturnStatusCancelled, out.Status)

	// The cancelled rows must stop consuming the sale's returnable quantity,
	// so the customer can be served a fresh return afterwards.
	returned, err = repo.ReturnedQtyBySale(ctx, dbPool, f.saleID, 0)
	require.NoError(t, err)
	assert.Zero(t, returned[f.itemID])

	loaded, err := repo.GetReturn(ctx, nil, pending.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.Approval)
	assert.Equal(t, "rejected", loaded.Approval.Status)
	require.NotNil(t, loaded.Approval.RejectedReason)
	assert.Equal(t, "di luar kebijakan", *loaded.Approval.RejectedReason)

	assert.Zero(t, adjuster.calls, "a rejection moves no money")
	assert.Empty(t, bus.topics, "and publishes no event")
}

func TestRejectReturn_Guards(t *testing.T) {
	_ = shared.TruncateTestData(dbPool)
	ctx := context.Background()
	f := seedSaleFixture(t)
	repo := NewRepository(dbPool)
	s, _, _ := newFlowService(f, map[int]int{}, &stubDeduct{}, "500000")

	t.Run("empty reason is rejected before any write", func(t *testing.T) {
		_, err := s.RejectReturn(ctx, flowActor(f), nil, 1, "")
		assert.ErrorIs(t, err, ErrInvalidReturn)
		assert.Contains(t, err.Error(), "rejection reason is required")
	})

	t.Run("unknown return is not found", func(t *testing.T) {
		_, err := s.RejectReturn(ctx, flowActor(f), nil, 99999999, "x")
		assert.ErrorIs(t, err, ErrReturnNotFound)
	})

	t.Run("another store's return is not found", func(t *testing.T) {
		pending := seedPendingReturn(t, f, time.Now().Add(5*time.Minute))
		_, err := s.RejectReturn(ctx, flowActor(f), ptrInt(2), pending.ID, "x")
		assert.ErrorIs(t, err, ErrReturnNotFound, "the store check runs before any decision")

		loaded, lerr := repo.GetReturn(ctx, nil, pending.ID)
		require.NoError(t, lerr)
		assert.Equal(t, ReturnStatusPendingApproval, loaded.Status, "the row must stay untouched")
	})

	t.Run("a completed return is not pending", func(t *testing.T) {
		done := insertReturnWithItem(ctx, t, repo, f, ReturnStatusCompleted, nil)
		_, err := s.RejectReturn(ctx, flowActor(f), nil, done.ID, "x")
		assert.ErrorIs(t, err, ErrReturnNotPending)

		loaded, lerr := repo.GetReturn(ctx, nil, done.ID)
		require.NoError(t, lerr)
		assert.Equal(t, ReturnStatusCompleted, loaded.Status, "the completed row must stay untouched")
	})
}

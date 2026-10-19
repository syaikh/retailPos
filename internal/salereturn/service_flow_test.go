package salereturn

import (
	"context"
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

type stubSaleReader struct{ sale *shared.ReturnSale }

func (m stubSaleReader) SaleForReturn(_ context.Context, _ pgx.Tx, _ int) (*shared.ReturnSale, error) {
	return m.sale, nil
}

type stubShiftFinder struct{ shiftID int }

func (m stubShiftFinder) ActiveShiftID(_ context.Context, _ int) (int, error) {
	return m.shiftID, nil
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

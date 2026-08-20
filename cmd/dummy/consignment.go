package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// consignmentSeedTerm is the agreed price/share for a consigned SKU,
// mirroring the consignment_terms table.
type consignmentSeedTerm struct {
	price           int
	storeShareType  string
	storeShareValue float64
}

// injectConsignment seeds the Konsinyasi Supplier module. It must run AFTER
// injectDailySales / injectShifts (consignment sale items are backfilled from
// real sale_items) and BEFORE injectStockOpnames (consignment-owned SKUs are
// excluded from the opname snapshot). Steps:
//  1. flag selected suppliers as is_consignment,
//  2. open an active arrangement per consignment supplier,
//  3. set price/share terms for each consigned SKU,
//  4. inject 1-2 receipts per arrangement (ownership ledger + product_stock),
//  5. backfill consignment_sale_items from sales after each SKU's first
//     receipt, capped at the received quantity,
//  6. create open pending returns, then a formal return per supplier,
//  7. settle each supplier's unsettled consignment sales (some paid + payout),
//  8. advance the CR-/RT-/CS-/CP- sequences through nextval on every document.
func injectConsignment(ctx context.Context, db *sql.DB, startDate, endDate time.Time, numConsignment int) error {
	storeIDs := getIDs(ctx, db, "stores")
	if len(storeIDs) == 0 {
		return fmt.Errorf("no stores found for consignment")
	}
	storeID := storeIDs[0]

	userIDs := getUserIDsByRoles(ctx, db, "admin", "manager")
	if len(userIDs) == 0 {
		userIDs = getIDs(ctx, db, "users")
	}
	if len(userIDs) == 0 {
		return fmt.Errorf("no users found for consignment")
	}
	createdBy := userIDs[0]

	// Select suppliers for consignment: pick first N from suppliers table
	// that don't already have an active arrangement
	allSupplierRows, err := findAllSuppliers(ctx, db)
	if err != nil {
		return err
	}
	if len(allSupplierRows) == 0 {
		fmt.Println("   ⚠️  No suppliers found; skipping consignment")
		return nil
	}

	// Skip suppliers that already have an active arrangement (BR-47 single-active
	// invariant) so re-running with -truncate=false does not collide.
	availableRows, err := filterSuppliersWithActiveArrangement(ctx, db, allSupplierRows, storeIDs[0])
	if err != nil {
		return err
	}
	if len(availableRows) == 0 {
		fmt.Println("   ℹ️  All suppliers already have an active arrangement; skipping")
		return nil
	}

	// Cap to requested number
	if numConsignment > len(availableRows) {
		numConsignment = len(availableRows)
	}
	if numConsignment <= 0 {
		fmt.Println("   ℹ️  No consignment suppliers requested; skipping")
		return nil
	}

	supplierRows := availableRows[:numConsignment]
	supplierIDs := make([]int, 0, len(supplierRows))
	for _, s := range supplierRows {
		supplierIDs = append(supplierIDs, s.id)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin consignment tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE suppliers SET is_consignment = true WHERE id = ANY($1)`, supplierIDs); err != nil {
		return fmt.Errorf("mark suppliers consignment: %w", err)
	}

	// Assign consigned SKUs to each supplier, globally deduped so no SKU is
	// owned by two consignment suppliers (consignment_stock UNIQUE product_id).
	assignments, err := assignConsignmentProducts(ctx, db, supplierRows, tx)
	if err != nil {
		return err
	}

	// Load retail prices for all consigned SKUs.
	allProductIDs := make([]int, 0)
	for _, a := range assignments {
		allProductIDs = append(allProductIDs, a.productIDs...)
	}
	prices, err := loadProductPrices(ctx, db, allProductIDs)
	if err != nil {
		return err
	}

	// Load payment methods for payouts.
	paymentMethodIDs := getIDs(ctx, db, "payment_methods")
	if len(paymentMethodIDs) == 0 {
		return fmt.Errorf("no payment methods found")
	}

	productTerm := map[int]consignmentSeedTerm{}
	firstReceiptAt := map[int]time.Time{}
	receivedQty := map[int]int{}
	arrangementByProduct := map[int]consignmentArrangementSeed{}

	// Lower bound for receipt dates: real sales history. On a fresh seed sales
	// only span [startDate, endDate]; on a re-seed (-truncate=false) they reach
	// back much further, so receipts can spread over the last 15-40 days and
	// still have sales after them for the consignment-sale backfill.
	salesStart := startDate
	var minSale time.Time
	if err := tx.QueryRowContext(ctx, `SELECT min(created_at) FROM sales`).Scan(&minSale); err == nil && !minSale.IsZero() {
		salesStart = minSale.In(jakartaTZ)
	}

	// --- Arrangements + terms + receipts ---
	receiptCount := 0
	for _, a := range assignments {
		arrangementID, err := insertConsignmentArrangement(ctx, tx, a.supplierID, storeID, createdBy, endDate)
		if err != nil {
			return err
		}

		// Terms for every consigned SKU.
		for _, pid := range a.productIDs {
			price := prices[pid]
			shareType := "percentage"
			shareValue := float64(15 + rand.Intn(16)) // 15-30%
			if rand.Intn(100) < 20 {
				shareType = "fixed_amount"
				shareValue = float64(500 + rand.Intn(9)*500) // 500-4500 IDR/unit
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO consignment_terms (arrangement_id, product_id, price, store_share_type, store_share_value, created_by)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (arrangement_id, product_id) DO NOTHING`,
				arrangementID, pid, price, shareType, shareValue, createdBy); err != nil {
				return fmt.Errorf("insert term for product %d: %w", pid, err)
			}
			productTerm[pid] = consignmentSeedTerm{price: price, storeShareType: shareType, storeShareValue: shareValue}
			arrangementByProduct[pid] = consignmentArrangementSeed{
				id: arrangementID, supplierID: a.supplierID, storeID: storeID,
			}
		}

		// 1-2 receipts per arrangement.
		numReceipts := 1
		if rand.Intn(100) < 65 && len(a.productIDs) >= 4 {
			numReceipts = 2
		}
		for r := 0; r < numReceipts; r++ {
			receivedAt := endDate.AddDate(0, 0, -rand.Intn(26)-15)
			if r == 1 {
				receivedAt = receivedAt.AddDate(0, 0, 5+rand.Intn(10))
			}
			// Never before the sales history (fresh seeds have no sales earlier
			// than the run window) and never after yesterday.
			if receivedAt.Before(salesStart) {
				receivedAt = salesStart.AddDate(0, 0, rand.Intn(5))
			}
			if receivedAt.After(endDate.AddDate(0, 0, -1)) {
				receivedAt = endDate.AddDate(0, 0, -1)
			}

			recNum, err := nextConsignmentDocNumber(ctx, tx, "consignment_receipt_seq", "CR")
			if err != nil {
				return err
			}
			var recID int
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO consignment_receipts (receipt_number, supplier_id, store_id, arrangement_id, received_by, received_at, notes)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id`,
				recNum, a.supplierID, storeID, arrangementID, createdBy, receivedAt,
				"Penerimaan konsinyasi").Scan(&recID); err != nil {
				return fmt.Errorf("insert receipt %s: %w", recNum, err)
			}

			for _, pid := range a.productIDs {
				term := productTerm[pid]
				accepted := 4 + rand.Intn(21) // 4-24 units
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO consignment_receipt_items (consignment_receipt_id, product_id, accepted_qty, price, store_share_type, store_share_value, notes)
					VALUES ($1, $2, $3, $4, $5, $6, NULL)`,
					recID, pid, accepted, term.price, term.storeShareType, term.storeShareValue); err != nil {
					return fmt.Errorf("insert receipt item: %w", err)
				}

				// Ownership ledger (mirrors Repository.UpsertConsignmentStock).
				var avail int
				if err := tx.QueryRowContext(ctx, `
					INSERT INTO consignment_stock (product_id, supplier_id, arrangement_id, store_id, available_qty, pending_return_qty)
					VALUES ($1, $2, $3, $4, GREATEST($5, 0), 0)
					ON CONFLICT (product_id) DO UPDATE
					SET supplier_id = EXCLUDED.supplier_id,
					    arrangement_id = EXCLUDED.arrangement_id,
					    store_id = EXCLUDED.store_id,
					    available_qty = consignment_stock.available_qty + $5,
					    updated_at = now()
					RETURNING available_qty`,
					pid, a.supplierID, arrangementID, storeID, accepted).Scan(&avail); err != nil {
					return fmt.Errorf("upsert consignment stock product %d: %w", pid, err)
				}

				// Sellable product_stock (Model A) + movement.
				if err := seedProductStockDelta(ctx, tx, pid, accepted, createdBy, "consignment_receipt", recID, "consignment_receipts", "consignment receipt "+recNum, receivedAt); err != nil {
					return err
				}

				if _, ok := firstReceiptAt[pid]; !ok {
					firstReceiptAt[pid] = receivedAt
				}
				receivedQty[pid] += accepted
			}
			receiptCount++
		}

		// Visit stays fresh so the arrangement is not lazily Ended.
		if _, err := tx.ExecContext(ctx,
			`UPDATE consignment_arrangements SET last_visit_at = $2, updated_at = now() WHERE id = $1`,
			arrangementID, endDate.AddDate(0, 0, -rand.Intn(3))); err != nil {
			return err
		}
	}

	// --- Backfill consignment_sale_items from real sales ---
	soldQty, err := backfillConsignmentSales(ctx, tx, allProductIDs, firstReceiptAt, receivedQty, productTerm, arrangementByProduct)
	if err != nil {
		return err
	}

	// --- Pending returns (open) + formal returns ---
	pendingReturnCount := 0
	returnCount := 0
	for _, a := range assignments {
		for _, pid := range a.productIDs {
			var avail int
			if err := tx.QueryRowContext(ctx,
				`SELECT available_qty FROM consignment_stock WHERE product_id = $1 FOR UPDATE`, pid).Scan(&avail); err != nil {
				continue
			}
			if avail <= 0 {
				continue
			}
			// ~40% of SKUs get an open pending return.
			if rand.Intn(100) >= 40 {
				continue
			}
			qty := 1 + rand.Intn(3)
			if qty > avail {
				qty = avail
			}
			reason := []string{"damaged", "expired", "customer_return"}[rand.Intn(3)]
			createdAt := endDate.AddDate(0, 0, -rand.Intn(5)-1)
			var prID int
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO consignment_pending_returns (supplier_id, product_id, arrangement_id, store_id, qty, reason, notes, status, created_by, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, NULL, 'open', $7, $8)
				RETURNING id`,
				a.supplierID, pid, arrangementByProduct[pid].id, storeID, qty, reason, createdBy, createdAt).Scan(&prID); err != nil {
				return fmt.Errorf("insert pending return: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
				UPDATE consignment_stock
				SET available_qty = available_qty - $2, pending_return_qty = pending_return_qty + $2, updated_at = now()
				WHERE product_id = $1`, pid, qty); err != nil {
				return err
			}
			if err := seedProductStockDelta(ctx, tx, pid, -qty, createdBy, "consignment_pending_return", prID, "consignment_pending_returns", "pending return "+reason, createdAt); err != nil {
				return err
			}
			pendingReturnCount++
		}

		// Formal return resolving that supplier's open pending returns (~60%).
		if rand.Intn(100) >= 60 {
			continue
		}
		rows, err := tx.QueryContext(ctx, `
			SELECT pr.id, pr.product_id, pr.qty
			FROM consignment_pending_returns pr
			WHERE pr.arrangement_id = $1 AND pr.status = 'open'
			ORDER BY pr.id ASC`, arrangementByProduct[a.productIDs[0]].id)
		if err != nil {
			return err
		}
		var openPRs []struct {
			id, productID, qty int
		}
		for rows.Next() {
			var pr struct{ id, productID, qty int }
			if err := rows.Scan(&pr.id, &pr.productID, &pr.qty); err == nil {
				openPRs = append(openPRs, pr)
			}
		}
		_ = rows.Close()
		if len(openPRs) == 0 {
			continue
		}

		retNum, err := nextConsignmentDocNumber(ctx, tx, "consignment_return_seq", "RT")
		if err != nil {
			return err
		}
		returnedAt := endDate.AddDate(0, 0, -rand.Intn(2))
		var retID int
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO consignment_returns (return_number, supplier_id, store_id, arrangement_id, returned_by, returned_at, notes)
			VALUES ($1, $2, $3, $4, $5, $6, NULL)
			RETURNING id`,
			retNum, a.supplierID, storeID, arrangementByProduct[a.productIDs[0]].id, createdBy, returnedAt).Scan(&retID); err != nil {
			return fmt.Errorf("insert return %s: %w", retNum, err)
		}
		for _, pr := range openPRs {
			reason := "damaged"
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO consignment_return_items (consignment_return_id, product_id, qty, reason, pending_return_id, notes)
				VALUES ($1, $2, $3, $4, $5, NULL)`,
				retID, pr.productID, pr.qty, reason, pr.id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				UPDATE consignment_pending_returns SET status = 'returned', returned_at = $2 WHERE id = $1`,
				pr.id, returnedAt); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `
				UPDATE consignment_stock SET pending_return_qty = pending_return_qty - $2, updated_at = now()
				WHERE product_id = $1`, pr.productID, pr.qty); err != nil {
				return err
			}
			if err := seedProductStockDelta(ctx, tx, pr.productID, -pr.qty, createdBy, "consignment_return", retID, "consignment_returns", "return "+retNum, returnedAt); err != nil {
				return err
			}
		}
		returnCount++
	}

	// --- Settlements + payouts ---
	settlementCount, err := seedConsignmentSettlements(ctx, tx, assignments, storeID, createdBy, endDate, paymentMethodIDs)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit consignment: %w", err)
	}

	fmt.Printf("   🎲 %d consignment suppliers flagged (is_consignment=true)\n", len(supplierRows))
	fmt.Printf("   🎲 %d arrangements opened across %d consigned SKUs\n", len(assignments), len(allProductIDs))
	fmt.Printf("   🎲 %d receipts (CR-), %d pending returns, %d formal returns (RT-)\n", receiptCount, pendingReturnCount, returnCount)
	fmt.Printf("   🎲 %d consignment units backfilled, %d settlements (CS-) with payouts (CP-)\n", soldQty, settlementCount)
	return nil
}

type consignmentSupplierRow struct {
	id   int
	name string
}

type consignmentArrangementSeed struct {
	id         int
	supplierID int
	storeID    int
}

type consignmentProductAssignment struct {
	supplierID   int
	supplierName string
	productIDs   []int
}

// findAllSuppliers returns all suppliers ordered by id ascending.
func findAllSuppliers(ctx context.Context, db *sql.DB) ([]consignmentSupplierRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, name FROM suppliers ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query all suppliers: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []consignmentSupplierRow
	for rows.Next() {
		var s consignmentSupplierRow
		if err := rows.Scan(&s.id, &s.name); err == nil {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

// filterSuppliersWithActiveArrangement drops suppliers that already have an
// active arrangement for the given store, so re-seeding does not collide with
// the single-active-arrangement-per-supplier+store invariant (BR-47).
func filterSuppliersWithActiveArrangement(ctx context.Context, db *sql.DB, rows []consignmentSupplierRow, storeID int) ([]consignmentSupplierRow, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.id)
	}
	active := map[int]bool{}
	q, err := db.QueryContext(ctx, `
		SELECT supplier_id FROM consignment_arrangements
		WHERE store_id = $1 AND status = 'active' AND supplier_id = ANY($2)`, storeID, ids)
	if err != nil {
		return nil, fmt.Errorf("query active arrangements: %w", err)
	}
	for q.Next() {
		var sid int
		if err := q.Scan(&sid); err == nil {
			active[sid] = true
		}
	}
	_ = q.Close()

	var out []consignmentSupplierRow
	for _, r := range rows {
		if !active[r.id] {
			out = append(out, r)
		}
	}
	return out, nil
}

// assignConsignmentProducts picks a deterministic subset of SKUs supplied by
// each consignment supplier (preferred links first). SKUs are globally
// deduped so each product is owned by exactly one consignment supplier.
func assignConsignmentProducts(ctx context.Context, db *sql.DB, suppliers []consignmentSupplierRow, tx *sql.Tx) ([]consignmentProductAssignment, error) {
	used := map[int]bool{}
	assignments := make([]consignmentProductAssignment, 0, len(suppliers))
	for _, s := range suppliers {
		rows, err := tx.QueryContext(ctx, `
			SELECT ps.product_id
			FROM product_suppliers ps
			WHERE ps.supplier_id = $1
			ORDER BY ps.is_preferred DESC, ps.product_id ASC`, s.id)
		if err != nil {
			return nil, fmt.Errorf("query supplier products: %w", err)
		}
		var ids []int
		for rows.Next() {
			var pid int
			if err := rows.Scan(&pid); err == nil {
				if !used[pid] {
					used[pid] = true
					ids = append(ids, pid)
				}
			}
		}
		_ = rows.Close()

		// Cap per-supplier SKUs to keep the demo focused.
		if len(ids) > 16 {
			ids = ids[:16]
		}
		if len(ids) == 0 {
			continue
		}
		assignments = append(assignments, consignmentProductAssignment{
			supplierID:   s.id,
			supplierName: s.name,
			productIDs:   ids,
		})
	}
	return assignments, nil
}

func loadProductPrices(ctx context.Context, db *sql.DB, ids []int) (map[int]int, error) {
	out := map[int]int{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id, price FROM products WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("query product prices: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, price int
		if err := rows.Scan(&id, &price); err == nil {
			out[id] = price
		}
	}
	return out, rows.Err()
}

func insertConsignmentArrangement(ctx context.Context, tx *sql.Tx, supplierID, storeID, createdBy int, endDate time.Time) (int, error) {
	createdAt := endDate.AddDate(0, 0, -rand.Intn(30)-5)
	lastVisit := endDate.AddDate(0, 0, -rand.Intn(3))
	var id int
	err := tx.QueryRowContext(ctx, `
		INSERT INTO consignment_arrangements (supplier_id, store_id, status, last_visit_at, created_by, created_at, updated_at)
		VALUES ($1, $2, 'active', $3, $4, $5, $5)
		RETURNING id`,
		supplierID, storeID, lastVisit, createdBy, createdAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert arrangement: %w", err)
	}
	return id, nil
}

// nextConsignmentDocNumber mirrors Repository.nextDocumentNumber: nextval on the
// sequence + current Jakarta year, keeping the seeded sequences in sync.
func nextConsignmentDocNumber(ctx context.Context, tx *sql.Tx, seq, prefix string) (string, error) {
	var seqValue int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT nextval('%s')`, seq)).Scan(&seqValue); err != nil {
		return "", fmt.Errorf("next %s number: %w", prefix, err)
	}
	return fmt.Sprintf("%s-%d-%06d", prefix, time.Now().In(jakartaTZ).Year(), seqValue), nil
}

// seedProductStockDelta adjusts the global product_stock row and appends an
// inventory_movements ledger entry, mirroring ConsignmentAdjuster.
func seedProductStockDelta(ctx context.Context, tx *sql.Tx, productID, delta, userID int, movementType string, referenceID int, referenceTable, notes string, at time.Time) error {
	if delta == 0 {
		return nil
	}
	var current int
	err := tx.QueryRowContext(ctx, `
		SELECT quantity FROM product_stock
		WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
		FOR UPDATE`, productID).Scan(&current)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("lock global stock: %w", err)
	}
	newQty := current + delta
	if newQty < 0 {
		newQty = 0
	}
	tag, err := tx.ExecContext(ctx, `
		UPDATE product_stock SET quantity = $1, updated_at = $2
		WHERE product_id = $3 AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL`,
		newQty, at, productID)
	if err != nil {
		return err
	}
	rowsAffected, err := tag.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, $3)`,
			productID, newQty, at); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		productID, delta, movementType, referenceID, referenceTable, userID, notes, at); err != nil {
		return err
	}
	return nil
}

// backfillConsignmentSales records consignment_sale_items from real sales of
// consigned SKUs that occurred at/after their first receipt, capped at the
// received quantity (older sales were store-owned). It also deducts the sold
// quantity from consignment_stock.available_qty. Returns total sold qty.
func backfillConsignmentSales(
	ctx context.Context,
	tx *sql.Tx,
	productIDs []int,
	firstReceiptAt map[int]time.Time,
	receivedQty map[int]int,
	productTerm map[int]consignmentSeedTerm,
	arrangementByProduct map[int]consignmentArrangementSeed,
) (int, error) {
	if len(productIDs) == 0 {
		return 0, nil
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT s.id, si.product_id, s.invoice_number, s.created_at, si.quantity, si.unit_price, si.subtotal
		FROM sale_items si
		JOIN sales s ON s.id = si.sale_id
		WHERE si.product_id = ANY($1)
		ORDER BY s.created_at ASC, si.id ASC`, productIDs)
	if err != nil {
		return 0, fmt.Errorf("query consignment sales: %w", err)
	}
	type saleRow struct {
		saleID, productID, qty, unitPrice, subtotal int
		invoice                                     string
		createdAt                                   time.Time
	}
	var sales []saleRow
	for rows.Next() {
		var sr saleRow
		if err := rows.Scan(&sr.saleID, &sr.productID, &sr.invoice, &sr.createdAt, &sr.qty, &sr.unitPrice, &sr.subtotal); err == nil {
			sales = append(sales, sr)
		}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	consumed := map[int]int{}
	totalSold := 0
	for _, sr := range sales {
		first, ok := firstReceiptAt[sr.productID]
		if !ok {
			continue
		}
		if sr.createdAt.Before(first) {
			continue
		}
		remaining := receivedQty[sr.productID] - consumed[sr.productID]
		if remaining <= 0 {
			continue
		}
		qty := sr.qty
		if qty > remaining {
			qty = remaining
		}
		ap := arrangementByProduct[sr.productID]
		term := productTerm[sr.productID]
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO consignment_sale_items
				(sale_id, invoice_number, product_id, supplier_id, arrangement_id, store_id,
				 quantity, unit_price, subtotal, store_share_type, store_share_value)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			sr.saleID, sr.invoice, sr.productID, ap.supplierID, ap.id, ap.storeID,
			qty, sr.unitPrice, sr.unitPrice*qty, term.storeShareType, term.storeShareValue); err != nil {
			return 0, fmt.Errorf("insert consignment sale item: %w", err)
		}
		consumed[sr.productID] += qty
		totalSold += qty
	}

	for pid, qty := range consumed {
		if _, err := tx.ExecContext(ctx, `
			UPDATE consignment_stock SET available_qty = available_qty - $2, updated_at = now()
			WHERE product_id = $1`, pid, qty); err != nil {
			return 0, err
		}
	}
	return totalSold, nil
}

// seedConsignmentSettlements settles ALL unsettled consignment sale items of
// each supplier (BR-41), computing store share like computeStoreShare. ~60% of
// settlements are marked paid and get a CP- payout.
func seedConsignmentSettlements(
	ctx context.Context,
	tx *sql.Tx,
	assignments []consignmentProductAssignment,
	storeID, createdBy int,
	endDate time.Time,
	paymentMethodIDs []int,
) (int, error) {
	settlementCount := 0
	for _, a := range assignments {
		rows, err := tx.QueryContext(ctx, `
			SELECT i.id, i.quantity, i.unit_price, i.subtotal, i.store_share_type, i.store_share_value, i.created_at
			FROM consignment_sale_items i
			WHERE i.supplier_id = $1 AND i.settlement_id IS NULL
			ORDER BY i.created_at ASC, i.id ASC`, a.supplierID)
		if err != nil {
			return 0, fmt.Errorf("query unsettled items: %w", err)
		}
		type itemRow struct {
			id, quantity, unitPrice, subtotal int
			shareType                         string
			shareValue                        float64
			createdAt                         time.Time
		}
		var items []itemRow
		for rows.Next() {
			var it itemRow
			if err := rows.Scan(&it.id, &it.quantity, &it.unitPrice, &it.subtotal, &it.shareType, &it.shareValue, &it.createdAt); err == nil {
				items = append(items, it)
			}
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return 0, err
		}
		if len(items) == 0 {
			continue
		}

		lastSaleAt := items[len(items)-1].createdAt
		createdAt := lastSaleAt.Add(time.Duration(1+rand.Intn(48)) * time.Hour)
		if createdAt.After(endDate) {
			createdAt = endDate.AddDate(0, 0, -1)
		}

		var totalSale, totalShare int
		itemIDs := make([]int, 0, len(items))
		for _, it := range items {
			totalSale += it.subtotal
			totalShare += seedStoreShare(it.unitPrice, it.quantity, it.shareType, it.shareValue)
			itemIDs = append(itemIDs, it.id)
		}
		settlementNumber, err := nextConsignmentDocNumber(ctx, tx, "consignment_settlement_seq", "CS")
		if err != nil {
			return 0, err
		}

		paid := rand.Intn(100) < 60
		status := "pending_payment"
		var paidAt sql.NullTime
		if paid {
			status = "paid"
			paidAt.Time = createdAt.Add(time.Duration(1+rand.Intn(72)) * time.Hour)
			if paidAt.Time.After(endDate) {
				paidAt.Time = endDate.AddDate(0, 0, -1)
			}
			paidAt.Valid = true
		}

		var settlementID int
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO consignment_settlements (settlement_number, supplier_id, store_id, total_sale_value, total_store_share, total_payable, status, created_by, created_at, paid_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id`,
			settlementNumber, a.supplierID, storeID, totalSale, totalShare, totalSale-totalShare,
			status, createdBy, createdAt, paidAt).Scan(&settlementID); err != nil {
			return 0, fmt.Errorf("insert settlement %s: %w", settlementNumber, err)
		}

		for _, it := range items {
			share := seedStoreShare(it.unitPrice, it.quantity, it.shareType, it.shareValue)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO consignment_settlement_items (consignment_settlement_id, consignment_sale_item_id, product_id, quantity, unit_price, subtotal, store_share)
				VALUES ($1, $2, NULL, $3, $4, $5, $6)`,
				settlementID, it.id, it.quantity, it.unitPrice, it.subtotal, share); err != nil {
				return 0, err
			}
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE consignment_sale_items SET settlement_id = $1 WHERE id = ANY($2)`,
			settlementID, itemIDs); err != nil {
			return 0, err
		}

		if paid {
			payoutNumber, err := nextConsignmentDocNumber(ctx, tx, "consignment_payout_seq", "CP")
			if err != nil {
				return 0, err
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO consignment_payouts (payout_number, settlement_id, payment_method_id, amount, reference_number, paid_by, paid_at, notes)
				VALUES ($1, $2, $3, $4, $5, $6, $7, NULL)`,
				payoutNumber, settlementID, paymentMethodIDs[rand.Intn(len(paymentMethodIDs))],
				totalSale-totalShare,
				fmt.Sprintf("TRF-%d", 100000+rand.Intn(900000)), createdBy, paidAt.Time); err != nil {
				return 0, fmt.Errorf("insert payout %s: %w", payoutNumber, err)
			}
		}
		settlementCount++
	}
	return settlementCount, nil
}

// seedStoreShare mirrors internal/consignment computeStoreShare.
func seedStoreShare(unitPrice, quantity int, shareType string, shareValue float64) int {
	if shareType == "percentage" {
		return int(float64(unitPrice) * float64(quantity) * shareValue / 100.0)
	}
	return int(shareValue) * quantity
}

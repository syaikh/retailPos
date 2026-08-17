package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// injectStockOpnames seeds realistic stock opname sessions across the date range.
//
// Real-world distribution:
//   - Historical sessions are mostly completed (posted + closed, with full count
//     history, expected qty, differences, adjustment ledger documents and
//     inventory movements); a minority are left "posted" but not yet closed, and
//     ~20% cancelled.
//   - The most recent session is left in an active "counting" state with partial
//     counts so the module has an in-progress case to demo.
//
// Overlap is enforced per scope at creation, and the newest session is left in
// a "counting" state so the module has an in-progress case to demo.
func injectStockOpnames(ctx context.Context, db *sql.DB, startDate, endDate time.Time, numSessions int) error {
	// Pick counters (cashier/staff) and approvers/managers.
	counterUserIDs := getUserIDsByRoles(ctx, db, "cashier", "staff")
	managerUserIDs := getUserIDsByRoles(ctx, db, "superadmin", "admin", "manager")
	if len(counterUserIDs) == 0 {
		counterUserIDs = getUserIDsByRoles(ctx, db, "superadmin", "admin", "manager", "cashier", "staff")
	}
	if len(managerUserIDs) == 0 {
		managerUserIDs = append(managerUserIDs, counterUserIDs...)
	}
	if len(counterUserIDs) == 0 || len(managerUserIDs) == 0 {
		return fmt.Errorf("no users available for stock opname seeding")
	}

	// Default session count: roughly one per month over the date range.
	if numSessions <= 0 {
		numSessions = int(endDate.Sub(startDate).Hours()/24/30) + 1
		if numSessions < 3 {
			numSessions = 3
		}
		if numSessions > 10 {
			numSessions = 10
		}
	}

	// Sync so_seq so session numbers never collide with existing data.
	// On an empty DB (maxSeq == 0) the value is marked unused so the next
	// nextval() returns 1; otherwise nextval() continues from maxSeq+1.
	var maxSeq int
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(REGEXP_REPLACE(session_number, '^SO-[0-9]+-0*', '') AS bigint)), 0)
		FROM stock_opnames
		WHERE session_number ~ '^SO-[0-9]+-[0-9]+$'
	`).Scan(&maxSeq); err != nil {
		return fmt.Errorf("read max stock opname seq: %w", err)
	}
	if maxSeq == 0 {
		if _, err := db.ExecContext(ctx, `SELECT setval('so_seq', 1, false)`); err != nil {
			return fmt.Errorf("sync so_seq: %w", err)
		}
	} else if _, err := db.ExecContext(ctx, `SELECT setval('so_seq', $1)`, maxSeq); err != nil {
		return fmt.Errorf("sync so_seq: %w", err)
	}

	storeIDs := getIDs(ctx, db, "stores")
	categoryIDs := getIDs(ctx, db, "categories")

	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1
	if totalDays < 1 {
		totalDays = 1
	}

	ref := endDate.In(jakartaTZ)

	// Snapshot products once; physical counts and stock adjustments are derived
	// from it, so re-reading per session keeps realism without extra queries.
	products, err := loadStockOpnameSnapshot(ctx, db)
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return fmt.Errorf("no products in general stock to snapshot")
	}

	sessionsCreated := 0
	itemsCreated := 0
	countsCreated := 0
	movementsCreated := 0
	adjustmentsCreated := 0

	for i := 0; i < numSessions; i++ {
		// Spread session dates across the range, oldest first.
		dayOffset := (i * (totalDays - 1)) / numSessions
		if dayOffset > totalDays-1 {
			dayOffset = totalDays - 1
		}
		createdAt := startDate.AddDate(0, 0, dayOffset)
		createdAt = createdAt.Add(time.Duration(8*3600+rand.Intn(8*3600)) * time.Second)
		if createdAt.After(ref) {
			createdAt = ref.Add(-time.Duration(3600+rand.Intn(2*3600)) * time.Second)
		}

		// Only the most recent session is left active (for the in-progress demo).
		isMostRecent := i == numSessions-1
		status := "posted"
		if isMostRecent {
			status = "counting"
		} else if rand.Intn(100) < 20 {
			status = "cancelled"
		}
		isClosed := status == "posted" && rand.Intn(100) < 70
		if isClosed {
			status = "closed"
		}

		createdBy := managerUserIDs[rand.Intn(len(managerUserIDs))]

		scopeType := "store"
		scopeID := 1
		var warehouseID any
		switch r := rand.Intn(100); {
		case r < 60:
			if len(storeIDs) > 0 {
				scopeID = storeIDs[rand.Intn(len(storeIDs))]
			}
		case r < 80:
			scopeType = "category"
			if len(categoryIDs) > 0 {
				scopeID = categoryIDs[rand.Intn(len(categoryIDs))]
			}
		case r < 90:
			scopeType = "warehouse"
			scopeID = 1 + rand.Intn(3)
			warehouseID = scopeID
		default:
			scopeType = "product"
			scopeID = 1 + rand.Intn(1000)
		}

		blindCount := rand.Intn(100) < 25

		var seq int
		if err := db.QueryRowContext(ctx, `SELECT nextval('so_seq')`).Scan(&seq); err != nil {
			return fmt.Errorf("nextval so_seq: %w", err)
		}
		sessionNumber := fmt.Sprintf("SO-%d-%06d", createdAt.Year(), seq)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin stock opname tx: %w", err)
		}

		var sessionID int
		err = tx.QueryRowContext(ctx, `
			INSERT INTO stock_opnames
				(session_number, scope_type, scope_id, warehouse_id, blind_count, status, created_by, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)
			RETURNING id
		`, sessionNumber, scopeType, scopeID, warehouseID, blindCount, status, createdBy, createdAt).Scan(&sessionID)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert stock opname session: %w", err)
		}

		// Populate the extensible scopes table so the UI renders the session scope.
		var scopeName string
		if scopeType == "store" {
			_ = db.QueryRowContext(ctx, `
				SELECT COALESCE(name, '') FROM stores WHERE id = $1
			`, scopeID).Scan(&scopeName)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO stock_opname_session_scopes (stock_opname_id, scope_type, scope_id, scope_name)
			VALUES ($1,$2,$3,$4)
		`, sessionID, scopeType, scopeID, scopeName); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert stock opname scope: %w", err)
		}

		// Assignments: one supervisor + 1-3 counters.
		supervisorID := managerUserIDs[rand.Intn(len(managerUserIDs))]
		shuffledCounters := make([]int, len(counterUserIDs))
		copy(shuffledCounters, counterUserIDs)
		rand.Shuffle(len(shuffledCounters), func(i, j int) { shuffledCounters[i], shuffledCounters[j] = shuffledCounters[j], shuffledCounters[i] })
		numCounters := 1 + rand.Intn(3)
		if numCounters > len(shuffledCounters) {
			numCounters = len(shuffledCounters)
		}
		assignedAt := createdAt.Add(time.Duration(rand.Intn(3600)+600) * time.Second)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO stock_opname_assignments (stock_opname_id, user_id, role, assigned_at)
			VALUES ($1,$2,'supervisor',$3)
		`, sessionID, supervisorID, assignedAt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert supervisor assignment: %w", err)
		}
		for c := 0; c < numCounters; c++ {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO stock_opname_assignments (stock_opname_id, user_id, role, assigned_at)
				VALUES ($1,$2,'counter',$3)
			`, sessionID, shuffledCounters[c], assignedAt.Add(time.Duration(c+1)*time.Minute)); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert counter assignment: %w", err)
			}
		}

		var counterIdx int
		var approverID = managerUserIDs[rand.Intn(len(managerUserIDs))]
		var totalDiff, totalAdj float64

		// Collect item data first, then batch-insert to minimize round-trips.
		type itemSeed struct {
			productID  int
			name       string
			sku        string
			barcode    string
			uom        string
			qty        float64
			physical   float64
			diff       float64
			itemStatus string
			countedBy  int
			countedAt  time.Time
			remarks    string
		}
		var itemSeeds []itemSeed

		for _, p := range products {
			var physical float64
			var diff float64
			itemStatus := "pending"

			switch status {
			case "posted", "closed":
				physical = p.qty
				r := rand.Intn(100)
				if r >= 95 { // 5% large discrepancy
					delta := float64(4 + rand.Intn(8))
					if rand.Intn(2) == 0 {
						delta = -delta
					}
					physical = p.qty + delta
					diff = delta
				} else if r >= 85 { // 10% small discrepancy
					delta := float64(1 + rand.Intn(3))
					if rand.Intn(2) == 0 {
						delta = -delta
					}
					physical = p.qty + delta
					diff = delta
				}
				if physical < 0 {
					physical = 0
					diff = -p.qty
				}
				itemStatus = "counted"

			case "counting":
				ratio := 0.45 + rand.Float64()*0.30
				if rand.Float64() < ratio {
					delta := float64(rand.Intn(3) - 1)
					physical = p.qty + delta
					if physical < 0 {
						physical = 0
					}
					itemStatus = "counted"
				}

			default: // cancelled — no counts
				physical = 0
			}

			remarks := ""
			countedBy := 0
			var countedAt time.Time
			if itemStatus == "counted" {
				countedBy = shuffledCounters[counterIdx%numCounters]
				counterIdx++
				if diff != 0 {
					remarks = stockOpnameRemarks()
				}
				countedAt = createdAt.Add(time.Duration(rand.Intn(6*3600)+3600) * time.Second)
			}

			itemSeeds = append(itemSeeds, itemSeed{
				productID: p.productID, name: p.name, sku: p.sku, barcode: p.barcode,
				uom: p.uom, qty: p.qty, physical: physical, diff: diff,
				itemStatus: itemStatus, countedBy: countedBy, countedAt: countedAt, remarks: remarks,
			})
		}

		// --- Batch 1: Insert items (chunks of 500) ---
		const itemChunk = 500
		type itemRow struct {
			id         int
			productID  int
			physical   float64
			diff       float64
			itemStatus string
			countedBy  int
			countedAt  time.Time
			remarks    string
		}
		var allItems []itemRow

		for start := 0; start < len(itemSeeds); start += itemChunk {
			end := start + itemChunk
			if end > len(itemSeeds) {
				end = len(itemSeeds)
			}
			chunk := itemSeeds[start:end]

			var sb strings.Builder
			sb.WriteString(`INSERT INTO stock_opname_items
				(stock_opname_id, product_id, opening_qty, expected_qty, physical_qty,
				 difference_qty, adjustment_qty, status, product_name, sku, barcode, uom_name, created_at, updated_at)
			VALUES `)
			args := make([]interface{}, 0, len(chunk)*14)
			for j, is := range chunk {
				if j > 0 {
					sb.WriteString(", ")
				}
				p := len(args)
				args = append(args, sessionID, is.productID, is.qty,
					0.0, is.physical, 0.0, 0.0, is.itemStatus, is.name, is.sku, is.barcode, is.uom, createdAt, createdAt)
				sb.WriteString(fmt.Sprintf("($%d::int,$%d::int,$%d::numeric,$%d::numeric,$%d::numeric,$%d::numeric,$%d::numeric,$%d::varchar,$%d::varchar,$%d::varchar,$%d::varchar,$%d::varchar,$%d::timestamptz,$%d::timestamptz)",
					p+1, p+2, p+3, p+4, p+5, p+6, p+7, p+8, p+9, p+10, p+11, p+12, p+13, p+14))
			}
			sb.WriteString(` RETURNING id, product_id`)

			rows, err := tx.QueryContext(ctx, sb.String(), args...)
			if err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("batch insert stock opname items: %w", err)
			}
			idx := 0
			for rows.Next() {
				var ir itemRow
				if err := rows.Scan(&ir.id, &ir.productID); err != nil {
					_ = rows.Close()
					_ = tx.Rollback()
					return fmt.Errorf("scan stock opname item id: %w", err)
				}
				ir.physical = chunk[idx].physical
				ir.diff = chunk[idx].diff
				ir.itemStatus = chunk[idx].itemStatus
				ir.countedBy = chunk[idx].countedBy
				ir.countedAt = chunk[idx].countedAt
				ir.remarks = chunk[idx].remarks
				allItems = append(allItems, ir)
				idx++
			}
			_ = rows.Close()
			if err := rows.Err(); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("rows err stock opname items: %w", err)
			}
			itemsCreated += len(chunk)
		}

		// --- Batch 2: Insert counts (chunks of 500) ---
		type countSeed struct {
			itemID    int
			physical  float64
			countedBy int
			countedAt time.Time
			remarks   string
		}
		var countSeeds []countSeed
		for _, ir := range allItems {
			if ir.itemStatus != "counted" {
				continue
			}
			countSeeds = append(countSeeds, countSeed{
				itemID: ir.id, physical: ir.physical, countedBy: ir.countedBy,
				countedAt: ir.countedAt, remarks: ir.remarks,
			})
		}

		const countChunk = 500
		for start := 0; start < len(countSeeds); start += countChunk {
			end := start + countChunk
			if end > len(countSeeds) {
				end = len(countSeeds)
			}
			chunk := countSeeds[start:end]

			var sb strings.Builder
			sb.WriteString(`INSERT INTO stock_opname_counts
				(stock_opname_item_id, count_sequence, physical_qty, counted_by, counted_at, remarks)
			VALUES `)
			args := make([]interface{}, 0, len(chunk)*6)
			for j, cs := range chunk {
				if j > 0 {
					sb.WriteString(", ")
				}
				p := len(args)
				args = append(args, cs.itemID, cs.physical, cs.countedBy, cs.countedAt, cs.remarks)
				sb.WriteString(fmt.Sprintf("($%d, 1, $%d, $%d, $%d, $%d)", p+1, p+2, p+3, p+4, p+5))
			}
			if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("batch insert stock opname counts: %w", err)
			}
			countsCreated += len(chunk)
		}

		if status != "posted" && status != "closed" {
			// Counting/cancelled: set terminal state and commit.
			switch status {
			case "counting":
				openedAt := createdAt.Add(time.Duration(rand.Intn(3600)+600) * time.Second)
				if _, err := tx.ExecContext(ctx, `
					UPDATE stock_opnames SET opened_by = $1, opened_at = $2, updated_at = $2 WHERE id = $3
				`, approverID, openedAt, sessionID); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("set opened session: %w", err)
				}
			case "cancelled":
				cancelledAt := createdAt.Add(time.Duration(rand.Intn(6*3600)+1800) * time.Second)
				if _, err := tx.ExecContext(ctx, `
					UPDATE stock_opnames SET cancelled_at = $1, updated_at = $1 WHERE id = $2
				`, cancelledAt, sessionID); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("set cancelled session: %w", err)
				}
			}
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("commit stock opname tx: %w", err)
			}
			sessionsCreated++
			continue
		}

		// --- Batch 3: Update items with expected/diff (posted/closed) ---
		type updateSeed struct {
			itemID   int
			expected float64
			diff     float64
		}
		var updateSeeds []updateSeed
		for _, ir := range allItems {
			if ir.itemStatus != "counted" {
				continue
			}
			updateSeeds = append(updateSeeds, updateSeed{itemID: ir.id, expected: ir.physical - ir.diff, diff: ir.diff})
			totalDiff += ir.diff
			totalAdj += ir.diff
		}

		const updateChunk = 500
		for start := 0; start < len(updateSeeds); start += updateChunk {
			end := start + updateChunk
			if end > len(updateSeeds) {
				end = len(updateSeeds)
			}
			chunk := updateSeeds[start:end]

			var sb strings.Builder
			sb.WriteString(`UPDATE stock_opname_items SET
				expected_qty = v.expected_qty, difference_qty = v.difference_qty,
				adjustment_qty = v.difference_qty, updated_at = NOW()
				FROM (VALUES `)
			args := make([]interface{}, 0, len(chunk)*3)
			for j, us := range chunk {
				if j > 0 {
					sb.WriteString(", ")
				}
				p := len(args)
				args = append(args, us.itemID, us.expected, us.diff)
				sb.WriteString(fmt.Sprintf("($%d::int, $%d::numeric, $%d::numeric)", p+1, p+2, p+3))
			}
			sb.WriteString(`) AS v(id, expected_qty, difference_qty)
			WHERE stock_opname_items.id = v.id`)
			if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("batch update stock opname items: %w", err)
			}
		}

		// --- Batch 4: Update product_stock + insert movements ---
		type stockDelta struct {
			productID int
			newQty    int
			diff      int
			notes     string
		}
		var stockDeltas []stockDelta
		for _, ir := range allItems {
			if ir.itemStatus != "counted" || ir.diff == 0 {
				continue
			}
			notes := fmt.Sprintf("Stock opname %s: physical %.2f vs expected %.2f", sessionNumber, ir.physical, ir.physical-ir.diff)
			stockDeltas = append(stockDeltas, stockDelta{
				productID: ir.productID, newQty: int(ir.physical), diff: int(ir.diff), notes: notes,
			})
		}

		if len(stockDeltas) > 0 {
			// Batch product_stock updates
			const stockChunk = 500
			for start := 0; start < len(stockDeltas); start += stockChunk {
				end := start + stockChunk
				if end > len(stockDeltas) {
					end = len(stockDeltas)
				}
				chunk := stockDeltas[start:end]

				var sb strings.Builder
				sb.WriteString(`INSERT INTO product_stock (product_id, quantity, updated_at) VALUES `)
				args := make([]interface{}, 0, len(chunk)*2)
				for j, sd := range chunk {
					if j > 0 {
						sb.WriteString(", ")
					}
					p := len(args)
					args = append(args, sd.productID, sd.newQty)
					sb.WriteString(fmt.Sprintf("($%d, $%d, NOW())", p+1, p+2))
				}
				sb.WriteString(` ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET
					quantity = EXCLUDED.quantity, updated_at = NOW()`)
				if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("batch update product_stock: %w", err)
				}
			}

			// Batch inventory_movements inserts
			const movChunk = 500
			for start := 0; start < len(stockDeltas); start += movChunk {
				end := start + movChunk
				if end > len(stockDeltas) {
					end = len(stockDeltas)
				}
				chunk := stockDeltas[start:end]

				var sb strings.Builder
				sb.WriteString(`INSERT INTO inventory_movements
					(product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
				VALUES `)
				args := make([]interface{}, 0, len(chunk)*6)
				for j, sd := range chunk {
					movementAt := createdAt.Add(time.Duration(rand.Intn(12*3600)+8*3600) * time.Second)
					if j > 0 {
						sb.WriteString(", ")
					}
					p := len(args)
					args = append(args, sd.productID, sd.diff, sessionID, approverID, sd.notes, movementAt)
					sb.WriteString(fmt.Sprintf("($%d, $%d, 'stock_opname', $%d, 'stock_opnames', $%d, $%d, $%d)", p+1, p+2, p+3, p+4, p+5, p+6))
				}
				if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("batch insert inventory_movements: %w", err)
				}
				movementsCreated += len(chunk)
			}
		}

		// Terminal state fields (same as before).
		postedAt := createdAt.Add(time.Duration(rand.Intn(24*3600)+24*3600) * time.Second)
		verifiedAt := postedAt.Add(-time.Duration(1800+rand.Intn(3600)) * time.Second)
		openedAt := createdAt.Add(time.Duration(rand.Intn(3600)+600) * time.Second)
		if _, err := tx.ExecContext(ctx, `
			UPDATE stock_opnames
			SET opened_by = $1, opened_at = $2,
			    verified_by = $1, verified_at = $3,
			    posted_by = $1, posted_at = $4,
			    total_difference = $5, total_adjustment = $6,
			    updated_at = $4
			WHERE id = $7
		`, approverID, openedAt, verifiedAt, postedAt, totalDiff, totalAdj, sessionID); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("set posted session: %w", err)
		}

		// Collect adjustment items from the batch data.
		var adjustmentItems []adjustmentSeedLine
		for _, ir := range allItems {
			if ir.itemStatus == "counted" && ir.diff != 0 {
				adjustmentItems = append(adjustmentItems, adjustmentSeedLine{
					productID: ir.productID,
					expected:  ir.physical - ir.diff,
					physical:  ir.physical,
					diff:      ir.diff,
					reason:    fmt.Sprintf("Stock opname %s: physical %.2f vs expected %.2f", sessionNumber, ir.physical, ir.physical-ir.diff),
				})
			}
		}

		if len(adjustmentItems) > 0 {
			var iaSeq int
			if err := tx.QueryRowContext(ctx, `SELECT nextval('ia_seq')`).Scan(&iaSeq); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("nextval ia_seq: %w", err)
			}
			adjNumber := fmt.Sprintf("IA-%d-%06d", createdAt.Year(), iaSeq)
			adjNotes := fmt.Sprintf("Posted from stock opname %s", sessionNumber)
			var adjustmentID int
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO inventory_adjustments (adjustment_number, session_id, status, notes, created_by, created_at)
				VALUES ($1,$2,'posted',$3,$4,$5)
				RETURNING id
			`, adjNumber, sessionID, adjNotes, approverID, postedAt).Scan(&adjustmentID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert inventory adjustment: %w", err)
			}
			adjustmentsCreated++

			// Batch adjustment items
			const adjChunk = 500
			for start := 0; start < len(adjustmentItems); start += adjChunk {
				end := start + adjChunk
				if end > len(adjustmentItems) {
					end = len(adjustmentItems)
				}
				chunk := adjustmentItems[start:end]

				var sb strings.Builder
				sb.WriteString(`INSERT INTO inventory_adjustment_items
					(adjustment_id, product_id, warehouse_id, store_id, expected_qty, physical_qty,
					 difference_qty, adjustment_qty, unit_cost, line_total, reason, created_at)
				VALUES `)
				args := make([]interface{}, 0, len(chunk)*7)
				for j, line := range chunk {
					if j > 0 {
						sb.WriteString(", ")
					}
					p := len(args)
					args = append(args, adjustmentID, line.productID, line.expected, line.physical, line.diff, line.reason, postedAt)
					sb.WriteString(fmt.Sprintf("($%d,$%d,NULL,NULL,$%d,$%d,$%d,$%d,0,0,$%d,$%d)", p+1, p+2, p+3, p+4, p+5, p+5, p+6, p+7))
				}
				if _, err := tx.ExecContext(ctx, sb.String(), args...); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("batch insert inventory adjustment items: %w", err)
				}
			}
		}

		if status == "closed" {
			closedAt := postedAt.Add(time.Duration(rand.Intn(3600)+300) * time.Second)
			if _, err := tx.ExecContext(ctx, `
				UPDATE stock_opnames SET closed_by = $1, closed_at = $2, updated_at = $2 WHERE id = $3
			`, approverID, closedAt, sessionID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("set closed session: %w", err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit stock opname tx: %w", err)
		}

		sessionsCreated++
	}

	fmt.Printf("   🎲 Created %d stock opname sessions (%d items, %d counts, %d stock movements, %d adjustments)\n",
		sessionsCreated, itemsCreated, countsCreated, movementsCreated, adjustmentsCreated)
	return nil
}

// adjustmentSeedLine is one posted adjustment line (per SKU) for a completed session.
type adjustmentSeedLine struct {
	productID int
	expected  float64
	physical  float64
	diff      float64
	reason    string
}

// stockOpnameSnapshot mirrors the app's LoadSnapshotProducts query.
type stockOpnameSnapshot struct {
	productID int
	name      string
	sku       string
	barcode   string
	uom       string
	qty       float64
}

func loadStockOpnameSnapshot(ctx context.Context, db *sql.DB) ([]stockOpnameSnapshot, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ps.product_id, p.name, COALESCE(p.sku, ''), COALESCE(p.barcode, ''),
		       COALESCE(u.name, 'pcs'), COALESCE(ps.quantity, 0)
		FROM product_stock ps
		JOIN products p ON p.id = ps.product_id
		LEFT JOIN units_of_measure u ON u.id = p.unit_of_measure_id
		WHERE ps.warehouse_id IS NULL AND ps.store_id IS NULL
		  AND p.status = 'active' AND p.deleted_at IS NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM consignment_stock cs WHERE cs.product_id = ps.product_id
		  )
		ORDER BY p.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query stock opname snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []stockOpnameSnapshot
	for rows.Next() {
		var s stockOpnameSnapshot
		var qty sql.NullFloat64
		if err := rows.Scan(&s.productID, &s.name, &s.sku, &s.barcode, &s.uom, &qty); err != nil {
			return nil, fmt.Errorf("scan stock opname snapshot: %w", err)
		}
		if qty.Valid {
			s.qty = qty.Float64
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// getUserIDsByRoles returns active user IDs whose role name is in roles.
func getUserIDsByRoles(ctx context.Context, db *sql.DB, roles ...string) []int {
	if len(roles) == 0 {
		return nil
	}
	placeholders := make([]string, len(roles))
	args := make([]interface{}, len(roles))
	for i, r := range roles {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = r
	}
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT u.id
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.is_active = true AND r.name IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func stockOpnameRemarks() string {
	remarks := []string{
		"Rusak ringan",
		"Barang terselip di rak",
		"Selisih stok fisik",
		"Barang hilang",
		"Kesalahan pencatatan",
		"Kemasan rusak",
	}
	return remarks[rand.Intn(len(remarks))]
}

// placeholder to keep log import used when seeder logging needs tweaks
var _ = log.Printf

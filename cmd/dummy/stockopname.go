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
//   - Historical sessions are mostly approved (with full count history, expected
//     qty, differences, adjustments and inventory movements), ~20% cancelled.
//   - The most recent session is left in an active "counting" state with partial
//     counts so the module has an in-progress case to demo.
//
// Because BR-001 enforces a single active session globally, only the newest
// session may use a non-terminal status.
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
	// setval(..., false) marks the value as unused, so the next nextval() returns max+1.
	// When empty, set it back to 1 so a truncate produces SO-000001 again (so_seq has
	// a minimum value of 1, so setval(..., 0) is invalid).
	var maxSeq int
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(REGEXP_REPLACE(session_number, '^SO-[0-9]+-0*', '') AS bigint)), 0)
		FROM stock_opnames
		WHERE session_number ~ '^SO-[0-9]+-[0-9]+$'
	`).Scan(&maxSeq); err != nil {
		return fmt.Errorf("read max stock opname seq: %w", err)
	}
	if maxSeq == 0 {
		maxSeq = 1
	}
	if _, err := db.ExecContext(ctx, `SELECT setval('so_seq', $1, false)`, maxSeq); err != nil {
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

		// Only the most recent session may be active (BR-001 single active session).
		isMostRecent := i == numSessions-1
		status := "approved"
		if isMostRecent {
			status = "counting"
		} else if rand.Intn(100) < 20 {
			status = "cancelled"
		}

		createdBy := managerUserIDs[rand.Intn(len(managerUserIDs))]

		scopeType := "store"
		scopeID := 1
		var warehouseID any = nil
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

		itemStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO stock_opname_items
				(stock_opname_id, product_id, opening_qty, expected_qty, physical_qty,
				 difference_qty, adjustment_qty, status, product_name, sku, barcode, uom_name, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)
			RETURNING id
		`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("prepare stock opname item stmt: %w", err)
		}
		defer itemStmt.Close()

		countStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO stock_opname_counts
				(stock_opname_item_id, count_sequence, physical_qty, counted_by, counted_at, remarks)
			VALUES ($1, 1, $2, $3, $4, $5)
		`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("prepare stock opname count stmt: %w", err)
		}
		defer countStmt.Close()

		stockStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (product_id, warehouse_id, store_id) DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = NOW()
		`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("prepare stock update stmt: %w", err)
		}
		defer stockStmt.Close()

		productStockSyncStmt, err := tx.PrepareContext(ctx, `
			UPDATE products SET stock = $1, updated_at = NOW() WHERE id = $2
		`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("prepare product stock sync stmt: %w", err)
		}
		defer productStockSyncStmt.Close()

		movementStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO inventory_movements
				(product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
			VALUES ($1, $2, 'stock_opname', $3, 'stock_opnames', $4, $5, $6)
		`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("prepare inventory movement stmt: %w", err)
		}
		defer movementStmt.Close()

		var counterIdx int
		var approverID = managerUserIDs[rand.Intn(len(managerUserIDs))]

		for _, p := range products {
			var physical float64
			var diff float64
			itemStatus := "pending"

			switch status {
			case "approved":
				// All items counted; ~85% match expected, 10% small discrepancy,
				// 5% larger discrepancy (shrinkage / damage / overcount).
				physical = p.qty
				r := rand.Intn(100)
				if r >= 85 {
					delta := float64(1 + rand.Intn(3))
					if rand.Intn(2) == 0 {
						delta = -delta
					}
					physical = p.qty + delta
					diff = delta
				} else if r >= 95 {
					delta := float64(4 + rand.Intn(8))
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
				// Active session: 45-75% of items counted so far.
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

			var itemID int
			if err := itemStmt.QueryRowContext(ctx, sessionID, p.productID, p.qty,
				0, physical, 0, 0, itemStatus, p.name, p.sku, p.barcode, p.uom, createdAt,
			).Scan(&itemID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert stock opname item: %w", err)
			}
			itemsCreated++

			if itemStatus != "counted" {
				continue
			}

			// Count record (single pass). approved sessions get expected/diff set below.
			countedBy := shuffledCounters[counterIdx%numCounters]
			counterIdx++
			remarks := ""
			if diff != 0 {
				remarks = stockOpnameRemarks()
			}
			countedAt := createdAt.Add(time.Duration(rand.Intn(6*3600)+3600) * time.Second)
			if _, err := countStmt.ExecContext(ctx, itemID, physical, countedBy, countedAt, remarks); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert stock opname count: %w", err)
			}
			countsCreated++

			if status != "approved" {
				continue
			}

			// Approved: set expected/diff/adjustment on every counted item (mirrors
			// Service.ApproveSession). Expected qty is the snapshot stock at session time.
			expected := p.qty
			if _, err := tx.ExecContext(ctx, `
				UPDATE stock_opname_items
				SET expected_qty = $2, difference_qty = $3, adjustment_qty = $3, updated_at = NOW()
				WHERE id = $1
			`, itemID, expected, diff); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("update stock opname item adjustment: %w", err)
			}

			if diff == 0 {
				continue
			}

			// Approved with discrepancy: correct stock to physical and record a movement.
			newQty := int(physical)
			if _, err := stockStmt.ExecContext(ctx, p.productID, newQty); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("update product stock: %w", err)
			}
			if _, err := productStockSyncStmt.ExecContext(ctx, newQty, p.productID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("sync products.stock: %w", err)
			}

			notes := fmt.Sprintf("Stock opname %s: physical %.2f vs expected %.2f", sessionNumber, physical, expected)
			movementAt := createdAt.Add(time.Duration(rand.Intn(12*3600)+8*3600) * time.Second)
			if _, err := movementStmt.ExecContext(ctx, p.productID, int(diff), sessionID, approverID, notes, movementAt); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert inventory movement: %w", err)
			}
			movementsCreated++
		}

		// Terminal state fields.
		switch status {
		case "approved":
			approvedAt := createdAt.Add(time.Duration(rand.Intn(24*3600)+24*3600) * time.Second)
			if _, err := tx.ExecContext(ctx, `
				UPDATE stock_opnames SET approved_by = $1, approved_at = $2, updated_at = $2 WHERE id = $3
			`, approverID, approvedAt, sessionID); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("set approved session: %w", err)
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
	}

	fmt.Printf("   🎲 Created %d stock opname sessions (%d items, %d counts, %d stock movements)\n",
		sessionsCreated, itemsCreated, countsCreated, movementsCreated)
	return nil
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
		ORDER BY p.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query stock opname snapshot: %w", err)
	}
	defer rows.Close()

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
	defer rows.Close()

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

package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"retail-pos-system/internal/shared"
)

// StockApplier is the inventory-owned implementation of the stock opname
// module's StockApplier port (structural typing — no import of internal/
// stockopname needed). internal/inventory is the canonical single-writer of
// product_stock (ADR_Modular_Monolith_Module_Boundaries §2.8), so the product
// stock writes performed on stock opname posting live here rather than inside
// internal/stockopname.
//
// The stock opname posting path batches all of a session's stock writes into a
// fixed number of statements (SetProductStocks / ReconcileLocationStocks)
// instead of one round-trip per product. Stock opname posts one session = one
// location/warehouse/store with unique products, and the caller's transaction
// has already locked the affected rows, so the batched writes preserve the exact
// per-product arithmetic and locking semantics of the single-product path
// (backend review P3).
//
// Lock ordering: ReconcileLocationStocks locks ALL global rows before any rack
// rows (the single-product path it replaces locked rack-then-global). Every
// other product_stock locker must keep a compatible order: StockDeducer locks
// global rows only and TransferLocationStockTx locks rack rows only, so no
// cycle exists today. Any future path that locks a rack row and then the
// global row of the same product can deadlock against an opname posting — take
// global rows first.
type StockApplier struct{}

// SetProductStock sets one product's global stock to an absolute value. It is a
// thin wrapper over the batched SetProductStocks, kept for direct use and tests.
func (a StockApplier) SetProductStock(ctx context.Context, tx pgx.Tx, item shared.StockSetItem) error {
	return a.SetProductStocks(ctx, tx, []shared.StockSetItem{item})
}

// SetProductStocks sets many products' global stock to absolute values in two
// statements (a bulk UPDATE of existing rows, then a bulk INSERT of the rows
// that did not exist) rather than one round-trip per product. Each product's
// global row is targeted by the same predicate as the single-product path
// (warehouse_id IS NULL AND store_id IS NULL), so behaviour is identical.
func (a StockApplier) SetProductStocks(ctx context.Context, tx pgx.Tx, items []shared.StockSetItem) error {
	if len(items) == 0 {
		return nil
	}
	// Deduplicate by product, keeping the LAST value (matching the sequential
	// last-wins behaviour if a product were ever listed twice).
	byProduct := make(map[int]int, len(items))
	pids := make([]int, 0, len(items))
	for _, it := range items {
		if _, seen := byProduct[it.ProductID]; !seen {
			pids = append(pids, it.ProductID)
		}
		byProduct[it.ProductID] = it.Quantity
	}
	qtys := make([]int, len(pids))
	for i, pid := range pids {
		qtys[i] = byProduct[pid]
	}

	if _, err := tx.Exec(ctx, `
		UPDATE product_stock AS ps
		SET quantity = v.qty, updated_at = NOW()
		FROM unnest($1::int[], $2::int[]) AS v(pid, qty)
		WHERE ps.product_id = v.pid AND ps.warehouse_id IS NULL AND ps.store_id IS NULL
	`, pids, qtys); err != nil {
		return fmt.Errorf("failed to update product stock: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO product_stock (product_id, quantity, updated_at)
		SELECT v.pid, v.qty, NOW()
		FROM unnest($1::int[], $2::int[]) AS v(pid, qty)
		WHERE NOT EXISTS (
			SELECT 1 FROM product_stock ps
			WHERE ps.product_id = v.pid AND ps.warehouse_id IS NULL AND ps.store_id IS NULL
		)
	`, pids, qtys); err != nil {
		return fmt.Errorf("failed to insert product stock: %w", err)
	}
	return nil
}

// ReconcileLocationStock applies one product's signed delta to its rack row and
// reconciles the global row from the rack share. It is a thin wrapper over the
// batched ReconcileLocationStocks, kept for direct use and tests.
func (a StockApplier) ReconcileLocationStock(ctx context.Context, tx pgx.Tx, reconcile shared.LocationStockReconcile) error {
	return a.ReconcileLocationStocks(ctx, tx, []shared.LocationStockReconcile{reconcile})
}

// ReconcileLocationStocks applies signed deltas to many products' rack rows and
// reconciles each product's global row from the rack share, in a fixed number of
// statements rather than one round-trip per product. It reads (and locks) the
// current rack and global quantities in bulk, computes the new values in Go using
// the exact single-product arithmetic (newRack = rack + delta; newGlobal =
// max(global - rack, 0) + newRack, clamped at zero), then writes them in bulk.
//
// Repeated products are handled explicitly rather than assumed away: duplicates
// at the same location are merged by summing their deltas (equivalent to
// sequential application), and a product repeated across locations carries its
// computed global forward, so the final upsert writes exactly one row per
// product and never hits Postgres' "cannot affect row a second time" error.
// Rack rows are targeted by (product_id, location_id) and missing rows are
// inserted with the session's warehouse/store; global rows are upserted through
// the uq_product_stock unique constraint. Both match the single-product path.
func (a StockApplier) ReconcileLocationStocks(ctx context.Context, tx pgx.Tx, reconciles []shared.LocationStockReconcile) error {
	if len(reconciles) == 0 {
		return nil
	}

	// Lock + read the current global quantity for every product in one pass.
	allPids := make([]int, len(reconciles))
	for i, rc := range reconciles {
		allPids[i] = rc.ProductID
	}
	globalQty, err := lockQuantities(ctx, tx, allPids, `
		SELECT product_id, quantity FROM product_stock
		WHERE product_id = ANY($1::int[]) AND warehouse_id IS NULL AND store_id IS NULL AND location_id IS NULL
		FOR UPDATE
	`)
	if err != nil {
		return fmt.Errorf("failed to lock global stock: %w", err)
	}

	// Deduplicate to one reconcile per (location, product), summing deltas so a
	// repeated product at the same location behaves like sequential application
	// (newRack = rack + d1 + d2). Without this, duplicate keys would make the
	// unnest-driven rack UPDATE pick an arbitrary value.
	type locProduct struct{ loc, pid int }
	deltas := make(map[locProduct]int, len(reconciles))
	byLoc := make(map[int][]shared.LocationStockReconcile, len(reconciles))
	locOrder := make([]int, 0, len(reconciles))
	for _, rc := range reconciles {
		k := locProduct{rc.LocationID, rc.ProductID}
		if _, dup := deltas[k]; !dup {
			if _, locSeen := byLoc[rc.LocationID]; !locSeen {
				locOrder = append(locOrder, rc.LocationID)
			}
			byLoc[rc.LocationID] = append(byLoc[rc.LocationID], rc)
		}
		deltas[k] += rc.Delta
	}

	// One final global value per product; a product repeated across locations
	// gets its entry refreshed as each location group is processed.
	globalUpsert := make(map[int]int, len(globalQty))

	for _, locID := range locOrder {
		group := byLoc[locID]
		locPids := make([]int, len(group))
		for i, rc := range group {
			locPids[i] = rc.ProductID
		}
		// Lock + read the current rack quantity for every product at this location.
		rackQty, err := lockQuantities(ctx, tx, locPids, `
			SELECT product_id, quantity FROM product_stock
			WHERE product_id = ANY($1::int[]) AND location_id = $2
			FOR UPDATE
		`, locID)
		if err != nil {
			return fmt.Errorf("failed to lock location stock: %w", err)
		}

		// Compute new rack/global quantities and split rack rows into those that
		// already exist (UPDATE) and those that must be created (INSERT).
		updPids := make([]int, 0, len(group))
		updQtys := make([]int, 0, len(group))
		insPids := make([]int, 0, len(group))
		insQtys := make([]int, 0, len(group))
		for _, rc := range group {
			r, rackExists := rackQty[rc.ProductID]
			g := globalQty[rc.ProductID]
			newRack := r + deltas[locProduct{locID, rc.ProductID}]
			newGlobal := max(g-r, 0) + newRack
			if rackExists {
				updPids = append(updPids, rc.ProductID)
				updQtys = append(updQtys, newRack)
			} else {
				insPids = append(insPids, rc.ProductID)
				insQtys = append(insQtys, newRack)
			}
			// Carry the new global forward so a product repeated at another
			// location reconciles against its updated value, matching
			// sequential application.
			globalQty[rc.ProductID] = newGlobal
			globalUpsert[rc.ProductID] = newGlobal
		}

		// Bulk-update existing rack rows.
		if len(updPids) > 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE product_stock AS ps
				SET quantity = v.qty, updated_at = NOW()
				FROM unnest($1::int[], $2::int[]) AS v(pid, qty)
				WHERE ps.product_id = v.pid AND ps.location_id = $3
			`, updPids, updQtys, locID); err != nil {
				return fmt.Errorf("failed to update location stock: %w", err)
			}
		}
		// Bulk-insert missing rack rows using the session's warehouse/store.
		if len(insPids) > 0 {
			wh := group[0].WarehouseID
			st := group[0].StoreID
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_stock (product_id, quantity, warehouse_id, store_id, location_id, updated_at)
				SELECT v.pid, v.qty, $3, $4, $5, NOW()
				FROM unnest($1::int[], $2::int[]) AS v(pid, qty)
			`, insPids, insQtys, wh, st, locID); err != nil {
				return fmt.Errorf("failed to insert location stock: %w", err)
			}
		}
	}

	// Bulk-upsert global rows (one entry per distinct product) through the
	// unique constraint.
	if len(globalUpsert) > 0 {
		upsertPids := make([]int, 0, len(globalUpsert))
		upsertQtys := make([]int, 0, len(globalUpsert))
		for pid, qty := range globalUpsert {
			upsertPids = append(upsertPids, pid)
			upsertQtys = append(upsertQtys, qty)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at)
			SELECT v.pid, v.qty, NOW()
			FROM unnest($1::int[], $2::int[]) AS v(pid, qty)
			ON CONFLICT ON CONSTRAINT uq_product_stock
			DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = NOW()
		`, upsertPids, upsertQtys); err != nil {
			return fmt.Errorf("failed to update global stock: %w", err)
		}
	}

	return nil
}

// lockQuantities reads (product_id, quantity) for the given products under FOR
// UPDATE and returns a map keyed by product id (absent products are simply not
// in the map). The query must take the product-id array as $1; extra params
// (e.g. a location id) follow.
func lockQuantities(ctx context.Context, tx pgx.Tx, pids []int, query string, args ...interface{}) (map[int]int, error) {
	q := make(map[int]int, len(pids))
	rows, err := tx.Query(ctx, query, append([]interface{}{pids}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pid, qty int
		if err := rows.Scan(&pid, &qty); err != nil {
			return nil, err
		}
		q[pid] = qty
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return q, nil
}

import { execFileSync } from 'child_process';

// ============================================================================
// Direct PostgreSQL access for scenarios that cannot be driven through the API
// (cart expiry backdating) and for deterministic test-data cleanup.
// Connection defaults mirror the Go test suite (TEST_DB_* / DB_* env vars).
// ============================================================================

const DB_HOST = process.env.TEST_DB_HOST || process.env.DB_HOST || 'localhost';
const DB_PORT = process.env.TEST_DB_PORT || process.env.DB_PORT || '5433';
const DB_USER = process.env.TEST_DB_USER || process.env.DB_USER || 'pos';
const DB_PASSWORD = process.env.TEST_DB_PASSWORD || process.env.DB_PASSWORD || 'admin123';
const DB_NAME = process.env.TEST_DB_NAME || 'retail_pos';

function psql(args: string[], input?: string): string {
  return execFileSync(
    'psql',
    ['-h', DB_HOST, '-p', DB_PORT, '-U', DB_USER, '-d', DB_NAME, '-v', 'ON_ERROR_STOP=1', '-qAt', ...args],
    { input, env: { ...process.env, PGPASSWORD: DB_PASSWORD }, encoding: 'utf8' }
  );
}

export function execSQL(sql: string): void {
  psql(['-c', sql]);
}

export function querySQL<T = Record<string, unknown>>(sql: string): T[] {
  const out = psql(['-c', `SELECT COALESCE(json_agg(q), '[]'::json)::text FROM (${sql}) q`]);
  return JSON.parse(out.trim() || '[]') as T[];
}

export function scalarSQL(sql: string): string {
  return psql(['-c', sql]).trim();
}

/**
 * Backdate a cart's expiry so it is already expired without waiting for the
 * real TTL (default 24h). Used by CS-A6/A7.
 */
export function backdateCartExpiry(cartId: number): void {
  execSQL(`UPDATE cart_sessions SET expired_at = NOW() - INTERVAL '1 minute' WHERE id = ${cartId}`);
}

/**
 * Delete every held cart belonging to a cashier so UI flows that recall
 * `.first()` deterministically pick up the cart parked within the test.
 */
export function purgeHeldCarts(cashierId: number): void {
  execSQL(`DELETE FROM cart_items WHERE cart_session_id IN (SELECT id FROM cart_sessions WHERE cashier_id = ${cashierId} AND status = 'held')`);
  execSQL(`DELETE FROM cart_sessions WHERE cashier_id = ${cashierId} AND status = 'held'`);
}

function idList(ids: number[]): string {
  return ids.join(',');
}

/**
 * Tracks every entity created during a spec run and purges them in reverse
 * foreign-key order so seeded data is never touched.
 */
export class TestDataTracker {
  saleIds: number[] = [];
  cartIds: number[] = [];
  shiftIds: number[] = [];
  userIds: number[] = [];
  customerIds: number[] = [];
  brandIds: number[] = [];
  storeIds: number[] = [];
  productIds: number[] = [];

  trackSale(id: number | undefined | null): void {
    if (id) this.saleIds.push(id);
  }

  trackCart(id: number | undefined | null): void {
    if (id) this.cartIds.push(id);
  }

  trackShift(id: number | undefined | null): void {
    if (id) this.shiftIds.push(id);
  }

  trackUser(id: number | undefined | null): void {
    if (id) this.userIds.push(id);
  }

  trackCustomer(id: number | undefined | null): void {
    if (id) this.customerIds.push(id);
  }

  trackBrand(id: number | undefined | null): void {
    if (id) this.brandIds.push(id);
  }

  trackStore(id: number | undefined | null): void {
    if (id) this.storeIds.push(id);
  }

  trackProduct(id: number | undefined | null): void {
    if (id) this.productIds.push(id);
  }

  cleanup(): void {
    // Children first: sale_payments -> sale_items -> sales (tracked + created
    // by tracked users) -> cart_sessions -> shifts -> audit_logs -> customers
    // -> users.
    const users = this.userIds.length ? idList(this.userIds) : null;
    if (this.saleIds.length) {
      const sales = idList(this.saleIds);
      execSQL(`DELETE FROM sale_payments WHERE sale_id IN (${sales})`);
      execSQL(`DELETE FROM sale_items WHERE sale_id IN (${sales})`);
      execSQL(`DELETE FROM sales WHERE id IN (${sales})`);
    }
    if (users) {
      // Sales completed by tracked cashiers block user deletion (RESTRICT).
      execSQL(`DELETE FROM sale_payments WHERE sale_id IN (SELECT id FROM sales WHERE cashier_id IN (${users}))`);
      execSQL(`DELETE FROM sale_items WHERE sale_id IN (SELECT id FROM sales WHERE cashier_id IN (${users}))`);
      execSQL(`DELETE FROM sales WHERE cashier_id IN (${users})`);
    }
    if (this.cartIds.length) {
      execSQL(`DELETE FROM cart_sessions WHERE id IN (${idList(this.cartIds)})`);
    }
    if (this.shiftIds.length) {
      // Sales reference shifts via shift_id FK — must delete them first.
      const shifts = idList(this.shiftIds);
      execSQL(`DELETE FROM sale_payments WHERE sale_id IN (SELECT id FROM sales WHERE shift_id IN (${shifts}))`);
      execSQL(`DELETE FROM sale_items WHERE sale_id IN (SELECT id FROM sales WHERE shift_id IN (${shifts}))`);
      execSQL(`DELETE FROM sales WHERE shift_id IN (${shifts})`);
      // Held leftovers created by helpers may reference tracked shifts.
      execSQL(`DELETE FROM cart_sessions WHERE shift_id IN (${shifts})`);
      execSQL(`DELETE FROM shifts WHERE id IN (${shifts})`);
    }
    if (users) {
      // Delete ALL shifts for tracked users (including untracked ones created
      // by ensureOpenShift) before deleting users — shifts_user_id_fkey is
      // RESTRICT and would otherwise block user deletion.
      execSQL(`DELETE FROM sale_payments WHERE sale_id IN (SELECT id FROM sales WHERE shift_id IN (SELECT id FROM shifts WHERE user_id IN (${users})))`);
      execSQL(`DELETE FROM sale_items WHERE sale_id IN (SELECT id FROM sales WHERE shift_id IN (SELECT id FROM shifts WHERE user_id IN (${users})))`);
      execSQL(`DELETE FROM sales WHERE shift_id IN (SELECT id FROM shifts WHERE user_id IN (${users}))`);
      execSQL(`DELETE FROM cart_sessions WHERE shift_id IN (SELECT id FROM shifts WHERE user_id IN (${users}))`);
      execSQL(`DELETE FROM shifts WHERE user_id IN (${users})`);
    }
    if (users) {
      // audit_logs has an append-only trigger; bypass it via the GUC the
      // migration (034) introduced for maintenance operations. Combine SET
      // and DELETE in one psql invocation so the GUC persists.
      psql(['-c', `SET app.allow_audit_mod = 'on'; DELETE FROM audit_logs WHERE user_id IN (${users})`]);
      execSQL(`DELETE FROM users WHERE id IN (${users})`);
    }
    if (this.customerIds.length) {
      execSQL(`DELETE FROM customers WHERE id IN (${idList(this.customerIds)})`);
    }
    if (this.productIds.length) {
      const pids = idList(this.productIds);
      // Test products may have sales — delete orphaned sales first to keep counts clean
      execSQL(`DELETE FROM sale_payments WHERE sale_id IN (SELECT sale_id FROM sale_items WHERE product_id IN (${pids}))`);
      execSQL(`DELETE FROM sales WHERE id IN (SELECT sale_id FROM sale_items WHERE product_id IN (${pids}))`);
      execSQL(`DELETE FROM sale_items WHERE product_id IN (${pids})`);
      execSQL(`DELETE FROM cart_items WHERE product_id IN (${pids})`);
      execSQL(`DELETE FROM products WHERE id IN (${pids})`);
    }
    if (this.brandIds.length) {
      const bids = idList(this.brandIds);
      // Schema is SET NULL for products.brand_id, but delete test products to avoid leaks
      execSQL(`DELETE FROM sale_payments WHERE sale_id IN (SELECT sale_id FROM sale_items WHERE product_id IN (SELECT id FROM products WHERE brand_id IN (${bids})))`);
      execSQL(`DELETE FROM sales WHERE id IN (SELECT sale_id FROM sale_items WHERE product_id IN (SELECT id FROM products WHERE brand_id IN (${bids})))`);
      execSQL(`DELETE FROM sale_items WHERE product_id IN (SELECT id FROM products WHERE brand_id IN (${bids}))`);
      execSQL(`DELETE FROM cart_items WHERE product_id IN (SELECT id FROM products WHERE brand_id IN (${bids}))`);
      execSQL(`DELETE FROM products WHERE brand_id IN (${bids})`);
      execSQL(`DELETE FROM brands WHERE id IN (${bids})`);
    }
    if (this.storeIds.length) {
      execSQL(`DELETE FROM stores WHERE id IN (${idList(this.storeIds)})`);
    }
  }
}

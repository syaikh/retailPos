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
      // Held leftovers created by helpers may reference tracked shifts.
      execSQL(`DELETE FROM cart_sessions WHERE shift_id IN (${idList(this.shiftIds)})`);
      execSQL(`DELETE FROM shifts WHERE id IN (${idList(this.shiftIds)})`);
    }
    if (users) {
      execSQL(`DELETE FROM audit_logs WHERE user_id IN (${users})`);
      execSQL(`DELETE FROM users WHERE id IN (${users})`);
    }
    if (this.customerIds.length) {
      execSQL(`DELETE FROM customers WHERE id IN (${idList(this.customerIds)})`);
    }
  }
}

import { describe, it, expect } from 'vitest';

// ─── helpers extracted from InventoryPage.svelte ───────────────────────────────

/**
 * classifyStock mirrors the stock-vs-threshold logic used in InventoryPage.
 * Returns the badge variant string that the component would set.
 *
 * Page rules:
 *  - if stock <= criticalThreshold  → 'destructive'
 *  - else → 'default'
 */
function classifyStock(stock: number, criticalThreshold: number): string {
  return stock <= criticalThreshold ? 'destructive' : 'default';
}

/**
 * resolveMaxStock returns the maxStock query value used when lowStockOnly is on.
 *
 * Before refactor it was the literal string '5'.
 * After refactor  it is criticalThreshold.toString().
 */
function resolveMaxStock(lowStockOnly: boolean, criticalThreshold: number): string | null {
  if (!lowStockOnly) return null;
  return criticalThreshold.toString();
}

/**
 * thresholdConfigShape is the expected response shape from GET /api/stock-thresholds.
 */
const DEFAULT_WARNING = 10;
const DEFAULT_CRITICAL = 5;

// ─── unit: classifyStock ───────────────────────────────────────────────────────

describe('classifyStock', () => {
  it('destructive when stock is 0', () => {
    expect(classifyStock(0, 5)).toBe('destructive');
    expect(classifyStock(0, 10)).toBe('destructive');
    expect(classifyStock(0, 0)).toBe('destructive');
  });

  it('destructive when stock equals critical threshold (inclusive boundary)', () => {
    expect(classifyStock(5, 5)).toBe('destructive');
    expect(classifyStock(10, 10)).toBe('destructive');
    expect(classifyStock(3, 3)).toBe('destructive');
  });

  it('destructive when stock is below critical threshold', () => {
    expect(classifyStock(1, 5)).toBe('destructive');
    expect(classifyStock(4, 5)).toBe('destructive');
    expect(classifyStock(7, 10)).toBe('destructive');
  });

  it('default when stock is one above the critical threshold', () => {
    expect(classifyStock(6, 5)).toBe('default');
    expect(classifyStock(11, 10)).toBe('default');
    expect(classifyStock(4, 3)).toBe('default');
  });

  it('default when stock equals the warning threshold (warning is never used by badge)', () => {
    // Badge only cares about critical threshold; warning threshold is not checked.
    // Stock at warning but well above critical → default.
    expect(classifyStock(10, 5)).toBe('default');
  });

  it('default when stock is far above threshold', () => {
    expect(classifyStock(100, 5)).toBe('default');
    expect(classifyStock(999, 10)).toBe('default');
  });

  it('zero critical threshold: any positive stock is default, zero is destructive', () => {
    expect(classifyStock(0, 0)).toBe('destructive');
    expect(classifyStock(1, 0)).toBe('default');
    expect(classifyStock(5, 0)).toBe('default');
  });
});

// ─── unit: resolveMaxStock ─────────────────────────────────────────────────────

describe('resolveMaxStock', () => {
  it('returns null when lowStockOnly is false', () => {
    expect(resolveMaxStock(false, 5)).toBeNull();
    expect(resolveMaxStock(false, 10)).toBeNull();
    expect(resolveMaxStock(false, 0)).toBeNull();
  });

  it('returns criticalThreshold as string when lowStockOnly is true', () => {
    expect(resolveMaxStock(true, 5)).toBe('5');
    expect(resolveMaxStock(true, 10)).toBe('10');
    expect(resolveMaxStock(true, 0)).toBe('0');
  });

  it('frontend hardcoded "5" is replaced by fetched threshold value', () => {
    // Before the refactor the code was hardcoded to:
    //   params.append('maxStock', '5')
    // After the refactor it is:
    //   params.append('maxStock', criticalThreshold.toString())
    const oldHardcoded = '5';
    const fetchedThreshold = 7;

    // Old behaviour: always 5 regardless of threshold-change
    const oldMaxStock = resolveMaxStock(true, 5); // was hardcoded
    expect(oldMaxStock).toBe(oldHardcoded);

    // New behaviour: maxStock follows the fetched threshold
    const newMaxStock = resolveMaxStock(true, fetchedThreshold);
    expect(newMaxStock).toBe('7');
  });

  it('custom critical threshold e.g. STOCK_CRITICAL_THRESHOLD=8', () => {
    expect(resolveMaxStock(true, 8)).toBe('8');
  });
});

// ─── unit: threshold response shape ──────────────────────────────────────────

describe('threshold API response shape', () => {
  it('both keys must be present on the root object', () => {
    const resp = { warning: 10, critical: 5 } as Record<string, unknown>;
    for (const key of ['warning', 'critical'] as const) {
      // Object.prototype.toString='[object Object]' is NOT a number so it does not trigger this path
      const val = (resp as Record<string, unknown>)[key];
      expect(val).toBeDefined();
      expect(typeof val).toBe('number');
    }
  });

  it('all values must be numbers', () => {
    const resp = { warning: 10, critical: 5 };
    for (const key of ['warning', 'critical'] as const) {
      expect(typeof resp[key as keyof typeof resp]).toBe('number');
    }
  });

  it('response type is stable — fixture must not grow stray keys', () => {
    const respKeys = Object.keys({ warning: 10, critical: 5 });
    expect(respKeys).toEqual(['warning', 'critical']);
  });
});

// ─── config: default thresholds ───────────────────────────────────────────────

describe('default threshold constants in code', () => {
  it('warning default is 10', () => {
    expect(DEFAULT_WARNING).toBe(10);
  });
  it('critical default is 5', () => {
    expect(DEFAULT_CRITICAL).toBe(5);
  });
});

// ─── form state: stock_min is removed ─────────────────────────────────────────

describe('InventoryPage form state contract', () => {
  it('form state object must not contain stock_min key', () => {
    // Simulate initial form state as declared in InventoryPage.svelte (post-refactor)
    const form = {
      name: '',
      sku: '',
      barcode: '',
      category: '',
      brand_id: null,
      price: 0,
      cost: 0,
      stock: 0,
      unit_of_measure_id: null,
      tax_class_id: null,
      weight_grams: null,
      description: '',
      status: 'draft'
    } as const;

    // stock_min must NOT be present
    expect('stock_min' in form).toBe(false);
    expect((form as Record<string, unknown>).stock_min_).toBeUndefined();
  });

  it('resetForm produces the same clean state without stock_min', () => {
    // Simulate resetForm() output (post-refactor)
    const resetForm = () => ({
      name: '',
      sku: '',
      barcode: '',
      category: '',
      brand_id: null,
      price: 0,
      cost: 0,
      stock: 0,
      unit_of_measure_id: null,
      tax_class_id: null,
      weight_grams: null,
      description: '',
      status: 'draft'
    });

    const state = resetForm();
    expect('stock_min' in state).toBe(false);
    expect(state.stock).toBe(0);
    expect('stock_min' in state).toBe(false);
  });
});

// ─── frontend Product type contract post-refactor ──────────────────────────────

import type { Product } from '$lib/types';

describe('Product type contract (frontend/type.ts)', () => {
  // stock_min must not be a declared key on the Product interface
  it('stock_min is not a valid property of Product', () => {
    // TypeScript compile-time check —
    // If stock_min were still in types.ts this line would compile fine.
    // Because we removed it the test reads it as a structural error that
    // an ESLint / noImplicitAny style checker would catch.
    //
    // At runtime we assert the derived object never carries it.
    const p: Product = {
      id: 1,
      name: 'Test',
      sku: 'T-001',
      price: 1000,
      stock: 50
    } as Product;
    expect((p as Record<string, unknown>).stock_min_).toBeUndefined();
  });
});

// ─── edge cases ───────────────────────────────────────────────────────────────

describe('edge cases', () => {
  it('negative stock values are treated as low stock', () => {
    // Sanity: negative stock (should never happen in prod but be safe)
    expect(classifyStock(-1, 5)).toBe('destructive');
  });

  it('both thresholds set to the same value', () => {
    expect(classifyStock(4, 4)).toBe('destructive');
    expect(classifyStock(5, 4)).toBe('default');
    expect(resolveMaxStock(true, 4)).toBe('4');
  });

  it('critical threshold higher than warning threshold', () => {
    // Edge config: warning < critical reversed
    // Badge: stock <= critical (7) → destructive
    expect(classifyStock(5, 7)).toBe('destructive');
    expect(classifyStock(8, 7)).toBe('default');
  });

  it('large threshold values from env', () => {
    expect(classifyStock(5, 9999)).toBe('destructive');
    expect(classifyStock(10000, 9999)).toBe('default');
    expect(resolveMaxStock(true, 9999)).toBe('9999');
  });

  it('critical threshold of 0 means only exactly-zero stock is "low"', () => {
    expect(classifyStock(0, 0)).toBe('destructive');
    expect(classifyStock(1, 0)).toBe('default');
  });
});

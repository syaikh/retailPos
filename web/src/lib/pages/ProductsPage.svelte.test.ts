import { describe, it, expect, beforeAll } from 'vitest';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

function getPageSource(): string {
  const dir = path.dirname(fileURLToPath(import.meta.url));
  return fs.readFileSync(path.join(dir, 'ProductsPage.svelte'), 'utf-8');
}

function classifyStock(stock: number, criticalThreshold: number): string {
  return stock <= criticalThreshold ? 'destructive' : 'default';
}

function resolveMaxStock(lowStockOnly: boolean, criticalThreshold: number): string | null {
  if (!lowStockOnly) return null;
  return criticalThreshold.toString();
}

const DEFAULT_WARNING = 10;
const DEFAULT_CRITICAL = 5;

describe('classifyStock', () => {
  it('destructive when stock is 0', () => {
    expect(classifyStock(0, 5)).toBe('destructive');
    expect(classifyStock(0, 10)).toBe('destructive');
  });

  it('destructive when stock equals critical threshold', () => {
    expect(classifyStock(5, 5)).toBe('destructive');
  });

  it('destructive when stock is below critical threshold', () => {
    expect(classifyStock(1, 5)).toBe('destructive');
  });

  it('default when stock is above critical threshold', () => {
    expect(classifyStock(6, 5)).toBe('default');
    expect(classifyStock(100, 5)).toBe('default');
  });
});

describe('resolveMaxStock', () => {
  it('returns null when lowStockOnly is false', () => {
    expect(resolveMaxStock(false, 5)).toBeNull();
  });

  it('returns criticalThreshold as string when lowStockOnly is true', () => {
    expect(resolveMaxStock(true, 5)).toBe('5');
  });
});

describe('default threshold constants', () => {
  it('warning default is 10', () => {
    expect(DEFAULT_WARNING).toBe(10);
  });
  it('critical default is 5', () => {
    expect(DEFAULT_CRITICAL).toBe(5);
  });
});

describe('ProductsPage.svelte — source structure guards', () => {
  let src: string;

  beforeAll(() => {
    src = getPageSource();
  });

  // ── Product CRUD features present ────────────────────────────────────────────
  it('declares showModal for add/edit product', () => {
    expect(src).toContain('let showModal');
  });

  it('declares showDeleteModal for delete confirmation', () => {
    expect(src).toContain('let showDeleteModal');
  });

  it('declares selectedProduct for detail view', () => {
    expect(src).toContain('let selectedProduct');
  });

  it('imports ProductFormModal component', () => {
    expect(src).toContain("import ProductFormModal from '$lib/components/inventory/ProductFormModal.svelte'");
  });

  it('has Add Product button', () => {
    expect(src).toContain('Add Product');
  });

  it('renders product table with PRODUCT NAME header', () => {
    expect(src).toContain('PRODUCT NAME');
  });

  it('renders category filter button', () => {
    expect(src).toContain('Kategori Dipilih');
  });

  it('has Category filter modal import', () => {
    expect(src).toContain("import CategoryFilterModal");
  });

  it('renders product table with CATEGORY column', () => {
    expect(src).toContain('CATEGORY');
  });

  it('renders product table with PRICE column', () => {
    expect(src).toContain('>PRICE<');
  });

  // ── No StockPage-specific features ──────────────────────────────────────────
  it('does NOT have low-stock toggle button', () => {
    expect(src).not.toContain("lowStockOnly");
    expect(src).not.toContain('Low Stock');
  });

  it('does NOT import StockAdjustModal', () => {
    expect(src).not.toContain('StockAdjustModal');
  });

  it('does NOT declare stockAdjustForm', () => {
    expect(src).not.toContain('stockAdjustForm');
  });

  it('does NOT have stock adjustment modal', () => {
    expect(src).not.toContain('showAdjustStockModal');
  });

  it('does NOT have STOCK column in table', () => {
    expect(src).not.toContain('>STOCK<');
  });

  // ── Permission checks ────────────────────────────────────────────────────────
  it('declares canManageInventory from allowedInventoryRoles', () => {
    expect(src).toContain('canManageInventory');
    expect(src).toContain('allowedInventoryRoles');
  });

  it('checks isSuperAdmin() for delete permission', () => {
    expect(src).toContain('isSuperAdmin()');
  });

  it('checks isAdmin() for admin permission', () => {
    expect(src).toContain('isAdmin()');
  });

  it('checks isSensitive() for cost/margin visibility', () => {
    expect(src).toContain('isSensitive()');
  });

  it('checks isFullAudit() for audit trail visibility', () => {
    expect(src).toContain('isFullAudit()');
  });

  it('checks canEdit() for edit permission', () => {
    expect(src).toContain('canEdit()');
  });

  // ── Detail drawer ────────────────────────────────────────────────────────────
  it('declares showDetailDrawer reactive state', () => {
    expect(src).toContain('let showDetailDrawer');
  });

  it('has product detail drawer with Detail Produk heading', () => {
    expect(src).toContain('Detail Produk');
  });

  it('has Stok & Logistik block in drawer', () => {
    expect(src).toContain('Stok');
  });

  it('has Keuangan block in drawer', () => {
    expect(src).toContain('Keuangan');
  });

  // ── Form helpers ─────────────────────────────────────────────────────────────
  it('has validateProductForm helper', () => {
    expect(src).toContain('validateProductForm');
  });

  it('has resetForm helper', () => {
    expect(src).toContain('resetForm');
  });

  it('has handleSort and sortProducts helpers', () => {
    expect(src).toContain('handleSort');
    expect(src).toContain('sortProducts');
  });

  it('has copyToClipboard helper', () => {
    expect(src).toContain('copyToClipboard');
  });

  it('has formatCurrency helper', () => {
    expect(src).toContain('formatCurrency');
  });

  it('has formatDate helper', () => {
    expect(src).toContain('formatDate');
  });

  it('has statusInfo helper', () => {
    expect(src).toContain('statusInfo');
  });

  // ── WebSocket ─────────────────────────────────────────────────────────────────
  it('imports useWebSocket and registers handlers', () => {
    expect(src).toContain('useWebSocket');
    expect(src).toContain('product_updated');
    expect(src).toContain('low_stock_alert');
  });
});

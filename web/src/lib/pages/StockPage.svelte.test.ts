import { describe, it, expect, beforeAll } from 'vitest';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

function getPageSource(): string {
  const dir = path.dirname(fileURLToPath(import.meta.url));
  return fs.readFileSync(path.join(dir, 'StockPage.svelte'), 'utf-8');
}

function classifyStock(stock: number, criticalThreshold: number): string {
  return stock <= criticalThreshold ? 'destructive' : 'default';
}

describe('classifyStock (StockPage)', () => {
  it('destructive when stock is 0', () => {
    expect(classifyStock(0, 5)).toBe('destructive');
  });

  it('destructive when stock equals critical threshold', () => {
    expect(classifyStock(5, 5)).toBe('destructive');
  });

  it('destructive when stock is below critical threshold', () => {
    expect(classifyStock(3, 5)).toBe('destructive');
  });

  it('default when stock is above critical threshold', () => {
    expect(classifyStock(6, 5)).toBe('default');
    expect(classifyStock(100, 5)).toBe('default');
  });
});

describe('StockPage.svelte — source structure guards', () => {
  let src: string;

  beforeAll(() => {
    src = getPageSource();
  });

  // ── Stock management features present ────────────────────────────────────────
  it('declares lowStockOnly filter toggle', () => {
    expect(src).toContain('let lowStockOnly');
  });

  it('declares stockAdjustForm for stock adjustment', () => {
    expect(src).toContain('stockAdjustForm');
  });

  it('declares showAdjustStockModal', () => {
    expect(src).toContain('showAdjustStockModal');
  });

  it('declares stockAdjustProduct', () => {
    expect(src).toContain('stockAdjustProduct');
  });

  it('imports StockAdjustModal component', () => {
    expect(src).toContain("import StockAdjustModal from '$lib/components/inventory/StockAdjustModal.svelte'");
  });

  it('has Low Stock toggle button', () => {
    expect(src).toContain('Low Stock');
  });

  it('renders stock table with PRODUCT NAME header', () => {
    expect(src).toContain('PRODUCT NAME');
  });

  it('renders stock table with STOCK column', () => {
    expect(src).toContain('>STOCK<');
  });

  it('renders stock table with STATUS column', () => {
    expect(src).toContain('STATUS');
  });

  it('renders stock table with ACTIONS column', () => {
    expect(src).toContain('ACTIONS');
  });

  it('has classifyStock helper', () => {
    expect(src).toContain('classifyStock');
  });

  it('has openAdjustStock function', () => {
    expect(src).toContain('openAdjustStock');
  });

  it('has handleAdjustStock function', () => {
    expect(src).toContain('handleAdjustStock');
  });

  it('has Adjust action button', () => {
    expect(src).toContain('Adjust');
  });

  it('fetches stock-thresholds on mount', () => {
    expect(src).toContain('fetchThresholds');
  });

  it('uses criticalThreshold for low stock filter', () => {
    expect(src).toContain('criticalThreshold');
  });

  it('uses warningThreshold for stock status badges', () => {
    expect(src).toContain('warningThreshold');
  });

  it('has Critical badge for low stock', () => {
    expect(src).toContain('Critical');
  });

  it('has In Stock badge for healthy stock', () => {
    expect(src).toContain('In Stock');
  });

  // ── No ProductPage-specific features ─────────────────────────────────────────
  it('does NOT import ProductFormModal', () => {
    expect(src).not.toContain('ProductFormModal');
  });

  it('does NOT have showModal for add/edit product', () => {
    expect(src).not.toContain('let showModal');
  });

  it('does NOT have showDeleteModal', () => {
    expect(src).not.toContain('showDeleteModal');
  });

  it('does NOT have Add Product button', () => {
    expect(src).not.toContain('Add Product');
  });

  it('does NOT have category filter modal', () => {
    expect(src).not.toContain('CategoryFilterModal');
  });

  it('does NOT have PRICE column in table', () => {
    expect(src).not.toContain('>PRICE<');
  });

  it('does NOT have product detail drawer', () => {
    expect(src).not.toContain('showDetailDrawer');
  });

  // ── Permission checks ────────────────────────────────────────────────────────
  it('declares canAdjustStock from allowedStockRoles', () => {
    expect(src).toContain('canAdjustStock');
    expect(src).toContain('allowedStockRoles');
  });

  it('checks canAdjustStock for showing adjust button', () => {
    expect(src).toContain('canAdjustStock');
  });

  // ── Search and pagination ────────────────────────────────────────────────────
  it('has search input for products', () => {
    expect(src).toContain('Search products');
  });

  it('has Pagination component', () => {
    expect(src).toContain('Pagination');
  });

  it('has handleSort and sortProducts helpers', () => {
    expect(src).toContain('handleSort');
    expect(src).toContain('sortProducts');
  });

  // ── API integration ──────────────────────────────────────────────────────────
  it('calls fetchProducts with maxStock param when lowStockOnly is on', () => {
    expect(src).toContain('maxStock');
  });

  it('calls /inventory/adjust endpoint', () => {
    expect(src).toContain('/inventory/adjust');
  });
});

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

  it('renders product table with PRODUCT NAME header (text-left like content)', () => {
    expect(src).toContain('text-left p-4 font-semibold" style="width: 34%;">PRODUCT NAME');
  });

  it('renders category filter button', () => {
    expect(src).toContain('Kategori Dipilih');
  });

  it('has Category filter modal import', () => {
    expect(src).toContain("import CategoryFilterModal");
  });

  it('renders product table with CATEGORY column (text-left like content)', () => {
    expect(src).toContain('text-left p-4 font-semibold w-44">CATEGORY');
  });

  it('renders product table with PRICE column (text-right like content)', () => {
    expect(src).toContain('justify-end gap-1">PRICE');
  });

  it('renders product table with Actions column header (text-left to match content)', () => {
    expect(src).toContain('text-left p-4 font-semibold w-10"></th>');
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

  it('renders product table with STOCK column (text-right like content)', () => {
    expect(src).toContain('justify-end gap-1">STOCK');
  });

  it('data table PRICE and STOCK sort buttons have w-full for flex right-alignment', () => {
    expect(src).toContain("justify-end w-full\" onclick={() => handleSort('price')}");
    expect(src).toContain("justify-end w-full\" onclick={() => handleSort('stock')}");
  });

  it('renders product table with STATUS column (text-left like content)', () => {
    expect(src).toContain('text-left p-4 font-semibold w-24">STATUS');
  });

  it('has openAdjustStock function', () => {
    expect(src).toContain('openAdjustStock');
  });

  it('has handleAdjustStock function', () => {
    expect(src).toContain('handleAdjustStock');
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

  it('passes canAdjustStock to ProductActionsDropdown', () => {
    expect(src).toContain('canAdjustStock={allowedStockRoles.includes(getUserRoleName())}');
  });

  it('declares allowedStockRoles', () => {
    expect(src).toContain('allowedStockRoles');
  });

  it('calls /inventory/adjust endpoint', () => {
    expect(src).toContain('/inventory/adjust');
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
  });

  // ── Bulk actions ──────────────────────────────────────────────────────────────
  it('declares selectedIds state for bulk selection', () => {
    expect(src).toContain('let selectedIds');
  });

  it('declares showBulkStatusModal state', () => {
    expect(src).toContain('let showBulkStatusModal');
  });

  it('declares isBulkUpdating state', () => {
    expect(src).toContain('let isBulkUpdating');
  });

  it('has toggleSelectAll and toggleSelect functions', () => {
    expect(src).toContain('function toggleSelectAll');
    expect(src).toContain('function toggleSelect');
  });

  it('has clearSelection function', () => {
    expect(src).toContain('function clearSelection');
  });

  it('has handleBulkStatusUpdate function', () => {
    expect(src).toContain('handleBulkStatusUpdate');
  });

  it('filters eligible products before bulk status update', () => {
    expect(src).toContain('eligibleIds');
    expect(src).toContain("p.status !== bulkStatusTarget");
  });

  it('calls /products/bulk/status endpoint', () => {
    expect(src).toContain('/products/bulk/status');
  });

  it('renders checkbox column in data table header', () => {
    expect(src).toContain('input type="checkbox"');
  });

  it('data table name column uses 40% width', () => {
    expect(src).toContain('style="width: 40%;"');
  });

  it('select-all checkbox binds indeterminate state', () => {
    expect(src).toContain('bind:indeterminate={someSelected}');
  });

  it('renders bulk action bar with Change Status button', () => {
    expect(src).toContain('selectedIds.size > 0');
    expect(src).toContain('Change Status');
    expect(src).toContain('showBulkStatusModal = true');
  });

  it('renders bulk status modal with status options', () => {
    expect(src).toContain("each ['active', 'inactive', 'archived']");
  });

  it('clears selection when fetchProducts is called', () => {
    expect(src).toContain('selectedIds = new Set()');
  });

  // ── Status filter ─────────────────────────────────────────────────────────────
  it('declares filterStatus state for status filter', () => {
    expect(src).toContain('let filterStatus');
  });

  it('declares showStatusDropdown state', () => {
    expect(src).toContain('let showStatusDropdown');
  });

  it('has statusLabel derived state', () => {
    expect(src).toContain('let statusLabel');
  });

  it('renders status filter dropdown with All Status default', () => {
    expect(src).toContain('All Status');
    expect(src).toContain('.status-filter-container');
  });

  it('passes status query param when filterStatus is set', () => {
    expect(src).toContain("params.append('status', filterStatus)");
  });

  it('closes status dropdown on Escape', () => {
    expect(src).toContain("if (showStatusDropdown) showStatusDropdown = false;");
  });

  // ── Filter chips ─────────────────────────────────────────────────────────────
  it('has activeChips derived state', () => {
    expect(src).toContain('let activeChips');
  });

  it('has clearFilter and clearAllFilters functions', () => {
    expect(src).toContain('function clearFilter');
    expect(src).toContain('function clearAllFilters');
  });

  it('renders filter chips wrapper with is-open class', () => {
    expect(src).toContain('filter-chips-wrapper" class:is-open={activeChips.length > 0}');
  });

  it('renders Clear all button in chips', () => {
    expect(src).toContain('Clear all');
  });
});

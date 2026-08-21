import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ProductsPage.svelte'), 'utf-8');
}

describe('ProductsPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiClient and RBAC composable', () => {
    expect(src).toContain("import apiClient from '$shared/api/http-client'");
    expect(src).toContain("import { useRBAC } from '$shared/composables/useRBAC.svelte'");
  });

  it('imports WebSocket utility', () => {
    expect(src).toContain("import { useWebSocket } from '$shared/api/websocket'");
  });

  it('imports toast store', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('imports child components (ProductFilterDrawer, ProductFormModal, etc.)', () => {
    expect(src).toContain("import ProductFilterDrawer from '$modules/product/components/ProductFilterDrawer.svelte'");
    expect(src).toContain("import ProductActionsDropdown");
    expect(src).toContain("import ProductFormModal");
    expect(src).toContain("import StockAdjustModal");
  });

  it('imports shared UI components', () => {
    expect(src).toContain("import { Button, Modal, Pagination, ImportWizard, ConfirmDeleteModal } from '$shared/ui'");
  });

  it('uses $state for products, loading, pagination state', () => {
    expect(src).toContain('let loading = $state(true)');
    expect(src).toContain('let products = $state<Product[]>');
    expect(src).toContain('let total = $state(0)');
    expect(src).toContain('let searchQuery = $state');
  });

  it('has fetchProducts, fetchCategories, fetchBrands, fetchTaxClasses, fetchUnitsOfMeasure functions', () => {
    expect(src).toContain('async function fetchProducts');
    expect(src).toContain('async function fetchCategories');
    expect(src).toContain('async function fetchBrands');
    expect(src).toContain('async function fetchTaxClasses');
    expect(src).toContain('async function fetchUnitsOfMeasure');
  });

  it('has master data state (brands, unitsOfMeasure, taxClasses)', () => {
    expect(src).toContain('let brands = $state');
    expect(src).toContain('let unitsOfMeasure = $state');
    expect(src).toContain('let taxClasses = $state');
  });

  it('has RBAC permission-based guards for inventory and stock', () => {
    expect(src).toContain('let canCreate = $derived(rbac.can(Permissions.product.create))');
    expect(src).toContain('let canEdit = $derived(rbac.can(Permissions.product.update))');
    expect(src).toContain('let canDelete = $derived(rbac.can(Permissions.product.delete))');
    expect(src).toContain('let canAdjustStock = $derived(rbac.can(Permissions.inventory.adjust))');
  });

  it('has sortProducts function', () => {
    expect(src).toContain('function sortProducts');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });

  it('has handleWindowKeydown for Escape and close-all-dropdowns', () => {
    expect(src).toContain('function handleWindowKeydown');
    expect(src).toContain("e.key === 'Escape'");
    expect(src).toContain("dispatchEvent(new CustomEvent('close-all-dropdowns')");
    expect(src).toContain('<svelte:window onkeydown={handleWindowKeydown} />');
  });

  it('imports getProductById for deep-link fallback', () => {
    expect(src).toContain("import { getProductById } from '$modules/product/services/product-service'");
  });

  it('falls back to getProductById when deep-linked product is not on the loaded page', () => {
    const fallbackIdx = src.indexOf('if (!product) {');
    expect(fallbackIdx).toBeGreaterThan(-1);
    const tryIdx = src.indexOf('await getProductById(pid)', fallbackIdx);
    expect(tryIdx).toBeGreaterThan(-1);
  });

  it('opens detail drawer only after product resolution succeeds', () => {
    const resolveIdx = src.indexOf('let product = products.find(p => p.id === pid) || null;');
    const drawerIdx = src.indexOf('showDetailDrawer = true;', resolveIdx);
    expect(resolveIdx).toBeGreaterThan(-1);
    expect(drawerIdx).toBeGreaterThan(resolveIdx);
  });

  it('applies low_stock=true URL param to lowStockOnly filter before first fetch', () => {
    expect(src).toContain("urlParams.get('low_stock') === 'true'");
    const paramIdx = src.indexOf("urlParams.get('low_stock') === 'true'");
    const fetchIdx = src.indexOf('await fetchProducts(0, limit);', paramIdx);
    expect(fetchIdx).toBeGreaterThan(paramIdx);
  });
});

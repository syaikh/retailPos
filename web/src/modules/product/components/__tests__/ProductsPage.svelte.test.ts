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

  it('imports apiClient and auth store', () => {
    expect(src).toContain("import apiClient from '$shared/api/http-client'");
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('imports WebSocket utility', () => {
    expect(src).toContain("import { useWebSocket } from '$shared/api/websocket'");
  });

  it('imports toast store', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports child components (CategoryFilterModal, ProductFormModal, etc.)', () => {
    expect(src).toContain("import CategoryFilterModal");
    expect(src).toContain("import ProductActionsDropdown");
    expect(src).toContain("import ProductFormModal");
    expect(src).toContain("import StockAdjustModal");
  });

  it('imports shared UI components', () => {
    expect(src).toContain("import { Button, Modal, Pagination, ImportWizard, HistoryDialog } from '$shared/ui'");
  });

  it('uses $state for products, loading, pagination state', () => {
    expect(src).toContain('let loading = $state(true)');
    expect(src).toContain('let products = $state([])');
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

  it('has RBAC role-based guards for inventory and stock', () => {
    expect(src).toContain('allowedInventoryRoles');
    expect(src).toContain('allowedStockRoles');
  });

  it('has sortProducts function', () => {
    expect(src).toContain('function sortProducts');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});

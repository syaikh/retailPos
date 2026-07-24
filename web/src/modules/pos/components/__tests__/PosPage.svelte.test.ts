import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PosPage.svelte'), 'utf-8');
}

describe('PosPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiClient from shared/api', () => {
    expect(src).toContain("import apiClient from '$shared/api/http-client'");
  });

  it('imports goto from $app/router', () => {
    expect(src).toContain("import { goto } from '$app/router'");
  });

  it('imports toast store', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports printReceipt store', () => {
    expect(src).toContain("import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte'");
  });

  it('imports auth store and shift store', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
    expect(src).toContain("import { useShiftStore } from '$modules/shifts'");
  });

  it('imports ShoppingCart from lucide-svelte for paymentOptions', () => {
    expect(src).toContain("import { ShoppingCart, Hand, RotateCcw } from 'lucide-svelte'");
  });

  it('imports extracted child components', () => {
    expect(src).toContain("import ProductSearchPanel from './ProductSearchPanel.svelte'");
    expect(src).toContain("import PosProductTable from './PosProductTable.svelte'");
    expect(src).toContain("import CartPanel from './CartPanel.svelte'");
    expect(src).toContain("import CheckoutModal from './CheckoutModal.svelte'");
    expect(src).toContain("import CustomerSelectModal from './CustomerSelectModal.svelte'");
  });

  it('uses $state for cart, products, search state', () => {
    expect(src).toContain('let cart');
    expect(src).toContain('$state');
    expect(src).toContain('let products');
    expect(src).toContain('let searchQuery = $state');
    expect(src).toContain('let loading = $state');
  });

  it('uses $derived for subtotal, tax, total, change', () => {
    expect(src).toContain('const subtotal = $derived');
    expect(src).toContain('const taxAmount = $derived');
    expect(src).toContain('const totalAmount = $derived');
    expect(src).toContain('changeDue = $derived(cashReceived - totalAmount)');
  });

  it('has addToCart, removeFromCart, updateQty functions', () => {
    expect(src).toContain('function addToCart(product');
    expect(src).toContain('function handleSearchSubmit');
    expect(src).toContain('function removeFromCart(id');
    expect(src).toContain('function updateQty(id');
  });

  it('has displayProducts derived for stock calculation', () => {
    expect(src).toContain('displayProducts');
  });

  it('includes shift_id in checkout payload', () => {
    expect(src).toContain('shift_id: activeShift?.id || null');
  });

  it('redirects cashier to /shifts if no active shift on mount', () => {
    expect(src).toContain("goto('/shifts')");
  });

  it('has processCheckout and finalizeSale functions', () => {
    expect(src).toContain('async function processCheckout');
    expect(src).toContain('function finalizeSale');
  });

  it('has payment method options and state', () => {
    expect(src).toContain("let paymentMethod = $state('Cash')");
    expect(src).toContain('let paymentOptions = $state');
    expect(src).toContain('let checkingOut = $state');
  });

  it('handles customer selection', () => {
    expect(src).toContain('let customers: Customer[] = $state');
    expect(src).toContain('let selectedCustomerId');
  });

  it('has keyboard navigation state and scroll function', () => {
    expect(src).toContain('let selectedProductIndex = $state');
    expect(src).toContain('let productTableEl');
    expect(src).toContain('function scrollSelectedIntoView');
    expect(src).toContain('scrollIntoView');
  });

  it('handles arrow key navigation in product list', () => {
    expect(src).toContain("event.key === 'ArrowDown'");
    expect(src).toContain("event.key === 'ArrowUp'");
    expect(src).toContain("event.key === 'Enter' && selectedProductIndex >= 0");
  });

  it('binds selectedIndex and element to PosProductTable', () => {
    expect(src).toContain('bind:selectedIndex={selectedProductIndex}');
    expect(src).toContain('bind:element={productTableEl}');
  });

  it('renders child component tags', () => {
    expect(src).toContain('<ProductSearchPanel');
    expect(src).toContain('<PosProductTable');
    expect(src).toContain('<CartPanel');
    expect(src).toContain('<CheckoutModal');
    expect(src).toContain('<CustomerSelectModal');
  });
});

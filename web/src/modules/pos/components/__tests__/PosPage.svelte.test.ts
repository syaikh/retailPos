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

  it('imports toast store', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports printReceipt store', () => {
    expect(src).toContain("import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte'");
  });

  it('imports auth store', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  it('imports ShoppingCart from lucide-svelte for paymentOptions', () => {
    expect(src).toContain("import { ShoppingCart } from 'lucide-svelte'");
  });

  it('imports extracted child components', () => {
    expect(src).toContain("import ProductSearchPanel from './ProductSearchPanel.svelte'");
    expect(src).toContain("import PosProductTable from './PosProductTable.svelte'");
    expect(src).toContain("import CartPanel from './CartPanel.svelte'");
    expect(src).toContain("import CheckoutModal from './CheckoutModal.svelte'");
    expect(src).toContain("import CustomerSelectModal from './CustomerSelectModal.svelte'");
  });

  it('uses $state for cart, products, search state', () => {
    expect(src).toContain('let cart = $state([])');
    expect(src).toContain('let products = $state([])');
    expect(src).toContain('let searchQuery = $state');
    expect(src).toContain('let loading = $state');
  });

  it('uses $derived for subtotal, tax, total, change', () => {
    expect(src).toContain('const subtotal = $derived');
    expect(src).toContain('const taxAmount = $derived');
    expect(src).toContain('const totalAmount = $derived');
    expect(src).toContain('let changeDue = $derived(cashReceived - totalAmount)');
  });

  it('has addToCart, removeFromCart, updateQty functions', () => {
    expect(src).toContain('function addToCart(product)');
    expect(src).toContain('function handleSearchSubmit');
    expect(src).toContain('function removeFromCart(id)');
    expect(src).toContain('function updateQty(id, delta)');
  });

  it('has processCheckout and finalizeSale functions', () => {
    expect(src).toContain('async function processCheckout');
    expect(src).toContain('function finalizeSale');
  });

  it('has payment method options and state', () => {
    expect(src).toContain("let paymentMethod = $state('Cash')");
    expect(src).toContain('const paymentOptions');
    expect(src).toContain('let checkingOut = $state');
  });

  it('handles customer selection', () => {
    expect(src).toContain('let customers: Customer[] = $state');
    expect(src).toContain('let selectedCustomerId');
  });

  it('renders child component tags', () => {
    expect(src).toContain('<ProductSearchPanel');
    expect(src).toContain('<PosProductTable');
    expect(src).toContain('<CartPanel');
    expect(src).toContain('<CheckoutModal');
    expect(src).toContain('<CustomerSelectModal');
  });
});

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

describe('pos-store', () => {
  let store: ReturnType<typeof import('../pos-store.svelte').usePosStore>;

  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns expected API shape', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    expect(store).toHaveProperty('cart');
    expect(store).toHaveProperty('subtotal');
    expect(store).toHaveProperty('taxAmount');
    expect(store).toHaveProperty('totalAmount');
    expect(store).toHaveProperty('totalItems');
    expect(store).toHaveProperty('changeDue');
    expect(store).toHaveProperty('paymentMethod');
    expect(store).toHaveProperty('paymentOptions');
    expect(store).toHaveProperty('addToCart');
    expect(store).toHaveProperty('removeFromCart');
    expect(store).toHaveProperty('updateQty');
    expect(store).toHaveProperty('clearCart');
    expect(store).toHaveProperty('setPaymentMethod');
  });

  it('adds new item to cart', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    expect(store.cart).toHaveLength(1);
    expect(store.cart[0].quantity).toBe(1);
    expect(store.cart[0].name).toBe('Item1');
  });

  it('increments existing item in cart', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    expect(store.cart).toHaveLength(1);
    expect(store.cart[0].quantity).toBe(2);
  });

  it('does not add beyond max stock', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    for (let i = 0; i < 10; i++) {
      store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    }
    expect(store.cart[0].quantity).toBe(5);
  });

  it('removes item from cart', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.removeFromCart(1);
    expect(store.cart).toHaveLength(0);
  });

  it('updates quantity with positive delta', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 }); // starts at 1
    store.updateQty(1, 4); // 1 + 4 = 5
    expect(store.cart[0].quantity).toBe(5);
  });

  it('updates quantity with negative delta', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 }); // now 2
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 }); // now 3
    store.updateQty(1, -1);
    expect(store.cart[0].quantity).toBe(2);
  });

  it('removes item when quantity drops to zero', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.updateQty(1, -1);
    expect(store.cart).toHaveLength(0);
  });

  it('clamps quantity to max stock', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.updateQty(1, 10); // exceeds stock of 5, should clamp to 5
    expect(store.cart[0].quantity).toBe(5);
  });

  it('clears cart', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.clearCart();
    expect(store.cart).toHaveLength(0);
  });

  it('sets payment method', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.setPaymentMethod('Card');
    expect(store.paymentMethod).toBe('Card');
  });

  it('calculates subtotal from cart', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 });
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 }); // now 2
    expect(store.subtotal).toBe(200);
  });

  it('calculates tax amount with tax rate', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5, tax_rate: 11 });
    // tax = price * quantity - dpp = 100 - 90 = 10 (rounded)
    expect(store.taxAmount).toBe(10);
  });

  it('calculates total items', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 }); // adds with quantity 1
    store.addToCart({ id: 1, name: 'Item1', price: 100, quantity: 1, stock: 5 }); // increments to 2
    store.addToCart({ id: 2, name: 'Item2', price: 50, quantity: 1, stock: 5 }); // adds with quantity 1
    store.addToCart({ id: 2, name: 'Item2', price: 50, quantity: 1, stock: 5 }); // increments to 2
    store.addToCart({ id: 2, name: 'Item2', price: 50, quantity: 1, stock: 5 }); // increments to 3
    expect(store.totalItems).toBe(5);
  });

  it('returns payment options', async () => {
    const { usePosStore } = await import('../pos-store.svelte');
    store = usePosStore();
    expect(store.paymentOptions).toHaveLength(3);
    expect(store.paymentOptions).toContainEqual({ id: 'Cash', label: 'Cash', icon: 'wallet' });
    expect(store.paymentOptions).toContainEqual({ id: 'Card', label: 'Card', icon: 'card' });
    expect(store.paymentOptions).toContainEqual({ id: 'E-Wallet', label: 'E-Wallet', icon: 'wallet' });
  });
});
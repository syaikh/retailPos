import type { CartItem, PaymentOption } from '../types';

let cart = $state<CartItem[]>([]);
let paymentMethod = $state('Cash');
let checkingOut = $state(false);
let cashReceived = $state(0);
let selectedCustomerId = $state<number | null>(null);

let subtotal = $derived(cart.reduce((sum, item) => sum + item.price * item.quantity, 0));
let taxAmount = $derived(cart.reduce((sum, item) => {
  const rate = item.tax_rate || 0;
  if (rate <= 0) return sum;
  const lineTotal = item.price * item.quantity;
  const dpp = Math.round(lineTotal * 100 / (100 + rate));
  return sum + (lineTotal - dpp);
}, 0));
let totalAmount = $derived(subtotal);
let totalItems = $derived(cart.reduce((sum, item) => sum + item.quantity, 0));
let changeDue = $derived(cashReceived - totalAmount);

const paymentOptions: PaymentOption[] = [
  { id: 'Cash', label: 'Cash', icon: 'wallet' },
  { id: 'Card', label: 'Card', icon: 'card' },
  { id: 'E-Wallet', label: 'E-Wallet', icon: 'wallet' },
];

export function usePosStore() {
  function addToCart(product: CartItem) {
    const existing = cart.find((item) => item.id === product.id);
    if (existing) {
      const maxStock = existing.stock || 999;
      if (existing.quantity >= maxStock) return;
      existing.quantity++;
      cart = [...cart];
    } else {
      cart = [...cart, { ...product, quantity: 1 }];
    }
  }

  function removeFromCart(id: number) {
    cart = cart.filter((item) => item.id !== id);
  }

  function updateQty(id: number, delta: number) {
    const item = cart.find((i) => i.id === id);
    if (!item) return;
    const newQty = item.quantity + delta;
    const maxStock = item.stock || 999;
    if (newQty <= 0) {
      removeFromCart(id);
    } else if (newQty > maxStock) {
      item.quantity = maxStock;
    } else {
      item.quantity = newQty;
    }
    cart = [...cart];
  }

  function clearCart() {
    cart = [];
  }

  function setPaymentMethod(method: string) {
    paymentMethod = method;
  }

  function setCheckingOut(v: boolean) {
    checkingOut = v;
  }

  function setCashReceived(v: number) {
    cashReceived = v;
  }

  function setSelectedCustomerId(id: number | null) {
    selectedCustomerId = id;
  }

  return {
    get cart() { return cart; },
    get paymentMethod() { return paymentMethod; },
    set paymentMethod(v: string) { paymentMethod = v; },
    get checkingOut() { return checkingOut; },
    set checkingOut(v: boolean) { checkingOut = v; },
    get cashReceived() { return cashReceived; },
    set cashReceived(v: number) { cashReceived = v; },
    get selectedCustomerId() { return selectedCustomerId; },
    set selectedCustomerId(v: number | null) { selectedCustomerId = v; },
    get subtotal() { return subtotal; },
    get taxAmount() { return taxAmount; },
    get totalAmount() { return totalAmount; },
    get totalItems() { return totalItems; },
    get changeDue() { return changeDue; },
    get paymentOptions() { return paymentOptions; },
    addToCart,
    removeFromCart,
    updateQty,
    clearCart,
    setPaymentMethod,
    setCheckingOut,
    setCashReceived,
    setSelectedCustomerId,
  };
}

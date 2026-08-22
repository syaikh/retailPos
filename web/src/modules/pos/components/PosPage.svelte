<script lang="ts">
   import { onMount } from 'svelte';
   import { goto } from '$app/router';
   import apiClient from '$shared/api/http-client';
   import { toast } from '$shared/stores/toast.svelte';
   import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte';
   import { debounce } from '$shared/utils/debounce';
   import { useWebSocket } from '$shared/api/websocket';
   import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime';
   import {
     createCart, getOpenCart, getHeldCarts,
     addCartItem, updateCartItemQuantity, removeCartItem,
      holdCart, resumeCart, cancelCart, checkoutCart, updateCartCustomer,
    } from '../services/pos-service';
   import type { PaymentAllocation, CartItem, CartSession } from '../types';
   import type { Sale, SaleItem } from '$modules/sales/types';
  import type { Customer } from '$modules/customers/types';

    import { ShoppingCart, Hand, RotateCcw } from 'lucide-svelte';
    import { useAuthStore } from '$modules/auth';
    import { useShiftStore } from '$modules/shifts';
    import { labels, t } from '$shared/i18n';
    import ProductSearchPanel from './ProductSearchPanel.svelte';
    import PosProductTable from './PosProductTable.svelte';
    import CartPanel from './CartPanel.svelte';
    import CheckoutModal from './CheckoutModal.svelte';
    import CustomerSelectModal from './CustomerSelectModal.svelte';
    import ParkedSalesModal from './ParkedSalesModal.svelte';

const authStore = useAuthStore();

  interface DisplayCartItem {
    id: number;
    product_id: number;
    name: string;
    price: number;
    original_price: number;
    quantity: number;
    stock: number;
    tax_rate?: number;
    sku?: string;
    barcode?: string;
    pricing_rule_id?: number;
    pricing_rule_name?: string;
    pricing_rule_type?: string;
    pricing_type?: string;
    discount?: number;
    snapshot_created_at?: string;
    [key: string]: unknown;
  }

   let products: any[] = $state([]);
let total: number = $state(0);
   let searchQuery = $state('');
   let loading = $state(false);
   let limit = $state(20);
   let offset = $state(0);
   let isInitialMount = $state(true);
   let isSearching = $state(false);
   let lastSale: Sale | null = $state(null);
   let ws = useWebSocket();
let selectedProductIndex = $state(-1);

  let unsubscribeStock: (() => void) | null = null;
  let unsubscribeSale: (() => void) | null = null;

  let previousSearchQuery = '';
  let showCopySuccess: Set<string> | null = $state(null);
  let warningThreshold = $state(10);
  let criticalThreshold = $state(5);

    let paymentOptions = $state<Array<{ id: string; label: string; icon: any; requiresReference?: boolean }>>([]);
   let checkingOut = $state(false);

    let showCheckoutModal = $state(false);
    let showCart = $state(false);
    let capturedPayments = $state<PaymentAllocation[]>([]);

  let heldCarts = $state<CartSession[]>([]);
  let showParkedModal = $state(false);
  let holdingSale = $state(false);

  let activeCartId = $state<number | null>(null);
  let cartSession = $state<CartSession | null>(null);
  let cartItems: CartItem[] = $state([]);
  const cart = $derived(cartItems.map(ci => toDisplayItem(ci)));
  let cartLoading = $state(false);

   let customers: Customer[] = $state([]);
   let selectedCustomerId = $state<number | null>(null);
   let showCustomerModal = $state(false);
   let customerSearch = $state('');
   let customerResults: Customer[] = $state([]);
   let customerSearching = $state(false);
   let selectedCustomerLabel = $derived((() => {
     if (!selectedCustomerId) return labels.walkInGeneral;
     return customers.find(c => c.id === selectedCustomerId)?.name
       || customerResults.find(c => c.id === selectedCustomerId)?.name
       || '';
   })());

  const subtotal = $derived(cartSession?.subtotal ?? 0);
  const taxAmount = $derived(cartSession?.tax ?? 0);
  const taxDisplay = $derived(taxAmount);
  const totalAmount = $derived(cartSession?.total_amount ?? 0);
  const dppDisplay = $derived(subtotal - taxAmount);
  const totalItems = $derived(cartItems.reduce((sum, item) => sum + item.quantity, 0));

  const displayProducts = $derived(products.map(p => ({
    ...p,
    stock: Math.max(0, p.stock - cartItems.filter(c => c.product_id === p.id).reduce((sum, c) => sum + c.quantity, 0)),
  })));

  function toDisplayItem(ci: CartItem): DisplayCartItem {
    const product = products.find(p => p.id === ci.product_id);
    return {
      id: ci.id,
      product_id: ci.product_id,
      name: ci.product_name,
      price: ci.unit_price,
      original_price: ci.original_price,
      quantity: ci.quantity,
      stock: product?.stock ?? 999,
      tax_rate: ci.tax_rate,
      sku: product?.sku || '',
      barcode: product?.barcode,
      pricing_rule_id: ci.pricing_rule_id,
      pricing_rule_name: ci.pricing_rule_name,
      pricing_rule_type: ci.pricing_rule_type,
      pricing_type: ci.pricing_type,
      discount: ci.discount,
      snapshot_created_at: ci.snapshot_created_at,
    };
  }

  function applyCartSession(session: CartSession) {
    cartSession = session;
    cartItems = session.items || [];
    activeCartId = session.id;
  }

  async function fetchProducts(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const r = await apiClient.get(`/products?limit=${limit}&offset=${offset}&search=${searchQuery}&status=active`);
      products = r.data.data || [];
      total = r.data.total || 0;
      selectedProductIndex = products.length > 0 ? 0 : -1;
    } catch (err) {
      toast.error(labels.toastFailedToLoadProducts);
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts(true);
  }, 400);

  async function fetchCustomers() {
    try {
      const r = await apiClient.get('/customers?limit=200');
      customers = r.data.data || [];
    } catch (err) {
      console.warn('Failed to load customers', err);
    }
  }

  async function loadPaymentMethods() {
    try {
      const r = await apiClient.get('/payment-methods');
      const methods = (r.data.data || r.data || []) as Array<{ code: string; name: string; is_active?: boolean; requires_reference?: boolean }>;
      const active = methods.filter(m => m.is_active !== false);
      if (active.length > 0) {
        paymentOptions = active.map(m => ({ id: m.code, label: m.name, icon: ShoppingCart, requiresReference: m.requires_reference }));
      }
    } catch (err) {
      console.warn('Failed to load payment methods', err);
      paymentOptions = [
        { id: 'CASH', label: 'Cash', icon: ShoppingCart },
        { id: 'CARD', label: 'Card', icon: ShoppingCart, requiresReference: true },
        { id: 'E_WALLET', label: 'E-Wallet', icon: ShoppingCart, requiresReference: true },
      ];
    }
  }

  async function searchCustomers() {
    if (!customerSearch.trim()) { customerResults = []; return; }
    customerSearching = true;
    try {
      const r = await apiClient.get('/customers', { params: { search: customerSearch.trim(), limit: 10 } });
      customerResults = r.data.data || [];
    } catch (e) { console.warn('customer search failed', e); }
    customerSearching = false;
  }

  $effect(() => {
    if (isInitialMount) return;
    if (previousSearchQuery === searchQuery) return;
    previousSearchQuery = searchQuery;
    if (searchQuery === '') {
      offset = 0;
      isSearching = false;
      fetchProducts(false);
    } else {
      isSearching = true;
      debouncedSearch();
    }
  });

  const debouncedCustomerSearch = debounce(() => {
    if (customerSearch.trim()) {
      searchCustomers();
    } else {
      customerResults = [];
    }
  }, 400);

  $effect(() => {
    if (customerSearch.trim()) {
      debouncedCustomerSearch();
    } else {
      customerResults = [];
    }
  });

  function handleSearchSubmit() {
    if (products.length > 0) {
      addToCart(products[selectedProductIndex >= 0 ? selectedProductIndex : 0]);
    }
  }

  async function ensureCart(): Promise<number> {
    if (activeCartId) return activeCartId;
    const shiftStore = useShiftStore();
    const payload: { store_id?: number; shift_id?: number; customer_id?: number } = {
      store_id: (authStore.user as any)?.store_id,
    };
    if (shiftStore.activeShift?.id) payload.shift_id = shiftStore.activeShift.id;
    if (selectedCustomerId) payload.customer_id = selectedCustomerId;
    const cart = await createCart(payload);
    applyCartSession(cart);
    return cart.id;
  }

  async function addToCart(product: CartItem) {
    if (cartLoading) return;
    cartLoading = true;
    try {
      const cartId = await ensureCart();
      const existing = cartItems.find(ci => ci.product_id === product.id);
      const selectedCustomer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : null;
      const customerGroupId = (selectedCustomer as any)?.customer_group_id || undefined;
      const shiftStore = useShiftStore();
      const shiftId = shiftStore.activeShift?.id;
      let session: CartSession;
      if (existing) {
        session = await updateCartItemQuantity(cartId, existing.id, existing.quantity + 1);
      } else {
        session = await addCartItem(cartId, {
          product_id: product.id,
          quantity: 1,
          customer_group_id: customerGroupId,
          store_id: cartSession?.store_id,
          shift_id: shiftId,
          customer_id: selectedCustomerId || undefined,
        });
      }
      applyCartSession(session);
    } catch (err: any) {
      const errMsg = err.response?.data?.error || err.message || labels.toastFailedToAddItem;
      toast.error(typeof errMsg === 'string' ? errMsg : errMsg?.message || labels.toastFailedToAddItem);
    } finally {
      cartLoading = false;
    }
  }

  async function removeFromCart(id: number) {
    if (!activeCartId) return;
    cartLoading = true;
    try {
      const session = await removeCartItem(activeCartId, id);
      applyCartSession(session);
    } catch (err: any) {
      const errMsg = err.response?.data?.error || err.message || labels.toastFailedToRemoveItem;
      toast.error(typeof errMsg === 'string' ? errMsg : errMsg?.message || labels.toastFailedToRemoveItem);
    } finally {
      cartLoading = false;
    }
  }

  async function updateQty(id: number, delta: number) {
    const item = cartItems.find(i => i.id === id);
    if (!item || !activeCartId) return;
    const newQty = item.quantity + delta;
    const maxStock = products.find(p => p.id === item.product_id)?.stock ?? 999;
    if (newQty <= 0) {
      await removeFromCart(id);
      return;
    }
    const finalQty = newQty > maxStock ? maxStock : newQty;
    if (finalQty !== item.quantity) {
      cartLoading = true;
      try {
        const session = await updateCartItemQuantity(activeCartId, id, finalQty);
        applyCartSession(session);
      } catch (err: any) {
        const errMsg = err.response?.data?.error || err.message || labels.toastFailedToUpdateQuantity;
        toast.error(typeof errMsg === 'string' ? errMsg : errMsg?.message || labels.toastFailedToUpdateQuantity);
      } finally {
        cartLoading = false;
      }
    }
  }

  function copyToClipboard(value: string, field: string, ms = 2000): void {
    navigator.clipboard.writeText(value).then(() => {
      const base = showCopySuccess || new Set();
      const next = new Set(base);
      next.add(field);
      showCopySuccess = next;
      setTimeout(() => {
        const removed = new Set(next);
        removed.delete(field);
        showCopySuccess = removed;
      }, ms);
    });
  }

   async function processCheckout(payments?: PaymentAllocation[]) {
    if (!activeCartId || cartItems.length === 0) {
      toast.error(labels.toastCartIsEmpty);
      return;
    }
    checkingOut = true;
    try {
      const response = await checkoutCart(activeCartId, payments || []);
      lastSale = response as Sale;
      if (payments) {
        capturedPayments = payments;
      }
      toast.success(labels.toastSaleCompleted);
      cartSession = null;
      cartItems = [];
      activeCartId = null;
      await fetchProducts(false);
    } catch (err: any) {
      const errData = err.response?.data?.error;
      const errMsg = typeof errData === 'string' ? errData : errData?.message || labels.toastCheckoutFailed;
      toast.error(errMsg);
      throw err;
    } finally {
      checkingOut = false;
    }
  }

  async function clearCart() {
    if (!activeCartId) return;
    const cartId = activeCartId;
    cartLoading = true;
    try {
      let lastSession: CartSession | null = null;
      for (const item of cartItems) {
        lastSession = await removeCartItem(cartId, item.id);
      }
      if (lastSession) applyCartSession(lastSession);
    } catch (err: any) {
      const errMsg = err.response?.data?.error || err.message || labels.toastFailedToClearCart;
      toast.error(typeof errMsg === 'string' ? errMsg : errMsg?.message || labels.toastFailedToClearCart);
    } finally {
      cartLoading = false;
    }
  }

  async function holdSale() {
    if (!activeCartId || cartItems.length === 0) {
      toast.error(labels.toastCartIsEmpty);
      return;
    }
    holdingSale = true;
    try {
      await holdCart(activeCartId);
      toast.success(labels.toastSaleHeld);
      cartSession = null;
      cartItems = [];
      activeCartId = null;
      await fetchHeldCarts();
    } catch (err: any) {
      const errData = err.response?.data?.error;
      toast.error(typeof errData === 'string' ? errData : errData?.message || labels.toastFailedToHoldSale);
    } finally {
      holdingSale = false;
    }
  }

  async function fetchHeldCarts() {
    try {
      heldCarts = await getHeldCarts();
    } catch (err) {
      console.warn('Failed to load held carts', err);
    }
  }

  async function recallSale(cartId: number) {
    try {
      const session = await resumeCart(cartId);
      applyCartSession(session);
      showParkedModal = false;
      toast.success(labels.toastSaleResumed);
      await fetchHeldCarts();
    } catch (err: any) {
      const errData = err.response?.data?.error;
      toast.error(typeof errData === 'string' ? errData : errData?.message || labels.toastFailedToResumeSale);
    }
  }

  async function cancelSale(cartId: number) {
    try {
      await cancelCart(cartId);
      toast.success(labels.toastSaleDiscarded);
      await fetchHeldCarts();
    } catch (err: any) {
      const errData = err.response?.data?.error;
      toast.error(typeof errData === 'string' ? errData : errData?.message || labels.toastFailedToDiscardSale);
    }
  }

   function buildReceiptPayload(sale: Sale, payments: PaymentAllocation[], customer: Customer | undefined) {
    const saleTaxAmount = sale.tax || 0;
    const paymentsList = payments.length > 0
      ? payments.map(p => `${p.payment_method_code}: ${labels.currencySymbol} ${p.amount.toLocaleString('id-ID')}`).join(', ')
      : (sale.payment_method || labels.cash);
    return {
      invoice_number: sale.invoice_number,
      created_at: sale.created_at,
      items: (sale.items || []).map((item) => ({
        name: item.name || '',
        quantity: item.quantity,
        unit_price: item.unit_price,
        original_price: item.original_price,
        pricing_rule_name: item.pricing_rule_name,
        pricing_type: item.pricing_type,
      })),
      total_amount: sale.total_amount,
      subtotal_dpp: sale.total_amount - saleTaxAmount,
      tax: saleTaxAmount,
      paymentMethod: paymentsList,
      payments: payments.map(p => ({ method: p.payment_method_code, amount: p.amount })),
      cashReceived: payments.find(p => p.payment_method_code === 'CASH')?.amount || sale.total_amount,
      changeDue: 0,
      customer_name: customer?.name,
      total_savings: (sale.items || []).reduce((sum: number, item: any) => {
        if (item.original_price && item.original_price > item.unit_price) {
          return sum + (item.original_price - item.unit_price) * item.quantity;
        }
        return sum;
      }, 0),
    };
  }

   async function printReceipt() {
    if (!lastSale || !lastSale.invoice_number) return;
    let sale = lastSale;
    if (!sale.items || sale.items.length === 0) {
      try {
        const detail = await apiClient.get(`/sales/${sale.id}`);
        sale = detail.data?.data || detail.data;
      } catch (_) { return; }
    }
    if (!sale || !sale.items || sale.items.length === 0) return;
    const customer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : undefined;
    printReceiptStore.set(buildReceiptPayload(sale, paymentsListFromSale(sale), customer));
    setTimeout(() => {
      window.print();
      setTimeout(() => printReceiptStore.set(null), 1000);
    }, 300);
   }

  function paymentsListFromSale(sale: Sale): PaymentAllocation[] {
    if (capturedPayments.length > 0) return capturedPayments;
    if (sale.payment_method) {
      return sale.payment_method.split(',').map((code: string) => ({
        payment_method_code: code.trim(),
        amount: sale.total_amount,
      }));
    }
    return [];
  }

  function handlePageChange(newOffset: number) {
    offset = newOffset;
    fetchProducts(false);
  }

   function openCheckoutModal() {
    if (cartItems.length === 0) {
      toast.error(labels.toastCartIsEmpty);
      return;
    }
    showCheckoutModal = true;
   }

   function closeCheckoutModal() {
    showCheckoutModal = false;
    capturedPayments = [];
   }

    function finalizeSale(payments: PaymentAllocation[]) {
     if (cartItems.length === 0) return;
     const customer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : undefined;
     capturedPayments = payments;
     closeCheckoutModal();
     processCheckout(payments).then(() => {
       if (lastSale && lastSale.items) {
         const taxAmt = lastSale.tax || 0;
         const paymentLines = payments.map(p => `${p.payment_method_code}: ${labels.currencySymbol} ${p.amount.toLocaleString('id-ID')}`).join(', ');
         printReceiptStore.set(buildReceiptPayload(lastSale, payments, customer));
         setTimeout(() => {
           window.print();
           setTimeout(() => printReceiptStore.set(null), 1000);
         }, 300);
       }
      }).catch((err: any) => {
        // processCheckout already surfaced the specific server error toast;
        // avoid a second generic toast for the same failure.
        console.error('Checkout failed:', err);
      });
    }

   $effect(() => {
    if (showCheckoutModal) {
      // CheckoutModal handles its own focus via onkeydown
    }
   });

  let productTableEl: HTMLElement | undefined = $state();

  function handleGlobalKeydown(event: KeyboardEvent) {
    if (event.altKey && event.key === 'Delete') {
      event.preventDefault();
      if (cartItems.length > 0) {
        clearCart();
        toast.info(labels.toastCartCleared);
      }
      return;
    }
    if (event.key === 'F2') {
      event.preventDefault();
      const input = document.getElementById('pos-search-input');
      if (input) {
        input.focus();
        (input as HTMLInputElement).select();
      }
      return;
    }
    // ESC clears search when not in modal
    if (event.key === 'Escape' && !showCheckoutModal && !showParkedModal && !showCustomerModal) {
      if (searchQuery) {
        searchQuery = '';
        fetchProducts(false);
      }
      return;
    }
    if (event.key === 'F4') {
      event.preventDefault();
      openCheckoutModal();
      return;
    }
    if (event.key === 'F7') {
      // While a modal is open it owns F7 (checkout uses it for "Exact");
      // only open the parked-sales modal from the bare POS screen.
      if (showCheckoutModal || showParkedModal || showCustomerModal) return;
      event.preventDefault();
      fetchHeldCarts().then(() => { showParkedModal = true; });
      return;
    }
    if (event.key === 'F6') {
      event.preventDefault();
      if (cartItems.length > 0) {
        holdSale();
      }
      return;
    }
    if (showCheckoutModal) {
      return;
    }
    if (showParkedModal || showCustomerModal) return;

    if (products.length === 0 || loading) return;

    const target = event.target as HTMLElement | null;
    const isSearchInput = target?.id === 'pos-search-input';
    const tag = target?.tagName ?? '';
    const isEditable = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';
    const isInteractive = tag === 'BUTTON' || tag === 'A';

    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      if (isEditable && !isSearchInput) return;
      event.preventDefault();
      selectedProductIndex = event.key === 'ArrowDown'
        ? Math.min(selectedProductIndex + 1, products.length - 1)
        : Math.max(selectedProductIndex - 1, 0);
      scrollSelectedIntoView();
      return;
    }
    if (event.key === 'Enter' && selectedProductIndex >= 0 && !isEditable && !isInteractive) {
      event.preventDefault();
      addToCart(products[selectedProductIndex]);
      return;
    }
  }

  function scrollSelectedIntoView() {
    if (!productTableEl) return;
    const rows = productTableEl.querySelectorAll('tbody tr');
    const row = rows[selectedProductIndex] as HTMLElement | undefined;
    if (row) row.scrollIntoView({ block: 'nearest' });
  }

  async function fetchThresholds() {
    try {
      const r = await apiClient.get('/stock-thresholds');
      warningThreshold = r.data.warning ?? 10;
      criticalThreshold = r.data.critical ?? 5;
    } catch {
      warningThreshold = 10;
      criticalThreshold = 5;
    }
  }

  function focusSearch() {
    setTimeout(() => {
      const input = document.getElementById('pos-search-input');
      if (input) {
        input.focus();
        (input as HTMLInputElement).select();
      }
    }, 50);
  }

  onMount(() => {
    (async () => {
      const shiftStore = useShiftStore();
      await shiftStore.loadActiveShift();
      // @display-only — flow guard navigasi UX: cashier tanpa shift aktif diarahkan ke /shifts.
      const userRole = typeof authStore.user?.role === 'string' ? authStore.user.role : authStore.user?.role?.name ?? '';
      if (userRole === 'cashier' && !shiftStore.activeShift) {
        toast.error(labels.toastMustOpenShiftFirst);
        goto('/shifts');
        return;
      }

      isInitialMount = true;
      await Promise.all([fetchProducts(false), fetchThresholds(), loadPaymentMethods()]);
      fetchCustomers();
      isInitialMount = false;
      focusSearch();
      try {
        const open = await getOpenCart();
        applyCartSession(open);
      } catch (_) { /* no open cart yet */ }
      try {
        const endDate = getTodayInJakarta();
        const startDate = getDateNDaysAgoInJakarta(7);
        const r = await apiClient.get(`/sales?limit=1&offset=0&startDate=${startDate}&endDate=${endDate}`);
        const body = r.data;
        let data = body?.data || body;
        if (Array.isArray(data) && data.length > 0) {
          lastSale = data[0];
        } else if (data && !Array.isArray(data) && (data as any).id) {
          lastSale = data;
        }
        if (lastSale && !lastSale.items) {
          const detail = await apiClient.get(`/sales/${lastSale.id}`);
          const detailData = detail.data?.data || detail.data;
          if (detailData?.items) {
            lastSale = detailData;
          }
        }
      } catch (err: any) {
        console.warn('[POS] Failed to load last sale:', err?.response?.data?.error || err?.message);
      }
      unsubscribeStock = ws.on('stock_update', (data: any) => {
        const product = products.find(p => p.id === data.id);
        if (product) {
          product.stock = data.stock;
        }
      });
      unsubscribeSale = ws.on('sale_created', (data: any) => {
        if (data) {
          lastSale = data as Sale;
        }
      });
    })();
    return () => {
      if (unsubscribeStock) unsubscribeStock();
      if (unsubscribeSale) unsubscribeSale();
    };
  });
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="flex flex-col lg:flex-row gap-4 lg:gap-6">
  <!-- Product area -->
  <div class="flex-1 flex flex-col gap-4">
    <ProductSearchPanel bind:searchQuery onsearchsubmit={handleSearchSubmit} />
    <div class="card flex-1 overflow-hidden flex flex-col p-0">
      <PosProductTable
        products={displayProducts}
        {loading}
        {total}
        {limit}
        {offset}
        {warningThreshold}
        {criticalThreshold}
        bind:showCopySuccess
        bind:selectedIndex={selectedProductIndex}
        bind:element={productTableEl}
        onaddtocart={addToCart}
        oncopy={copyToClipboard}
        onpagechange={handlePageChange}
      />
    </div>
  </div>

  <!-- Cart: toggleable bottom panel on tablet/mobile, side panel on desktop -->
  <div class="lg:hidden">
    <button
      type="button"
      class="w-full flex items-center justify-between px-4 py-3 rounded-xl border border-border bg-surface-default text-sm font-medium text-text-primary"
      onclick={() => showCart = !showCart}
    >
      <span class="flex items-center gap-2">
        <span class="text-primary-light">{labels.cart}</span>
        {#if totalItems > 0}
          <span class="px-2 py-0.5 rounded-full bg-primary-subtle text-primary-light text-xs font-semibold">{t('itemsCount', { count: totalItems })}</span>
        {/if}
      </span>
      <span class="text-text-muted">{showCart ? labels.hide : labels.show}</span>
    </button>
    {#if showCart}
      <div class="mt-2">
        <CartPanel
          {cart}
          {totalAmount}
          {totalItems}
          {subtotal}
          {taxAmount}
          {dppDisplay}
          {lastSale}
          {checkingOut}
          parkedSaleCount={heldCarts.length}
          onupdateqty={updateQty}
          onremovefromcart={removeFromCart}
          onclearcart={clearCart}
          oncheckout={openCheckoutModal}
          onprintreceipt={printReceipt}
          onholdsale={holdSale}
          onopenparkedmodal={() => { fetchHeldCarts().then(() => { showParkedModal = true; }); }}
          class="!h-auto !max-h-none !sticky-none"
        />
      </div>
    {/if}
  </div>

  <!-- Desktop cart (side panel) -->
  <div class="hidden lg:block w-[420px] shrink-0">
    <CartPanel
      {cart}
      {totalAmount}
      {totalItems}
      {subtotal}
      {taxAmount}
      {dppDisplay}
      {lastSale}
      {checkingOut}
      parkedSaleCount={heldCarts.length}
      onupdateqty={updateQty}
      onremovefromcart={removeFromCart}
      onclearcart={clearCart}
      oncheckout={openCheckoutModal}
      onprintreceipt={printReceipt}
      onholdsale={holdSale}
      onopenparkedmodal={() => { fetchHeldCarts().then(() => { showParkedModal = true; }); }}
    />
  </div>
</div>

<CheckoutModal
  bind:showCheckoutModal
  {cart}
  {totalAmount}
  {subtotal}
  {taxAmount}
  {dppDisplay}
  {paymentOptions}
  {selectedCustomerLabel}
  {checkingOut}
  onfinalize={finalizeSale}
  onselectcustomer={() => { showCustomerModal = true; }}
/>

<CustomerSelectModal
  bind:showCustomerModal
  bind:customerSearch
  {customerResults}
  {customerSearching}
  onselectcustomer={(id) => {
    selectedCustomerId = id;
    if (id) {
      const found = customerResults.find(c => c.id === id);
      if (found && !customers.find(c => c.id === id)) {
        customers = [...customers, found];
      }
    }
    if (activeCartId) {
      updateCartCustomer(activeCartId, id).then(applyCartSession).catch(() => {});
    }
    showCustomerModal = false;
  }}
/>

<ParkedSalesModal
  bind:showModal={showParkedModal}
  {heldCarts}
  onrecall={recallSale}
  oncancel={cancelSale}
/>

<style>
</style>

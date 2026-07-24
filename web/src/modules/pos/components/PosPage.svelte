<script lang="ts">
   import { onMount } from 'svelte';
   import { goto } from '$app/router';
   import apiClient from '$shared/api/http-client';
   import { toast } from '$shared/stores/toast.svelte';
   import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte';
   import { debounce } from '$shared/utils/debounce';
   import { useWebSocket } from '$shared/api/websocket';
   import { getTodayInJakarta } from '$shared/utils/jakartaTime';
   import { resolvePrices } from '$modules/pricing/services/pricing-service';
   import { parkSale, listParkedSales, recallParkedSale, cancelParkedSale } from '../services/pos-service';
   import type { Sale, SaleItem } from '$modules/sales/types';
  import type { Customer } from '$modules/customers/types';

    import { ShoppingCart, Hand, RotateCcw } from 'lucide-svelte';
    import { useAuthStore } from '$modules/auth';
    import { useShiftStore } from '$modules/shifts';
    import ProductSearchPanel from './ProductSearchPanel.svelte';
    import PosProductTable from './PosProductTable.svelte';
    import CartPanel from './CartPanel.svelte';
    import CheckoutModal from './CheckoutModal.svelte';
    import CustomerSelectModal from './CustomerSelectModal.svelte';
    import ParkedSalesModal from './ParkedSalesModal.svelte';

const authStore = useAuthStore();

  interface CartItem {
    id: number;
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
    [key: string]: unknown;
  }

   let cart: CartItem[] = $state([]);
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

   let paymentOptions = $state<Array<{ id: string; label: string; icon: any }>>([]);
  let paymentMethod = $state('Cash');
  let checkingOut = $state(false);

   let showCheckoutModal = $state(false);
   let showCart = $state(false);
   let cashReceived = $state(0);

  let parkedSales = $state<any[]>([]);
  let showParkedModal = $state(false);
  let holdingSale = $state(false);
  let recalledSaleId = $state<number | null>(null);

   let customers: Customer[] = $state([]);
   let selectedCustomerId = $state<number | null>(null);
   let showCustomerModal = $state(false);
   let customerSearch = $state('');
   let customerResults: Customer[] = $state([]);
   let customerSearching = $state(false);
   let selectedCustomerLabel = $derived((() => {
     if (!selectedCustomerId) return 'Walk-in / General';
     return customers.find(c => c.id === selectedCustomerId)?.name
       || customerResults.find(c => c.id === selectedCustomerId)?.name
       || '';
   })());

  const subtotal = $derived(cart.reduce((sum, item) => sum + item.price * item.quantity, 0));
  const taxAmount = $derived(cart.reduce((sum, item) => {
    const rate = item.tax_rate || 0;
    if (rate <= 0) return sum;
    const lineTotal = item.price * item.quantity;
    const dpp = Math.round(lineTotal * 100 / (100 + rate));
    return sum + (lineTotal - dpp);
  }, 0));
  const taxDisplay = $derived(taxAmount); 
  const totalAmount = $derived(subtotal);
  const changeDue = $derived(cashReceived - totalAmount);
  const dppDisplay = $derived(subtotal - taxAmount);
  const totalItems = $derived(cart.reduce((sum, item) => sum + item.quantity, 0));

  const displayProducts = $derived(products.map(p => ({
    ...p,
    stock: Math.max(0, p.stock - cart.filter(c => c.id === p.id).reduce((sum, c) => sum + c.quantity, 0)),
  })));

  async function fetchProducts(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const r = await apiClient.get(`/products?limit=${limit}&offset=${offset}&search=${searchQuery}&status=active`);
      products = r.data.data || [];
      total = r.data.total || 0;
      selectedProductIndex = products.length > 0 ? 0 : -1;
    } catch (err) {
      toast.error('Failed to load products');
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
      const methods = (r.data.data || r.data || []) as Array<{ code: string; name: string; is_active?: boolean }>;
      const active = methods.filter(m => m.is_active !== false);
      if (active.length > 0) {
        paymentOptions = active.map(m => ({ id: m.code, label: m.name, icon: ShoppingCart }));
        if (!paymentOptions.some(p => p.id === paymentMethod)) {
          paymentMethod = paymentOptions[0].id;
        }
      }
    } catch (err) {
      console.warn('Failed to load payment methods', err);
      paymentOptions = [
        { id: 'CASH', label: 'Cash', icon: ShoppingCart },
        { id: 'CARD', label: 'Card', icon: ShoppingCart },
        { id: 'E_WALLET', label: 'E-Wallet', icon: ShoppingCart },
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
      addToCart(products[0]);
    }
  }

  function addToCart(product: CartItem) {
    const existing = cart.find((item) => item.id === product.id);
    if (existing) {
      const maxStock = existing.stock || 999;
      if (existing.quantity >= maxStock) {
        toast.info(`Max stock reached: ${existing.name} (${maxStock})`);
        return;
      }
      existing.quantity++;
      cart = cart;
    } else {
      cart.push({ ...product, quantity: 1, original_price: product.price });
      cart = cart;
    }
    resolveCartPrices();
  }

  function removeFromCart(id: number) {
    const idx = cart.findIndex((item) => item.id === id);
    if (idx !== -1) cart.splice(idx, 1);
    cart = cart;
    resolveCartPrices();
  }

  function updateQty(id: number, delta: number) {
    const item = cart.find((i) => i.id === id);
    if (item) {
      const newQty = item.quantity + delta;
      const maxStock = item.stock || 999;
      if (newQty <= 0) {
        removeFromCart(id);
      } else if (newQty > maxStock) {
        item.quantity = maxStock;
        toast.info(`Max stock for ${item.name} is ${maxStock}`);
      } else {
        item.quantity = newQty;
      }
      cart = cart;
      resolveCartPrices();
    }
  }

  let pricingResolving = $state(false);

  async function resolveCartPrices() {
    if (cart.length === 0) return;
    pricingResolving = true;
    try {
      const selectedCustomer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : null;
      const storeId = (authStore.user as any)?.store_id || undefined;
      const items = cart.map((item) => ({
        product_id: item.id,
        quantity: item.quantity,
        customer_group_id: (selectedCustomer as any)?.customer_group_id || undefined,
        store_id: storeId
      }));
      const results = await resolvePrices(items);
      for (let i = 0; i < cart.length; i++) {
        const result = results[i];
        if (result) {
          cart[i].price = result.unit_price;
          cart[i].original_price = result.original_price;
          cart[i].discount = result.discount;
          cart[i].pricing_type = result.pricing_type;
          if (result.rule) {
            cart[i].pricing_rule_id = result.rule.id;
            cart[i].pricing_rule_name = result.rule.name;
            cart[i].pricing_rule_type = result.rule.pricing_type;
          } else {
            cart[i].pricing_rule_id = undefined;
            cart[i].pricing_rule_name = undefined;
            cart[i].pricing_rule_type = undefined;
          }
        }
      }
      cart = cart;
    } catch (err) {
      console.warn('Pricing resolution failed, using base prices', err);
    } finally {
      pricingResolving = false;
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

  async function processCheckout(parkedSaleId?: number | null) {
    if (cart.length === 0) {
      toast.error('Cart is empty');
      return;
    }
    checkingOut = true;
    try {
      const items = cart.map((item) => ({
        product_id: item.id,
        quantity: item.quantity,
        unit_price: item.price,
        subtotal: item.price * item.quantity,
        ...(item.pricing_rule_id ? {
          pricing_rule_id: item.pricing_rule_id,
          pricing_rule_name: item.pricing_rule_name,
          pricing_rule_type: item.pricing_rule_type,
          pricing_type: item.pricing_type,
          original_price: item.original_price,
        } : {}),
      }));
      const activeShift = useShiftStore().activeShift;
      const response = await apiClient.post('/sales', {
        cashier_id: (authStore.user as any)?.id || 1,
        store_id: (authStore.user as any)?.store_id || null,
        shift_id: activeShift?.id || null,
        subtotal,
        discount: 0,
        tax: taxAmount,
        total_amount: totalAmount,
        payment_method: paymentMethod,
        customer_id: selectedCustomerId,
        status: 'completed',
        items,
        ...(parkedSaleId ? { parked_sale_id: parkedSaleId } : {}),
      });
      lastSale = response.data?.data || response.data;
      recalledSaleId = null;
      toast.success('Sale completed');
      cart = [];
      await fetchProducts(false);
    } catch (err: any) {
      const errMsg = err.response?.data?.error || 'Checkout failed';
      toast.error(errMsg);
    } finally {
      checkingOut = false;
    }
  }

  function clearCart() {
    cart = [];
    recalledSaleId = null;
  }

  async function holdSale() {
    if (cart.length === 0) {
      toast.error('Cart is empty');
      return;
    }
    holdingSale = true;
    try {
      const items = cart.map((item) => ({
        product_id: item.id,
        quantity: item.quantity,
        subtotal: item.price * item.quantity,
      }));
      await parkSale({
        items,
        payment_method: paymentMethod,
        recalled_sale_id: recalledSaleId,
      });
      toast.success('Sale parked');
      cart = [];
      recalledSaleId = null;
      await fetchParkedSales();
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to park sale');
    } finally {
      holdingSale = false;
    }
  }

  async function fetchParkedSales() {
    try {
      parkedSales = await listParkedSales();
    } catch (err) {
      console.warn('Failed to load parked sales', err);
    }
  }

  async function recallSale(saleId: number) {
    try {
      recalledSaleId = null;
      const recalled = await recallParkedSale(saleId);
      recalledSaleId = saleId;
      if (recalled.items && recalled.items.length > 0) {
        cart = recalled.items.map((item: any) => {
          const product = products.find((p: any) => p.id === item.product_id);
          return {
            id: item.product_id,
            name: product?.name || item.name || `Product #${item.product_id}`,
            price: item.unit_price,
            original_price: item.unit_price,
            quantity: item.quantity,
            stock: product?.stock ?? 999,
            sku: product?.sku || '',
          };
        });
        resolveCartPrices();
      }
      showParkedModal = false;
      toast.success('Sale recalled');
      await fetchParkedSales();
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to recall sale');
    }
  }

  async function cancelParked(id: number) {
    try {
      await cancelParkedSale(id);
      toast.success('Parked sale cancelled');
      await fetchParkedSales();
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to cancel parked sale');
    }
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
    const customer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : null;
    const saleTaxAmount = sale.tax || 0;
    printReceiptStore.set({
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
      paymentMethod: sale.payment_method || paymentMethod,
      cashReceived: sale.cash_received || cashReceived,
      changeDue: sale.change_due || changeDue,
      customer_name: customer?.name,
      total_savings: (sale.items || []).reduce((sum: number, item: any) => {
        if (item.original_price && item.original_price > item.unit_price) {
          return sum + (item.original_price - item.unit_price) * item.quantity;
        }
        return sum;
      }, 0),
    });
    setTimeout(() => {
      window.print();
      setTimeout(() => printReceiptStore.set(null), 1000);
    }, 300);
  }

  function handlePageChange(newOffset: number) {
    offset = newOffset;
    fetchProducts(false);
  }

  function openCheckoutModal() {
    if (cart.length === 0) {
      toast.error('Cart is empty');
      return;
    }
    showCheckoutModal = true;
    cashReceived = 0;
  }

  function closeCheckoutModal() {
    showCheckoutModal = false;
    cashReceived = 0;
    recalledSaleId = null;
  }

  function finalizeSale() {
    if (cart.length === 0 || changeDue < 0) return;
    const capturedCash = cashReceived;
    const capturedChange = changeDue;
    const capturedPayment = paymentMethod;
    const capturedRecalledSaleId = recalledSaleId;
    const customer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : null;
    closeCheckoutModal();
    processCheckout(capturedRecalledSaleId).then(() => {
      if (lastSale && lastSale.items) {
        const taxAmt = lastSale.tax || 0;
        printReceiptStore.set({
          invoice_number: lastSale.invoice_number,
          created_at: lastSale.created_at,
          items: lastSale.items.map((item) => ({
            name: item.name || '',
            quantity: item.quantity,
            unit_price: item.unit_price,
            original_price: item.original_price,
            pricing_rule_name: item.pricing_rule_name,
            pricing_type: item.pricing_type,
          })),
          total_amount: lastSale.total_amount,
          subtotal_dpp: lastSale.total_amount - taxAmt,
          tax: taxAmt,
          paymentMethod: capturedPayment,
          cashReceived: capturedCash,
          changeDue: capturedChange,
          customer_name: customer?.name,
          total_savings: lastSale.items.reduce((sum: number, item: any) => {
            if (item.original_price && item.original_price > item.unit_price) {
              return sum + (item.original_price - item.unit_price) * item.quantity;
            }
            return sum;
          }, 0),
        });
        setTimeout(() => {
          window.print();
          setTimeout(() => printReceiptStore.set(null), 1000);
        }, 300);
      }
    }).catch((err: any) => {
      console.error('Checkout failed:', err);
      toast.error('Checkout failed. Please try again.');
    });
  }

  $effect(() => {
    if (showCheckoutModal) {
      setTimeout(() => {
        const cashEl = document.getElementById('cash-received-input');
        const nonCashEl = document.getElementById('card-ewallet-amount-input');
        if (cashEl) cashEl.focus();
        if (nonCashEl) nonCashEl.focus();
      }, 0);
    }
  });

  let productTableEl: HTMLElement | undefined = $state();

  function handleGlobalKeydown(event: KeyboardEvent) {
    if (event.altKey && event.key === 'Delete') {
      event.preventDefault();
      if (cart.length > 0) {
        clearCart();
        toast.info('Cart cleared');
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
    if (event.key === 'F5') {
      event.preventDefault();
      fetchParkedSales().then(() => { showParkedModal = true; });
      return;
    }
    if (event.key === 'F6') {
      event.preventDefault();
      if (cart.length > 0) {
        holdSale();
      }
      return;
    }
    if (showCheckoutModal) {
      if (event.key === 'Escape' || event.key === 'F3') {
        event.preventDefault();
        closeCheckoutModal();
        return;
      }
      if (event.key === 'Enter') {
        event.preventDefault();
        if (changeDue >= 0) {
          finalizeSale();
        }
        return;
      }
      return;
    }
    if (showParkedModal || showCustomerModal) return;

    if (products.length === 0 || loading) return;

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      selectedProductIndex = Math.min(selectedProductIndex + 1, products.length - 1);
      scrollSelectedIntoView();
      return;
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      selectedProductIndex = Math.max(selectedProductIndex - 1, 0);
      scrollSelectedIntoView();
      return;
    }
    if (event.key === 'Enter' && selectedProductIndex >= 0) {
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
      const userRole = typeof authStore.user?.role === 'string' ? authStore.user.role : authStore.user?.role?.name ?? '';
      if (userRole === 'cashier') {
        try {
          const res = await apiClient.get('/shifts/active');
          if (!res.data.data) {
            toast.error('Anda harus membuka shift terlebih dahulu');
            goto('/shifts');
            return;
          }
        } catch {
          toast.error('Anda harus membuka shift terlebih dahulu');
          goto('/shifts');
          return;
        }
      }

      isInitialMount = true;
      await Promise.all([fetchProducts(false), fetchThresholds(), loadPaymentMethods()]);
      fetchCustomers();
      isInitialMount = false;
      focusSearch();
      try {
        const endDate = getTodayInJakarta();
        const startDate = '2025-01-01';
        const r = await apiClient.get(`/sales?limit=1&offset=0&startDate=${startDate}&endDate=${endDate}`);
        const body = r.data;
        let data = body?.data || body;
        if (Array.isArray(data) && data.length > 0) {
          lastSale = data[0];
        } else if (data && !Array.isArray(data) && (data as any).id) {
          lastSale = data;
        }
        if (lastSale) {
          console.log('[POS] Last sale loaded:', lastSale.invoice_number, 'items:', lastSale.items?.length);
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
        <span class="text-primary-light">Cart</span>
        {#if totalItems > 0}
          <span class="px-2 py-0.5 rounded-full bg-primary-subtle text-primary-light text-xs font-semibold">{totalItems} items</span>
        {/if}
      </span>
      <span class="text-text-muted">{showCart ? 'Hide' : 'Show'}</span>
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
          parkedSaleCount={parkedSales.filter(s => s.status === 'parked').length}
          onupdateqty={updateQty}
          onremovefromcart={removeFromCart}
          onclearcart={clearCart}
          oncheckout={openCheckoutModal}
          onprintreceipt={printReceipt}
          onholdsale={holdSale}
          onopenparkedmodal={() => { fetchParkedSales().then(() => { showParkedModal = true; }); }}
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
      parkedSaleCount={parkedSales.filter(s => s.status === 'parked').length}
      onupdateqty={updateQty}
      onremovefromcart={removeFromCart}
      onclearcart={clearCart}
      oncheckout={openCheckoutModal}
      onprintreceipt={printReceipt}
      onholdsale={holdSale}
      onopenparkedmodal={() => { fetchParkedSales().then(() => { showParkedModal = true; }); }}
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
  bind:paymentMethod
  {paymentOptions}
  {selectedCustomerLabel}
  bind:cashReceived
  {changeDue}
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
    showCustomerModal = false;
  }}
/>

<ParkedSalesModal
  bind:showModal={showParkedModal}
  {parkedSales}
  onrecall={recallSale}
  oncancel={cancelParked}
/>

<style>
</style>
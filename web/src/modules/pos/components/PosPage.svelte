<script lang="ts">
   import { onMount } from 'svelte';
   import apiClient from '$shared/api/http-client';
   import { toast } from '$shared/stores/toast.svelte';
   import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte';
   import { debounce } from '$shared/utils/debounce';
   import { useWebSocket } from '$shared/api/websocket';
   import { getTodayInJakarta } from '$shared/utils/jakartaTime';
   import type { Sale, SaleItem } from '$modules/sales/types';
  import type { Customer } from '$modules/customers/types';

    import { ShoppingCart } from 'lucide-svelte';
    import { useAuthStore } from '$modules/auth';
    import ProductSearchPanel from './ProductSearchPanel.svelte';
    import PosProductTable from './PosProductTable.svelte';
    import CartPanel from './CartPanel.svelte';
    import CheckoutModal from './CheckoutModal.svelte';
    import CustomerSelectModal from './CustomerSelectModal.svelte';

const authStore = useAuthStore();

   let cart = $state([]);
   let products = $state([]);
   let total = $state(0);
   let searchQuery = $state('');
   let loading = $state(false);
   let limit = $state(20);
   let offset = $state(0);
   let isInitialMount = $state(true);
   let isSearching = $state(false);
   let lastSale: Sale | null = $state(null);
   let ws = useWebSocket();

  let unsubscribeStock = null;
  let unsubscribeSale = null;

  let previousSearchQuery = '';
  let showCopySuccess = $state(null);
  let warningThreshold = $state(10);
  let criticalThreshold = $state(5);

  const paymentOptions = [
    { id: 'Cash', label: 'Cash', icon: ShoppingCart },
    { id: 'Card', label: 'Card', icon: ShoppingCart },
    { id: 'E-Wallet', label: 'E-Wallet', icon: ShoppingCart },
  ];
  let paymentMethod = $state('Cash');
  let checkingOut = $state(false);

   let showCheckoutModal = $state(false);
   let cashReceived = $state(0);
   let changeDue = $derived(cashReceived - totalAmount);

   let customers: Customer[] = $state([]);
   let selectedCustomerId = $state<number | null>(null);
   let showCustomerModal = $state(false);
   let customerSearch = $state('');
   let customerResults: Customer[] = $state([]);
   let customerSearching = $state(false);
   let selectedCustomerLabel = $derived(selectedCustomerId ? (customers.find(c => c.id === selectedCustomerId)?.name || '') : 'Walk-in / General');

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
  const dppDisplay = $derived(subtotal - taxAmount);
  const totalItems = $derived(cart.reduce((sum, item) => sum + item.quantity, 0));

  async function fetchProducts(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const r = await apiClient.get(`/products?limit=${limit}&offset=${offset}&search=${searchQuery}&status=active`);
      products = r.data.data || [];
      total = r.data.total || 0;
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

  function addToCart(product) {
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
      cart.push({ ...product, quantity: 1 });
      cart = cart;
    }
  }

  function removeFromCart(id) {
    const idx = cart.findIndex((item) => item.id === id);
    if (idx !== -1) cart.splice(idx, 1);
    cart = cart;
  }

  function updateQty(id, delta) {
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

  async function processCheckout() {
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
      }));
      const response = await apiClient.post('/sales', {
        cashier_id: (authStore.user as any)?.id || 1,
        store_id: (authStore.user as any)?.store_id || null,
        subtotal,
        discount: 0,
        tax: taxAmount,
        total_amount: totalAmount,
        payment_method: paymentMethod,
        customer_id: selectedCustomerId,
        status: 'completed',
        items,
      });
      lastSale = response.data?.data || response.data;
      toast.success('Sale completed');
      cart = [];
      await fetchProducts(false);
    } catch (err) {
      const errMsg = err.response?.data?.error || 'Checkout failed';
      toast.error(errMsg);
    } finally {
      checkingOut = false;
    }
  }

  function clearCart() {
    cart = [];
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
        name: item.name,
        quantity: item.quantity,
        unit_price: item.unit_price,
      })),
      total_amount: sale.total_amount,
      subtotal_dpp: sale.total_amount - saleTaxAmount,
      tax: saleTaxAmount,
      paymentMethod: sale.payment_method || paymentMethod,
      cashReceived: sale.cash_received || cashReceived,
      changeDue: sale.change_due || changeDue,
      customer_name: customer?.name,
    });
    setTimeout(() => {
      window.print();
      setTimeout(() => printReceiptStore.set(null), 1000);
    }, 300);
  }

  function handlePageChange(newOffset) {
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
  }

  function finalizeSale() {
    if (cart.length === 0 || changeDue < 0) return;
    closeCheckoutModal();
    processCheckout().then(() => {
      if (lastSale && lastSale.items) {
        const customer = selectedCustomerId ? customers.find(c => c.id === selectedCustomerId) : null;
        const taxAmount = lastSale.tax || 0;
        printReceiptStore.set({
          invoice_number: lastSale.invoice_number,
          created_at: lastSale.created_at,
          items: lastSale.items.map((item) => ({
            name: item.name,
            quantity: item.quantity,
            unit_price: item.unit_price,
          })),
          total_amount: lastSale.total_amount,
          subtotal_dpp: lastSale.total_amount - taxAmount,
          tax: taxAmount,
          paymentMethod: paymentMethod,
          cashReceived: cashReceived,
          changeDue: changeDue,
          customer_name: customer?.name,
        });
      }
    setTimeout(() => {
      window.print();
      // Clear the receipt store after print dialog closes so the
      // hidden receipt container doesn't linger in the DOM
      setTimeout(() => printReceiptStore.set(null), 1000);
    }, 300);
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

  function handleGlobalKeydown(event) {
    if (event.altKey && event.key === 'Delete') {
      event.preventDefault();
      if (cart.length > 0 && confirm('Kosongkan seluruh keranjang belanja?')) {
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
        input.select();
      }
      return;
    }
    // ESC clears search when not in modal
    if (event.key === 'Escape' && !showCheckoutModal) {
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
    if (!showCheckoutModal) return;
    if (event.key === 'Escape' || event.key === 'F3') {
      event.preventDefault();
      closeCheckoutModal();
      return;
    }
    if (event.key === 'F6') {
      event.preventDefault();
      cashReceived = totalAmount;
      return;
    }
    if (event.key === 'Enter') {
      event.preventDefault();
      if (changeDue >= 0) {
        finalizeSale();
      }
      return;
    }
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
        input.select();
      }
    }, 50);
  }

  onMount(async () => {
    isInitialMount = true;
    await Promise.all([fetchProducts(false), fetchThresholds()]);
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
    unsubscribeStock = ws.on('stock_update', (data) => {
      const product = products.find(p => p.id === data.id);
      if (product) {
        product.stock = data.stock;
      }
    });
    unsubscribeSale = ws.on('sale_created', (data) => {
      if (data) {
        lastSale = data;
      }
    });
    return () => {
      if (unsubscribeStock) unsubscribeStock();
      if (unsubscribeSale) unsubscribeSale();
    };
  });
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="space-y-6">
  <div class="flex gap-6">
    <div class="flex-1 flex flex-col gap-4">
      <ProductSearchPanel bind:searchQuery />
      <div class="card flex-1 overflow-hidden flex flex-col p-0">
        <PosProductTable
          {products}
          {loading}
          {total}
          {limit}
          {offset}
          {warningThreshold}
          {criticalThreshold}
          bind:showCopySuccess
          onaddtocart={addToCart}
          oncopy={copyToClipboard}
          onpagechange={handlePageChange}
        />
      </div>
    </div>
    <div class="w-[340px] shrink-0 flex flex-col relative">
      <CartPanel
        {cart}
        {totalAmount}
        {totalItems}
        {subtotal}
        {taxAmount}
        {dppDisplay}
        bind:paymentMethod
        {paymentOptions}
        {selectedCustomerLabel}
        {lastSale}
        {checkingOut}
        onupdateqty={updateQty}
        onremovefromcart={removeFromCart}
        onclearcart={clearCart}
        oncheckout={openCheckoutModal}
        onprintreceipt={printReceipt}
        onselectcustomer={() => (showCustomerModal = true)}
      />
    </div>
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
  onselectcustomer={() => { showCheckoutModal = false; showCustomerModal = true; }}
/>

<CustomerSelectModal
  bind:showCustomerModal
  bind:customerSearch
  {customerResults}
  {customerSearching}
  onselectcustomer={(id) => { selectedCustomerId = id; showCustomerModal = false; customerSearch = ''; }}
/>

<style>
/* Component-scoped styles for screen only */
</style>
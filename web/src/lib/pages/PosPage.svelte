<script>
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { useWebSocket } from '$lib/composables/useWebSocket';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search, Plus, Minus, ShoppingCart, X, Package, Copy, Printer } from 'lucide-svelte';
  import { auth } from '$lib/stores/auth';
  import { slide } from 'svelte/transition';
  import { flip } from 'svelte/animate';

  let cart = $state([]);
  let products = $state([]);
  let total = $state(0);
  let searchQuery = $state('');
  let loading = $state(false);
  let limit = $state(20);
  let offset = $state(0);
  let isInitialMount = $state(true);
  let isSearching = $state(false);
  let lastSale = $state(null);
  let ws = useWebSocket();
  
  // Listen for stock updates via WebSocket
  let unsubscribeStock = null;
  let unsubscribeSale = null;
  
  // Track previous search to avoid duplicate fetches
  let previousSearchQuery = '';

  const paymentOptions = [
    { id: 'Cash', label: 'Cash', icon: ShoppingCart },
    { id: 'Card', label: 'Card', icon: ShoppingCart },
    { id: 'E-Wallet', label: 'E-Wallet', icon: ShoppingCart },
  ];
  let paymentMethod = $state('Cash');
  let checkingOut = $state(false);

  const subtotal = $derived(cart.reduce((sum, item) => sum + item.price * item.quantity, 0));
  // const tax = $derived(Math.round(subtotal * 0.1)); // Temporarily removed
  const totalAmount = $derived(subtotal); // No tax for now
  const totalItems = $derived(cart.reduce((sum, item) => sum + item.quantity, 0));

  async function fetchProducts(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const r = await apiClient.get(`/products?limit=${limit}&offset=${offset}&search=${searchQuery}`);
      products = r.data.data || [];
      total = r.data.total || 0;
    } catch (err) {
      toast.error('Failed to load products');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  // Debounced search
  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts(true);
  }, 400);

  // Watch for search query changes and trigger debounced search
  $effect(() => {
    // Skip the initial render to prevent double fetch
    if (isInitialMount) return;
    
    // Only proceed if searchQuery actually changed
    if (previousSearchQuery === searchQuery) return;
    
    previousSearchQuery = searchQuery;

    if (searchQuery === '') {
      // Immediate fetch when clearing search
      offset = 0;
      isSearching = false;
      fetchProducts(false);
    } else {
      isSearching = true;
      debouncedSearch();
    }
  });

  function addToCart(product) {
    const existing = cart.find((item) => item.id === product.id);
    if (existing) {
      existing.quantity++;
      cart = [...cart];
    } else {
      cart = [...cart, { ...product, quantity: 1 }];
    }
  }

  function removeFromCart(id) {
    cart = cart.filter((item) => item.id !== id);
  }

  function updateQty(id, delta) {
    const item = cart.find((i) => i.id === id);
    if (item) {
      item.quantity += delta;
      if (item.quantity <= 0) removeFromCart(id);
      cart = [...cart];
    }
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
        invoice_number: `INV-${Date.now()}`,
        cashier_id: $auth.user?.id || 1,
        store_id: $auth.user?.store_id || null,
        subtotal,
        discount: 0,
        tax: 0, // No tax for now
        total_amount: totalAmount,
        payment_method: paymentMethod,
        status: 'completed',
        items,
      });

      lastSale = response.data;
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

  function printReceipt() {
    if (!lastSale) return;
    window.print();
  }

  function handlePageChange(newOffset) {
    offset = newOffset;
    fetchProducts(false);
  }

  onMount(async () => {
    isInitialMount = true;
    await fetchProducts(false);
    isInitialMount = false;

    // WebSocket event handlers for real-time updates
    unsubscribeStock = ws.on('stock_update', (data) => {
      const product = products.find(p => p.id === data.id);
      if (product) {
        product.stock = data.stock;
        toast.info(`Stock updated: ${product.name} now has ${data.stock} units`);
      }
    });

    unsubscribeSale = ws.on('sale_created', (data) => {
      toast.success(`New sale: ${data.invoice} (${data.total.toLocaleString('id-ID')})`);
    });

    // Cleanup on unmount
    return () => {
      if (unsubscribeStock) unsubscribeStock();
      if (unsubscribeSale) unsubscribeSale();
    };
  });
</script>

<div class="space-y-6">
  <div class="flex gap-6">
    <!-- Products -->
    <div class="flex-1 flex flex-col gap-4">
      <div class="card p-4">
        <div class="relative">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input
            type="text"
            placeholder="Search products..."
            bind:value={searchQuery}
            class="input pl-10 pr-10"
          />
          {#if searchQuery}
            <button
              onclick={() => searchQuery = ''}
              class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              title="Clear search"
            >
              <X size={16} />
            </button>
          {/if}
        </div>
      </div>

      <div class="card flex-1 overflow-hidden flex flex-col p-0">

        {#if loading}
          <div class="flex-1 overflow-y-auto">
            {#each { length: 8 } as _}
              <div class="flex items-center gap-4 px-4 py-3 border-b border-border">
                <div class="skeleton h-4 w-40"></div>
                <div class="skeleton h-4 w-20 ml-auto"></div>
                <div class="skeleton h-4 w-16"></div>
                <div class="skeleton h-7 w-14 rounded-xl"></div>
              </div>
            {/each}
          </div>
        {:else if products.length === 0}
          <div class="px-4 py-12 text-center">
            <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
              <Package size={32} class="text-text-muted" />
            </div>
            <p class="text-text-primary font-semibold mt-4">No products found</p>
            <p class="text-text-muted text-sm mt-1">Add products to start selling</p>
          </div>
        {:else}
          <div class="flex-1 overflow-y-auto">
            <table class="w-full table-fixed">
              <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
                <tr>
                  <th class="p-4 w-64">PRODUCT NAME</th>
                  <th class="p-4 text-center w-24">Stock</th>
                  <th class="p-4 text-right w-28">Price</th>
                  <th class="p-4 w-20"></th>
                </tr>
              </thead>
              <tbody>
                {#each products as product (product.id)}
                  <tr class="hover:bg-surface-hover/50 transition-colors">
                    <td class="p-4 w-64">
                      <!-- Product name (normal size) -->
                      <div class="font-medium truncate w-full text-text-primary" title={product.name}>
                        {product.name}
                      </div>

                      <!-- SKU and Barcode details (smaller font) -->
                      <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
                        <!-- SKU with copy button -->
                        <span class="flex items-center gap-1">
                          {product.sku}
                          <button
                            class="p-0.5 hover:text-primary transition-colors"
                            title="Salin SKU"
                            onclick={() => {
                              navigator.clipboard.writeText(product.sku).then(() => {
                                toast.info(`SKU copied: ${product.sku}`, 2000);
                              });
                            }}
                          >
                            <Copy size={14} class="text-text-muted hover:text-primary" />
                          </button>
                        </span>

                        <!-- Barcode with copy button (only if barcode exists) -->
                        {#if product.barcode}
                          <span class="flex items-center gap-1 ml-4">
                            {product.barcode}
                            <button
                              class="p-0.5 hover:text-primary transition-colors"
                              title="Salin barcode"
                              onclick={() => {
                                navigator.clipboard.writeText(product.barcode).then(() => {
                                  toast.info(`Barcode copied: ${product.barcode}`, 2000);
                                });
                              }}
                            >
                              <Copy size={14} class="text-text-muted hover:text-primary" />
                            </button>
                          </span>
                        {/if}
                      </div>
                    </td>
                    <td class="p-4 text-center w-24">
                      {#if product.stock === 0}
                        <Badge variant="destructive">Out of stock</Badge>
                      {:else if product.stock <= 5}
                        <Badge variant="warning">Low: {product.stock}</Badge>
                      {:else}
                        <Badge variant="success">{product.stock}</Badge>
                      {/if}
                    </td>
                    <td class="p-4 text-right font-semibold text-text-primary w-28">
                      {product.price?.toLocaleString('id-ID')}
                    </td>
                    <td class="p-4 text-right w-20">
                      <button
                        class="btn btn-primary btn-sm"
                        onclick={() => addToCart(product)}
                        disabled={product.stock === 0}
                      >
                        <Plus size={14} /> Add
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          <div class="p-3 border-t border-border bg-surface-subtle/20">
            <Pagination {total} {limit} {offset} onPageChange={handlePageChange} />
          </div>
        {/if}
      </div>
    </div>

    <!-- Cart -->
    <div class="w-[340px] shrink-0 flex flex-col relative">
      <div class="card flex flex-col overflow-hidden p-0 sticky top-0 h-[calc(100vh-120px)] max-h-[800px]">
        <!-- Cart Header -->
        <div class="px-4 py-3.5 border-b border-border flex items-center justify-between shrink-0">
          <div class="flex items-center gap-2">
            <ShoppingCart size={18} class="text-primary-light" />
            <span class="font-semibold text-text-primary">Cart</span>
            {#if totalItems > 0}
              <span class="badge badge-primary">{totalItems}</span>
            {/if}
          </div>
          {#if cart.length > 0}
            <button onclick={clearCart} class="btn btn-ghost btn-icon btn-sm text-danger hover:bg-danger-subtle" title="Clear cart">
              <X size={14} />
            </button>
          {/if}
        </div>

        <!-- Cart Items Area -->
        {#if cart.length === 0}
          <div class="flex-1 p-4 flex flex-col items-center justify-center text-center overflow-y-auto">
            <div class="empty-state-icon bg-surface w-20 h-20 rounded-2xl mb-4 border border-dashed border-border-strong flex items-center justify-center animate-pulse">
              <ShoppingCart size={32} class="text-text-muted opacity-50" />
            </div>
            <p class="text-text-secondary font-medium">Your cart is empty</p>
            <p class="text-text-muted text-xs mt-1">Add products to start selling</p>
          </div>
        {:else}
          <div class="flex-1 overflow-y-auto divide-y divide-border overflow-x-hidden">
            {#each cart as item (item.id)}
              <div 
                class="flex items-start gap-3 px-4 py-3 hover:bg-surface-hover/50 transition-colors"
                animate:flip={{ duration: 300 }}
                transition:slide={{ duration: 250 }}
              >
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-text-primary truncate">{item.name}</p>
                  <p class="text-xs text-text-muted mt-0.5">
                    {item.price.toLocaleString('id-ID')} × {item.quantity}
                    = <span class="text-text-secondary font-medium">{(item.price * item.quantity).toLocaleString('id-ID')}</span>
                  </p>
                </div>
                <div class="flex items-center gap-1.5 shrink-0 mt-0.5">
                  <button
                    class="w-8 h-8 rounded-lg bg-surface hover:bg-surface-hover text-text-secondary flex items-center justify-center transition-colors border border-border active:scale-95"
                    onclick={() => updateQty(item.id, -1)}
                  >
                    <Minus size={14} />
                  </button>
                  <span class="w-7 text-center text-sm font-semibold text-text-primary">{item.quantity}</span>
                  <button
                    class="w-8 h-8 rounded-lg bg-surface hover:bg-surface-hover text-text-secondary flex items-center justify-center transition-colors border border-border active:scale-95"
                    onclick={() => updateQty(item.id, 1)}
                  >
                    <Plus size={14} />
                  </button>
                  <button
                    class="w-8 h-8 rounded-lg hover:bg-danger-subtle text-text-muted hover:text-danger flex items-center justify-center transition-colors ml-1 active:scale-95"
                    onclick={() => removeFromCart(item.id)}
                  >
                    <X size={14} />
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}

        <!-- Summary Section (Always Visible) -->
        <div class="border-t border-border p-4 space-y-3 bg-bg-secondary shrink-0">
          <!-- Total -->
          <div class="flex justify-between font-bold text-text-primary">
            <span>Total</span>
            <span class="text-white text-base">{totalAmount.toLocaleString('id-ID')}</span>
          </div>

          <!-- Payment Method -->
          <div>
            <p class="text-xs text-text-muted mb-2 font-medium">Payment method</p>
            <div class="grid grid-cols-3 gap-2">
              {#each paymentOptions as opt}
                <button
                  class="flex flex-col items-center gap-1 py-2 rounded-xl border text-xs font-medium transition-all {paymentMethod === opt.id ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
                  onclick={() => paymentMethod = opt.id}
                >
                  <opt.icon size={16} />
                  {opt.label}
                </button>
              {/each}
            </div>
          </div>

          <!-- Complete Purchase Button -->
          <button
            class="btn btn-success w-full py-3"
            onclick={processCheckout}
            disabled={checkingOut || cart.length === 0}
          >
            {#if checkingOut}
              <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
              Processing…
            {:else}
              Complete Purchase · {totalAmount.toLocaleString('id-ID')}
            {/if}
          </button>

          {#if lastSale}
            <button
              class="btn btn-ghost w-full py-2 mt-2"
              onclick={printReceipt}
            >
              <Printer size={16} />
              Print Receipt
            </button>
          {/if}
        </div>
      </div>
    </div>
  </div>
</div>

{#if lastSale}
  <div class="receipt-print-area" id="receipt-print-area">
    <div class="receipt-header">
      <h2>RETAIL POS</h2>
      <p>Invoice: {lastSale.invoice_number}</p>
      <p>{new Date(lastSale.created_at).toLocaleString('id-ID')}</p>
    </div>
    <div class="receipt-divider"></div>
    {#each lastSale.items as item}
      <div class="receipt-item">
        <span>{item.name} x{item.quantity}</span>
        <span>{(item.unit_price * item.quantity).toLocaleString('id-ID')}</span>
      </div>
    {/each}
    <div class="receipt-divider"></div>
    <div class="receipt-total receipt-item">
      <span>TOTAL</span>
      <span>{lastSale.total_amount.toLocaleString('id-ID')}</span>
    </div>
    <div class="receipt-header" style="margin-top: 5mm;">
      <p>Thank you for your purchase!</p>
    </div>
  </div>
{/if}

<style>
@media print {
  body > *:not(#receipt-print-area) { display: none !important; }
  #receipt-print-area {
    display: block !important;
    position: absolute;
    top: 0;
    left: 0;
    width: 58mm;
    padding: 5mm;
    font-family: 'Courier New', monospace;
    font-size: 10pt;
  }
  .receipt-header { text-align: center; margin-bottom: 3mm; }
  .receipt-item { display: flex; justify-content: space-between; margin-bottom: 2mm; }
  .receipt-divider { border-top: 1px dashed #000; margin: 3mm 0; }
  .receipt-total { font-weight: bold; font-size: 12pt; }
}
</style>
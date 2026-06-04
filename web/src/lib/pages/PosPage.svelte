<script lang="ts">
   import { onMount } from 'svelte';
   import apiClient from '$lib/api/client';
   import { toast } from '$lib/stores/toast';
   import { debounce } from '$lib/utils/debounce';
   import { useWebSocket } from '$lib/composables/useWebSocket';
   import type { Sale, SaleItem } from '$lib/types';

   import Badge from '$lib/components/ui/Badge.svelte';
   import Pagination from '$lib/components/ui/Pagination.svelte';
   import { Search, Plus, Minus, ShoppingCart, X, Package, Copy, Printer, Wallet, Check, Receipt } from 'lucide-svelte';
   import { auth } from '$lib/stores/auth';
   import { slide } from 'svelte/transition';
   import { flip } from 'svelte/animate';
   import { fly } from 'svelte/transition';

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
   let lastSalePrintData: { paymentMethod: string; cashReceived: number; changeDue: number } | null = $state(null);
   let ws = useWebSocket();

  let unsubscribeStock = null;
  let unsubscribeSale = null;

  let previousSearchQuery = '';
  let showCopySuccess = $state(null);

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

  const subtotal = $derived(cart.reduce((sum, item) => sum + item.price * item.quantity, 0));
  const totalAmount = $derived(subtotal);
  const totalItems = $derived(cart.reduce((sum, item) => sum + item.quantity, 0));

  const quickCashPresets = [50000, 100000, 150000, 200000];

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

  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts(true);
  }, 400);

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

  function addToCart(product) {
    const existing = cart.find((item) => item.id === product.id);
    if (existing) {
      const maxStock = existing.stock || 999;
      if (existing.quantity >= maxStock) {
        toast.info(`Max stock reached: ${existing.name} (${maxStock})`);
        return;
      }
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
      cart = [...cart];
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
        cashier_id: $auth.user?.id || 1,
        store_id: $auth.user?.store_id || null,
        subtotal,
        discount: 0,
        tax: 0,
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

  function openCheckoutModal() {
    if (cart.length === 0) {
      toast.error('Cart is empty');
      return;
    }
    showCheckoutModal = true;
    cashReceived = 0;
    paymentMethod = 'Cash';
  }

  function closeCheckoutModal() {
    showCheckoutModal = false;
    cashReceived = 0;
  }

  function finalizeSale() {
    if (cart.length === 0 || changeDue < 0) return;
    lastSalePrintData = {
      paymentMethod: paymentMethod,
      cashReceived: cashReceived,
      changeDue: changeDue
    };
    closeCheckoutModal();
    processCheckout().then(() => {
      setTimeout(() => {
        window.print();
      }, 100);
    });
  }

  $effect(() => {
    if (showCheckoutModal) {
      setTimeout(() => {
        const el = document.getElementById('cash-received-input');
        if (el) el.focus();
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
    await fetchProducts(false);
    isInitialMount = false;
    focusSearch();
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
    return () => {
      if (unsubscribeStock) unsubscribeStock();
      if (unsubscribeSale) unsubscribeSale();
    };
  });
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="space-y-6 main-content">
   <div class="flex gap-6">
     <!-- Products -->
     <div class="flex-1 flex flex-col gap-4">
       <div class="card p-4">
         <div class="flex items-center gap-2">
           <kbd class="px-1.5 py-0.5 text-[10px] font-medium text-primary/60 bg-primary-subtle/30 rounded border border-primary/20 select-none">F2</kbd>
           <div class="relative flex-1">
             <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
             <input
               type="text"
               placeholder="Search products..."
               id="pos-search-input"
               bind:value={searchQuery}
               class="input pl-10 pr-10 w-full"
               autocomplete="off"
              spellcheck="false"
            />
            {#if searchQuery}
              <button
                onclick={() => searchQuery = ''}
                class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
                title="Clear search [ESC]"
              >
                <X size={16} />
              </button>
            {/if}
          </div>
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
                      <div class="font-medium truncate w-full text-text-primary" title={product.name}>
                        {product.name}
                      </div>
                      <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
                        <span class="flex items-center gap-1">
                          {product.sku}
                          <button
                            class="p-0.5 hover:text-primary transition-colors"
                            title="Salin SKU"
                            onclick={() => copyToClipboard(product.sku, `sku_${product.id}`)}
                          >
                            {#if showCopySuccess?.has(`sku_${product.id}`)}
                              <span class="text-sm text-primary font-semibold">✓</span>
                            {:else}
                              <Copy size={14} class="text-text-muted hover:text-primary" />
                            {/if}
                          </button>
                        </span>
                        {#if product.barcode}
                          <span class="flex items-center gap-1 ml-4">
                            {product.barcode}
                            <button
                              class="p-0.5 hover:text-primary transition-colors"
                              title="Salin barcode"
                              onclick={() => copyToClipboard(product.barcode, `barcode_${product.id}`)}
                            >
                              {#if showCopySuccess?.has(`barcode_${product.id}`)}
                                <span class="text-sm text-primary font-semibold">✓</span>
                              {:else}
                                <Copy size={14} class="text-text-muted hover:text-primary" />
                              {/if}
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
        <div class="px-4 py-3.5 border-b border-border flex items-center justify-between shrink-0">
          <div class="flex items-center gap-2">
            <ShoppingCart size={18} class="text-primary-light" />
            <span class="font-semibold text-text-primary">Cart</span>
            {#if totalItems > 0}
              <span class="badge badge-primary">{totalItems}</span>
            {/if}
          </div>
          {#if cart.length > 0}
            <div class="flex items-center gap-1">
              <kbd class="px-1 py-0.5 text-[10px] font-medium text-danger/60 bg-danger-subtle/30 rounded border border-danger/20 select-none">ALT+DEL</kbd>
              <button onclick={clearCart} class="btn btn-ghost btn-icon btn-sm text-danger hover:bg-danger-subtle" title="Clear cart [ALT+DEL]">
                <X size={14} />
              </button>
            </div>
          {/if}
        </div>

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
                  <input
                    type="number"
                    min="1"
                    max={item.stock || 999}
                    bind:value={item.quantity}
                    oninput={() => {
                      const maxStock = item.stock || 999;
                      if (item.quantity > maxStock) {
                        item.quantity = maxStock;
                        cart = [...cart];
                      }
                    }}
                    onblur={() => {
                      if (item.quantity <= 0) removeFromCart(item.id);
                      else {
                        const maxStock = item.stock || 999;
                        if (item.quantity > maxStock) {
                          item.quantity = maxStock;
                        }
                        cart = [...cart];
                      }
                    }}
                    onkeydown={(e) => {
                      if (e.key === 'Enter') {
                        e.target.blur();
                        focusSearch();
                      }
                    }}
                    class="w-12 text-center text-sm font-semibold text-text-primary bg-surface border border-border rounded-lg px-1 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary"
                  />
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

        <div class="border-t border-border p-4 space-y-3 bg-bg-secondary shrink-0">
          <div class="flex justify-between font-bold text-text-primary">
            <span>Total</span>
            <span class="text-white text-base">{totalAmount.toLocaleString('id-ID')}</span>
          </div>

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

          <button
            class="btn btn-success w-full py-3"
            onclick={openCheckoutModal}
            disabled={checkingOut || cart.length === 0}
          >
            {#if checkingOut}
              <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
              Processing…
            {:else}
              <Wallet size={16} />
              Bayar [F4] · Rp {totalAmount.toLocaleString('id-ID')}
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

{#if showCheckoutModal}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center print-modal-overlay"
    transition:fly={{ y: 40, duration: 300 }}
  >
    <div
      class="absolute inset-0 bg-black/70 backdrop-blur-sm"
      onclick={closeCheckoutModal}
      role="presentation"
    ></div>

    <div
      role="dialog"
      aria-modal="true"
      aria-label="Pembayaran Selesai"
      class="relative z-[55] w-full max-w-xl rounded-2xl border border-border-default bg-bg-card shadow-modal p-6"
    >
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-xl font-bold text-text-primary">Pembayaran Selesai</h2>
        <button
          onclick={closeCheckoutModal}
          class="w-9 h-9 flex items-center justify-center rounded-xl text-text-muted hover:text-text-primary hover:bg-surface-hover/50 transition-colors"
          title="Tutup [F3 / Esc]"
        >
          <X size={20} />
        </button>
      </div>

      <div class="mb-6 text-center">
        <p class="text-sm text-text-muted mb-1 font-medium">Total Tagihan</p>
        <p class="text-4xl font-extrabold text-purple-400">
          {totalAmount.toLocaleString('id-ID')}
        </p>
      </div>

      <p class="text-xs text-text-muted mb-2 font-medium">Metode Pembayaran</p>
      <div class="grid grid-cols-3 gap-2 mb-6">
        {#each paymentOptions as opt}
          <button
            class="flex flex-col items-center gap-1 py-2.5 rounded-xl border text-xs font-medium transition-all {paymentMethod === opt.id ? 'border-primary bg-primary-subtle text-primary-light' : 'border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
            onclick={() => paymentMethod = opt.id}
          >
            <opt.icon size={18} />
            {opt.label}
          </button>
        {/each}
      </div>

      {#if paymentMethod === 'Cash'}
        <div class="mb-4">
          <label for="cash-received-input" class="text-xs text-text-muted mb-1.5 font-medium block">
            Cash Received [F6] = Total
          </label>
          <input
            id="cash-received-input"
            type="text"
            inputmode="numeric"
            bind:value={cashReceived}
            class="input text-lg font-bold text-text-primary [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
            placeholder="0"
          />
        </div>

        <div class="flex flex-wrap gap-2 mb-4">
          {#each quickCashPresets as preset}
            <button
              class="px-3.5 py-1.5 rounded-lg border border-border text-xs font-semibold text-text-secondary hover:border-primary-light hover:text-primary-light transition-colors"
              onclick={() => cashReceived = preset}
            >
              Rp {preset.toLocaleString('id-ID')}
            </button>
          {/each}
        </div>

        <div
          class="flex items-center justify-between p-4 rounded-xl
            {changeDue >= 0
              ? 'bg-success-subtle border border-success-default/20'
              : 'bg-danger-subtle border border-danger-default/20'}"
        >
          <span class="text-sm font-medium text-text-secondary">Uang Kembali</span>
          <span
            class="text-2xl font-extrabold
              {changeDue >= 0 ? 'text-emerald-400' : 'text-danger-light'}"
          >
            {Math.abs(changeDue).toLocaleString('id-ID')}
            {#if changeDue < 0}
              <span class="text-xs font-semibold text-danger-light ml-1">(kurang)</span>
            {/if}
          </span>
        </div>
      {:else}
        <div
          class="flex items-center justify-between p-4 rounded-xl
            bg-info-subtle border border-info-default/20"
        >
          <span class="text-sm font-medium text-text-secondary">Metode</span>
          <span class="text-lg font-bold text-info-light">{paymentMethod}</span>
        </div>
      {/if}

      <div class="flex gap-3 mt-6">
        <button
          class="btn btn-secondary flex-1"
          onclick={closeCheckoutModal}
        >
          Batal [F3]
        </button>
        <button
          class="btn btn-success flex-1"
          disabled={cart.length === 0 || changeDue < 0}
          onclick={finalizeSale}
        >
          <Check size={16} />
          Selesai &amp; Cetak [Enter]
        </button>
      </div>
    </div>
  </div>
{/if}

{#if lastSale}
<div class="thermal-receipt hidden" id="thermal-receipt">
  <div class="thermal-shop-name">RETAIL POS</div>
  <div class="thermal-row">
    <span class="thermal-label">Invoice:</span>
    <span class="thermal-value">{lastSale.invoice_number}</span>
  </div>
  <div class="thermal-row">
    <span class="thermal-label">Waktu:</span>
    <span class="thermal-value">{new Date(lastSale.created_at || Date.now()).toLocaleString('id-ID')}</span>
  </div>
  <div class="thermal-divider"></div>
  {#each lastSale.items as item}
    <div class="thermal-item">
      <div class="thermal-item-name">{item.name} x{item.quantity}</div>
      <div class="thermal-item-price">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</div>
    </div>
  {/each}
  <div class="thermal-divider"></div>
  <div class="thermal-item thermal-item-total">
    <span>TOTAL</span>
    <span>{lastSale.total_amount.toLocaleString('id-ID')}</span>
  </div>
  <div class="thermal-row">
    <span class="thermal-label">Pembayaran:</span>
    <span class="thermal-value">{lastSalePrintData?.paymentMethod ?? 'Cash'}</span>
  </div>
  <div class="thermal-row">
    <span class="thermal-label">Uang Tunai:</span>
    <span class="thermal-value">{lastSalePrintData?.cashReceived?.toLocaleString('id-ID') ?? '—'}</span>
  </div>
  <div class="thermal-row">
    <span class="thermal-label">Kembali:</span>
    <span class="thermal-value">{lastSalePrintData?.changeDue?.toLocaleString('id-ID') ?? '—'}</span>
  </div>
  <div class="thermal-divider"></div>
  <div class="thermal-footer">
    <p>Terima kasih atas kunjungan Anda!</p>
    <p>Barang yang sudah dibeli tidak dapat dikembalikan.</p>
  </div>
</div>
{/if}

<style>
@media print {
  body {
    background: white !important;
    margin: 0 !important;
    padding: 0 !important;
  }

  /* Hide all app UI elements */
  .main-content,
  .card,
  table,
  button.btn,
  .print-modal-overlay,
  .sidebar-shell,
  .topbar,
  header,
  nav[aria-label="Breadcrumb"] {
    display: none !important;
  }

  /* Show thermal receipt with proper thermal paper dimensions */
  .thermal-receipt,
  .thermal-receipt.hidden {
    display: block !important;
    position: fixed !important;
    top: 0 !important;
    left: 0 !important;
    width: 80mm !important;
    padding: 2mm !important;
    background: white !important;
    color: #000 !important;
    font-family: 'Courier New', 'Courier', monospace !important;
    font-size: 10pt !important;
    line-height: 1.3 !important;
  }

  .thermal-receipt * {
    visibility: visible !important;
  }

  .thermal-shop-name {
    font-size: 14pt !important;
    font-weight: bold !important;
    text-align: center !important;
    margin-bottom: 2mm !important;
  }

  .thermal-row {
    display: flex !important;
    justify-content: space-between !important;
    margin-bottom: 1.5mm !important;
  }

  .thermal-item {
    display: flex !important;
    justify-content: space-between !important;
    margin-bottom: 1.5mm !important;
  }

  .thermal-item-name {
    flex: 1 !important;
    word-break: break-word !important;
    padding-right: 2mm !important;
  }

  .thermal-item-price {
    white-space: nowrap !important;
    font-weight: bold !important;
  }

  .thermal-item-total {
    font-size: 12pt !important;
    font-weight: bold !important;
    margin-top: 2mm !important;
    padding-top: 1mm !important;
    border-top: 1px dashed #000 !important;
  }

  .thermal-divider {
    border-top: 1px dashed #000 !important;
    margin: 3mm 0 !important;
  }

  .thermal-footer {
    text-align: center !important;
    font-size: 9pt !important;
    margin-top: 3mm !important;
    line-height: 1.4 !important;
  }

  @page {
    margin: 0 !important;
    size: auto;
  }
}</style>
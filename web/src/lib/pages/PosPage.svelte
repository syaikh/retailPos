<script>
  import { onMount, onDestroy } from 'svelte';
  import { useWebSocket } from '$lib/composables/useWebSocket';
  import Badge from '$lib/components/ui/Badge.svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import {
    Search, ShoppingCart, Trash2, Plus, Minus,
    X, CreditCard, Banknote, QrCode, Tag,
  } from 'lucide-svelte';

  // ── State ────────────────────────────────────────────────────
  let loading = $state(true);
  let products = $state([]);
  let filteredProducts = $state([]);
  let searchQuery = $state('');
  let selectedCategory = $state('Semua');
  let cart = $state([]);
  let categories = $state(['Semua', 'Makanan', 'Minuman', 'Snack', 'Lainnya']);
  let paymentMethod = $state('cash');
  let checkingOut = $state(false);

  const ws = useWebSocket();

  // ── Derived ──────────────────────────────────────────────────
  let subtotal = $derived(cart.reduce((s, i) => s + i.price * i.quantity, 0));
  let tax      = $derived(Math.floor(subtotal * 0.1));
  let total    = $derived(subtotal + tax);
  let totalItems = $derived(cart.reduce((s, i) => s + i.quantity, 0));

  // ── Products ─────────────────────────────────────────────────
  async function fetchProducts() {
    try {
      loading = true;
      const r = await apiFetch('/api/products');
      if (r.ok) {
        const data = await r.json();
        products = data.data || [];
      }
    } catch (e) {
      toast.error('Failed to load products');
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    let result = products;
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      result = result.filter(p =>
        (p.name || '').toLowerCase().includes(q) ||
        (p.sku  || '').toLowerCase().includes(q)
      );
    }
    if (selectedCategory && selectedCategory !== 'Semua') {
      result = result.filter(p => p.category === selectedCategory);
    }
    filteredProducts = result;
  });

  // ── Cart ─────────────────────────────────────────────────────
  function addToCart(product) {
    if (product.stock === 0) return;
    const existing = cart.find(i => i.id === product.id);
    if (existing) {
      existing.quantity += 1;
      cart = [...cart];
    } else {
      cart = [...cart, { ...product, quantity: 1 }];
    }
  }

  function updateQty(productId, delta) {
    const item = cart.find(i => i.id === productId);
    if (!item) return;
    const newQty = item.quantity + delta;
    if (newQty <= 0) {
      cart = cart.filter(i => i.id !== productId);
    } else {
      item.quantity = newQty;
      cart = [...cart];
    }
  }

  function removeFromCart(productId) {
    cart = cart.filter(i => i.id !== productId);
  }

  function clearCart() { cart = []; }

  // ── Checkout ─────────────────────────────────────────────────
  async function processCheckout() {
    if (cart.length === 0 || checkingOut) return;
    checkingOut = true;
    try {
      const saleData = {
        invoice_number: `INV-${Date.now()}`,
        cashier_id: 1,
        store_id: 1,
        subtotal,
        discount: 0,
        tax,
        total_amount: total,
        payment_method: paymentMethod,
        items: cart.map(i => ({
          product_id: i.id,
          quantity: i.quantity,
          unit_price: i.price,
          subtotal: i.price * i.quantity,
        })),
      };

      const response = await apiFetch('/api/sales', {
        method: 'POST',
        body: JSON.stringify(saleData),
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.message || 'Checkout failed');
      }

      clearCart();
      toast.success('Transaction completed successfully!');
      await fetchProducts();
    } catch (e) {
      toast.error(e.message || 'Failed to process checkout');
    } finally {
      checkingOut = false;
    }
  }

  // ── WebSocket ────────────────────────────────────────────────
  function handleStockUpdate(payload) {
    const product = products.find(p => p.id === payload.id);
    if (product) { product.stock = payload.stock; products = [...products]; }
  }

  onMount(async () => {
    await fetchProducts();
    ws.wsEvents?.on('stock_update', handleStockUpdate);
  });

  onDestroy(() => {
    ws.wsEvents?.off('stock_update', handleStockUpdate);
  });

  // Payment method options
  const paymentOptions = [
    { id: 'cash',  label: 'Cash',  icon: Banknote  },
    { id: 'qris',  label: 'QRIS',  icon: QrCode    },
    { id: 'card',  label: 'Card',  icon: CreditCard },
  ];
</script>

<!-- POS Layout: two-column flex, full viewport height minus topbar -->
<div class="flex gap-5 h-[calc(100vh-var(--topbar-height)-48px)]">

  <!-- ── LEFT: Product Panel ───────────────────────────────── -->
  <div class="flex flex-col flex-1 min-w-0 gap-4">

    <!-- Search + Category filter -->
    <div class="card p-4 flex flex-col sm:flex-row gap-3">
      <div class="relative flex-1">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <input
          type="text"
          placeholder="Search by name or SKU…"
          class="input pl-9"
          bind:value={searchQuery}
        />
      </div>

      <!-- Category pill tabs -->
      <div class="flex items-center gap-2 overflow-x-auto no-scrollbar">
        {#each categories as cat}
          <button
            class={selectedCategory === cat ? 'pill-tab-active' : 'pill-tab'}
            onclick={() => selectedCategory = cat}
          >
            {cat}
          </button>
        {/each}
      </div>
    </div>

    <!-- Product table -->
    <div class="card flex-1 overflow-hidden flex flex-col p-0">
      <!-- Header row -->
      <div class="px-4 py-3 border-b border-border flex items-center justify-between">
        <p class="text-sm font-semibold text-text-primary">Products</p>
        {#if !loading}
          <span class="badge badge-muted">{filteredProducts.length} items</span>
        {/if}
      </div>

      <div class="flex-1 overflow-y-auto">
        {#if loading}
          <div class="flex flex-col gap-0 divide-y divide-border">
            {#each { length: 8 } as _}
              <div class="flex items-center gap-4 px-4 py-3">
                <div class="skeleton h-4 w-40"></div>
                <div class="skeleton h-4 w-20 ml-auto"></div>
                <div class="skeleton h-4 w-16"></div>
                <div class="skeleton h-7 w-14 rounded-xl"></div>
              </div>
            {/each}
          </div>
        {:else if filteredProducts.length === 0}
          <div class="empty-state">
            <div class="empty-state-icon bg-surface">
              <Tag size={28} class="text-text-muted" />
            </div>
            <p class="text-text-primary font-medium">No products found</p>
            <p class="text-text-muted text-sm mt-1">
              {searchQuery ? `No results for "${searchQuery}"` : 'No products available'}
            </p>
          </div>
        {:else}
          <table>
            <thead>
              <tr>
                <th>Product</th>
                <th>SKU</th>
                <th class="text-center">Stock</th>
                <th class="text-right">Price</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {#each filteredProducts as product (product.id)}
                <tr>
                  <td>
                    <div class="font-medium text-text-primary">{product.name}</div>
                    <div class="text-xs text-text-muted">{product.category || '—'}</div>
                  </td>
                  <td class="text-text-muted font-mono text-xs">{product.sku}</td>
                  <td class="text-center">
                    {#if product.stock === 0}
                      <Badge variant="danger">Out of stock</Badge>
                    {:else if product.stock <= 5}
                      <Badge variant="warning">Low: {product.stock}</Badge>
                    {:else}
                      <Badge variant="success">{product.stock}</Badge>
                    {/if}
                  </td>
                  <td class="text-right font-semibold text-text-primary">
                    Rp {product.price.toLocaleString('id-ID')}
                  </td>
                  <td class="text-right">
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
        {/if}
      </div>
    </div>
  </div>

  <!-- ── RIGHT: Cart Panel ─────────────────────────────────── -->
  <div class="w-[340px] flex-shrink-0 flex flex-col gap-4">
    <div class="card flex-1 flex flex-col overflow-hidden p-0">
      <!-- Cart header -->
      <div class="px-4 py-3.5 border-b border-border flex items-center justify-between">
        <div class="flex items-center gap-2">
          <ShoppingCart size={18} class="text-primary-light" />
          <span class="font-semibold text-text-primary">Cart</span>
          {#if totalItems > 0}
            <span class="badge badge-primary">{totalItems}</span>
          {/if}
        </div>
        {#if cart.length > 0}
          <button onclick={clearCart} class="btn-ghost btn-icon btn-sm text-danger hover:bg-danger-subtle" title="Clear cart">
            <Trash2 size={14} />
          </button>
        {/if}
      </div>

      <!-- Cart items -->
      <div class="flex-1 overflow-y-auto">
        {#if cart.length === 0}
          <div class="empty-state h-full">
            <div class="empty-state-icon bg-surface w-20 h-20 rounded-2xl">
              <ShoppingCart size={32} class="text-text-muted opacity-50" />
            </div>
            <p class="text-text-secondary font-medium">Cart is empty</p>
            <p class="text-text-muted text-xs mt-1">Add products to get started</p>
          </div>
        {:else}
          <div class="divide-y divide-border">
            {#each cart as item (item.id)}
              <div class="flex items-start gap-3 px-4 py-3 hover:bg-surface/40 transition-colors">
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-text-primary truncate">{item.name}</p>
                  <p class="text-xs text-text-muted mt-0.5">
                    Rp {item.price.toLocaleString('id-ID')} × {item.quantity}
                    = <span class="text-text-secondary font-medium">Rp {(item.price * item.quantity).toLocaleString('id-ID')}</span>
                  </p>
                </div>
                <div class="flex items-center gap-1.5 flex-shrink-0 mt-0.5">
                  <button
                    class="w-6 h-6 rounded-lg bg-surface hover:bg-surface-hover text-text-secondary flex items-center justify-center transition-colors border border-border"
                    onclick={() => updateQty(item.id, -1)}
                  >
                    <Minus size={11} />
                  </button>
                  <span class="w-7 text-center text-sm font-semibold text-text-primary">{item.quantity}</span>
                  <button
                    class="w-6 h-6 rounded-lg bg-surface hover:bg-surface-hover text-text-secondary flex items-center justify-center transition-colors border border-border"
                    onclick={() => updateQty(item.id, 1)}
                  >
                    <Plus size={11} />
                  </button>
                  <button
                    class="w-6 h-6 rounded-lg hover:bg-danger-subtle text-text-muted hover:text-danger flex items-center justify-center transition-colors ml-1"
                    onclick={() => removeFromCart(item.id)}
                  >
                    <X size={11} />
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Totals + checkout -->
      {#if cart.length > 0}
        <div class="border-t border-border p-4 space-y-3 bg-bg-secondary">
          <div class="space-y-2 text-sm">
            <div class="flex justify-between text-text-muted">
              <span>Subtotal</span>
              <span>Rp {subtotal.toLocaleString('id-ID')}</span>
            </div>
            <div class="flex justify-between text-text-muted">
              <span>Tax (10%)</span>
              <span>Rp {tax.toLocaleString('id-ID')}</span>
            </div>
            <div class="flex justify-between font-bold text-text-primary border-t border-border pt-2">
              <span>Total</span>
              <span class="text-primary-light text-base">Rp {total.toLocaleString('id-ID')}</span>
            </div>
          </div>

          <!-- Payment method -->
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
            onclick={processCheckout}
            disabled={checkingOut}
          >
            {#if checkingOut}
              <span class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
              Processing…
            {:else}
              Complete Purchase · Rp {total.toLocaleString('id-ID')}
            {/if}
          </button>
        </div>
      {/if}
    </div>
  </div>
</div>

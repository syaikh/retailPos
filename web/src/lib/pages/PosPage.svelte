<script>
  import { onMount, onDestroy } from 'svelte';
  import { useWebSocket } from '$lib/composables/useWebSocket';
  import Card from '$lib/components/ui/Card.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import { apiFetch } from '$lib/api/client';

  // Reactive state with runes
  let loading = $state(true);
  let products = $state([]);
  let filteredProducts = $state([]);
  let searchQuery = $state('');
  let selectedCategory = $state('Semua');
  let cart = $state([]);
  let categories = $state(['Semua', 'Makanan', 'Minuman', 'Snack', 'Lainnya']);

  const ws = useWebSocket();

  // Computed values with runes
  let cartTotal = $derived(cart.reduce((sum, item) => sum + (item.price * item.quantity), 0));
  let totalItems = $derived(cart.reduce((sum, item) => sum + item.quantity, 0));

  // Fetch products
  async function fetchProducts() {
    try {
      loading = true;
      const r = await apiFetch('/api/products');
      if (r.ok) {
        const data = await r.json();
        products = data.data || [];
      }
    } catch (e) {
      console.error('Failed to fetch products:', e);
    } finally {
      loading = false;
    }
  }

  // Filter products based on search and category
  function filterProducts() {
    let result = products;
    
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      result = result.filter(p => 
        (p.name || '').toLowerCase().includes(q) || 
        (p.sku || '').toLowerCase().includes(q)
      );
    }
    
    if (selectedCategory && selectedCategory !== 'Semua') {
      result = result.filter(p => (p.category || '') === selectedCategory);
    }
    
    filteredProducts = result;
  }

  $effect(() => {
    filterProducts();
  });

  // Cart operations
  function addToCart(product) {
    const existing = cart.find(item => item.id === product.id);
    if (existing) {
      existing.quantity += 1;
      cart = [...cart];
    } else {
      cart = [...cart, { ...product, quantity: 1 }];
    }
  }

  function updateQuantity(productId, delta) {
    const item = cart.find(i => i.id === productId);
    if (item) {
      item.quantity = Math.max(1, item.quantity + delta);
      if (item.quantity <= 0) {
        cart = cart.filter(i => i.id !== productId);
      } else {
        cart = [...cart];
      }
    }
  }

  function removeFromCart(productId) {
    cart = cart.filter(i => i.id !== productId);
  }

  function clearCart() {
    cart = [];
  }

  // Checkout
  async function processCheckout(paymentMethod) {
    if (cart.length === 0) return;

    const receiptItems = cart.map(item => ({
      name: item.name,
      quantity: item.quantity,
      unit_price: item.price,
      subtotal: item.price * item.quantity
    }));

    try {
      const subtotal = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
      const discount = 0;
      const tax = Math.floor(subtotal * 0.1);
      const totalAmount = subtotal + tax - discount;

      const saleData = {
        invoice_number: `INV-${Date.now()}`,
        cashier_id: 1,
        store_id: 1,
        subtotal,
        discount,
        tax,
        total_amount: totalAmount,
        payment_method: paymentMethod,
        items: cart.map(item => ({
          product_id: item.id,
          quantity: item.quantity,
          unit_price: item.price,
          subtotal: item.price * item.quantity
        }))
      };

      const response = await apiFetch('/api/sales', {
        method: 'POST',
        body: JSON.stringify(saleData)
      });

      if (!response.ok) {
        const error = await response.json();
        throw new Error(error.message || 'Failed to create sale');
      }

      const result = await response.json();
      clearCart();

      // Show receipt
      generateReceipt(result.data);
      await fetchProducts();
      
    } catch (error) {
      console.error('Checkout error:', error);
      alert(error.message || 'Failed to process checkout');
    }
  }

  function generateReceipt(sale) {
    const receipt = `
================================
     RETAIL POS RECEIPT
================================
Invoice: ${sale.invoice_number}
Date: ${new Date().toLocaleString()}
--------------------------------
${sale.items.map(item => 
  `${item.name} x${item.quantity}
   Rp${(item.unit_price * item.quantity).toLocaleString()}`
).join('\n')}
--------------------------------
Total: Rp${sale.total_amount.toLocaleString()}
================================
    THANK YOU FOR SHOPPING!
================================
`;
    console.log(receipt);
  }

  // WebSocket event handlers
  function handleStockUpdate(payload) {
    const product = products.find(p => p.id === payload.id);
    if (product) product.stock = payload.stock;
  }

  // Lifecycle
  onMount(async () => {
    await fetchProducts();
    ws.wsEvents?.on('stock_update', handleStockUpdate);
  });

  onDestroy(() => {
    ws.wsEvents?.off('stock_update', handleStockUpdate);
  });
</script>

<div class="pos-page p-6 bg-bg min-h-screen">
  <div class="flex items-center justify-between mb-6">
    <div>
      <h2 class="text-2xl font-bold text-white">Point of Sale</h2>
      <p class="text-slate-400 text-sm">Process transactions and manage sales</p>
    </div>
    <div class="flex items-center gap-4">
      <span class="px-3 py-1 bg-emerald-500/20 text-emerald-300 rounded-full text-sm font-medium border border-emerald-500/30">
        {totalItems} items
      </span>
    </div>
  </div>

    <div class="grid lg:grid-cols-3 gap-6">
      <!-- Product List -->
    <div class="lg:col-span-2">
      <Card class="p-4 mb-4">
        <div class="flex flex-col lg:flex-row gap-4">
          <input
            type="text"
            placeholder="Search products (name or SKU)..."
            class="input flex-1"
            bind:value={searchQuery}
          />
          <select class="input w-full lg:w-auto" bind:value={selectedCategory}>
            {#each categories as cat}
              <option value={cat}>{cat}</option>
            {/each}
          </select>
        </div>
      </Card>

      <Card class="p-0 overflow-hidden">
        <div class="overflow-x-auto">
          {#if loading}
            <div class="flex justify-center py-12">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
            </div>
          {:else if filteredProducts.length === 0}
            <div class="text-center py-12 text-slate-400">
              {#if searchQuery}
                No products found matching "{searchQuery}"
              {:else}
                No products available
              {/if}
            </div>
          {:else}
            <table class="w-full">
              <thead class="bg-slate-800/50">
                <tr>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase text-slate-400">Product</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase text-slate-400">SKU</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase text-slate-400">Stock</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase text-slate-400">Price</th>
                  <th class="text-left px-4 py-3 text-xs font-semibold uppercase text-slate-400">Action</th>
                </tr>
              </thead>
              <tbody>
                {#each filteredProducts as product (product.id)}
                  <tr class="border-t border-slate-700/50 hover:bg-slate-700/30 transition-colors">
                    <td class="px-4 py-3">
                      <div class="font-medium text-white">{product.name}</div>
                      <div class="text-xs text-slate-400">{product.category}</div>
                    </td>
                    <td class="px-4 py-3 text-slate-400 text-sm">{product.sku}</td>
                    <td class="px-4 py-3">
                      {#if product.stock <= 5}
                        <Badge variant="danger" class="text-xs">Low: {product.stock}</Badge>
                      {:else}
                        <Badge variant="success" class="text-xs">{product.stock}</Badge>
                      {/if}
                    </td>
                    <td class="px-4 py-3 text-white font-medium">Rp {product.price.toLocaleString()}</td>
                    <td class="px-4 py-3">
                      <button
                        class="btn btn-primary text-sm py-1 px-3"
                        on:click={() => addToCart(product)}
                        disabled={product.stock === 0}
                      >
                        Add
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}
        </div>
      </Card>
      </div>

      <!-- Cart Panel -->
    <div class="lg:col-span-1">
      <Card class="p-0 overflow-hidden">
        <div class="bg-slate-800/50 px-4 py-3 border-b border-slate-700">
          <h3 class="font-semibold text-white">Shopping Cart</h3>
        </div>

        {#if cart.length === 0}
          <div class="text-center py-12 text-slate-400">
            <svg class="w-12 h-12 mx-auto mb-3 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z"/>
            </svg>
            <p>Your cart is empty</p>
          </div>
        {:else}
          <div class="max-h-96 overflow-y-auto">
            {#each cart as item (item.id)}
              <div class="flex items-center gap-3 p-4 border-b border-slate-700/50 hover:bg-slate-700/30 transition-colors">
                <div class="flex-1 min-w-0">
                  <p class="font-medium text-white text-sm truncate">{item.name}</p>
                  <p class="text-xs text-slate-400">Rp {item.price.toLocaleString()}</p>
                </div>
                <div class="flex items-center gap-2">
                  <button
                    class="w-7 h-7 rounded bg-slate-700 text-white flex items-center justify-center hover:bg-slate-600 text-sm"
                    on:click={() => updateQuantity(item.id, -1)}
                  >-</button>
                  <span class="w-8 text-center text-sm font-medium">{item.quantity}</span>
                  <button
                    class="w-7 h-7 rounded bg-slate-700 text-white flex items-center justify-center hover:bg-slate-600 text-sm"
                    on:click={() => updateQuantity(item.id, 1)}
                  >+</button>
                  <button
                    class="w-7 h-7 rounded bg-red-500/20 text-red-400 flex items-center justify-center hover:bg-red-500 hover:text-white ml-1 text-sm"
                    on:click={() => removeFromCart(item.id)}
                  >×</button>
                </div>
              </div>
            {/each}
          </div>

          <div class="p-4 border-t border-slate-700 bg-slate-800/50">
            <div class="flex justify-between items-center mb-3">
              <span class="text-slate-400 text-sm">Subtotal</span>
              <span class="text-white font-medium">Rp {cartTotal.toLocaleString()}</span>
            </div>
            <div class="flex justify-between items-center mb-3">
              <span class="text-slate-400 text-sm">Tax (10%)</span>
              <span class="text-white font-medium">Rp {Math.floor(cartTotal * 0.1).toLocaleString()}</span>
            </div>
            <div class="flex justify-between items-center pt-3 border-t border-slate-700">
              <span class="text-lg font-bold text-white">Total</span>
              <span class="text-xl font-bold text-blue-400">Rp {(cartTotal + Math.floor(cartTotal * 0.1)).toLocaleString()}</span>
            </div>
            <button
              class="btn btn-success w-full mt-4"
              on:click={() => processCheckout('cash')}
            >
              Complete Purchase
            </button>
          </div>
        {/if}
      </Card>
    </div>
  </div>
</div>

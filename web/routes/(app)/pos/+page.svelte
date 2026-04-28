<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte';
  import { writable } from 'svelte/store';
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth';
  import { useWebSocket } from '$lib/composables/useWebSocket';
  import ProductTable from '$lib/components/pos/ProductTable.svelte';
  import CartPanel from '$lib/components/pos/CartPanel.svelte';
  import CheckoutPanel from '$lib/components/pos/CheckoutPanel.svelte';
  import { Card } from '$lib/components/ui';
  import { Badge } from '$lib/components/ui';

  // ... existing code up to processCheckout

  let products: any[] = [];
  let filteredProducts: any[] = [];
  let loading = true;
  let searchQuery = '';
  let selectedCategory = '';
  let categories = ['Semua', 'Makanan', 'Minuman', 'Snack', 'Lainnya'];
  let receipt = null;

  // Cart state
  let cart = writable<Array<any>>([]);
  let showCheckout = writable(false);

  // WebSocket
  const ws = useWebSocket();

  // Fetch products
  async function fetchProducts() {
    try {
      loading = true;
      const response = await fetch('/api/products');
      if (!response.ok) throw new Error('Gagal memuat produk');
      const data = await response.json();
      products = data.data || [];
      filteredProducts = products;
    } catch (error) {
      console.error('Error fetching products:', error);
    } finally {
      loading = false;
    }
  }

  // Filter products
  $: {
    let filtered = products;
    
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(p => 
        p.name.toLowerCase().includes(query) || 
        p.sku.toLowerCase().includes(query)
      );
    }
    
    if (selectedCategory && selectedCategory !== 'Semua') {
      // In real app, filter by actual category
      filtered = filtered;
    }
    
    filteredProducts = filtered;
  }

  // Cart operations
  function addToCart(product: any) {
    const existing = $cart.find(item => item.id === product.id);
    
    if (existing) {
      cart.update(items => 
        items.map(item => 
          item.id === product.id 
            ? { ...item, quantity: item.quantity + 1 }
            : item
        )
      );
    } else {
      cart.update(items => [...items, { ...product, quantity: 1 }]);
    }
  }

  function updateQuantity(productId: number, delta: number) {
    cart.update(items => {
      const updated = items.map(item => 
        item.id === productId 
          ? { ...item, quantity: Math.max(1, item.quantity + delta) }
          : item
      ).filter(item => item.quantity > 0);
      
      return updated;
    });
  }

  function removeFromCart(productId: number) {
    cart.update(items => items.filter(item => item.id !== productId));
  }

  function clearCart() {
    cart.set([]);
  }

   // Checkout
   async function processCheckout(paymentMethod: string) {
     if ($cart.length === 0) return;

     // Capture cart items for receipt (including names)
     const receiptItems = $cart.map(item => ({
       name: item.name,
       quantity: item.quantity,
       unit_price: item.price,
       subtotal: item.price * item.quantity
     }));

     try {
       const subtotal = $cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
       const discount = 0;
       const tax = Math.floor(subtotal * 0.1);
       const totalAmount = subtotal + tax - discount;

       const saleData = {
         invoice_number: `INV-${Date.now()}`,
         cashier_id: $auth.user?.id,
         store_id: $auth.user?.store_id,
         subtotal,
         discount,
         tax,
         total_amount: totalAmount,
         payment_method: paymentMethod,
         items: $cart.map(item => ({
           product_id: item.id,
           quantity: item.quantity,
           unit_price: item.price,
           subtotal: item.price * item.quantity
         }))
       };

       const response = await fetch('/api/sales', {
         method: 'POST',
         headers: { 'Content-Type': 'application/json' },
         body: JSON.stringify(saleData)
       });

       if (!response.ok) {
         const error = await response.json();
         throw new Error(error.message || 'Gagal memproses penjualan');
       }

       const result = await response.json();
       
       // Clear cart
       clearCart();
       $showCheckout = false;
       
       // Set receipt data and print
       receipt = {
         invoice_number: result.data.invoice_number,
         total_amount: result.data.total_amount,
         payment_method: paymentMethod,
         items: receiptItems
       };
       await tick();
       window.print();
       
       // Show success toast
       ui.success(`Penjualan berhasil! Invoice: ${result.data.invoice_number}`);
       
       // Refresh products to get updated stock
       await fetchProducts();
       
     } catch (error) {
       console.error('Checkout error:', error);
       ui.error(error instanceof Error ? error.message : 'Gagal memproses penjualan');
     }
   }

  // Real-time updates from WebSocket
  function handleStockUpdate(payload: any) {
    const product = products.find(p => p.id === payload.id);
    if (product) {
      product.stock = payload.stock;
      products = [...products];
      
      // Update filtered products too
      const filteredProduct = filteredProducts.find(p => p.id === payload.id);
      if (filteredProduct) {
        filteredProduct.stock = payload.stock;
        filteredProducts = [...filteredProducts];
      }
    }
  }

  function handleSaleCreated(payload: any) {
    // Refresh products stock if this sale affects our displayed products
    fetchProducts();
  }

  onMount(async () => {
    await fetchProducts();

    // Subscribe to WebSocket events
    ws.status.subscribe(s => {
      if (s === 'connected') {
        console.log('POS WebSocket connected');
      }
    });

    ws.wsEvents?.on('stock_update', handleStockUpdate);
    ws.wsEvents?.on('sale_created', handleSaleCreated);
  });

  onDestroy(() => {
    ws.wsEvents?.off('stock_update', handleStockUpdate);
    ws.wsEvents?.off('sale_created', handleSaleCreated);
  });

  // Check authentication
  $: if (!$auth.isAuthenticated) {
    goto('/login');
  }

  $: cartTotal = $cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
  $: totalItems = $cart.reduce((sum, item) => sum + item.quantity, 0);
</script>

<div class="pos-container">
  {#if loading}
    <div class="flex justify-center items-center h-64">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
    </div>
  {:else}
    <div class="pos-grid">
      <!-- Header -->
      <div class="pos-header">
        <div>
          <h1 class="text-2xl font-bold text-gray-800">Point of Sale</h1>
          <p class="text-gray-600">Kelola penjualan dan stok produk</p>
        </div>
        <div class="flex items-center space-x-4">
          <Badge variant="outline">
            <span class="w-2 h-2 bg-green-500 rounded-full mr-2"></span>
            {$auth.onlineUsers} kasir online
          </Badge>
          <Badge variant="outline">
            {$cart.length} item di keranjang
          </Badge>
        </div>
      </div>

      <!-- Search & Filter -->
      <Card class="pos-search">
        <div class="flex gap-4">
          <div class="flex-1">
            <input
              type="text"
              bind:value={searchQuery}
              placeholder="Cari produk (nama atau SKU)..."
              class="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
          <select
            bind:value={selectedCategory}
            class="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            {#each categories as cat}
              <option value={cat}>{cat}</option>
            {/each}
          </select>
        </div>
      </Card>

      <!-- Products Grid -->
      <Card class="pos-products">
        <h2 class="text-lg font-semibold mb-4">Daftar Produk</h2>
        {#if filteredProducts.length === 0}
          <div class="text-center py-12 text-gray-500">
            {#if searchQuery}
              Tidak ada produk yang cocok dengan "{searchQuery}"
            {:else}
              Belum ada produk
            {/if}
          </div>
        {:else}
          <ProductTable
            products={filteredProducts}
            onAddToCart={addToCart}
          />
        {/if}
      </Card>

      <!-- Cart Panel -->
      <Card class="pos-cart">
        <CartPanel
          items={$cart}
          total={cartTotal}
          totalItems={totalItems}
          onUpdateQuantity={updateQuantity}
          onRemoveItem={removeFromCart}
          onCheckout={() => $showCheckout = true}
        />
      </Card>
    </div>
  {/if}

  <!-- Checkout Modal -->
  {#if $showCheckout}
    <CheckoutPanel
      items={$cart}
      total={cartTotal}
      onClose={() => $showCheckout = false}
      onConfirm={processCheckout}
    />
   {/if}
 </div>

 {#if receipt}
   <div id="printable-receipt" style="display:none">
     <div class="receipt-header">
       <strong>RetailPOS</strong><br />
       {new Date().toLocaleString('id-ID')}<br />
       --------------------------
     </div>
     {#each receipt.items as item}
       <div class="receipt-item">
         <span>{item.name} x{item.quantity}</span>
         <span>Rp{(item.unit_price * item.quantity).toLocaleString()}</span>
       </div>
     {/each}
     <div class="receipt-total">
       <div class="receipt-item"><span>TOTAL:</span><span>Rp{receipt.total_amount.toLocaleString()}</span></div>
       <div class="receipt-item"><span>Pembayaran:</span><span>{receipt.payment_method}</span></div>
     </div>
     <div class="receipt-header">
       Terima kasih<br />
       {receipt.invoice_number}
     </div>
   </div>
 {/if}

 <style>
  .pos-container {
    padding: 24px;
    background-color: #f3f4f6;
    min-height: 100vh;
  }

  .pos-grid {
    display: grid;
    grid-template-columns: 1fr 380px;
    gap: 24px;
    max-width: 1600px;
    margin: 0 auto;
  }

  .pos-header {
    grid-column: 1 / -1;
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 8px;
  }

  .pos-search {
    grid-column: 1 / -1;
  }

  .pos-products {
    min-height: 600px;
  }

  .pos-cart {
    position: sticky;
    top: 24px;
    height: fit-content;
  }

  @media (max-width: 1024px) {
    .pos-grid {
      grid-template-columns: 1fr;
    }

    .pos-cart {
      position: static;
    }
  }
</style>

<script>
  import { onMount, onDestroy } from 'svelte';
  import { cart as cartStore, products as productsStore } from '$lib/stores.js';
  import api from '$lib/api.js';
  import { 
    Barcode, 
    Trash2, 
    Plus, 
    Minus, 
    CreditCard, 
    Banknote,
    Search
  } from 'lucide-svelte';

  let barcodeInput = $state('');
  let searchQuery = $state('');
  let ws;
  let checkoutLoading = $state(false);
  let paymentMethod = $state('cash');

  let total = $derived($cartStore.reduce((sum, item) => sum + (item.price * item.quantity), 0));

  onMount(() => {
    // 1. Fetch initial products
    fetchProducts();

    // 2. Set up WebSocket for real-time updates
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = import.meta.env.VITE_WS_URL || 'localhost:8080/api/ws';
    ws = new WebSocket(`${protocol}//${host}`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'stock_updated' || data.type === 'product_updated') {
        fetchProducts(); // Refresh product list
      }
    };

    // 3. Barcode Scanner Handler (HID Keyboard)
    window.addEventListener('keydown', handleGlobalKeydown);
  });

  onDestroy(() => {
    if (ws) ws.close();
    window.removeEventListener('keydown', handleGlobalKeydown);
  });

  async function fetchProducts() {
    const resp = await api.get('/products');
    productsStore.set(resp.data);
  }

  let barcodeBuffer = '';
  let lastKeyTime = Date.now();

  function handleGlobalKeydown(e) {
    // Basic Barcode listener logic:
    // Scanners usually send keys very fast and end with 'Enter'
    const currentTime = Date.now();
    
    if (currentTime - lastKeyTime > 50) {
      barcodeBuffer = '';
    }
    
    if (e.key === 'Enter') {
      if (barcodeBuffer.length > 2) {
        findAndAddByBarcode(barcodeBuffer);
        barcodeBuffer = '';
      }
    } else if (e.key.length === 1) {
      barcodeBuffer += e.key;
    }
    
    lastKeyTime = currentTime;
  }

  async function findAndAddByBarcode(code) {
    try {
      const resp = await api.get(`/products?barcode=${code}`);
      addToCart(resp.data);
    } catch (e) {
      alert(`Barcode ${code} tidak ditemukan`);
    }
  }

  function addToCart(product) {
    cartStore.update(items => {
      const existing = items.find(i => i.id === product.id);
      if (existing) {
        if (existing.quantity >= product.stock) {
          alert('Stok tidak cukup');
          return items;
        }
        return items.map(i => i.id === product.id ? { ...i, quantity: i.quantity + 1 } : i);
      }
      if (product.stock <= 0) {
        alert('Stok habis');
        return items;
      }
      return [...items, { ...product, quantity: 1 }];
    });
  }

  function removeFromCart(id) {
    cartStore.update(items => items.filter(i => i.id !== id));
  }

  function updateQty(id, delta) {
    cartStore.update(items => {
      return items.map(i => {
        if (i.id === id) {
          const newQty = i.quantity + delta;
          if (newQty <= 0) return i;
          if (newQty > i.stock) {
            alert('Stok tidak cukup');
            return i;
          }
          return { ...i, quantity: newQty };
        }
        return i;
      });
    });
  }

  async function handleCheckout() {
    if ($cartStore.length === 0) return;
    checkoutLoading = true;
    try {
      const saleData = {
        total_amount: total,
        payment_method: paymentMethod,
        items: $cartStore.map(i => ({
          product_id: i.id,
          quantity: i.quantity,
          price_at_sale: i.price
        }))
      };
      await api.post('/sales', saleData);
      cartStore.set([]);
      alert('Transaksi Berhasil!');
    } catch (e) {
      alert('Transaksi Gagal: ' + (e.response?.data?.error || e.message));
    } finally {
      checkoutLoading = false;
    }
  }

  let filteredProducts = $derived($productsStore.filter(p => 
    p.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
    p.sku.includes(searchQuery)
  ));
</script>

<div class="pos-container">
  <!-- Left Side: Product Selection -->
  <div class="product-area">
    <div class="search-bar premium-card glass">
      <Search size={20} class="icon" />
      <input type="text" placeholder="Cari produk atau scan barcode..." bind:value={searchQuery} />
    </div>

    <div class="product-grid">
      {#each filteredProducts as product}
        <button class="product-card premium-card" on:click={() => addToCart(product)} disabled={product.stock <= 0}>
          <div class="stock-badge" class:out={product.stock <= 0}>
            Stok: {product.stock}
          </div>
          <div class="sku">{product.sku}</div>
          <div class="name">{product.name}</div>
          <div class="price">Rp {product.price.toLocaleString()}</div>
        </button>
      {/each}
    </div>
  </div>

  <!-- Right Side: Cart & Checkout -->
  <aside class="cart-area premium-card glass">
    <div class="cart-header">
      <h2>Keranjang</h2>
      <button class="clear-btn" on:click={() => cartStore.set([])}>Clear</button>
    </div>

    <div class="cart-items">
      {#if $cartStore.length === 0}
        <div class="empty-state">
          <Barcode size={48} />
          <p>Scan barcode atau ketik nama produk</p>
        </div>
      {:else}
        {#each $cartStore as item}
          <div class="cart-item">
            <div class="item-info">
              <div class="item-name">{item.name}</div>
              <div class="item-price">Rp {item.price.toLocaleString()}</div>
            </div>
            <div class="item-actions">
              <button class="qty-btn" on:click={() => updateQty(item.id, -1)}><Minus size={14}/></button>
              <span class="qty">{item.quantity}</span>
              <button class="qty-btn" on:click={() => updateQty(item.id, 1)}><Plus size={14}/></button>
              <button class="remove-btn" on:click={() => removeFromCart(item.id)}><Trash2 size={16}/></button>
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <div class="cart-footer">
      <div class="payment-methods">
        <button class="method-btn" class:active={paymentMethod === 'cash'} on:click={() => paymentMethod = 'cash'}>
          <Banknote size={18} /> Tunai
        </button>
        <button class="method-btn" class:active={paymentMethod === 'card'} on:click={() => paymentMethod = 'card'}>
          <CreditCard size={18} /> Kartu
        </button>
      </div>

      <div class="total-row">
        <span>Total</span>
        <span class="total-price">Rp {total.toLocaleString()}</span>
      </div>

      <button class="checkout-btn" disabled={checkoutLoading || $cartStore.length === 0} on:click={handleCheckout}>
        {checkoutLoading ? 'Processing...' : 'BAYAR SEKARANG'}
      </button>
    </div>
  </aside>
</div>

<style>
  .pos-container {
    display: grid;
    grid-template-columns: 1fr 400px;
    gap: 24px;
    height: calc(100vh - 120px);
  }

  .product-area {
    display: flex;
    flex-direction: column;
    gap: 24px;
    overflow-y: auto;
  }

  .search-bar {
    position: relative;
    padding: 12px 20px;
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .search-bar input {
    flex: 1;
    background: transparent;
    border: none;
    font-size: 1.1rem;
  }

  .product-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 16px;
  }

  .product-card {
    text-align: left;
    position: relative;
    padding: 16px;
    transition: transform 0.2s, border-color 0.2s;
  }

  .product-card:hover:not(:disabled) {
    transform: translateY(-4px);
    border-color: var(--primary);
  }

  .product-card:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .stock-badge {
    position: absolute;
    top: 12px;
    right: 12px;
    font-size: 0.75rem;
    padding: 2px 8px;
    border-radius: 99px;
    background: var(--success);
    color: white;
  }

  .stock-badge.out {
    background: var(--danger);
  }

  .sku {
    color: var(--text-secondary);
    font-size: 0.75rem;
    margin-bottom: 4px;
  }

  .name {
    font-weight: 600;
    font-size: 1rem;
    margin-bottom: 8px;
  }

  .price {
    font-weight: 800;
    color: var(--accent);
  }

  .cart-area {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .cart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--border);
  }

  .clear-btn {
    font-size: 0.875rem;
    color: var(--text-secondary);
    background: transparent;
  }

  .cart-items {
    flex: 1;
    overflow-y: auto;
    padding: 16px 0;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    text-align: center;
    gap: 16px;
  }

  .cart-item {
    padding: 12px;
    border-bottom: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .item-name {
    font-weight: 600;
  }

  .item-price {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .item-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .qty-btn {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-main);
    color: white;
  }

  .remove-btn {
    color: var(--danger);
    background: transparent;
    margin-left: 8px;
  }

  .cart-footer {
    padding-top: 16px;
    border-top: 2px solid var(--border);
  }

  .payment-methods {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-bottom: 20px;
  }

  .method-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px;
    background: var(--bg-main);
    color: var(--text-secondary);
  }

  .method-btn.active {
    background: var(--primary);
    color: white;
  }

  .total-row {
    display: flex;
    justify-content: space-between;
    font-size: 1.25rem;
    font-weight: 700;
    margin-bottom: 20px;
  }

  .total-price {
    color: var(--accent);
  }

  .checkout-btn {
    width: 100%;
    padding: 16px;
    background: var(--success);
    color: white;
    font-weight: 800;
    font-size: 1.1rem;
    letter-spacing: 0.05em;
  }

  .checkout-btn:disabled {
    opacity: 0.5;
    background: var(--text-secondary);
  }
</style>

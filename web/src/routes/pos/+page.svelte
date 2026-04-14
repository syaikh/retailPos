<script>
  import { onMount, onDestroy } from 'svelte';
  import { tick } from 'svelte';
  import { cart as cartStore, products as productsStore } from '$lib/stores.js';
  import api from '$lib/api.js';
  import { 
    Barcode, 
    Trash2, 
    Plus, 
    Minus, 
    CreditCard, 
    Banknote,
    Search,
    Package,
    X,
    AlertCircle
  } from 'lucide-svelte';
  import Pagination from '$lib/components/Pagination.svelte';

  let barcodeInput = $state('');
  let searchQuery = $state('');
  let ws;
  let checkoutLoading = $state(false);
  let paymentMethod = $state('cash');
  let searchInput;

  // Pagination for POS
  let posLimit = $state(10);
  let posOffset = $state(0);
  let posTotal = $state(0);
  let posLoading = $state(false);
  let posProducts = $state([]);

  let total = $derived($cartStore.reduce((sum, item) => sum + (item.price * item.quantity), 0));

  onMount(async () => {
    // 1. Initial empty or minimal products if needed
    // fetchProducts(); 

    // 2. Set up WebSocket for real-time updates
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = import.meta.env.VITE_WS_URL || 'localhost:8080/api/ws';
    ws = new WebSocket(`${protocol}//${host}`);

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'stock_updated' || data.type === 'product_updated') {
        fetchProducts(); // Refresh current page
      }
    };

    // 3. Barcode Scanner Handler (HID Keyboard)
    window.addEventListener('keydown', handleGlobalKeydown);

    // 4. Autofocus search input
    await tick();
    if (searchInput) {
      searchInput.focus();
    }
  });

  onDestroy(() => {
    cleanup();
    if (ws) ws.close();
    window.removeEventListener('keydown', handleGlobalKeydown);
  });

  async function fetchProducts() {
    const q = searchQuery.trim();
    if (q.length < 3) {
      posProducts = [];
      posTotal = 0;
      posLoading = false;
      return;
    }

    posLoading = true;
    try {
      const resp = await api.get(`/products?limit=${posLimit}&offset=${posOffset}&search=${q}`);
      posProducts = resp.data.data || [];
      posTotal = resp.data.total || 0;
    } catch (e) {
      console.error('Failed to fetch products:', e);
    } finally {
      posLoading = false;
    }
  }

  // Debounce timer
  let searchDebounceTimer = null;

  // Effect to trigger search when query or page changes
  $effect(() => {
    const q = searchQuery;
    const offset = posOffset;
    const limit = posLimit;
    
    // Clear previous timer
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }
    
    // Debounce search
    searchDebounceTimer = setTimeout(() => {
      fetchProducts();
    }, 300);
  });

  function cleanup() {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }
  }

  function handlePosPageChange(newOffset, newLimit) {
    if (newLimit !== undefined) posLimit = newLimit;
    posOffset = newOffset;
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
      const resp = await api.get(`/products?search=${code}&limit=1`);
      const results = resp.data.data;
      if (results && results.length > 0) {
        addToCart(results[0]);
      } else {
        alert(`Barcode ${code} tidak ditemukan`);
      }
    } catch (e) {
      alert(`Gagal mencari barcode ${code}`);
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

  function setQty(id, event) {
    const val = event.target.value;
    cartStore.update(items => {
      return items.map(i => {
        if (i.id === id) {
          let newQty = parseInt(val, 10);
          if (isNaN(newQty) || newQty <= 0) {
            newQty = 1;
            event.target.value = newQty;
          } else if (newQty > i.stock) {
            alert(`Stok tidak cukup. Maksimal: ${i.stock}`);
            newQty = i.stock;
            event.target.value = newQty;
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

  // Remove filteredProducts derived state as we are using server-side search
</script>

<div class="pos-container">
  <!-- Left Side: Product Selection -->
  <div class="product-area">
    <div class="search-bar premium-card glass">
      <span class="icon"><Search size={20} /></span>
      <input 
        type="text" 
        placeholder="Cari produk atau scan barcode..." 
        bind:value={searchQuery}
        bind:this={searchInput}
        oninput={() => posOffset = 0}
      />
      {#if searchQuery}
        <button class="clear-search-btn" aria-label="Hapus pencarian" onclick={() => { searchQuery = ''; posOffset = 0; searchInput.focus(); }}>
          <X size={18} />
        </button>
      {/if}
    </div>

    <div class="product-table-container premium-card">
      {#if !searchQuery.trim()}
        <div class="empty-search-state">
          <Search size={64} />
          <h3>Mulai Pencarian</h3>
          <p>Ketik nama produk atau scan barcode untuk menemukan item</p>
        </div>
      {:else if searchQuery.trim().length < 3}
        <div class="empty-search-state warning">
          <AlertCircle size={64} color="var(--accent)" />
          <h3>Teks Terlalu Pendek</h3>
          <p>Masukkan minimal <strong>3 karakter</strong> untuk memulai pencarian produk.</p>
        </div>
      {:else if posLoading}
        <div class="empty-search-state">
          <div class="loading-spinner"></div>
          <p>Mencari produk...</p>
        </div>
      {:else if posProducts.length === 0}
        <div class="empty-search-state">
          <Package size={64} />
          <h3>Produk Tidak Ditemukan</h3>
          <p>Coba kata kunci lain atau scan barcode secara langsung</p>
        </div>
      {:else}
        <table class="product-table">
          <thead>
            <tr>
              <th>SKU</th>
              <th>Barcode</th>
              <th>Nama Produk</th>
              <th>Harga</th>
              <th>Stok</th>
              <th>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {#each posProducts as product}
              <tr class:disabled={product.stock <= 0}>
              <td><code>{product.sku}</code></td>
              <td>
                {#if product.barcode}
                  <code>{product.barcode}</code>
                {:else}
                  <span class="text-dim">-</span>
                {/if}
              </td>
                <td><strong>{product.name}</strong></td>
                <td class="price">Rp {product.price.toLocaleString()}</td>
                <td>
                  <span class="stock-badge" class:out={product.stock <= 0}>
                    {product.stock} {product.stock <= 0 ? 'habis' : 'pcs'}
                  </span>
                </td>
                <td>
                  <button class="add-cart-btn" onclick={() => addToCart(product)} disabled={product.stock <= 0}>
                    <Plus size={16} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
        <Pagination 
          total={posTotal} 
          limit={posLimit} 
          offset={posOffset} 
          onPageChange={handlePosPageChange} 
        />
      {/if}
    </div>
  </div>

  <!-- Right Side: Cart & Checkout -->
  <aside class="cart-area premium-card glass">
    <div class="cart-header">
      <h2>Keranjang</h2>
      <button class="clear-btn" onclick={() => cartStore.set([])}>Clear</button>
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
              <button class="qty-btn" onclick={() => updateQty(item.id, -1)}><Minus size={14}/></button>
              <input 
                type="number" 
                class="qty-input" 
                value={item.quantity} 
                min="1"
                max={item.stock}
                onchange={(e) => setQty(item.id, e)} 
              />
              <button class="qty-btn" onclick={() => updateQty(item.id, 1)}><Plus size={14}/></button>
              <button class="remove-btn" onclick={() => removeFromCart(item.id)}><Trash2 size={16}/></button>
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <div class="cart-footer">
      <div class="payment-methods">
        <button class="method-btn" class:active={paymentMethod === 'cash'} onclick={() => paymentMethod = 'cash'}>
          <Banknote size={18} /> Tunai
        </button>
        <button class="method-btn" class:active={paymentMethod === 'card'} onclick={() => paymentMethod = 'card'}>
          <CreditCard size={18} /> Kartu
        </button>
      </div>

      <div class="total-row">
        <span>Total</span>
        <span class="total-price">Rp {total.toLocaleString()}</span>
      </div>

      <button class="checkout-btn" disabled={checkoutLoading || $cartStore.length === 0} onclick={handleCheckout}>
        {checkoutLoading ? 'Processing...' : 'BAYAR SEKARANG'}
      </button>
    </div>
  </aside>
</div>

<style>
  .pos-container {
    display: grid;
    grid-template-columns: 1fr 320px;
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
    color: white;
  }

  .search-bar input:focus {
    outline: none;
  }

  .clear-search-btn {
    background: transparent;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 4px;
    border-radius: 50%;
    transition: all 0.2s;
  }

  .clear-search-btn:hover {
    color: white;
    background: rgba(255, 255, 255, 0.1);
  }

  .product-table-container {
    overflow: auto;
    max-height: 100%;
  }

  .empty-search-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    min-height: 300px;
    color: var(--text-secondary);
    text-align: center;
    gap: 16px;
    padding: 40px 20px;
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid rgba(99, 102, 241, 0.2);
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .empty-search-state h3 {
    font-size: 1.25rem;
    color: #e2e8f0;
    margin: 0;
  }

  .empty-search-state p {
    max-width: 300px;
    margin: 0;
  }

  .empty-search-state.warning h3 {
    color: var(--accent);
  }

  .product-table {
    width: 100%;
    border-collapse: collapse;
    background: #1e293b;
  }

  .product-table thead {
    background: #0f172a;
    position: sticky;
    top: 0;
    z-index: 10;
  }

  .product-table th {
    padding: 14px 12px;
    text-align: left;
    font-weight: 600;
    color: var(--accent);
    border-bottom: 2px solid var(--primary);
    font-size: 0.875rem;
    text-transform: uppercase;
  }

  .product-table tbody tr {
    border-bottom: 1px solid rgba(148, 163, 184, 0.1);
    transition: background-color 0.2s;
  }

  .product-table tbody tr:hover:not(.disabled) {
    background: rgba(99, 102, 241, 0.1);
  }

  .product-table tbody tr.disabled {
    opacity: 0.5;
    background: rgba(239, 68, 68, 0.05);
  }

  .product-table td {
    padding: 12px;
    color: #e2e8f0;
    font-size: 0.95rem;
  }

  .product-table code {
    background: #0f172a;
    padding: 4px 8px;
    border-radius: 4px;
    color: var(--accent);
    font-family: monospace;
    font-size: 0.85rem;
    display: inline-block;
  }


  .product-table .price {
    font-weight: 700;
    color: var(--accent);
  }

  .stock-badge {
    display: inline-block;
    font-size: 0.8rem;
    padding: 4px 10px;
    border-radius: 99px;
    background: rgba(16, 185, 129, 0.2);
    color: #10b981;
    font-weight: 600;
  }

  .stock-badge.out {
    background: rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .add-cart-btn {
    background: var(--primary);
    color: white;
    padding: 6px 10px;
    border-radius: 4px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: background 0.2s;
  }

  .add-cart-btn:hover:not(:disabled) {
    background: #5b61f5;
  }

  .add-cart-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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

  .qty-input {
    width: 48px;
    height: 28px;
    text-align: center;
    background: transparent;
    border: 1px solid var(--border);
    color: white;
    font-weight: 600;
    appearance: textfield;
  }

  /* Hilangkan panah spinner bawaan browser pada input number */
  .qty-input::-webkit-outer-spin-button,
  .qty-input::-webkit-inner-spin-button {
    -webkit-appearance: none;
    appearance: none;
    margin: 0;
  }
  .qty-input[type=number] {
    -moz-appearance: textfield;
    appearance: textfield;
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

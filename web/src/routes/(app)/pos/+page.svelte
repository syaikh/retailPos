<script lang="ts">
	import { onDestroy } from 'svelte';
	import { get } from 'svelte/store';
	import type { Product, CartItem } from '$lib/domain/entities';
	import { useCart } from '$lib/composables/useCart';
	import { useCheckout } from '$lib/composables/useCheckout';
	import { useProductSearch } from '$lib/composables/useProductSearch';
	import { useWebSocket } from '$lib/composables/useWebSocket';
	import { auth, type User } from '$lib/stores/auth';
	import { ui } from '$lib/stores/ui';
	import Toast from '$lib/components/ui/Toast.svelte';
	import ProductTable from '$lib/components/pos/ProductTable.svelte';
	import CartPanel from '$lib/components/pos/CartPanel.svelte';
	import CheckoutPanel from '$lib/components/pos/CheckoutPanel.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { calculateTotal } from '$lib/domain/services/CartService';

	// Initialize composables
	const cart = useCart();
	const productSearch = useProductSearch();
	const checkout = useCheckout();

	// Destructure stores and functions
	const { cartItems, addToCart, removeFromCart, incrementQuantity, decrementQuantity, setQuantity, getTotal } = cart;
	const { products: productList, total: productTotal, isLoading: productLoading, search, findByBarcode, getAll } = productSearch;
	const { isLoading: checkoutLoading, checkout: doCheckout, checkoutAndClear } = checkout;

	// Get storeId from user (should be loaded by handle hook)
	const currentUser = get(auth).user;
	const storeId = currentUser?.store_id?.toString() || '1';
	if (!currentUser?.store_id) {
		console.warn('[POS] User store_id not found, using fallback "1"');
	}

	// Initialize WebSocket at top-level (required by Svelte composable pattern)
	const ws = useWebSocket(storeId);

	// Reactive connection status
	let wsStatus = $state<'disconnected' | 'connecting' | 'connected' | 'reconnecting'>('disconnected');
	$effect(() => {
		const unsub = ws.status.subscribe((value) => {
			wsStatus = value;
		});
		return () => unsub();
	});

	// Register WebSocket event handlers
	ws.setStockUpdateHandler(handleStockUpdate);
	ws.setSaleCreatedHandler(handleSaleCreated);
	ws.setReconnectedHandler(handleReconnected);

	// Local state
	let searchQuery = $state('');
	let paymentMethod = $state<'cash' | 'card'>('cash');
	let posLimit = $state(10);
	let posOffset = $state(0);
	let searchInput!: HTMLInputElement;

	// Reactive total calculation
	let cartTotal = $derived(calculateTotal($cartItems));

	// Barcode scanner state
	let barcodeBuffer = '';
	let lastKeyTime = Date.now();

	onDestroy(() => {
		ws.disconnect();
	});

	// Auto-focus search on mount
	$effect(() => {
		const timer = setTimeout(() => {
			searchInput?.focus();
		}, 0);
		return () => clearTimeout(timer);
	});

	// Search effect with debounce
	$effect(() => {
		const q = searchQuery;
		const offset = posOffset;
		const limit = posLimit;

		const timer = setTimeout(() => {
			search(q, limit, offset);
		}, 300);

		return () => clearTimeout(timer);
	});

	function handleSearchInput() {
		posOffset = 0;
	}

	function handlePageChange(newOffset: number, newLimit?: number) {
		if (newLimit !== undefined) posLimit = newLimit;
		posOffset = newOffset;
	}

	async function handleAddToCart(product: Product) {
		await addToCart(product);
	}

	function handleRemoveFromCart(productId: number) {
		removeFromCart(productId);
	}

	async function handleIncrement(productId: number) {
		await incrementQuantity(productId);
	}

	async function handleDecrement(productId: number) {
		await decrementQuantity(productId);
	}

	async function handleSetQuantity(productId: number, quantity: number) {
		await setQuantity(productId, quantity);
	}

	async function handleCheckout() {
		const success = await doCheckout($cartItems, paymentMethod);
		if (success) {
			cart.clearCart();
		}
	}

	// Barcode scanner
	function handleGlobalKeydown(e: KeyboardEvent) {
		const currentTime = Date.now();

		if (currentTime - lastKeyTime > 50) {
			barcodeBuffer = '';
		}

		if (e.key === 'Enter') {
			if (barcodeBuffer.length > 2) {
				handleBarcodeScan(barcodeBuffer);
				barcodeBuffer = '';
			}
		} else if (e.key.length === 1) {
			barcodeBuffer += e.key;
		}

		lastKeyTime = currentTime;
	}

	async function handleBarcodeScan(code: string) {
		const product = await findByBarcode(code);
		if (product) {
			await addToCart(product);
		}
	}

	// WebSocket event handlers
	function handleStockUpdate(payload: { product_id: number }) {
		if (searchQuery.length >= 3) {
			productSearch.search(searchQuery, posLimit, posOffset);
		}
	}

	function handleSaleCreated(sale: any) {
		ui.success(`Transaksi #${sale.transaction_code} berhasil`);
	}

	function handleReconnected(payload: { stale: boolean }) {
		if (payload.stale) {
			if (searchQuery.length >= 3) {
				productSearch.search(searchQuery, posLimit, posOffset);
			} else {
				productSearch.getAll(posLimit, posOffset);
			}
			ui.info('Koneksi pulih. Data stok telah diperbarui.', 3000);
		}
	}

	// Visibility change for tab refocus
	$effect(() => {
		const handler = () => {
			if (document.visibilityState === 'visible' && ws) {
				if (wsStatus === 'connected') {
					if (searchQuery.length >= 3) {
						productSearch.search(searchQuery, posLimit, posOffset);
					} else {
						productSearch.getAll(posLimit, posOffset);
					}
				}
			}
		};
		document.addEventListener('visibilitychange', handler);
		return () => document.removeEventListener('visibilitychange', handler);
	});

	// Computed WebSocket status for UI
	function getWsStatusClass(): string {
		switch (wsStatus) {
			case 'connected': return 'connected';
			case 'connecting':
			case 'reconnecting': return 'reconnecting';
			default: return 'disconnected';
		}
	}

	function getWsStatusText(): string {
		switch (wsStatus) {
			case 'connected': return 'Terhubung';
			case 'connecting': return 'Menghubungkan...';
			case 'reconnecting': return 'Menghubungkan kembali...';
			case 'disconnected': return 'Stok tidak real-time';
			default: return 'Tidak terhubung';
		}
	}
</script>

<div class="pos-container">
	<!-- Left: Product Selection -->
	<div class="product-area">
		<div class="search-bar premium-card glass">
			<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
			<input
				type="text"
				class="search-input"
				placeholder="Cari produk atau scan barcode..."
				bind:value={searchQuery}
				bind:this={searchInput}
				oninput={handleSearchInput}
			/>
			{#if searchQuery}
				<button
					class="clear-search-btn"
					aria-label="Hapus pencarian"
					onclick={() => { searchQuery = ''; posOffset = 0; searchInput?.focus(); }}
				>
					<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
				</button>
			{/if}
			<div class="ws-indicator" title={getWsStatusText()}>
				<span class="dot {getWsStatusClass()}"></span>
				{#if getWsStatusClass() === 'disconnected'}
					<span class="ws-warning">Stok tidak real-time</span>
				{/if}
				{#if getWsStatusClass() === 'reconnecting'}
					<span class="ws-reconnecting">Menghubungkan kembali...</span>
				{/if}
			</div>
		</div>

		<div class="product-table-container premium-card">
			<ProductTable
				products={$productList}
				loading={$productLoading}
				total={$productTotal}
				limit={posLimit}
				offset={posOffset}
				searchQuery={searchQuery}
				onPageChange={handlePageChange}
				onAddToCart={handleAddToCart}
			/>
		</div>
	</div>

	<!-- Right: Cart & Checkout -->
	<aside class="cart-area premium-card glass">
		<CartPanel
			items={$cartItems}
			onRemove={handleRemoveFromCart}
			onIncrement={handleIncrement}
			onDecrement={handleDecrement}
			onSetQuantity={handleSetQuantity}
			onClear={cart.clearCart}
		/>

		<CheckoutPanel
			total={cartTotal}
			{paymentMethod}
			isLoading={$checkoutLoading}
			onCheckout={handleCheckout}
		/>
	</aside>
</div>

<Toast />

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
		overflow: hidden;
	}

	.search-bar {
		position: relative;
		padding: 12px 20px;
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.search-input {
		flex: 1;
		background: transparent;
		border: none;
		font-size: 1.1rem;
		color: white;
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
		cursor: pointer;
		border: none;
	}

	.clear-search-btn:hover {
		color: white;
		background: rgba(255, 255, 255, 0.1);
	}

	.product-table-container {
		flex: 1;
		overflow: hidden;
		border-radius: 12px;
	}

	.cart-area {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.ws-indicator {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-left: auto;
		padding-left: 12px;
		font-size: 0.75rem;
		color: var(--text-secondary);
	}

	.ws-indicator .dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		transition: background-color 0.3s ease;
	}

	.ws-indicator .dot.connected {
		background-color: #22c55e;
	}

	.ws-indicator .dot.reconnecting {
		background-color: #eab308;
		animation: pulse 1s infinite;
	}

	.ws-indicator .dot.disconnected {
		background-color: #ef4444;
	}

	.ws-warning, .ws-reconnecting {
		font-size: 0.7rem;
		color: var(--text-secondary);
		white-space: nowrap;
	}

	@keyframes pulse {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.5; }
	}
</style>

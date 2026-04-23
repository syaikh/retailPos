<script lang="ts">
	import type { Product } from '$lib/domain/entities';
	import { Search, Package, AlertCircle, X, Plus, Copy } from 'lucide-svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { ui } from '$lib/stores/ui';

	let {
		products,
		loading = false,
		total = 0,
		limit = 10,
		offset = 0,
		searchQuery = '',
		onPageChange,
		onAddToCart
	}: {
		products: Product[];
		loading?: boolean;
		total?: number;
		limit?: number;
		offset?: number;
		searchQuery?: string;
		onPageChange?: (newOffset: number, newLimit?: number) => void;
		onAddToCart: (product: Product) => void;
	} = $props();

	async function copyToClipboard(text: string, label: string) {
		try {
			await navigator.clipboard.writeText(text);
			ui.success(`${label} disalin`);
		} catch (e) {
			ui.error(`Gagal menyalin ${label}`);
		}
	}
</script>

<div class="product-table-container">
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
	{:else if loading}
		<div class="empty-search-state">
			<div class="loading-spinner"></div>
			<p>Mencari produk...</p>
		</div>
	{:else if products.length === 0}
		<div class="empty-search-state">
			<Package size={64} />
			<h3>Produk Tidak Ditemukan</h3>
			<p>Coba kata kunci lain atau scan barcode secara langsung</p>
		</div>
	{:else}
		<div class="table-content">
			<table class="product-table">
				<thead>
					<tr>
						<th class="col-product">Nama Produk / SKU / Barcode</th>
						<th class="col-price">Harga</th>
						<th class="col-stock">Stok</th>
						<th class="col-action">Aksi</th>
					</tr>
				</thead>
				<tbody>
					{#each products as product (product.id)}
						<tr class:disabled={product.stock <= 0}>
							<td class="col-product">
								<div class="product-name">{product.name}</div>
								<div class="product-meta">
									<div class="meta-item">
										<span class="meta-label">SKU:</span>
										<code>{product.sku}</code>
										<button 
											class="copy-btn" 
											onclick={() => copyToClipboard(product.sku, 'SKU')}
											title="Salin SKU"
										>
											<Copy size={12} />
										</button>
									</div>
									<span class="meta-divider">|</span>
									<div class="meta-item">
										<span class="meta-label">BC:</span>
										{#if product.barcode}
											<code>{product.barcode}</code>
											<button 
												class="copy-btn" 
												onclick={() => copyToClipboard(product.barcode || '', 'Barcode')}
												title="Salin Barcode"
											>
												<Copy size={12} />
											</button>
										{:else}
											<span class="text-dim">-</span>
										{/if}
									</div>
								</div>
							</td>
							<td class="col-price price">Rp {product.price.toLocaleString('id-ID')}</td>
							<td class="col-stock">
								<Badge variant={product.stock > 0 ? 'success' : 'danger'}>
									{product.stock} {product.stock <= 0 ? 'habis' : 'pcs'}
								</Badge>
							</td>
							<td class="col-action">
								<Button
									size="sm"
									class="add-icon-btn"
									onclick={() => onAddToCart(product)}
									disabled={product.stock <= 0}
									title="Tambah ke Keranjang"
								>
									<Plus size={18} />
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
		{#if onPageChange}
			<div class="pagination-wrapper">
				<Pagination {total} {limit} {offset} {onPageChange} />
			</div>
		{/if}
	{/if}
</div>

<style>
	.product-table-container {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
		background: var(--color-bg-main);
	}

	.table-content {
		overflow: auto;
		flex: 1;
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

	.empty-search-state.warning h3 {
		color: var(--accent);
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
		color: var(--text-primary);
		margin: 0;
	}

	.empty-search-state p {
		max-width: 300px;
		margin: 0;
		color: var(--text-secondary);
	}

	.product-table {
		width: 100%;
		border-collapse: collapse;
		table-layout: fixed;
	}

	.product-table thead {
		background: var(--color-bg-elevated);
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
		font-size: 0.8rem;
		text-transform: uppercase;
		white-space: nowrap;
	}

	.product-table tbody tr {
		border-bottom: 1px solid var(--border);
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
		color: var(--text-primary);
		font-size: 0.95rem;
		vertical-align: middle;
	}

	/* Column Widths - Balanced */
	.col-product { width: 55%; }
	.col-price { width: 160px; text-align: right; }
	.col-stock { width: 100px; text-align: center; }
	.col-action { width: 70px; text-align: center; }

	.product-name {
		font-weight: 600;
		color: white;
		margin-bottom: 4px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.product-meta {
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 8px;
	}

	.meta-divider {
		color: var(--border);
		font-size: 0.75rem;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 0.75rem;
		color: var(--text-secondary);
	}

	.meta-label {
		color: var(--text-dim);
		font-weight: 500;
	}

	.product-table code {
		background: rgba(255, 255, 255, 0.05);
		padding: 1px 4px;
		border-radius: 4px;
		color: var(--accent);
		font-family: monospace;
		font-size: 0.75rem;
		white-space: nowrap;
	}

	.copy-btn {
		background: transparent;
		border: none;
		color: var(--text-secondary);
		cursor: pointer;
		padding: 2px;
		display: flex;
		align-items: center;
		transition: color 0.2s;
	}

	.copy-btn:hover {
		color: var(--primary);
	}

	.product-table .price {
		font-weight: 700;
		color: var(--accent);
		text-align: right;
	}

	/* Fix for Badge alignment */
	:global(.col-stock .badge) {
		display: inline-block;
		min-width: 70px;
		text-align: center;
	}

	:global(.add-icon-btn) {
		padding: 8px !important;
		min-width: unset !important;
		border-radius: 8px !important;
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.pagination-wrapper {
		padding: 12px;
		border-top: 1px solid var(--border);
		background: var(--color-bg-elevated);
		margin-top: auto;
	}
</style>

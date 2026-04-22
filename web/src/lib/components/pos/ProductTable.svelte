<script lang="ts">
	import type { Product } from '$lib/domain/entities';
	import { Search, Package, AlertCircle, X, Plus } from 'lucide-svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

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
</script>

<div class="product-table-wrapper">
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
		<div class="table-container">
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
					{#each products as product (product.id)}
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
								<Badge variant={product.stock > 0 ? 'success' : 'danger'}>
									{product.stock} {product.stock <= 0 ? 'habis' : 'pcs'}
								</Badge>
							</td>
							<td>
								<Button
									size="sm"
									onclick={() => onAddToCart(product)}
									disabled={product.stock <= 0}
								>
									<Plus size={14} />
									Tambah
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
			{#if onPageChange}
				<Pagination {total} {limit} {offset} {onPageChange} />
			{/if}
		</div>
	{/if}
</div>

<style>
	.product-table-wrapper {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	.table-container {
		overflow: auto;
		height: 100%;
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
		font-size: 0.875rem;
		text-transform: uppercase;
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
	}

	.product-table code {
		background: var(--color-bg-elevated);
		padding: 4px 8px;
		border-radius: 4px;
		color: var(--accent);
		font-family: monospace;
		font-size: 0.85rem;
	}

	.product-table .price {
		font-weight: 700;
		color: var(--accent);
	}

	.text-dim {
		color: var(--text-secondary);
	}
</style>

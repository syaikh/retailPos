import { get } from 'svelte/store';
import { writable } from 'svelte/store';
import { productApi } from '$lib/api/productApi';
import type { Product, ProductPage } from '$lib/domain/entities';
import { ui } from '$lib/stores/ui';

/**
 * Product Search composable - handles product search with loading/error states
 */

export function useProductSearch() {
	let products = writable<Product[]>([]);
	let total = writable(0);
	let isLoading = writable(false);
	let error = writable<string | null>(null);

	/**
	 * Search products by query string
	 */
	async function search(query: string, limit = 20, offset = 0) {
		const trimmed = query.trim();
		if (trimmed.length < 3) {
			products.set([]);
			total.set(0);
			isLoading.set(false);
			error.set(null);
			return;
		}

		isLoading.set(true);
		error.set(null);

		try {
			const { data } = await productApi.search(trimmed, limit, offset);
			products.set(data.data || []);
			total.set(data.total || 0);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Gagal mencari produk';
			error.set(message);
			ui.error(message);
			products.set([]);
			total.set(0);
		} finally {
			isLoading.set(false);
		}
	}

	/**
	 * Find single product by barcode
	 */
	async function findByBarcode(barcode: string): Promise<Product | null> {
		try {
			const { data } = await productApi.findByBarcode(barcode);
			return data;
		} catch (err) {
			ui.error('Produk tidak ditemukan');
			return null;
		}
	}

	/**
	 * Get all products (no search)
	 */
	async function getAll(limit = 50, offset = 0) {
		isLoading.set(true);
		error.set(null);

		try {
			const { data } = await productApi.getAll(limit, offset);
			products.set(data.data || []);
			total.set(data.total || 0);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Gagal memuat produk';
			error.set(message);
			ui.error(message);
		} finally {
			isLoading.set(false);
		}
	}

	/**
	 * Clear results
	 */
	function clear() {
		products.set([]);
		total.set(0);
		error.set(null);
	}

	/**
	 * Get current product list
	 */
	function getProducts(): Product[] {
		return get(products);
	}

	/**
	 * Get total count
	 */
	function getTotal(): number {
		return get(total);
	}

	return {
		products,
		total,
		isLoading,
		error,
		search,
		findByBarcode,
		getAll,
		clear,
		getProducts,
		getTotal
	};
}

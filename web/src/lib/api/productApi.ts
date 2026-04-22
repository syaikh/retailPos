import client from './client';
import type { Product, ProductPage } from '$lib/domain/entities';

export const productApi = {
	/**
	 * Search products with query
	 */
	search: (query: string, limit = 20, offset = 0) =>
		client.get<ProductPage>(`/products?search=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`),

	/**
	 * Get product by barcode
	 */
	findByBarcode: (barcode: string) =>
		client.get<Product | null>(`/products/barcode/${encodeURIComponent(barcode)}`),

	/**
	 * Get all products with optional pagination
	 */
	getAll: (limit = 50, offset = 0) =>
		client.get<ProductPage>(`/products?limit=${limit}&offset=${offset}`),

	/**
	 * Get single product by ID
	 */
	getById: (id: number) =>
		client.get<Product>(`/products/${id}`),

	/**
	 * Create new product
	 */
	create: (data: Partial<Product>) =>
		client.post<Product>('/products', data),

	/**
	 * Update product
	 */
	update: (id: number, data: Partial<Product>) =>
		client.patch<Product>(`/products/${id}`, data),

	/**
	 * Delete product
	 */
	delete: (id: number) =>
		client.delete(`/products/${id}`)
};

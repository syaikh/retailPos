import { get } from 'svelte/store';
import { productApi } from '$lib/api/productApi';
import { addItem, removeItem, updateQuantity as updateQty, setQuantity as setQty, calculateTotal, clearCart as clearCartItems } from '$lib/domain/services/CartService';
import type { CartItem, Product } from '$lib/domain/entities';
import { ui } from '$lib/stores/ui';
import { cart } from '$lib/stores/cart';

/**
 * Cart composable - provides cart operations with Svelte reactivity
 * Bridges domain CartService with store state
 */

export function useCart() {
	/**
	 * Add product to cart
	 */
	async function addToCart(product: Product) {
		try {
			const { data: freshProduct } = await productApi.findByBarcode(product.barcode || '');
			if (!freshProduct) {
				ui.error('Produk tidak ditemukan');
				return;
			}
			cart.update(items => addItem(items, freshProduct));
			ui.success('Produk ditambahkan ke keranjang');
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Gagal menambahkan produk';
			ui.error(message);
			throw err;
		}
	}

	/**
	 * Remove product from cart
	 */
	function removeFromCart(productId: number) {
		cart.update(items => removeItem(items, productId));
	}

	/**
	 * Increase quantity by 1
	 */
	async function incrementQuantity(productId: number) {
		try {
			cart.update(items => updateQty(items, productId, 1));
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Gagal mengupdate kuantitas';
			ui.error(message);
		}
	}

	/**
	 * Decrease quantity by 1
	 */
	async function decrementQuantity(productId: number) {
		try {
			cart.update(items => updateQty(items, productId, -1));
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Gagal mengupdate kuantitas';
			ui.error(message);
		}
	}

	/**
	 * Set exact quantity
	 */
	async function setQuantity(productId: number, quantity: number) {
		if (quantity <= 0) {
			removeFromCart(productId);
			return;
		}

		try {
			cart.update(items => setQty(items, productId, quantity));
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Gagal mengupdate kuantitas';
			ui.error(message);
			throw err;
		}
	}

	/**
	 * Clear cart
	 */
	function clearCart() {
		cart.set([]);
	}

	/**
	 * Get current cart items (read-only)
	 */
	function getCart(): CartItem[] {
		return get(cart);
	}

	/**
	 * Get cart total
	 */
	function getTotal(): number {
		return calculateTotal(get(cart));
	}

	return {
		cartItems: cart,
		addToCart,
		removeFromCart,
		incrementQuantity,
		decrementQuantity,
		setQuantity,
		clearCart,
		getCart,
		getTotal
	};
}

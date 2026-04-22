import { writable } from 'svelte/store';
import { saleApi } from '$lib/api/saleApi';
import { calculateTotal } from '$lib/domain/services/CartService';
import type { CartItem, SaleRequest } from '$lib/domain/entities';
import { ui } from '$lib/stores/ui';

/**
 * Checkout composable - handles sale/checkout flow with loading state
 */

export function useCheckout() {
	const isLoading = writable(false);

	/**
	 * Process checkout - creates sale and clears cart on success
	 */
	async function checkout(cartItems: CartItem[], paymentMethod: 'cash' | 'card'): Promise<boolean> {
		if (cartItems.length === 0) {
			ui.error('Keranjang kosong');
			return false;
		}

		const total = calculateTotal(cartItems);
		const saleData: SaleRequest = {
			total_amount: total,
			payment_method: paymentMethod,
			items: cartItems.map(item => ({
				product_id: item.id,
				quantity: item.quantity,
				price_at_sale: item.price
			}))
		};

		isLoading.set(true);
		ui.setLoading(true);

		try {
			const { data: sale } = await saleApi.checkout(saleData);
			ui.success(`Transaksi berhasil! Kode: ${sale.transaction_code}`);
			return true;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Transaksi gagal';
			ui.error(message);
			return false;
		} finally {
			isLoading.set(false);
			ui.setLoading(false);
		}
	}

	/**
	 * Process checkout and clear cart on success
	 */
	async function checkoutAndClear(cartItems: CartItem[], paymentMethod: 'cash' | 'card'): Promise<boolean> {
		const success = await checkout(cartItems, paymentMethod);
		if (success) {
			// Cart clearing is handled by the caller
		}
		return success;
	}

	return {
		isLoading,
		checkout,
		checkoutAndClear
	};
}

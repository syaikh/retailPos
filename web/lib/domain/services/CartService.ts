import type { CartItem, Product } from '../entities';

/**
 * Pure cart business logic - no Svelte dependencies
 * All functions are pure and return new arrays
 */

export function addItem(items: CartItem[], product: Product): CartItem[] {
	const existing = items.find(i => i.id === product.id);

	if (existing) {
		if (existing.quantity >= product.stock) {
			throw new Error('Stok tidak cukup');
		}
		return items.map(i =>
			i.id === product.id
				? { ...i, quantity: i.quantity + 1 }
				: i
		);
	}

	if (product.stock <= 0) {
		throw new Error('Stok habis');
	}

	return [...items, { ...product, quantity: 1 }];
}

export function removeItem(items: CartItem[], productId: number): CartItem[] {
	return items.filter(i => i.id !== productId);
}

export function updateQuantity(items: CartItem[], productId: number, delta: number): CartItem[] {
	return items.map(i => {
		if (i.id !== productId) return i;

		const newQty = i.quantity + delta;
		if (newQty <= 0) return i;
		if (newQty > i.stock) {
			throw new Error(`Stok tidak cukup. Maksimal: ${i.stock}`);
		}
		return { ...i, quantity: newQty };
	});
}

export function setQuantity(items: CartItem[], productId: number, quantity: number): CartItem[] {
	if (quantity <= 0) {
		throw new Error('Quantity must be positive');
	}

	return items.map(i => {
		if (i.id !== productId) return i;
		if (quantity > i.stock) {
			throw new Error(`Stok tidak cukup. Maksimal: ${i.stock}`);
		}
		return { ...i, quantity };
	});
}

export function calculateTotal(items: CartItem[]): number {
	return items.reduce((sum, item) => sum + (item.price * item.quantity), 0);
}

export function clearCart(items: CartItem[]): CartItem[] {
	return [];
}

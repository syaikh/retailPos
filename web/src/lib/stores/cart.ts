import { writable } from 'svelte/store';
import type { CartItem } from '$lib/domain/entities';

export const cart = writable<CartItem[]>([]);

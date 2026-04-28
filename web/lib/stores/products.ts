import { writable } from 'svelte/store';
import type { Product } from '$lib/domain/entities';

export const products = writable<Product[]>([]);

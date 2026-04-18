import { writable } from 'svelte/store';

export const user = writable(null);
export const isAuthenticated = writable(false);

export const cart = writable([]);
export const products = writable([]);

export function logout() {
  user.set(null);
  isAuthenticated.set(false);
  cart.set([]);
}
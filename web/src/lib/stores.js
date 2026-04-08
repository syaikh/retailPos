import { writable } from 'svelte/store';

export const user = writable(JSON.parse(localStorage.getItem('user')) || null);
export const token = writable(localStorage.getItem('token') || null);

export const cart = writable([]);
export const products = writable([]);

export function logout() {
  user.set(null);
  token.set(null);
  localStorage.removeItem('user');
  localStorage.removeItem('token');
}

import { writable } from 'svelte/store';
import { browser } from '$app/environment';

function createPersistentStore(key, defaultValue) {
  const store = writable(defaultValue);

  if (browser) {
    const stored = localStorage.getItem(key);
    if (stored !== null) {
      try {
        store.set(JSON.parse(stored));
      } catch (e) {
        console.warn(`Failed to parse stored value for ${key}:`, e);
      }
    }

    store.subscribe(value => {
      if (browser) {
        if (value === null || value === undefined) {
          localStorage.removeItem(key);
        } else {
          localStorage.setItem(key, JSON.stringify(value));
        }
      }
    });
  }

  return store;
}

export const user = writable(null);
export const isAuthenticated = writable(false);

export const cart = writable([]);
export const products = writable([]);

export function logout() {
  user.set(null);
  isAuthenticated.set(false);
  cart.set([]);
}
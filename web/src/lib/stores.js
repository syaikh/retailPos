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
      if (value === null || value === undefined) {
        localStorage.removeItem(key);
      } else {
        localStorage.setItem(key, JSON.stringify(value));
      }
    });
  }

  return store;
}

export const user = createPersistentStore('user', null);
export const token = createPersistentStore('token', null);

export const cart = writable([]);
export const products = writable([]); // Initialize with empty array to prevent null errors

export function logout() {
  user.set(null);
  token.set(null);
}

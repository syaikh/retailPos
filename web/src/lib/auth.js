import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { user } from '$lib/stores.js';
import { writable } from 'svelte/store';

// Signals that the initial auth check has completed, preventing flash of wrong content.
export const authReady = writable(false);

export function protectRoute() {
    if (!browser) return;

    // In a hash-routed SPA, $page.url.pathname is always '/'
    // We must read the hash to determine the current "route".
    const hash = window.location.hash; // e.g. '#/login', '#/', ''
    const isLoginRoute = hash === '#/login' || hash === '#/login/';

    let currentUser;
    user.subscribe(val => currentUser = val)();

    if (!currentUser && !isLoginRoute) {
        goto('/login');
    } else if (currentUser && isLoginRoute) {
        goto('/');
    }

    authReady.set(true);
}

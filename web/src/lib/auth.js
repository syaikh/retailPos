import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { user, isAuthenticated } from '$lib/stores.js';
import { writable, get } from 'svelte/store';
import { checkAuth } from '$lib/api.js';

export const authReady = writable(false);

// Track if we've already processed to prevent loops
let isProcessing = false;

export async function protectRoute() {
    if (!browser || isProcessing) return;

    const hash = window.location.hash;
    const isLoginRoute = hash === '#/login' || hash === '#/login/';

    // Prevent multiple concurrent calls
    isProcessing = true;

    try {
        const isAuth = await checkAuth();

        // Only redirect if state actually changed or we're on wrong page
        if (!isAuth && !isLoginRoute) {
            goto('/login');
        } else if (isAuth && isLoginRoute) {
            goto('/');
        }
    } finally {
        // Always mark auth as ready, even if checkAuth fails
        authReady.set(true);
        isProcessing = false;
    }
}

export function getUser() {
    return get(user);
}

export function getIsAuthenticated() {
    return get(isAuthenticated);
}
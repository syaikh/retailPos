import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { writable, get } from 'svelte/store';
import { checkAuth } from '$lib/api/auth';

export const authReady = writable(false);

let isProcessing = false;

export async function protectRoute() {
    if (!browser || isProcessing) return;

    const hash = window.location.hash;
    const isLoginRoute = hash === '#/login' || hash === '#/login/';

    isProcessing = true;

    try {
        const isAuth = await checkAuth();

        if (!isAuth && !isLoginRoute) {
            goto('/login');
        } else if (isAuth && isLoginRoute) {
            goto('/');
        }
    } finally {
        authReady.set(true);
        isProcessing = false;
    }
}

export function getUser() {
    return get(authReady);
}
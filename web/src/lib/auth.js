import { browser } from '$app/environment';
import { goto } from '$app/navigation';
import { user } from '$lib/stores.js';

export function protectRoute(pathname) {
    if (!browser) return;

    const publicRoutes = ['/login'];
    const isPublic = publicRoutes.includes(pathname);
    
    let currentUser;
    user.subscribe(val => currentUser = val)();

    if (!currentUser && !isPublic) {
        goto('/login');
    } else if (currentUser && isPublic) {
        goto('/');
    }
}

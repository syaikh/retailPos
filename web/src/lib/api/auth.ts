import client from './client';
import { auth } from '$lib/stores/auth';
import { goto } from '$app/navigation';

export async function checkAuth() {
    try {
        const { data } = await client.get('/auth/validate');
        if (data.user) {
            auth.setUser(data.user);
            return true;
        }
        auth.reset();
        return false;
    } catch {
        auth.reset();
        return false;
    }
}

export async function logout() {
    try {
        await client.post('/logout');
    } catch {
        // Ignore errors
    }
    sessionStorage.removeItem('refresh_token');
    auth.reset();
    goto('/login');
}
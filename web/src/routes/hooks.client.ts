import { checkAuth } from '$lib/api/auth';
import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';

export async function handle({ event, resolve }) {
	const path = event.url.pathname;
	
	// Allow static assets and login
	if (path.startsWith('/_app') || path === '/login') {
		return resolve(event);
	}
	
	const isAuth = await checkAuth();
	const state = get(auth);
	
	// Must be authenticated
	if (!isAuth || !state.user) {
		return Response.redirect('/login', 303);
	}
	
	// Admin can access all routes
	if (state.user.role === 'admin') {
		return resolve(event);
	}
	
	// Cashier: restricted to /pos only
	if (state.user.role === 'cashier' && !path.startsWith('/pos')) {
		return Response.redirect('/pos', 303);
	}
	
	return resolve(event);
}
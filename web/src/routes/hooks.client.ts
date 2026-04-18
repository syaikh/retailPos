import { checkAuth } from '$lib/api/auth';
import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';

const ROLE_PERMISSIONS: Record<string, string[]> = {
	admin: ['/dashboard', '/inventory', '/reports', '/pos'],
	cashier: ['/pos']
};

export async function handle({ event, resolve }) {
	const path = event.url.pathname;
	
	if (path === '/login') {
		return resolve(event);
	}
	
	const isAuth = await checkAuth();
	const state = get(auth);
	
	if (!isAuth || !state.user) {
		return Response.redirect('/login', 303);
	}
	
	const allowedRoutes = ROLE_PERMISSIONS[state.user.role] || ['/pos'];
	const hasAccess = allowedRoutes.some(route => path.startsWith(route));
	
	if (!hasAccess) {
		return Response.redirect(allowedRoutes[0] || '/pos', 303);
	}
	
	return resolve(event);
}
import { checkAuth } from '$lib/api/auth';
import { get } from 'svelte/store';
import { auth } from '$lib/stores/auth';

const routePermissions: Record<string, string[]> = {
	'/': ['dashboard:read'],
	'/pos': ['pos:access'],
	'/inventory': ['inventory:read'],
	'/inventory/groups': ['inventory:group:read'],
	'/reports': ['reports:read'],
	'/admin/users': ['users:read'],
	'/admin/roles': ['users:roles:manage'],
};

function hasAllPermissions(userPerms: string[], required: string[]): boolean {
	return required.every(p => userPerms.includes(p));
}

function getRequiredPermissions(path: string): string[] | null {
	if (routePermissions[path]) {
		return routePermissions[path];
	}
	for (const route in routePermissions) {
		if (path.startsWith(route)) {
			return routePermissions[route];
		}
	}
	return null;
}

function getFallbackRedirect(userPerms: string[]): string {
	if (userPerms.includes('pos:access')) {
		return '/pos';
	}
	if (userPerms.includes('dashboard:read')) {
		return '/';
	}
	return '/pos';
}

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
	
	// Check permissions for this route
	const requiredPerms = getRequiredPermissions(path);
	const userPerms = state.user.permissions || [];
	
	if (requiredPerms && !hasAllPermissions(userPerms, requiredPerms)) {
		return Response.redirect(getFallbackRedirect(userPerms), 303);
	}
	
	return resolve(event);
}
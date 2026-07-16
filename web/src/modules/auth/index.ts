export { login, logout, restoreSession, refreshAccessToken, checkAuth, setupAxiosInterceptors, startProactiveRefresh, stopProactiveRefresh } from './services/auth-service';
export { useAuthStore } from './stores/auth-store.svelte';
export { getAuthToken } from './lib/session';
export type { User, AuthState, LoginResult } from './types';

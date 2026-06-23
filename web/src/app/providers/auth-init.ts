import { restoreSession, useAuthStore } from '$modules/auth';

export async function initAuth(): Promise<void> {
  const store = useAuthStore();
  const result = await restoreSession();
  if (result.success && result.user) {
    store.user = result.user;
    store.isAuthenticated = true;
  } else {
    store.isAuthenticated = false;
    store.user = null;
  }
  store.loading = false;
}

import { writable } from 'svelte/store';

function createAuthStore() {
  const { subscribe, set, update } = writable({
    isAuthenticated: false,
    user: null,
    loading: true
  });

  return {
    subscribe,
    setUser: (user) => update(() => ({ isAuthenticated: true, user, loading: false })),
    clearUser: () => update(() => ({ isAuthenticated: false, user: null, loading: false })),
    setLoading: (loading) => update(state => ({ ...state, loading }))
  };
}

export const auth = createAuthStore();

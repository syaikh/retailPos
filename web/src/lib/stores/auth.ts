import { writable } from 'svelte/store';
import type { User } from '$lib/types';

function createAuthStore() {
  const { subscribe, set, update } = writable({
    isAuthenticated: false,
    user: null as User | null,
    loading: true
  });

  // Initialize from sessionStorage on page load
  const token = sessionStorage.getItem('access_token');
  if (token) {
    // Jika ada token di storage, anggap sudah login
    // (Di aplikasi sungguhan, ini bisa panggil /api/validate)
    update(() => ({ isAuthenticated: true, user: null, loading: false }));
  } else {
    update(() => ({ isAuthenticated: false, user: null, loading: false }));
  }

  return {
    subscribe,
    setUser: (user: User) => update(() => ({ isAuthenticated: true, user, loading: false })),
    clearUser: () => {
      sessionStorage.removeItem('access_token');
      localStorage.removeItem('access_token');
      // Catatan: refresh_token dihapus oleh backend via cookie, atau kita biarkan expire
      update(() => ({ isAuthenticated: false, user: null, loading: false }));
    },
    setLoading: (loading: boolean) => update(state => ({ ...state, loading })),
    // Helper untuk dapatkan token (dipakai oleh api client)
    getToken: () => sessionStorage.getItem('access_token') || localStorage.getItem('access_token')
  };
}

export const auth = createAuthStore();

// Ekspor fungsi helper agar bisa dipakai di luar store (misal: axios client)
export function getAuthToken(): string | null {
  if (typeof window !== 'undefined') {
    return sessionStorage.getItem('access_token');
  }
  return null;
}

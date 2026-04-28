import { writable, get } from 'svelte/store';

export interface User {
  id: number;
  username: string;
  email: string;
  password?: string;
  role_id: number;
  role: string;
  store_id?: number;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  loading: boolean;
  onlineUsers: number;
}

function createAuthStore() {
  const { subscribe, set, update } = writable<AuthState>({
    user: null,
    isAuthenticated: false,
    loading: true,
    onlineUsers: 0
  });

  return {
    subscribe,
    setUser: (user: User | null) => update(s => ({ ...s, user, isAuthenticated: !!user, loading: false })),
    setLoading: (loading: boolean) => update(s => ({ ...s, loading })),
    setOnlineUsers: (count: number) => update(s => ({ ...s, onlineUsers: count })),
    reset: () => set({ user: null, isAuthenticated: false, loading: false, onlineUsers: 0 }),
    getUser: () => get({ subscribe }).user,
    getOnlineUsers: () => get({ subscribe }).onlineUsers
  };
}

export const auth = createAuthStore();

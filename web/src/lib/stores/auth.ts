import { writable, get } from 'svelte/store';

export interface User {
    id: number;
    username: string;
    role: string;
}

interface AuthState {
    user: User | null;
    isAuthenticated: boolean;
    loading: boolean;
}

function createAuthStore() {
    const { subscribe, set, update } = writable<AuthState>({
        user: null,
        isAuthenticated: false,
        loading: true
    });

    return {
        subscribe,
        setUser: (user: User | null) => update(s => ({ ...s, user, isAuthenticated: !!user, loading: false })),
        setLoading: (loading: boolean) => update(s => ({ ...s, loading })),
        reset: () => set({ user: null, isAuthenticated: false, loading: false }),
        getUser: () => get({ subscribe }).user
    };
}

export const auth = createAuthStore();
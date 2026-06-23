import type { User } from '../types';

let user = $state<User | null>(null);
let isAuthenticated = $state(false);
let loading = $state(true);

let initialized = false;

export function useAuthStore() {
  if (!initialized) {
    const token = sessionStorage.getItem('access_token');
    if (token) {
      isAuthenticated = true;
    }
    loading = false;
    initialized = true;
  }

  return {
    get user() { return user; },
    set user(value: User | null) { user = value; },
    get isAuthenticated() { return isAuthenticated; },
    set isAuthenticated(value: boolean) { isAuthenticated = value; },
    get loading() { return loading; },
    set loading(value: boolean) { loading = value; },
    setUser(u: User) {
      user = u;
      isAuthenticated = true;
      loading = false;
    },
    clearUser() {
      sessionStorage.removeItem('access_token');
      user = null;
      isAuthenticated = false;
      loading = false;
    },
    getToken: (): string | null => sessionStorage.getItem('access_token'),
  };
}

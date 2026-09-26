import type { User } from "../types";

let user = $state<User | null>(null);
let isAuthenticated = $state(false);
let loading = $state(true);
// True while the account still owes a first-login password rotation: every API
// call outside the allowlist returns 428 until it happens, so the app renders
// a blocking change-password modal instead of a page.
let mustChangePassword = $state(false);

let initialized = false;

export function useAuthStore() {
  if (!initialized) {
    const token = sessionStorage.getItem("access_token");
    if (token) {
      isAuthenticated = true;
    }
    loading = false;
    initialized = true;
  }

  return {
    get user() {
      return user;
    },
    set user(value: User | null) {
      user = value;
      mustChangePassword = !!value?.must_change_password;
    },
    get isAuthenticated() {
      return isAuthenticated;
    },
    set isAuthenticated(value: boolean) {
      isAuthenticated = value;
    },
    get loading() {
      return loading;
    },
    set loading(value: boolean) {
      loading = value;
    },
    get mustChangePassword() {
      return mustChangePassword;
    },
    set mustChangePassword(value: boolean) {
      mustChangePassword = value;
    },
    setUser(u: User) {
      user = u;
      mustChangePassword = !!u.must_change_password;
      isAuthenticated = true;
      loading = false;
    },
    clearUser() {
      sessionStorage.removeItem("access_token");
      user = null;
      isAuthenticated = false;
      loading = false;
      mustChangePassword = false;
    },
    getToken: (): string | null => sessionStorage.getItem("access_token"),
  };
}

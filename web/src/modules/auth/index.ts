export {
  login,
  logout,
  restoreSession,
  refreshAccessToken,
  checkAuth,
  setupAxiosInterceptors,
  startProactiveRefresh,
  stopProactiveRefresh,
  updatePreferences,
  handleCrossTabLogout,
  changePassword,
  isPasswordChangeRequired,
  markPasswordChangeRequired,
  PASSWORD_CHANGE_REQUIRED,
} from "./services/auth-service";
export { useAuthStore } from "./stores/auth-store.svelte";
export { getAuthToken } from "./lib/session";
export { default as PasswordChangeForm } from "./components/PasswordChangeForm.svelte";
export type { User, AuthState, LoginResult } from "./types";

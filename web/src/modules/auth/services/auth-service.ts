import axios from "axios";
import { useAuthStore } from "$shared/stores/auth.svelte";
import { setAccessToken } from "$shared/api/session";
import {
  authApi,
  refreshAccessToken,
  getAuthHeaders,
  stopProactiveRefresh,
} from "$shared/auth/refresh-kernel";
import type { LoginResult, User } from "../types";
import { applyTheme } from "$shared/utils/theme";
import { labels, setLocale } from "$shared/i18n";
import { onCrossTabLogout } from "$shared/utils/tab-coordination";

// The session/refresh machinery (token refresh, proactive refresh, axios
// interceptors, logout, 428 gate) lives in $shared/auth/refresh-kernel.ts so
// $shared/api/http-client can use it without importing this module. It is
// re-exported below to keep this module's public API stable — one source
// instance, verified by kernel-identity.test.ts.
export {
  refreshAccessToken,
  startProactiveRefresh,
  stopProactiveRefresh,
  PASSWORD_CHANGE_REQUIRED,
  isPasswordChangeRequired,
  markPasswordChangeRequired,
  setupAxiosInterceptors,
  logout,
} from "$shared/auth/refresh-kernel";

function decodeTokenPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = parts[1];
    const decoded = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(decoded);
  } catch {
    return null;
  }
}

export async function checkAuth(): Promise<boolean> {
  try {
    await authApi.post("/validate", {}, { headers: getAuthHeaders() });
    return true;
  } catch (err: unknown) {
    if (axios.isAxiosError(err) && err.response?.status === 401) {
      const newToken = await refreshAccessToken();
      if (!newToken) return false;
      try {
        await authApi.post(
          "/validate",
          {},
          { headers: { Authorization: `Bearer ${newToken}` } },
        );
        return true;
      } catch {
        return false;
      }
    }
    return false;
  }
}

export async function restoreSession(): Promise<{
  success: boolean;
  user?: User;
}> {
  let accessToken: string | null;
  try {
    accessToken = sessionStorage.getItem("access_token");
    if (!accessToken) {
      return { success: false };
    }
  } catch {
    return { success: false };
  }

  const getHeaders = () => ({ Authorization: `Bearer ${accessToken}` });

  try {
    const response = await authApi.post(
      "/validate",
      {},
      { headers: getHeaders() },
    );
    if (response.data.user) {
      const user = response.data.user as User;
      if (response.data.permissions) {
        user.permissions = response.data.permissions;
      }
      return { success: true, user };
    }
    return { success: false };
  } catch (err) {
    if (axios.isAxiosError(err) && err.response?.status === 401) {
      const newToken = await refreshAccessToken();
      if (!newToken) return { success: false };
      try {
        const retry = await authApi.post(
          "/validate",
          {},
          { headers: { Authorization: `Bearer ${newToken}` } },
        );
        if (retry.data.user) {
          const user = retry.data.user as User;
          if (retry.data.permissions) {
            user.permissions = retry.data.permissions;
          }
          return { success: true, user };
        }
      } catch {
        // fallthrough
      }
    }
    return { success: false };
  }
}

export async function login(
  username: string,
  password: string,
): Promise<LoginResult> {
  try {
    const response = await authApi.post("/login", { username, password });
    const data = response.data;
    if (data.access_token) {
      sessionStorage.setItem("access_token", data.access_token);
      const claims = decodeTokenPayload(data.access_token);
      if (
        claims?.permissions &&
        Array.isArray(claims.permissions) &&
        data.user
      ) {
        data.user.permissions = claims.permissions as string[];
      }
    }
    if (data.user) {
      if (data.user.theme) applyTheme(data.user.theme);
      if (data.user.language) setLocale(data.user.language);
    }
    return { ok: true, data };
  } catch (err) {
    // 429 means the request never reached credential checks, so showing
    // "invalid credentials" would send a cashier re-typing a correct password.
    if (axios.isAxiosError(err) && err.response?.status === 429) {
      return {
        ok: false,
        reason: "rate_limited",
        retryAfter: parseRetryAfter(err.response.headers?.["retry-after"]),
      };
    }
    return { ok: false, reason: "invalid" };
  }
}

// parseRetryAfter normalizes the Retry-After header (seconds as a string on
// this backend) into a positive integer, falling back to 60 — the value the
// login limiter sends — when absent or malformed.
function parseRetryAfter(raw: unknown): number {
  const n = typeof raw === "string" ? Number.parseInt(raw, 10) : NaN;
  return Number.isFinite(n) && n > 0 ? n : 60;
}

// Completes a first-login (or any) password rotation. On success the backend
// re-issues the access token with the must_change_password claim cleared, so
// the session continues without a re-login.
export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<{ ok: true } | { ok: false; message: string }> {
  const post = (token: string | null) =>
    authApi.post(
      "/change-password",
      { current_password: currentPassword, new_password: newPassword },
      {
        headers: token
          ? { Authorization: `Bearer ${token}` }
          : getAuthHeaders(),
      },
    );

  try {
    let res;
    try {
      res = await post(null);
    } catch (err) {
      // authApi carries no 401 interceptor, and the access token can expire
      // while the forced-rotation modal waits for input. Refresh once and
      // retry so the user is not stranded behind a raw 401 with the only way
      // out being "sign out and start over".
      if (!axios.isAxiosError(err) || err.response?.status !== 401) {
        throw err;
      }
      const refreshed = await refreshAccessToken();
      if (!refreshed) {
        return { ok: false, message: labels.toastSessionExpired };
      }
      res = await post(refreshed);
    }
    const token = res.data?.access_token;
    if (token) setAccessToken(token);
    const store = useAuthStore();
    if (store.user) {
      store.user = { ...store.user, must_change_password: false };
    }
    store.mustChangePassword = false;
    return { ok: true };
  } catch (err) {
    if (axios.isAxiosError(err)) {
      const body = err.response?.data as
        { error?: { message?: string } | string } | undefined;
      const raw = body?.error;
      const message =
        typeof raw === "string"
          ? raw
          : raw?.message || err.message || "Failed to change password";
      return { ok: false, message };
    }
    return { ok: false, message: "Failed to change password" };
  }
}

export function handleCrossTabLogout(): void {
  onCrossTabLogout(() => {
    stopProactiveRefresh();
    const store = useAuthStore();
    store.clearUser();
    window.location.replace("/login");
  });
}

export async function updatePreferences(
  language: string,
  theme: string,
): Promise<boolean> {
  try {
    const res = await authApi.put(
      "/users/me/preferences",
      { language, theme },
      { headers: getAuthHeaders() },
    );
    if (res.status === 200) {
      const store = useAuthStore();
      if (store.user) {
        store.user = { ...store.user, language, theme };
      }
      applyTheme(theme as "light" | "dark");
      setLocale(language as "id" | "en");
      return true;
    }
  } catch {
    /* ignore */
  }
  return false;
}

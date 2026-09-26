import axios from "axios";
import type { AxiosInstance, AxiosError, AxiosRequestConfig } from "axios";
import { useAuthStore } from "../stores/auth-store.svelte";
import { setAccessToken, getAuthToken } from "../lib/session";
import type { User } from "../types";
import { applyTheme } from "$shared/utils/theme";
import { labels, setLocale } from "$shared/i18n";
import {
  initTabCoordination,
  destroyTabCoordination,
  isTabLeader,
  requestRefresh,
  onLeaderChange,
  onCrossTabLogout,
  broadcastLogout,
} from "$shared/utils/tab-coordination";

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

const authApi = axios.create({
  baseURL: "/api",
  withCredentials: true,
});

// --- Shared refresh lock (deduplication queue) ---
let refreshPromise: Promise<string | null> | null = null;
const refreshQueue: Array<{
  resolve: (token: string | null) => void;
  reject: (err: unknown) => void;
}> = [];

async function doRefresh(): Promise<string | null> {
  // Cross-tab: if follower, ask leader to refresh and wait for result
  if (!isTabLeader()) {
    try {
      const token = await requestRefresh();
      if (token) return token;
    } catch {
      // Leader refresh failed — fall through to direct refresh as fallback
    }
  }

  // Leader or fallback: refresh directly
  if (refreshPromise) {
    return new Promise<string | null>((resolve, reject) => {
      refreshQueue.push({ resolve, reject });
    });
  }

  refreshPromise = (async () => {
    try {
      const response = await authApi.post("/refresh");
      const newAccessToken = response.data.access_token;
      setAccessToken(newAccessToken);
      return newAccessToken;
    } catch (err) {
      // Notify all queued callers of the failure
      const queue = refreshQueue.splice(0);
      queue.forEach((cb) => cb.reject(err));
      refreshPromise = null;
      return null;
    }
  })();

  try {
    const result = await refreshPromise;
    // Notify all queued callers with the new token
    const queue = refreshQueue.splice(0);
    queue.forEach((cb) => cb.resolve(result));
    return result;
  } finally {
    refreshPromise = null;
  }
}

export async function refreshAccessToken(): Promise<string | null> {
  return doRefresh();
}

// --- Proactive token refresh ---
let proactiveRefreshTimer: ReturnType<typeof setInterval> | null = null;
const PROACTIVE_REFRESH_INTERVAL = 13 * 60 * 1000;

function startProactiveTimer() {
  stopProactiveTimer();
  proactiveRefreshTimer = setInterval(async () => {
    const token = getAuthToken();
    if (!token) {
      stopProactiveTimer();
      return;
    }
    await doRefresh();
  }, PROACTIVE_REFRESH_INTERVAL);
}

function stopProactiveTimer() {
  if (proactiveRefreshTimer) {
    clearInterval(proactiveRefreshTimer);
    proactiveRefreshTimer = null;
  }
}

export function startProactiveRefresh() {
  initTabCoordination();

  // Start timer only if this tab is the leader
  if (isTabLeader()) {
    startProactiveTimer();
  }

  // React to leadership changes
  onLeaderChange((leader) => {
    if (leader) {
      startProactiveTimer();
    } else {
      stopProactiveTimer();
    }
  });
}

export function stopProactiveRefresh() {
  stopProactiveTimer();
  destroyTabCoordination();
}
// The backend answers 428 Precondition Required with this code while a valid
// session still owes a first-login password rotation.
export const PASSWORD_CHANGE_REQUIRED = "PASSWORD_CHANGE_REQUIRED";

export function isPasswordChangeRequired(err: unknown): boolean {
  if (!axios.isAxiosError(err)) return false;
  if (err.response?.status !== 428) return false;
  const code = (err.response.data as { error?: { code?: string } } | undefined)
    ?.error?.code;
  return code === PASSWORD_CHANGE_REQUIRED;
}

export function markPasswordChangeRequired(): void {
  useAuthStore().mustChangePassword = true;
}

export function setupAxiosInterceptors(apiClient: AxiosInstance) {
  let failedQueue: Array<{
    resolve: (token: string) => void;
    reject: (err: unknown) => void;
  }> = [];

  const processQueue = (error: unknown, token: string | null = null) => {
    failedQueue.forEach((prom) => {
      if (error) {
        prom.reject(error);
      } else {
        prom.resolve(token as string);
      }
    });
    failedQueue = [];
  };

  apiClient.interceptors.response.use(
    (response) => response,
    async (error: AxiosError) => {
      // 428: the token is valid but the account still owes a first-login
      // password rotation. Surface it as a blocking modal instead of retrying
      // or logging the user out.
      if (isPasswordChangeRequired(error)) {
        markPasswordChangeRequired();
        return Promise.reject(error);
      }

      const originalRequest = error.config as AxiosRequestConfig & {
        _retry?: boolean;
      };

      if (error.response?.status === 401 && !originalRequest._retry) {
        if (refreshPromise) {
          return new Promise<string>((resolve, reject) => {
            failedQueue.push({ resolve, reject });
          })
            .then((token: string) => {
              originalRequest.headers = originalRequest.headers || {};
              originalRequest.headers["Authorization"] = "Bearer " + token;
              return apiClient(originalRequest);
            })
            .catch((err) => Promise.reject(err));
        }

        originalRequest._retry = true;

        try {
          const newToken = await doRefresh();
          if (!newToken) {
            processQueue(new Error("Refresh failed"), null);
            logout();
            return Promise.reject(new Error("Refresh failed"));
          }

          processQueue(null, newToken);

          originalRequest.headers = originalRequest.headers || {};
          originalRequest.headers["Authorization"] = "Bearer " + newToken;
          return apiClient(originalRequest);
        } catch (refreshError) {
          processQueue(refreshError, null);
          logout();
          return Promise.reject(refreshError);
        }
      }

      return Promise.reject(error);
    },
  );
}

function getAuthHeaders(): Record<string, string> {
  const accessToken = sessionStorage.getItem("access_token");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function checkAuth(): Promise<boolean> {
  try {
    await authApi.post("/validate", {}, { headers: getAuthHeaders() });
    return true;
  } catch (err: unknown) {
    if (axios.isAxiosError(err) && err.response?.status === 401) {
      const newToken = await doRefresh();
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
      const newToken = await doRefresh();
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
): Promise<
  { access_token: string; refresh_token: string; user: User } | false
> {
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
    return data;
  } catch {
    return false;
  }
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
      const refreshed = await doRefresh();
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

export async function logout(): Promise<void> {
  stopProactiveRefresh();
  broadcastLogout();
  try {
    await authApi.post("/logout", {}, { headers: getAuthHeaders() });
  } catch (_err) {
    // Ignore errors on logout
  }
  const store = useAuthStore();
  store.clearUser();
  window.location.replace("/login");
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

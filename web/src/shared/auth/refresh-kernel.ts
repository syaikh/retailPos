// Session kernel: token refresh, proactive refresh, axios 401 interception,
// logout and the 428 password-rotation gate.
//
// This lives in $shared because $shared/api/http-client depends on it — the
// import-boundary rule forbids shared code from importing $modules. It is the
// session/refresh machinery only; login, session restore and password change
// flows remain in modules/auth/services/auth-service.ts.
//
// Single-instance invariant: refreshPromise/refreshQueue live here and ONLY
// here. modules/auth re-exports these functions, so importing via
// $modules/auth or $shared/auth/refresh-kernel resolves to this one module —
// asserted by the identity test in
// modules/auth/services/__tests__/kernel-identity.test.ts.
import axios from "axios";
import type { AxiosInstance, AxiosError, AxiosRequestConfig } from "axios";
import { useAuthStore } from "$shared/stores/auth.svelte";
import { setAccessToken, getAuthToken } from "$shared/api/session";
import {
  initTabCoordination,
  destroyTabCoordination,
  isTabLeader,
  requestRefresh,
  onLeaderChange,
  broadcastLogout,
} from "$shared/utils/tab-coordination";

export const authApi = axios.create({
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

export function getAuthHeaders(): Record<string, string> {
  const accessToken = sessionStorage.getItem("access_token");
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
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

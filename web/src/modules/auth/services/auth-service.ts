import axios from 'axios';
import type { AxiosInstance, AxiosError, AxiosRequestConfig } from 'axios';
import { useAuthStore } from '../stores/auth-store.svelte';
import { setAccessToken, removeAccessToken, getAuthToken } from '../lib/session';
import type { User } from '../types';
import { applyTheme } from '$shared/utils/theme';
import { setLocale } from '$shared/i18n';
import {
  initTabCoordination,
  destroyTabCoordination,
  isTabLeader,
  requestRefresh,
  onLeaderChange,
  onCrossTabLogout,
  broadcastLogout,
} from '$shared/utils/tab-coordination';

function decodeTokenPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = parts[1];
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(decoded);
  } catch {
    return null;
  }
}

const authApi = axios.create({
  baseURL: '/api',
  withCredentials: true,
});

// --- Shared refresh lock (deduplication queue) ---
let refreshPromise: Promise<string | null> | null = null;
let refreshQueue: Array<{ resolve: (token: string | null) => void; reject: (err: unknown) => void }> = [];

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
      const response = await authApi.post('/refresh');
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
      const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean };

      if (error.response?.status === 401 && !originalRequest._retry) {
        if (refreshPromise) {
          return new Promise<string>((resolve, reject) => {
            failedQueue.push({ resolve, reject });
          })
            .then((token: string) => {
              originalRequest.headers = originalRequest.headers || {};
              originalRequest.headers['Authorization'] = 'Bearer ' + token;
              return apiClient(originalRequest);
            })
            .catch((err) => Promise.reject(err));
        }

        originalRequest._retry = true;

        try {
          const newToken = await doRefresh();
          if (!newToken) {
            processQueue(new Error('Refresh failed'), null);
            logout();
            return Promise.reject(new Error('Refresh failed'));
          }

          processQueue(null, newToken);

          originalRequest.headers = originalRequest.headers || {};
          originalRequest.headers['Authorization'] = 'Bearer ' + newToken;
          return apiClient(originalRequest);
        } catch (refreshError) {
          processQueue(refreshError, null);
          logout();
          return Promise.reject(refreshError);
        }
      }

      return Promise.reject(error);
    }
  );
}

function getAuthHeaders(): Record<string, string> {
  const accessToken = sessionStorage.getItem('access_token');
  return accessToken ? { Authorization: `Bearer ${accessToken}` } : {};
}

export async function checkAuth(): Promise<boolean> {
  try {
    await authApi.post('/validate', {}, { headers: getAuthHeaders() });
    return true;
  } catch (err: unknown) {
    if (axios.isAxiosError(err) && err.response?.status === 401) {
      const newToken = await doRefresh();
      if (!newToken) return false;
      try {
        await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${newToken}` } });
        return true;
      } catch {
        return false;
      }
    }
    return false;
  }
}

export async function restoreSession(): Promise<{ success: boolean; user?: User }> {
  let accessToken: string | null;
  try {
    accessToken = sessionStorage.getItem('access_token');
    if (!accessToken) {
      return { success: false };
    }
  } catch {
    return { success: false };
  }

  const getHeaders = () => ({ Authorization: `Bearer ${accessToken}` });

  try {
    const response = await authApi.post('/validate', {}, { headers: getHeaders() });
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
        const retry = await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${newToken}` } });
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

export async function login(username: string, password: string): Promise<{ access_token: string; refresh_token: string; user: User } | false> {
  try {
    const response = await authApi.post('/login', { username, password });
    const data = response.data;
    if (data.access_token) {
      sessionStorage.setItem('access_token', data.access_token);
      const claims = decodeTokenPayload(data.access_token);
      if (claims?.permissions && Array.isArray(claims.permissions) && data.user) {
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

export async function logout(): Promise<void> {
  stopProactiveRefresh();
  broadcastLogout();
  try {
    await authApi.post('/logout', {}, { headers: getAuthHeaders() });
  } catch (err) {
    // Ignore errors on logout
  }
  const store = useAuthStore();
  store.clearUser();
  window.location.replace('/login');
}

export function handleCrossTabLogout(): void {
  onCrossTabLogout(() => {
    stopProactiveRefresh();
    const store = useAuthStore();
    store.clearUser();
    window.location.replace('/login');
  });
}

export async function updatePreferences(language: string, theme: string): Promise<boolean> {
  try {
    const res = await authApi.put('/users/me/preferences', { language, theme }, { headers: getAuthHeaders() });
    if (res.status === 200) {
      const store = useAuthStore();
      if (store.user) {
        store.user = { ...store.user, language, theme };
      }
      applyTheme(theme as 'light' | 'dark');
      setLocale(language as 'id' | 'en');
      return true;
    }
  } catch { /* ignore */ }
  return false;
}

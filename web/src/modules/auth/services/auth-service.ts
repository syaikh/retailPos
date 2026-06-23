import axios from 'axios';
import { useAuthStore } from '../stores/auth-store.svelte';
import { setAccessToken, removeAccessToken, getAuthToken } from '../lib/session';
import type { User } from '../types';

const authApi = axios.create({
  baseURL: '/api',
  withCredentials: true,
});

export async function refreshAccessToken(): Promise<string | null> {
  try {
    const response = await authApi.post('/refresh');
    const newAccessToken = response.data.access_token;
    setAccessToken(newAccessToken);
    return newAccessToken;
  } catch (err) {
    logout();
    return null;
  }
}

export async function refreshTokenSilently(): Promise<string | null> {
  try {
    const response = await authApi.post('/refresh');
    const newAccessToken = response.data.access_token;
    setAccessToken(newAccessToken);
    return newAccessToken;
  } catch (err) {
    return null;
  }
}

export function setupAxiosInterceptors(apiClient: axios.AxiosInstance) {
  let isRefreshing = false;
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
    async (error: axios.AxiosError) => {
      const originalRequest = error.config as axios.AxiosRequestConfig & { _retry?: boolean };

      if (error.response?.status === 401 && !originalRequest._retry) {
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
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
        isRefreshing = true;

        try {
          const newToken = await refreshAccessToken();
          if (!newToken) throw new Error('Refresh failed');

          processQueue(null, newToken);

          originalRequest.headers = originalRequest.headers || {};
          originalRequest.headers['Authorization'] = 'Bearer ' + newToken;
          return apiClient(originalRequest);
        } catch (refreshError) {
          processQueue(refreshError, null);
          return Promise.reject(refreshError);
        } finally {
          isRefreshing = false;
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
    const response = await authApi.post('/validate', {}, { headers: getAuthHeaders() });
    return response.status === 200;
  } catch (err: unknown) {
    if (axios.isAxiosError(err) && err.response?.status === 401) {
      const newToken = await refreshAccessToken();
      if (!newToken) {
        return false;
      }
      try {
        const response = await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${newToken}` } });
        return response.status === 200;
      } catch (_err) {
        return false;
      }
    }
    return false;
  }
}

export async function restoreSession(): Promise<{ success: boolean; user?: User }> {
  try {
    const accessToken = sessionStorage.getItem('access_token');
    if (!accessToken) {
      return { success: false };
    }

    try {
      const response = await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${accessToken}` } });
      if (response.status === 200 && response.data.user) {
        const user = response.data.user as User;
        if (response.data.permissions) {
          user.permissions = response.data.permissions;
        }
        return { success: true, user };
      }
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 401) {
        const newToken = await refreshTokenSilently();
        if (!newToken) {
          return { success: false };
        }
        const response = await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${newToken}` } });
        if (response.status === 200 && response.data.user) {
          const user = response.data.user as User;
          if (response.data.permissions) {
            user.permissions = response.data.permissions;
          }
          return { success: true, user };
        }
      }
    }

    return { success: false };
  } catch (err) {
    return { success: false };
  }
}

export async function login(username: string, password: string): Promise<{ access_token: string; refresh_token: string; user: User } | false> {
  try {
    const response = await authApi.post('/login', { username, password });

    if (response.status === 200) {
      const data = response.data;
      if (data.access_token) {
        sessionStorage.setItem('access_token', data.access_token);
      }
      return data;
    } else {
      return false;
    }
  } catch (err) {
    return false;
  }
}

export async function logout(): Promise<void> {
  try {
    await authApi.post('/logout', {}, { headers: getAuthHeaders() });
  } catch (err) {
    // Ignore errors on logout
  }
  const store = useAuthStore();
  store.clearUser();
}

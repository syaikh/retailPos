// Auth API for checking and handling authentication
import axios from 'axios';
import { auth } from '$lib/stores/auth';

// Buat instance Axios terpisah untuk request Auth (agar tidak terjadi loop)
const authApi = axios.create({
  baseURL: '/api',
  withCredentials: true,
});

// Fungsi untuk merefresh token
export async function refreshAccessToken(): Promise<string | null> {
  try {
    // Request ke /api/refresh, cookie HttpOnly otomatis terkirim
    const response = await authApi.post('/refresh');
    const newAccessToken = response.data.access_token;

    // Simpan ke sessionStorage (safe dari XSS, refresh token ada di HttpOnly cookie)
    sessionStorage.setItem('access_token', newAccessToken);
    return newAccessToken;
  } catch (err) {
    // Jika gagal refresh, logout user
    logout();
    return null;
  }
}

// Function to refresh token without automatic logout (for session restoration)
export async function refreshTokenSilently(): Promise<string | null> {
  try {
    const response = await authApi.post('/refresh');
    const newAccessToken = response.data.access_token;
    sessionStorage.setItem('access_token', newAccessToken);
    return newAccessToken;
  } catch (err) {
    return null;
  }
}

// Fungsi untuk mengatur Interceptor pada API Client utama
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

      // Jika error 401 dan belum mencoba refresh
      if (error.response?.status === 401 && !originalRequest._retry) {
        if (isRefreshing) {
          // Jika sedang refresh, masukkan request ke antrean
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

          // Set header baru dan ulangi request awal
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
  // Use sessionStorage for access token to reduce XSS risk
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

// Fungsi khusus untuk restore session saat refresh halaman
export async function restoreSession(): Promise<{ success: boolean; user?: User }> {
  try {
    const accessToken = sessionStorage.getItem('access_token');
    if (!accessToken) {
      return { success: false };
    }

    // First try to validate the existing token
    try {
      const response = await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${accessToken}` } });
      if (response.status === 200 && response.data.user) {
        return { success: true, user: response.data.user };
      }
    } catch (err) {
      // If validation fails with 401, try to refresh the token
      if (axios.isAxiosError(err) && err.response?.status === 401) {
        const newToken = await refreshTokenSilently();
        if (!newToken) {
          return { success: false };
        }
        // Validate the new token
        const response = await authApi.post('/validate', {}, { headers: { Authorization: `Bearer ${newToken}` } });
        if (response.status === 200 && response.data.user) {
          return { success: true, user: response.data.user };
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
      // Store access token to sessionStorage (refresh token is HttpOnly cookie)
      if (data.access_token) {
        sessionStorage.setItem('access_token', data.access_token);
      }
      if (data.refresh_token) {
        // Sudah otomatis di cookie via backend, tapi kita bisa simpan di state jika perlu
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
  auth.clearUser();
}

// Web API Client (Axios)
import axios from 'axios';
import { getAuthToken } from '$lib/stores/auth';

// 1. Buat instance Axios untuk aplikasi
const apiClient = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

// 2. Tambahkan Request Interceptor untuk menyuntikkan token
apiClient.interceptors.request.use(
  (config) => {
    const token = getAuthToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 3. Setup Response Interceptor untuk menangani Auto-Refresh 401
// (Fungsi ini menggunakan instance axios khusus untuk menghindari loop)
// This will be called from main.js to avoid the static import warning
export const setupApiInterceptors = async () => {
  const { setupAxiosInterceptors } = await import('./auth');
  setupAxiosInterceptors(apiClient);
};

export default apiClient;

// 4. (Opsional) Helper khusus untuk GET biasa jika tidak mau pakai async/await di store
export const apiFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {
  // Setup interceptors on first API call
  await setupApiInterceptors();

  const token = getAuthToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  // Handle 401 - try refresh and retry once
  if (response.status === 401) {
    const { refreshAccessToken, logout } = await import('./auth');
    const newToken = await refreshAccessToken();
    if (newToken) {
      headers['Authorization'] = `Bearer ${newToken}`;
      return fetch(url, {
        ...options,
        headers,
      });
    } else {
      logout();
    }
  }

  return response;
};

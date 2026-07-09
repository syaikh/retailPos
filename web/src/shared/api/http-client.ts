// Web API Client (Axios)
import axios from 'axios';
import { getAuthToken, refreshAccessToken, logout, setupAxiosInterceptors } from '$modules/auth';

// 1. Buat instance Axios untuk aplikasi
const apiClient = axios.create({
  baseURL: '/api',
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
setupAxiosInterceptors(apiClient);

export default apiClient;

// 4. (Opsional) Helper khusus untuk GET biasa jika tidak mau pakai async/await di store
export const apiFetch = async (url: string, options: RequestInit = {}): Promise<Response> => {

  const token = getAuthToken();
  const isFormData = options.body instanceof FormData;
  const headers: Record<string, string> = {
    ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
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

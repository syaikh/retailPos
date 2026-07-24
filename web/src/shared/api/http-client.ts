// Web API Client (Axios)
import axios from 'axios';
import { getAuthToken, refreshAccessToken, setupAxiosInterceptors } from '$modules/auth';
import { getCached, setCache, invalidateCache } from './cache';

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

// 4. Cache GET responses and invalidate on mutations
function mutationCachePrefix(url: string): string {
  const path = url.split('?')[0];
  const segments = path.split('/').filter(Boolean);
  if (segments.length > 1 && /^\d+$/.test(segments[segments.length - 1])) {
    segments.pop();
  }
  return '/' + segments.join('/');
}

apiClient.interceptors.response.use(
  (response) => {
    if (response.config.method === 'get') {
      setCache(response.config.url!, response.data);
    }
    if (['post', 'put', 'patch', 'delete'].includes(response.config.method!)) {
      invalidateCache(mutationCachePrefix(response.config.url!));
    }
    return response;
  },
  (error) => Promise.reject(error)
);

export default apiClient;

// 5. GET helper with cache-first strategy
export const cachedGet = async <T>(url: string): Promise<T> => {
  const cached = getCached<T>(url);
  if (cached !== null) {
    return cached;
  }
  const response = await apiClient.get<T>(url);
  return response.data;
};

// 6. (Opsional) Helper khusus untuk GET biasa jika tidak mau pakai async/await di store
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
    }
    throw new Error('Session expired');
  }

  return response;
};

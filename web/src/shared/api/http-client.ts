// Web API Client (Axios)
import axios from "axios";
import {
  getAuthToken,
  refreshAccessToken,
  setupAxiosInterceptors,
  logout,
  markPasswordChangeRequired,
  PASSWORD_CHANGE_REQUIRED,
} from "$modules/auth";

// 1. Buat instance Axios untuk aplikasi
const apiClient = axios.create({
  baseURL: "/api",
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
  (error) => Promise.reject(error),
);

// 3. Setup Response Interceptor untuk menangani Auto-Refresh 401
setupAxiosInterceptors(apiClient);

// 4. No response interceptor: a previous one wrote every GET response into a module-level
// Map that nothing ever read, and never evicted (TTL expiry was only checked by the
// unread getCached). It retained every response payload for the session and gave back
// nothing. Reads are now always live.

export default apiClient;

// 5. (Opsional) Helper khusus untuk GET biasa jika tidak mau pakai async/await di store
export const apiFetch = async (
  url: string,
  options: RequestInit = {},
): Promise<Response> => {
  const token = getAuthToken();
  const isFormData = options.body instanceof FormData;
  const headers: Record<string, string> = {
    ...(isFormData ? {} : { "Content-Type": "application/json" }),
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  // Handle 401 - try refresh and retry once; logout on failure (matches apiClient behavior)
  if (response.status === 401) {
    const newToken = await refreshAccessToken();
    if (newToken) {
      headers["Authorization"] = `Bearer ${newToken}`;
      return fetch(url, {
        ...options,
        headers,
      });
    }
    logout();
    throw new Error("Session expired");
  }

  // 428 - the session still owes a first-login password rotation: raise the
  // blocking modal, but hand the response back so callers keep working.
  if (response.status === 428) {
    const body = await response
      .clone()
      .json()
      .catch(() => null);
    if (body?.error?.code === PASSWORD_CHANGE_REQUIRED) {
      markPasswordChangeRequired();
    }
  }

  return response;
};

import axios from 'axios';
import { user, isAuthenticated, logout } from './stores.js';
import { get } from 'svelte/store';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
  withCredentials: true,
});

// Flag to skip 401 interceptor during auth checks
let skipInterceptors = false;

api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Skip interceptor during auth validation
    if (skipInterceptors) {
      return Promise.reject(error);
    }
    
    if (error.response?.status === 401) {
      logout();
      if (typeof window !== 'undefined') {
        window.location.hash = '#/login';
      }
    }
    return Promise.reject(error);
  }
);

export default api;

export async function checkAuth() {
  // Temporarily skip interceptors to prevent loop
  skipInterceptors = true;
  try {
    const resp = await api.get('/auth/validate');
    user.set(resp.data.user);
    isAuthenticated.set(true);
    return true;
  } catch (e) {
    isAuthenticated.set(false);
    return false;
  } finally {
    skipInterceptors = false;
  }
}

export async function doLogout() {
  try {
    await api.post('/logout');
  } catch (e) {
    // Ignore logout API errors
  }
  logout();
}
import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api',
});

api.interceptors.request.use((config) => {
  const tokenStr = localStorage.getItem('token');
  if (tokenStr) {
    try {
      const token = JSON.parse(tokenStr);
      config.headers.Authorization = `Bearer ${token}`;
    } catch (e) {
      config.headers.Authorization = `Bearer ${tokenStr}`;
    }
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '#/login';
    }
    return Promise.reject(error);
  }
);

export default api;

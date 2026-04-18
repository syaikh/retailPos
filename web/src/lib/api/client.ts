import axios from 'axios';
import { auth } from '$lib/stores/auth';
import { goto } from '$app/navigation';

const client = axios.create({
    baseURL: '/api',
    withCredentials: true,
    timeout: 10000
});

let isRefreshing = false;

client.interceptors.response.use(
    res => res,
    async error => {
        const originalRequest = error.config;
        
        if (error.response?.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;
            
            if (isRefreshing) {
                return Promise.reject(error);
            }
            
            isRefreshing = true;
            
            try {
                const refreshToken = sessionStorage.getItem('refresh_token');
                
                if (!refreshToken) {
                    throw new Error('No refresh token');
                }
                
                const response = await client.post('/refresh', {}, {
                    headers: { 'X-Refresh-Token': refreshToken }
                });
                
                if (response.data.refresh_token) {
                    sessionStorage.setItem('refresh_token', response.data.refresh_token);
                }
                
                return client(originalRequest);
            } catch (refreshError) {
                sessionStorage.removeItem('refresh_token');
                auth.reset();
                goto('/login');
                return Promise.reject(refreshError);
            } finally {
                isRefreshing = false;
            }
        }
        
        return Promise.reject(error);
    }
);

export default client;
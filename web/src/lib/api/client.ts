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
                const response = await client.post('/refresh', {}, {
                    withCredentials: true,
                });
                
                return client(originalRequest);
            } catch (refreshError) {
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
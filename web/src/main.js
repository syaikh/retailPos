import './app.css';
import App from './app/main.svelte';
import { mount } from 'svelte';

// Import apiClient untuk Auto-Refresh Token
import apiClient from './shared/api/http-client';

// Suppress known Chrome extension errors (Receiving end does not exist)
window.addEventListener('unhandledrejection', (event) => {
  if (event.reason?.message?.includes('Receiving end does not exist')) {
    event.preventDefault();
  }
});

// Mount the app using mount() for client-only SPA (no hydration)
const target = document.getElementById('app');
const app = mount(App, { target });

// For Hot Module Replacement (HMR) in development
if (import.meta.hot) {
  import.meta.hot.accept();
}

export default app;

import './app.css';
import App from './lib/App.svelte';
import { mount } from 'svelte';

console.log('MAIN_JS_ENTRY_DEBUG_20260429');

// Mount the app using mount() for client-only SPA (no hydration)
const target = document.getElementById('app');
const app = mount(App, { target });

// For Hot Module Replacement (HMR) in development
if (import.meta.hot) {
  import.meta.hot.accept();
}

export default app;

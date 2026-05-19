import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { fileURLToPath, URL } from 'node:url';

export default defineConfig({
  plugins: [
    tailwindcss(),
    svelte()
  ],
  resolve: {
    alias: {
      '$lib': fileURLToPath(new URL('./src/lib', import.meta.url))
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:9095',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:9095',
        changeOrigin: true,
        ws: true
      },
      '/health': {
        target: 'http://localhost:9095',
        changeOrigin: true
      }
    }
  }
});
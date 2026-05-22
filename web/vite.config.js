import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { fileURLToPath, URL } from 'node:url';
import dotenv from 'dotenv';

dotenv.config({ path: '../.env' });

const frontendPort = Number(process.env.FRONTEND_PORT) || 5173;
const backendPort = Number(process.env.BACKEND_PORT) || 9095;

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
    port: frontendPort,
    proxy: {
      '/api': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true
      },
      '/ws': {
        target: `ws://localhost:${backendPort}`,
        changeOrigin: true,
        ws: true
      },
      '/health': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true
      }
    }
  },
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.{test,spec}.{js,ts}'],
    globals: true,
    setupFiles: ['./src/test-setup.ts']
  }
});
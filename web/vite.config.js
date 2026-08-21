import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { fileURLToPath, URL } from 'node:url';
import dotenv from 'dotenv';

dotenv.config({ path: '../.env' });

const frontendPort = Number(process.env.FRONTEND_PORT) || 5173;
const backendPort = Number(process.env.BACKEND_PORT) || 9095;

export default defineConfig({
  define: {
    __FRONTEND_PORT__: JSON.stringify(frontendPort),
    __BACKEND_PORT__: JSON.stringify(backendPort)
  },
  plugins: [
    tailwindcss(),
    svelte()
  ],
  resolve: {
    alias: {
      '$modules':  fileURLToPath(new URL('./src/modules', import.meta.url)),
      '$shared':   fileURLToPath(new URL('./src/shared', import.meta.url)),
      '$app':      fileURLToPath(new URL('./src/app', import.meta.url)),
    },
    // Testing-library/svelte needs the browser build of Svelte when running under Vitest
    ...(process.env.VITEST ? { conditions: ['browser'] } : {})
  },
  server: {
    port: frontendPort,
    hmr: {
      port: frontendPort
    },
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
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return;
          if (id.includes('chart.js')) return 'chart';
          if (id.includes('xlsx')) return 'xlsx';
          if (id.includes('jspdf-autotable')) return 'jspdf';
          if (id.includes('jspdf') && !id.includes('jspdf-autotable')) return 'jspdf-core';
          if (id.includes('html2canvas')) return 'html2canvas';
          if (id.includes('dompurify')) return 'purify';
        }
      }
    }
  }
});
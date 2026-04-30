/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: '#0f172a',
          secondary: '#1e293b',
        },
        border: '#334155',
        primary: {
          DEFAULT: '#3b82f6',
          hover: '#2563eb',
        },
        success: '#10b981',
        danger: '#ef4444',
        warning: '#f59e0b',
        text: {
          primary: '#f1f5f9',
          secondary: '#94a3b8',
        },
      },
    },
  },
  plugins: [],
}

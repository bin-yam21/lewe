import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In development the API runs separately on :8080; proxy it so the app can
// use same-origin paths exactly as it does when the Go server serves it.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
    },
  },
});

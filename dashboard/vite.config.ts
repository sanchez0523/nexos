import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    // Proxy /api and /ws to the Go ingestion service during `npm run dev`.
    // In production, Caddy handles this same routing.
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: false
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: false
      }
    }
  }
});

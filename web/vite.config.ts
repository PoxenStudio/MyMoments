import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: false,
      injectRegister: 'auto',
      workbox: {
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/openapi/],
        globPatterns: ['**/*.{js,css,html,png,svg,ico,webmanifest}'],
        runtimeCaching: [
          { urlPattern: /^\/api\/v1\//, handler: 'NetworkOnly' },
          {
            urlPattern: /\/assets\//,
            handler: 'CacheFirst',
            options: { cacheName: 'assets', expiration: { maxEntries: 120 } },
          },
          {
            urlPattern: /\/icons\//,
            handler: 'CacheFirst',
            options: { cacheName: 'icons', expiration: { maxEntries: 20 } },
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    port: 5173,
    proxy: { '/api': 'http://127.0.0.1:8080', '/healthz': 'http://127.0.0.1:8080' },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.test.ts'],
  },
});

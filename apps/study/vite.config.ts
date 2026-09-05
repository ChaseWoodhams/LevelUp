import path from 'path'

import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

/**
 * The study tool runs beside the web app, never inside it: its own dev server, its
 * own port (5174 — 5173 belongs to apps/web), its own module graph. The `@` alias
 * mirrors the web app's on purpose, so the modules copied under
 * `src/features/replay/` keep their import lines byte-identical to their origin.
 */
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  server: {
    port: 5174,
    /**
     * `/study` reaches study-server, the same way apps/web reaches the Go API through `/api`.
     *
     * A PROXY AND NOT AN ORIGIN IN THE CLIENT. Talking to `http://127.0.0.1:8100` from the
     * page would be a cross-origin request, and study-server deliberately sends no CORS
     * headers: it reads films and rosters of matches played by people who never published
     * them, and widening it for a dev server would be a strange price to pay for a prefix.
     * Through the proxy, everything the app fetches is same-origin.
     *
     * The target is the server's own default (`cmd/study-server`, loopback by default). A
     * server started on another address is reached by editing this line, which is the honest
     * shape of a tool whose backend is a process the reader starts by hand.
     */
    proxy: {
      '/study': {
        target: 'http://127.0.0.1:8100',
        changeOrigin: false,
        rewrite: (path) => path.replace(/^\/study/, ''),
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    exclude: ['**/node_modules/**', '**/dist/**'],
  },
})

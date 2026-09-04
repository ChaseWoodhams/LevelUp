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
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    exclude: ['**/node_modules/**', '**/dist/**'],
  },
})

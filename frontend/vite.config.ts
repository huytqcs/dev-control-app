import path from 'node:path'
// `vitest/config` re-exports Vite's `defineConfig` merged with the `test`
// option's types, so this file stays a single source of truth for both
// `vite` and `vitest run` without needing a separate vitest.config.ts.
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:4312',
      '/ws': {
        target: 'ws://localhost:4312',
        ws: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
})

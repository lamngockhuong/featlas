import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import type { Plugin } from 'vite'
import { defineConfig } from 'vitest/config'

const outDir = fileURLToPath(new URL('../internal/webui/static', import.meta.url))
const api = process.env.MUNINN_API || 'http://127.0.0.1:7078'

// emptyOutDir wipes the output directory, but `go:embed all:static` needs it
// to exist and be non-empty even without a UI build, so the placeholder is
// written back after every build.
function keepPlaceholder(): Plugin {
  return {
    name: 'muninn-keep-placeholder',
    apply: 'build',
    closeBundle() {
      writeFileSync(`${outDir}/.keep`, '')
    },
  }
}

export default defineConfig({
  base: '/',
  // Chunks import each other by relative URL, so lazy chunks also load when a
  // static export is hosted under a sub-path. index.html keeps absolute URLs
  // for client-side routes; the export rewrites them.
  experimental: {
    renderBuiltUrl: (_file, { hostType }) => (hostType === 'js' ? { relative: true } : undefined),
  },
  plugins: [react(), tailwindcss(), keepPlaceholder()],
  build: {
    outDir,
    emptyOutDir: true,
    assetsDir: 'assets',
    // mermaid's diagram engines (elk, cytoscape) are large lazy chunks that
    // load only when a body contains that diagram type.
    chunkSizeWarningLimit: 1600,
  },
  server: {
    port: Number(process.env.MUNINN_WEB_PORT) || 5174,
    strictPort: true,
    // The API refuses writes whose Origin differs from Host, so the Host header
    // must pass through unchanged (the string shorthand would set changeOrigin).
    proxy: {
      '/api': { target: api, changeOrigin: false },
      '/content': { target: api, changeOrigin: false },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
  },
})

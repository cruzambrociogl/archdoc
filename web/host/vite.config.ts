import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The built app is embedded into the archdoc binary from web/dist. During development,
// `archdoc serve --dev` proxies the page to this server and the page calls the engine's API,
// which Vite forwards to archdoc on its own port.
// base './' keeps every asset path relative, so the same build works served by archdoc at / and
// published under any subpath — a GitHub Pages project site, say.
export default defineConfig({
  base: './',
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  // 127.0.0.1, not localhost: the dev build of archdoc proxies to this exact address (web/embed_dev.go),
  // and localhost can resolve to IPv6 alone.
  server: { host: '127.0.0.1', port: 5173, strictPort: true, proxy: { '/api': 'http://127.0.0.1:7474' } },
})

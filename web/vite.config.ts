import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// The built app is embedded into the archdoc binary from web/dist. During development,
// `archdoc serve --dev` proxies the page to this server and the page calls the engine's API,
// which Vite forwards to archdoc on its own port.
export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: { port: 5173, strictPort: true, proxy: { '/api': 'http://127.0.0.1:7474' } },
})

import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import path from 'path'

// @tanstack/router-plugin generates `function $RefreshReg$(type, id) { return RefreshRuntime.register(...) }`
// in route files, but `register` is not exported from `/@react-refresh`. This plugin patches that.
const patchReactRefreshRegister = {
  name: 'patch-react-refresh-register',
  transform(code: string, id: string) {
    if (id === '/@react-refresh') {
      return { code: code + '\nexport { register }', map: null }
    }
  },
}

export default defineConfig({
  plugins: [
    TanStackRouterVite({ routesDirectory: './src/routes', generatedRouteTree: './src/routeTree.gen.ts' }),
    react(),
    patchReactRefreshRegister,
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        rewrite: (path) => path.replace(/^\/api/, ''),
        changeOrigin: true,
      },
    },
  },
})

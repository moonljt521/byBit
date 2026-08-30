import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 管理后台开发服务器：5174，/api 代理到后端
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})

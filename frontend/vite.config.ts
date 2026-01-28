import path from "path"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    open: true,
    host: '0.0.0.0',
    // 添加静态文件服务
    fs: {
      allow: ['..'],
    },

    proxy: {

      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            const originalPath = req.url;
            console.log(`Before restoring: ${originalPath}`);
            // @ts-expect-error
            proxyReq.path = originalPath.replace('%2F%2F', '//');
            console.log(`Restored path: ${proxyReq.path}`);
          });
        },
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})

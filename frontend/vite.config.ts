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
            // @ts-expect-error 代理请求对象缺少 path 类型定义
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
  build: {
    rollupOptions: {
      output: {
        // 入口文件
        entryFileNames: 'assets/[name]-[hash].js',
        // chunk 文件（你这个 _baseUniq 就是这里来的）
        chunkFileNames: 'assets/chunk-[name]-[hash].js',
        // 其它资源
        assetFileNames: 'assets/[name]-[hash][extname]',
      },
    },
  },
})

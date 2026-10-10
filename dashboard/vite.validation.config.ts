import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: { alias: { '@': resolve(__dirname, 'src') } },
  server: {
    host: '127.0.0.1',
    port: 3005,
    proxy: {
      '/api': { target: 'http://127.0.0.1:18441', changeOrigin: true },
      '/fyers': { target: 'http://127.0.0.1:18441', changeOrigin: true },
    },
  },
})

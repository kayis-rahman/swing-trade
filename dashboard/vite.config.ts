import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3003,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            proxyReq.setHeader('Origin', 'http://localhost:3003')
            console.warn('[VITE PROXY]', req.method, req.url, '->', proxyReq.path)
          })
          proxy.on('proxyRes', (proxyRes, req, res) => {
            const isSSE =
              req.url?.includes('/stream') ||
              proxyRes.headers['content-type']?.includes('text/event-stream')
            if (isSSE) {
              // Prevent buffering: forward headers immediately and remove content-length
              res.setHeader('Content-Type', 'text/event-stream')
              res.setHeader('Cache-Control', 'no-cache')
              res.setHeader('Connection', 'keep-alive')
              res.removeHeader('Content-Length')
              res.flushHeaders()
            }
            console.warn('[VITE PROXY]', req.method, req.url, '->', proxyRes.statusCode)
          })
          proxy.on('error', (err, req) => {
            console.error('[VITE PROXY ERROR]', req.method, req.url, err.message)
          })
        },
      },
      '/fyers': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
})

import { defineConfig } from 'vite'
import { createVuePlugin } from 'vite-plugin-vue2'
import path from 'path'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    createVuePlugin(),
    VitePWA({
      manifest: {
        name: 'DailyTxT',
        short_name: 'DailyTxT',
        description: 'Encrypted diary web-app',
        theme_color: '#2196f3',
        icons: [
          {
            src: '/android-icon-192x192.png',
            sizes: '192x192',
            type: 'image/png'
          },
          {
            src: '/android-icon-512x512.png',
            sizes: '512x512',
            type: 'image/png'
          }
        ]
      },
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      workbox: {
        cleanupOutdatedCaches: true,
        skipWaiting: true
      }
      /* devOptions: {
        enabled: true
      } */
    })
  ],
  server: {
    port: 8080
  },
  resolve: {
    alias: [
      {
        find: '@',
        replacement: path.resolve(__dirname, 'src')
      }
    ]
  },
  build: {
    //       chunkSizeWarningLimit: 600,
    cssCodeSplit: false
  }
})

// https://vitejs.dev/config/
/* export default defineConfig({
  plugins: [vue()]
})
 */

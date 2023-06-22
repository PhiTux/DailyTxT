import { defineConfig } from 'vite'
import { createVuePlugin } from 'vite-plugin-vue2'
import path from 'path'

export default defineConfig({
  plugins: [createVuePlugin()],
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

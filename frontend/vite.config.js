import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import mkcert from 'vite-plugin-mkcert';

export default defineConfig({
	plugins: [
		sveltekit(),
		mkcert(),
		SvelteKitPWA({
			registerType: 'autoUpdate',
			scope: '/',
			manifest: {
				name: 'DailyTxT',
				short_name: 'DailyTxT',
				start_url: '/',
				display: 'standalone',
				background_color: '#ffffff',
				theme_color: '#0d6efd',
				icons: [
					{ src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any maskable' },
					{ src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any maskable' }
				]
			},
			workbox: {
				// Don’t cache API calls; only precache built assets
				navigateFallbackDenylist: [/^\/users\//, /^\/logs\//, /^\/api\//],
				runtimeCaching: [
					{
						urlPattern: ({ url }) => url.origin === self.origin && url.pathname.startsWith('/build/'),
						handler: 'CacheFirst',
						options: {
							cacheName: 'app-assets',
							expiration: { maxEntries: 100, maxAgeSeconds: 60 * 60 * 24 * 30 },
						}
					}
				]
			},
			devOptions: { enabled: true }
		})
	],
	server: {
		port: 5173,
		https: false
	}
});

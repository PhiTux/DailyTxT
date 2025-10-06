import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';

export default defineConfig({
	plugins: [
		sveltekit(),
		SvelteKitPWA({
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
			devOptions: { enabled: true }
		})
	],
	server: {
		port: 5173,
		https: false
	},
	css: {
		preprocessorOptions: {
			scss: {
				silenceDeprecations: ['color-functions', 'import', 'global-builtin']
			}
		}
	}
});

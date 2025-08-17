<script>
	import { blur } from 'svelte/transition';
	import axios from 'axios';
	//import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import '../scss/styles.scss';
	import { useTrianglify, trianglifyOpacity } from '$lib/settingsStore.js';
	import { page } from '$app/state';
	import { API_URL } from '$lib/APIurl.js';
	import trianglify from 'trianglify';
	import { alwaysShowSidenav } from '$lib/helpers.js';
	import * as bootstrap from 'bootstrap';
	import {
		TolgeeProvider,
		Tolgee,
		DevTools,
		LanguageDetector,
		LanguageStorage
	} from '@tolgee/svelte';
	import { FormatIcu } from '@tolgee/format-icu';
	import { use } from 'marked';

	const tolgee = Tolgee()
		.use(DevTools())
		.use(FormatIcu())
		.use(LanguageStorage())
		.init({
			availableLanguages: ['en', 'de', 'fr'],
			defaultLanguage: 'en',

			// for development
			apiUrl: import.meta.env.VITE_TOLGEE_API_URL,
			apiKey: import.meta.env.VITE_TOLGEE_API_KEY
		});

	let { children } = $props();
	let inDuration = 150;
	let outDuration = 150;

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	let available_backup_codes = $state(0);

	axios.interceptors.response.use(
		(response) => {
			if (response.data && response.data.available_backup_codes >= 0) {
				available_backup_codes = response.data.available_backup_codes;
				// show toast
				if (available_backup_codes < 6) {
					let toast = new bootstrap.Toast(
						document.getElementById('toastAvailableBackupCodesWarning')
					);
					toast.show();
				}
			}
			return response;
		},
		(error) => {
			if (
				error.response &&
				error.response.status &&
				(error.response.status == 401 || error.response.status == 440)
			) {
				// logout
				axios
					.get(API_URL + '/users/logout')
					.then((response) => {
						localStorage.removeItem('user');
						goto(`/login?error=${error.response.status}`);
					})
					.catch((error) => {
						console.error(error);
					});
			}
			return Promise.reject(error);
		}
	);

	function createBackground() {
		if ($useTrianglify) {
			//remove old canvas
			const oldCanvas = document.querySelector('canvas');
			if (oldCanvas) {
				oldCanvas.remove();
			}

			//xColors: ['#F3F3F3', '#FEFEFE', '#E5E5E5'],
			const canvas = trianglify({
				width: window.innerWidth,
				height: window.innerHeight,
				xColors: ['#FA2'],
				fill: false,
				strokeWidth: 1,
				cellSize: 100
			});

			document.body.appendChild(canvas.toCanvas());
			document.querySelector('canvas').style =
				'position: fixed; top: 0; left: 0; z-index: -1; opacity: 0.4; width: 100%; height: 100%; background-color: #eaeaea;';
		}
	}

	$effect(() => {
		if ($trianglifyOpacity) {
			if (document.querySelector('canvas')) {
				document.querySelector('canvas').style.opacity = $trianglifyOpacity;
			}
		}
	});

	function calculateResize() {
		if (window.innerWidth > 840) {
			$alwaysShowSidenav = true;
		} else {
			$alwaysShowSidenav = false;
		}
	}

	/* trigger on window-resize */
	window.addEventListener('resize', () => {
		calculateResize();
	});

	onMount(() => {
		createBackground();
		calculateResize();
	});

	let routeToFromLoginKey = $derived(page.url.pathname === '/login');
</script>

<TolgeeProvider {tolgee}>
	<main class="d-flex flex-column">
		<div class="wrapper h-100">
			{#key routeToFromLoginKey}
				<div
					class="transition-wrapper h-100"
					out:blur={{ duration: outDuration }}
					in:blur={{ duration: inDuration, delay: outDuration }}
				>
					{@render children()}
				</div>
			{/key}
		</div>

		<div class="toast-container position-fixed bottom-0 end-0 p-3">
			<div
				id="toastAvailableBackupCodesWarning"
				class="toast align-items-center {available_backup_codes > 3
					? 'text-bg-warning'
					: 'text-bg-danger'}"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Noch {available_backup_codes} Backup-Codes verfügbar!</div>
				</div>
			</div>
		</div>
	</main>
</TolgeeProvider>

<style>
	main {
		height: 100vh;

		/* background-image: linear-gradient(#ff8a00, #e52e71); */
		/* background-image: linear-gradient(to right, violet, darkred, purple); */
		/* background: linear-gradient(40deg, #38bdf8, #fb7185, #84cc16); */
	}

	.wrapper {
		position: relative; /* Ensure the wrapper is the positioning context */
	}

	.transition-wrapper {
		position: absolute; /* Ensure the transition wrapper does not occupy space */
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
	}
</style>

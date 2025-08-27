<script>
	import { blur } from 'svelte/transition';
	import axios from 'axios';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import '../scss/styles.scss';
	import { page } from '$app/state';
	import { API_URL } from '$lib/APIurl.js';
	import { alwaysShowSidenav, generateNeonMesh } from '$lib/helpers.js';
	import * as bootstrap from 'bootstrap';
	import { TolgeeProvider, Tolgee, DevTools, LanguageStorage } from '@tolgee/svelte';
	import { FormatIcu } from '@tolgee/format-icu';

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
		calculateResize();

		// if on login page, generate neon mesh
		if (page.url.pathname === '/login') {
			generateNeonMesh();
		}
	});

	let routeToFromLoginKey = $derived(page.url.pathname === '/login');
</script>

<main class="d-flex flex-column background" use:focus={generateNeonMesh}>
	<TolgeeProvider {tolgee}>
		<div class="wrapper h-100" transition:blur={{ duration: inDuration * 2 }}>
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
	</TolgeeProvider>
</main>

<style>
	main {
		height: 100vh;
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

	:global(.glass) {
		backdrop-filter: blur(12px) saturate(130%);
		/* background-color: rgba(219, 219, 219, 0.45); */
		background-color: rgba(83, 83, 83, 0.73);
		border: 1px solid #62626278;
		color: #ececec;
	}

	:global(.glassLight) {
		backdrop-filter: blur(8px) saturate(130%);
		/* background-color: rgba(219, 219, 219, 0.45); */
		background-color: rgba(83, 83, 83, 0.445);
		border: 1px solid #62626278;
		color: #ececec;
	}
</style>

<script>
	import { blur } from 'svelte/transition';
	import axios from 'axios';
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';
	import '../scss/styles.scss';
	import { page } from '$app/state';
	import { API_URL } from '$lib/APIurl.js';
	import { alwaysShowSidenav, generateNeonMesh } from '$lib/helpers.js';
	import * as bootstrap from 'bootstrap';
	import { TolgeeProvider, Tolgee, DevTools, LanguageStorage } from '@tolgee/svelte';
	import { FormatIcu } from '@tolgee/format-icu';
	import { darkMode } from '$lib/settingsStore.js';
	import { registerSW } from 'virtual:pwa-register';
	import { resolve } from '$app/paths';

	const tolgee = Tolgee()
		.use(DevTools())
		.use(FormatIcu())
		.use(LanguageStorage())
		.init({
			availableLanguages: ['en', 'de', 'fr', 'cs', 'no'],
			defaultLanguage: 'en',
			fallbackLanguage: 'en',
			staticData: {
				en: () => import('../i18n/en.json'),
				de: () => import('../i18n/de.json'),
				fr: () => import('../i18n/fr.json'),
				cs: () => import('../i18n/cs.json'),
				no: () => import('../i18n/no.json')
			},

			// for development
			apiUrl: import.meta.env.VITE_TOLGEE_API_URL,
			apiKey: import.meta.env.VITE_TOLGEE_API_KEY
		});

	let { children } = $props();
	let inDuration = 150;
	let outDuration = 150;

	// Keep a reference to the SW registration to allow immediate skipWaiting
	let swRegistration = null;

	// PWA install prompt state
	let deferredInstallPrompt = $state(null);
	let showInstallToast = $state(false);

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
					.then(() => {
						localStorage.removeItem('user');
						goto(resolve(`/login?error=${error.response.status}`));
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

	let isStandalone = false;

	onMount(() => {
		calculateResize();

		// if on login page, generate neon mesh
		if (page.url.pathname.endsWith('/login')) {
			generateNeonMesh($darkMode);
		}

		// helper to present the PWA update toast and wire the reload button once
		function showPwaUpdateToast() {
			setTimeout(() => {
				const toast = new bootstrap.Toast(document.getElementById('toastPwaUpdate'), {
					autohide: false
				});
				toast.show();
				const btn = document.getElementById('btnPwaReload');
				let swReloadScheduled = false;
				if (btn) {
					btn.removeAttribute('disabled');
					btn.onclick = () => {
						if (swReloadScheduled) return;
						swReloadScheduled = true;
						btn.setAttribute('disabled', 'true');
						// Ask the waiting SW (if present) to activate immediately
						try {
							swRegistration?.waiting?.postMessage({ type: 'SKIP_WAITING' });
						} catch (_) {}
						// Request update with built-in reload once activated; don't await to keep UI responsive
						try {
							updateSW(true);
						} catch (_) {}
						// Fallback: force a reload after a brief delay in case the event doesn't fire on this platform
						setTimeout(() => {
							try {
								window.location.reload();
							} catch (_) {}
						}, 1200);
					};
				}
			}, 500);
		}

		// PWA auto-update with user prompt
		const updateSW = registerSW({
			onNeedRefresh() {
				showPwaUpdateToast();
			},
			onOfflineReady() {
				// not needed, we don't aim offline, skip toast
			},
			onRegisteredSW(_swUrl, registration) {
				// When the app stays open, explicitly ask the SW to check for updates
				// on focus, visibility regain, going online, and periodically.
				if (!registration) return;
				swRegistration = registration;
				const checkForUpdate = () => {
					try {
						registration.update();
					} catch (_) {}
				};
				// If a waiting SW already exists (e.g. app was open when update installed), show the toast now.
				if (registration.waiting) {
					showPwaUpdateToast();
				}
				// Initial delayed check after mount
				const initialTimer = setTimeout(checkForUpdate, 5000);
				// Check on app focus / tab visible / network back
				const onFocus = () => checkForUpdate();
				const onVisibility = () => {
					if (document.visibilityState === 'visible') checkForUpdate();
				};
				const onOnline = () => checkForUpdate();
				window.addEventListener('focus', onFocus);
				document.addEventListener('visibilitychange', onVisibility);
				window.addEventListener('online', onOnline);
				// Periodic check (e.g., every 30 minutes)
				const intervalId = setInterval(checkForUpdate, 30 * 60 * 1000);
				// Cleanup on destroy (main layout rarely unmounts, but keep things tidy)
				onDestroy(() => {
					clearTimeout(initialTimer);
					clearInterval(intervalId);
					window.removeEventListener('focus', onFocus);
					document.removeEventListener('visibilitychange', onVisibility);
					window.removeEventListener('online', onOnline);
				});
			},
			onRegistered(registration) {
				// Fallback for environments exposing onRegistered instead of onRegisteredSW
				if (!registration) return;
				swRegistration = registration;
				const checkForUpdate = () => {
					try {
						registration.update();
					} catch (_) {}
				};
				if (registration.waiting) showPwaUpdateToast();
				const initialTimer = setTimeout(checkForUpdate, 5000);
				const onFocus = () => checkForUpdate();
				const onVisibility = () => {
					if (document.visibilityState === 'visible') checkForUpdate();
				};
				const onOnline = () => checkForUpdate();
				window.addEventListener('focus', onFocus);
				document.addEventListener('visibilitychange', onVisibility);
				window.addEventListener('online', onOnline);
				const intervalId = setInterval(checkForUpdate, 30 * 60 * 1000);
				onDestroy(() => {
					clearTimeout(initialTimer);
					clearInterval(intervalId);
					window.removeEventListener('focus', onFocus);
					document.removeEventListener('visibilitychange', onVisibility);
					window.removeEventListener('online', onOnline);
				});
			}
		});

		// Detect standalone (already installed) and platforms where auto prompt won't show
		isStandalone =
			window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone;

		// Capture the install prompt event (Android/Chrome etc.)
		window.addEventListener('beforeinstallprompt', (e) => {
			// Prevent the mini-infobar and save for triggering later
			e.preventDefault();
			deferredInstallPrompt = e;
			if (!isStandalone) {
				// show toast
				setTimeout(() => {
					const toast = new bootstrap.Toast(document.getElementById('toastPwaInstall'), {
						autohide: false
					});
					toast.show();
				}, 500);
			}
		});

		// Hide install banner when app gets installed
		window.addEventListener('appinstalled', () => {
			deferredInstallPrompt = null;
			showInstallToast = false;
		});
	});

	async function installPWA() {
		if (!deferredInstallPrompt) return;
		deferredInstallPrompt.prompt();
		try {
			await deferredInstallPrompt.userChoice;
		} finally {
			// Only allow prompting once
			deferredInstallPrompt = null;
			showInstallToast = false;
		}
	}

	$effect(() => {
		if ($darkMode !== undefined) {
			document.body.setAttribute('data-bs-theme', $darkMode ? 'dark' : 'light');
		}
	});

	let routeToFromLoginKey = $derived(page.url.pathname.endsWith('/login'));
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
				id="toastPwaInstall"
				class="toast text-bg-primary"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex align-items-center">
					<div class="toast-body">{tolgee.t('toast.pwa.install_description')}</div>
					<button
						id="btnPwaInstall"
						type="button"
						class="btn btn-primary btn-sm me-2 m-2 toastBtn"
						onclick={installPWA}
					>
						{tolgee.t('toast.pwa.install_button')}
					</button>
					<button
						type="button"
						class="btn-close me-2 m-auto"
						data-bs-dismiss="toast"
						aria-label="Close"
						onclick={() => (showInstallToast = false)}
					></button>
				</div>
			</div>
			<div
				id="toastPwaUpdate"
				class="toast text-bg-info"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex align-items-center">
					<div class="toast-body">{tolgee.t('toast.pwa.update_available')}</div>
					<button id="btnPwaReload" type="button" class="btn btn-primary btn-sm me-2 m-2 toastBtn">
						{tolgee.t('toast.pwa.reload_button')}
					</button>
					<button
						type="button"
						class="btn-close me-2 m-auto"
						data-bs-dismiss="toast"
						aria-label="Close"
					></button>
				</div>
			</div>
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
					<div class="toast-body">
						{tolgee.t('toast.password.available_backup_codes', { count: available_backup_codes })}
					</div>
					<button
						type="button"
						class="btn-close me-2 m-auto"
						data-bs-dismiss="toast"
						aria-label="Close"
					></button>
				</div>
			</div>
		</div>
	</TolgeeProvider>
</main>

<style>
	:global(.glass-shadow) {
		box-shadow: 3px 3px 8px 2px rgba(0, 0, 0, 0.2);
	}

	.toastBtn {
		background-color: #f57c00 !important;
		color: black !important;
	}

	.toastBtn:hover {
		background-color: rgb(201, 100, 0) !important;
		color: black !important;
	}

	:global(.toast-container) {
		z-index: 9999;
	}

	:global(body[data-bs-theme='dark'] button) {
		color: #fbfbfe;
	}

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

	:global(.modal.show) {
		background-color: rgba(0, 0, 0, 0.3) !important;
	}

	:global(body[data-bs-theme='dark'] .modal-content) {
		backdrop-filter: blur(17px) saturate(150%);
		background-color: rgba(70, 70, 70, 0.5) !important;
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #ececec;
	}
	:global(body[data-bs-theme='light'] .modal-content) {
		backdrop-filter: blur(17px) saturate(150%);
		background-color: rgba(211, 211, 211, 0.5) !important;
		border: 1px solid rgba(255, 255, 255, 0.2);
		color: #161616;
	}

	:global(body[data-bs-theme='dark'] .glass) {
		backdrop-filter: blur(14px) saturate(130%);
		background-color: rgba(83, 83, 83, 0.4);
		border: 1px solid #62626278;
		color: #ececec;
	}
	:global(body[data-bs-theme='light'] .glass) {
		backdrop-filter: blur(14px) saturate(130%);
		background-color: rgba(158, 158, 158, 0.3);
		border: 1px solid #ccc;
		color: #222;
	}

	:global(body[data-bs-theme='dark'] .popover-body > span) {
		background-color: #444;
	}
</style>

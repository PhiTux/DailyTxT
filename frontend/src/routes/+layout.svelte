<script>
	import { fade, blur, slide } from 'svelte/transition';
	import axios from 'axios';
	//import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import '../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Fa from 'svelte-fa';
	import { readingMode } from '$lib/settingsStore.js';
	import { page } from '$app/state';
	import { API_URL } from '$lib/APIurl.js';
	import trianglify from 'trianglify';
	import { useTrianglify, trianglifyOpacity } from '$lib/settingsStore.js';

	import {
		faRightFromBracket,
		faGlasses,
		faPencil,
		faSliders
	} from '@fortawesome/free-solid-svg-icons';

	let { children } = $props();
	let inDuration = 150;
	let outDuration = 150;

	$effect(() => {
		if ($readingMode) {
			goto('/read');
		} else {
			goto('/write');
		}
	});

	function logout() {
		axios
			.get(API_URL + '/users/logout')
			.then((response) => {
				localStorage.removeItem('user');
				goto('/login');
			})
			.catch((error) => {
				console.error(error);
			});
	}

	function createBackground() {
		if ($useTrianglify) {
			//remove old canvas
			const oldCanvas = document.querySelector('canvas');
			if (oldCanvas) {
				oldCanvas.remove();
			}

			const canvas = trianglify({
				width: window.innerWidth,
				height: window.innerHeight
			});

			document.body.appendChild(canvas.toCanvas());
			document.querySelector('canvas').style =
				'position: fixed; top: 0; left: 0; z-index: -1; opacity: 0.8; width: 100%; height: 100%;';
		}
	}

	$effect(() => {
		if ($trianglifyOpacity) {
			if (document.querySelector('canvas')) {
				document.querySelector('canvas').style.opacity = $trianglifyOpacity;
			}
		}
	});

	onMount(() => {
		createBackground();

		document.getElementById('settingsModal').addEventListener('shown.bs.modal', function () {
			const height = document.getElementById('modal-body').clientHeight;
			document.getElementById('settings-content').style.height = 'calc(' + height + 'px - 2rem)';
			document.getElementById('settings-nav').style.height = 'calc(' + height + 'px - 2rem)';
			document.getElementById('settings-content').style.overflowY = 'auto';

			setTimeout(() => {
				const dataSpyList = document.querySelectorAll('[data-bs-spy="scroll"]');
				dataSpyList.forEach((dataSpyEl) => {
					bootstrap.ScrollSpy.getInstance(dataSpyEl).refresh();
				});
			}, 200);
		});
	});
</script>

<main class="d-flex flex-column">
	<nav class="navbar navbar-expand-lg bg-body-tertiary">
		<div class="row w-100">
			<div class="col-lg-4 col-sm-5 col d-flex flex-row justify-content-start align-items-center">
				<button
					class="btn d-md-none"
					type="button"
					data-bs-toggle="offcanvas"
					data-bs-target="#sidenav"
					aria-controls="sidenav">menü</button
				>

				<div class="form-check form-switch d-flex flex-row">
					<label class="me-3" for="flexSwitchCheckDefault"><Fa icon={faPencil} size="1.5x" /></label
					>
					<div class="form-check form-switch">
						<input
							class="form-check-input"
							bind:checked={$readingMode}
							type="checkbox"
							role="switch"
							id="flexSwitchCheckDefault"
							style="transform: scale(1.3);"
						/>
					</div>
					<label class="ms-2" for="flexSwitchCheckDefault"
						><Fa icon={faGlasses} size="1.5x" /></label
					>
				</div>
			</div>

			<div class="col-lg-4 col-sm-2 col d-flex flex-row justify-content-center align-items-center">
				Center-LOGO
			</div>

			<div class="col-lg-4 col-sm-5 col pe-0 d-flex flex-row justify-content-end">
				<button
					class="btn btn-outline-secondary me-2"
					data-bs-toggle="modal"
					data-bs-target="#settingsModal"><Fa icon={faSliders} /></button
				>
				<button class="btn btn-outline-secondary" onclick={logout}
					><Fa icon={faRightFromBracket} /></button
				>
			</div>
		</div>
	</nav>

	<div class="wrapper h-100">
		{#key page.data}
			<div
				class="transition-wrapper h-100"
				out:blur={{ duration: outDuration }}
				in:blur={{ duration: inDuration, delay: outDuration }}
			>
				{@render children()}
			</div>
		{/key}
	</div>

	<!-- Full screen modal -->
	<div class="modal fade" data-bs-backdrop="static" id="settingsModal">
		<div
			class="modal-dialog modal-dialog-scrollable modal-dialog-centered modal-xl modal-fullscreen-sm-down"
		>
			<!--  -->
			<div class="modal-content shadow-lg">
				<div class="modal-header">
					<h1>Settings</h1>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"
					></button>
				</div>
				<div class="modal-body" id="modal-body">
					<div class="row">
						<div class="col-4 overflow-y-auto">
							<nav class="flex-column align-items-stretch" id="settings-nav">
								<nav class="nav nav-pills flex-column">
									<a class="nav-link mb-1" href="#appearance">Aussehen</a>
									<a href="#lightdark" class="ms-3 mb-1 nav-link">Light/Dark-Mode</a>
									<a href="#background" class="ms-3 mb-1 nav-link">Hintergrund</a>
									<a class="nav-link mb-1" href="#functions">Funktionen</a>
									<nav class="nav nav-pills flex-column">
										<a href="#language" class="ms-3 mb-1 nav-link">Sprache</a>
										<a href="#timezone" class="ms-3 mb-1 nav-link">Zeitzone</a>
										<a href="#onthisday" class="ms-3 mb-1 nav-link">An diesem Tag</a>
										<a href="#loginonreload" class="ms-3 mb-1 nav-link">Login bei Reload</a>
									</nav>
									<a class="nav-link mb-1" href="#tags">Tags</a>
									<a class="nav-link mb-1" href="#templates">Vorlagen</a>
									<a class="nav-link mb-1" href="#data">Daten</a>
									<nav class="nav nav-pills flex-column">
										<a href="#export" class="ms-3 nav-link mb-1">Export</a>
										<a href="#import" class="ms-3 nav-link mb-1">Import</a>
									</nav>
									<a class="nav-link mb-1" href="#account">Account</a>
									<nav class="nav nav-pills flex-column">
										<a href="#password" class="ms-3 mb-1 nav-link">Password ändern</a>
										<a href="#backupkeys" class="ms-3 mb-1 nav-link">Backup-Keys</a>
										<a href="#username" class="ms-3 mb-1 nav-link">Username ändern</a>
										<a href="#deleteaccount" class="ms-3 mb-1 nav-link">Konto löschen</a>
									</nav>
								</nav>
							</nav>
						</div>
						<div class="col-8">
							<div
								class="settings-content overflow-y-auto"
								data-bs-spy="scroll"
								data-bs-target="#settings-nav"
								data-bs-smooth-scroll="true"
								id="settings-content"
							>
								<div id="appearance"><h4>Aussehen</h4></div>
								<div id="lightdark">
									<h5>Light/Dark-Mode</h5>
									Bla<br />
									blub <br />
									bla <br />
									blub <br />
								</div>
								<div id="background">
									<h5>Hintergrund</h5>
									<div class="d-flex flex-row justify-content-start">
										<label for="trianglifyOpacity" class="form-label"
											>Transparenz der Dreiecke</label
										>
										<input
											bind:value={$trianglifyOpacity}
											type="range"
											class="mx-3 form-range"
											id="trianglifyOpacity"
											min="0"
											max="1"
											step="0.01"
										/>
										<input
											bind:value={$trianglifyOpacity}
											type="number"
											id="trianglifyOpacityNumber"
										/>
									</div>
								</div>

								<div id="functions"><h4>Funktionen</h4></div>

								<div id="language">
									<h5>Sprache</h5>
									Bla<br />
									blub <br />
									bla <br />
									blub <br />
								</div>
								<div id="timezone">
									<h5>Zeitzone</h5>
									Bla<br />
									blub <br />
									bla <br />
									blub <br />
								</div>
								<div id="onthisday">
									<h5>An diesem Tag</h5>
									Bla<br />
									blub <br />
									bla <br />
									blub <br />
								</div>
								<div id="loginonreload">
									<h5>Login bei Reload</h5>
									Bla<br />
									blub <br />
									bla <br />
									blub <br />
								</div>

								<div id="tags"><h4>Tags</h4></div>

								<div id="templates"><h4>Vorlagen</h4></div>

								<div id="data"><h4>Daten</h4></div>
								<div id="export"><h5>Export</h5></div>
								<div id="import"><h5>Import</h5></div>

								<div id="account"><h4>Account</h4></div>
								<div id="password"><h5>Password ändern</h5></div>
								<div id="backupkeys"><h5>Backup-Keys</h5></div>
								<div id="username"><h5>Username ändern</h5></div>
								<div id="deleteaccount"><h5>Konto löschen</h5></div>
							</div>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-primary">Speichern</button>
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Abbrechen</button>
				</div>
			</div>
		</div>
	</div>
</main>

<style>
	#trianglifyOpacity {
		max-width: 300px;
	}

	#trianglifyOpacityNumber {
		width: 80px;
	}

	.modal-body {
		overflow-y: hidden;
	}

	.modal-content {
		backdrop-filter: blur(10px) saturate(150%);
		/* background-color: rgba(43, 56, 78, 0.75); */
		background-color: rgba(208, 208, 208, 0.61);
		/* color: rgb(22, 22, 22); */
	}

	.modal-header {
		border-bottom: 1px solid rgba(255, 255, 255, 0.2);
	}

	.modal-footer {
		border-top: 1px solid rgba(255, 255, 255, 0.2);
	}

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

<script>
	import { blur } from 'svelte/transition';
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
	import { useTrianglify, trianglifyOpacity, autoLoadImages } from '$lib/settingsStore.js';
	import { tags } from '$lib/tagStore.js';
	import TagModal from '$lib/TagModal.svelte';
	import { alwaysShowSidenav } from '$lib/helpers.js';

	import {
		faRightFromBracket,
		faGlasses,
		faPencil,
		faSliders,
		faTriangleExclamation
	} from '@fortawesome/free-solid-svg-icons';
	import Tag from '$lib/Tag.svelte';

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

	let settingsModal;
	function openSettingsModal() {
		settingsModal = new bootstrap.Modal(document.getElementById('settingsModal'));
		settingsModal.show();
	}

	$effect(() => {
		if ($trianglifyOpacity) {
			if (document.querySelector('canvas')) {
				document.querySelector('canvas').style.opacity = $trianglifyOpacity;
			}
		}
	});

	/* Important for development: convenient modal-handling with HMR */
	if (import.meta.hot) {
		import.meta.hot.dispose(() => {
			document.querySelectorAll('.modal-backdrop').forEach((el) => el.remove());
		});
	}

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

	let editTagModal;
	let editTag = $state({});
	let isSavingEditedTag = $state(false);

	function openTagModal(tagId) {
		$tags.forEach((tag) => {
			if (tag.id === tagId) {
				editTag.name = tag.name;
				editTag.color = tag.color;
				editTag.icon = tag.icon;
				editTag.id = tag.id;
				return;
			}
		});

		settingsModal.hide();
		editTagModal.open();
	}

	let deleteTagId = $state(null);
	function askDeleteTag(tagId) {
		if (deleteTagId === tagId) deleteTagId = null;
		else deleteTagId = tagId;
	}

	let isDeletingTag = $state(false);
	function deleteTag(tagId) {
		if (isDeletingTag) return;
		isDeletingTag = true;

		axios
			.get(API_URL + '/logs/deleteTag', { params: { id: tagId } })
			.then((response) => {
				if (response.data.success) {
					$tags = $tags.filter((tag) => tag.id !== tagId);
				}
			})
			.catch((error) => {
				console.error(error);

				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorDeleteTag'));
				toast.show();
			})
			.finally(() => {
				deleteTagId = null;
				isDeletingTag = false;
			});
	}

	function saveEditedTag() {
		if (isSavingEditedTag) return;
		isSavingEditedTag = true;

		axios
			.post(API_URL + '/logs/editTag', editTag)
			.then((response) => {
				if (response.data.success) {
					$tags = $tags.map((tag) => {
						if (tag.id === editTag.id) {
							tag.name = editTag.name;
							tag.color = editTag.color;
							tag.icon = editTag.icon;
						}
						return tag;
					});

					// show toast
					const toast = new bootstrap.Toast(document.getElementById('toastSuccessEditTag'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);

				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorEditTag'));
				toast.show();
			})
			.finally(() => {
				isSavingEditedTag = false;
				editTagModal.close();
				settingsModal.show();
			});
	}
</script>

<main class="d-flex flex-column">
	<nav class="navbar navbar-expand-lg bg-body-tertiary">
		<div class="row w-100">
			<div class="col-lg-4 col-sm-5 col d-flex flex-row justify-content-start align-items-center">
				{#if !$alwaysShowSidenav}
					<button
						class="btn d-xl-none"
						type="button"
						data-bs-toggle="offcanvas"
						data-bs-target="#sidenav"
						aria-controls="sidenav">menü</button
					>
				{/if}

				<div class="form-check form-switch d-flex flex-row">
					<label class="me-3" for="selectMode"><Fa icon={faPencil} size="1.5x" /></label>
					<div class="form-check form-switch">
						<input
							class="form-check-input"
							bind:checked={$readingMode}
							type="checkbox"
							role="switch"
							id="selectMode"
							style="transform: scale(1.3);"
						/>
					</div>
					<label class="ms-2" for="selectMode"><Fa icon={faGlasses} size="1.5x" /></label>
				</div>
			</div>

			<div class="col-lg-4 col-sm-2 col d-flex flex-row justify-content-center align-items-center">
				Center-LOGO
			</div>

			<div class="col-lg-4 col-sm-5 col pe-0 d-flex flex-row justify-content-end">
				<button class="btn btn-outline-secondary me-2" onclick={openSettingsModal}
					><Fa icon={faSliders} /></button
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

	<TagModal
		bind:this={editTagModal}
		createTag={false}
		bind:editTag
		isSaving={isSavingEditedTag}
		{saveEditedTag}
	/>

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
									<!-- <a href="#lightdark" class="ms-3 mb-1 nav-link">Light/Dark-Mode</a>
									<a href="#background" class="ms-3 mb-1 nav-link">Hintergrund</a> -->
									<a class="nav-link mb-1" href="#functions">Funktionen</a>
									<!-- <nav class="nav nav-pills flex-column">
										<a href="#language" class="ms-3 mb-1 nav-link">Sprache</a>
										<a href="#timezone" class="ms-3 mb-1 nav-link">Zeitzone</a>
										<a href="#onthisday" class="ms-3 mb-1 nav-link">An diesem Tag</a>
										<a href="#loginonreload" class="ms-3 mb-1 nav-link">Login bei Reload</a>
									</nav> -->
									<a class="nav-link mb-1" href="#tags">Tags</a>
									<a class="nav-link mb-1" href="#templates">Vorlagen</a>
									<a class="nav-link mb-1" href="#data">Daten</a>
									<!-- <nav class="nav nav-pills flex-column">
										<a href="#export" class="ms-3 nav-link mb-1">Export</a>
										<a href="#import" class="ms-3 nav-link mb-1">Import</a>
									</nav> -->
									<a class="nav-link mb-1" href="#security">Sicherheit</a>
									<!-- <nav class="nav nav-pills flex-column">
										<a href="#password" class="ms-3 mb-1 nav-link">Password ändern</a>
										<a href="#backupkeys" class="ms-3 mb-1 nav-link">Backup-Keys</a>
										<a href="#username" class="ms-3 mb-1 nav-link">Username ändern</a>
										<a href="#deleteaccount" class="ms-3 mb-1 nav-link">Konto löschen</a>
									</nav> -->
									<a class="nav-link mb-1" href="#about">About</a>
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
								<h3 id="appearance" class="text-primary">Aussehen</h3>
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

								<h3 id="functions" class="text-primary">Funktionen</h3>

								<div id="autoLoadImages">
									<h5>Bilder automatisch laden</h5>
									<ul>
										<li>
											Beim Laden eines Textes können hochgeladene Bilder (sofern vorhanden)
											automatisch geladen werden. <em>Erhöhter Datenverbrauch!</em>
										</li>
										<li>Alternativ wird ein Button zum Nachladen aller Bilder angezeigt.</li>
									</ul>

									<div class="form-check form-switch">
										<input
											class="form-check-input"
											bind:checked={$autoLoadImages}
											type="checkbox"
											role="switch"
											id="autoLoadImagesSwitch"
										/>
										<label class="form-check-label" for="autoLoadImagesSwitch">
											{#if $autoLoadImages}
												Bilder werden automatisch geladen
											{:else}
												Bilder werden nicht automatisch geladen
											{/if}</label
										>
									</div>
								</div>

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

								<h3 id="tags" class="text-primary">Tags</h3>
								<div>
									Hier können Tags bearbeitet oder auch vollständig aus DailyTxT gelöscht werden.
									<div class="d-flex flex-column tagColumn mt-1">
										{#each $tags as tag}
											<Tag
												{tag}
												isEditable
												editTag={openTagModal}
												isDeletable
												deleteTag={askDeleteTag}
											/>
											{#if deleteTagId === tag.id}
												<div class="alert alert-danger align-items-center" role="alert">
													<div>
														<Fa icon={faTriangleExclamation} fw /> <b>Tag dauerhaft löschen?</b> Dies
														kann einen Moment dauern, da jeder Eintrag nach potenziellen Verlinkungen
														durchsucht werden muss. Änderungen werden zudem u. U. erst nach einem Neuladen
														im Browser angezeigt.
													</div>
													<!-- svelte-ignore a11y_consider_explicit_label -->
													<div class="d-flex flex-row mt-2">
														<button class="btn btn-secondary" onclick={() => (deleteTagId = null)}
															>Abbrechen
														</button>
														<button
															disabled={isDeletingTag}
															class="btn btn-danger ms-3"
															onclick={() => deleteTag(tag.id)}
															>Löschen
															{#if isDeletingTag}
																<span
																	class="spinner-border spinner-border-sm ms-2"
																	role="status"
																	aria-hidden="true"
																></span>
															{/if}
														</button>
													</div>
												</div>
											{/if}
										{/each}
									</div>
								</div>

								<div id="templates"><h4>Vorlagen</h4></div>

								<div id="data">
									<h4>Daten</h4>
									<div id="export"><h5>Export</h5></div>
									<div id="import"><h5>Import</h5></div>
								</div>

								<div id="security">
									<h4>Sicherheit</h4>
									<div id="password"><h5>Password ändern</h5></div>
									<div id="backupkeys"><h5>Backup-Keys</h5></div>
									<div id="username"><h5>Username ändern</h5></div>
									<div id="deleteaccount"><h5>Konto löschen</h5></div>
								</div>

								<div id="about">
									<h4>About</h4>
									Version:<br />
									Changelog: <br />
									Link zu github
								</div>
							</div>
						</div>
					</div>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Abbrechen</button>
					<button type="button" class="btn btn-primary">Speichern</button>
				</div>
			</div>
		</div>
	</div>

	<div class="toast-container position-fixed bottom-0 end-0 p-3">
		<div
			id="toastSuccessEditTag"
			class="toast align-items-center text-bg-success"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">Änderungen wurden gespeichert!</div>
			</div>
		</div>

		<div
			id="toastErrorEditTag"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">Fehler beim Speichern der Änderungen!</div>
			</div>
		</div>

		<div
			id="toastErrorDeleteTag"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">Fehler beim Löschen des Tags!</div>
			</div>
		</div>
	</div>
</main>

<style>
	:global(.tagColumn > span) {
		width: min-content;
	}

	.tagColumn {
		gap: 0.5rem;
		/* width: min-content; */
	}

	#selectMode:checked {
		border-color: #da880e;
		background-color: #da880e;
	}

	#selectMode:not(:checked) {
		background-color: #2196f3;
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='-4 -4 8 8'%3e%3ccircle r='3' fill='rgba(255, 255, 255, 1)'/></svg>");
	}

	#settings-content > div {
		background-color: #bdbdbd5d;
		padding: 0.5rem;
		border-radius: 10px;
		margin-bottom: 1rem;
	}

	h3.text-primary {
		font-weight: 700;
	}

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

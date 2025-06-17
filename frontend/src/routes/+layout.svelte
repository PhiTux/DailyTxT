<script>
	import { blur, slide, fade } from 'svelte/transition';
	import axios from 'axios';
	//import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import '../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Fa from 'svelte-fa';
	import {
		readingMode,
		settings,
		tempSettings,
		autoLoadImagesThisDevice,
		useTrianglify,
		trianglifyOpacity
	} from '$lib/settingsStore.js';
	import { page } from '$app/state';
	import { API_URL } from '$lib/APIurl.js';
	import trianglify from 'trianglify';
	import { tags } from '$lib/tagStore.js';
	import TagModal from '$lib/TagModal.svelte';
	import { alwaysShowSidenav } from '$lib/helpers.js';
	import { templates } from '$lib/templateStore';
	import {
		faRightFromBracket,
		faGlasses,
		faPencil,
		faSliders,
		faTriangleExclamation,
		faTrash
	} from '@fortawesome/free-solid-svg-icons';
	import Tag from '$lib/Tag.svelte';

	let { children } = $props();
	let inDuration = 150;
	let outDuration = 150;

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	axios.interceptors.response.use(
		(response) => {
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

	$effect(() => {
		if ($readingMode === true && page.url.pathname !== '/read') {
			goto('/read');
		} else if ($readingMode === false) {
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
		$tempSettings = JSON.parse(JSON.stringify($settings));
		onThisDayYears = $settings.onThisDayYears.toString();

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

	let onThisDayYears = $state('');
	function getUserSettings() {
		axios
			.get(API_URL + '/users/getUserSettings')
			.then((response) => {
				$settings = response.data;
				onThisDayYears = $settings.onThisDayYears.toString();
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				if ($autoLoadImagesThisDevice === null || $autoLoadImagesThisDevice === undefined) {
					$autoLoadImagesThisDevice = $settings.autoloadImagesByDefault;
				}
			});
	}

	let onThisDayYearsInvalid = $state(false);
	// check if onThisDayYears is valid
	$effect(() => {
		onThisDayYearsInvalid = false;
		if ($tempSettings.useOnThisDay === false) {
			return;
		}

		//regex: years may only contain numbers and commas
		if (onThisDayYears.match(/[^0-9,]/)) {
			onThisDayYearsInvalid = true;
			return;
		}

		onThisDayYears
			.trim()
			.split(',')
			.forEach((year) => {
				if (!Number.isInteger(parseInt(year.trim()))) {
					onThisDayYearsInvalid = true;
				}
				return year;
			});
	});

	let settingsHaveChanged = $derived(
		JSON.stringify($settings) !== JSON.stringify($tempSettings) ||
			JSON.stringify($settings.onThisDayYears) !==
				JSON.stringify(
					onThisDayYears
						.trim()
						.split(',')
						.map((year) => parseInt(year.trim()))
				)
	);

	let isSaving = $state(false);
	function saveUserSettings() {
		if (isSaving) return;
		isSaving = true;

		$tempSettings.onThisDayYears = onThisDayYears
			.trim()
			.split(',')
			.map((year) => parseInt(year.trim()));

		axios
			.post(API_URL + '/users/saveUserSettings', $tempSettings)
			.then((response) => {
				if (response.data.success) {
					$settings = $tempSettings;

					// show toast
					const toast = new bootstrap.Toast(document.getElementById('toastSuccessSaveSettings'));
					toast.show();

					settingsModal.hide();
				} else {
					throw new Error('Error saving settings');
				}
			})
			.catch((error) => {
				console.error(error);

				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSaveSettings'));
				toast.show();
			})
			.finally(() => {
				isSaving = false;
			});
	}

	onMount(() => {
		createBackground();
		calculateResize();
		getUserSettings();
		getTemplates();

		if (page.url.pathname === '/read') {
			$readingMode = true;
		} else if (page.url.pathname === '/write') {
			$readingMode = false;
		}

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

	let selectedTemplate = $state(null);
	let templateName = $state('');
	let templateText = $state('');
	let oldTemplateName = $state('');
	let oldTemplateText = $state('');
	let confirmDeleteTemplate = $state(false);

	function getTemplates() {
		axios
			.get(API_URL + '/logs/getTemplates')
			.then((response) => {
				$templates = response.data;

				selectedTemplate = null;
				updateSelectedTemplate();
			})
			.catch((error) => {
				console.error(error);
			});
	}

	let isSavingTemplate = $state(false);
	function saveTemplate() {
		// check if name or text is empty
		if (templateName === '' || templateText === '') {
			// show toast
			const toast = new bootstrap.Toast(document.getElementById('toastErrorInvalidTemplateEmpty'));
			toast.show();
			return;
		}

		// check if template name already exists
		for (let i = 0; i < $templates.length; i++) {
			if ($templates[i].name === templateName && selectedTemplate !== i) {
				// show toast
				const toast = new bootstrap.Toast(
					document.getElementById('toastErrorInvalidTemplateDouble')
				);
				toast.show();
				return;
			}
		}

		if (isSavingTemplate) return;
		isSavingTemplate = true;

		if (selectedTemplate === '-1') {
			// add new template
			$templates.push({ name: templateName, text: templateText });
		} else {
			// update existing template
			$templates[selectedTemplate].name = templateName;
			$templates[selectedTemplate].text = templateText;
		}

		axios
			.post(API_URL + '/logs/saveTemplates', {
				templates: $templates
			})
			.then((response) => {
				if (response.data.success) {
					getTemplates();

					// show toast
					const toast = new bootstrap.Toast(document.getElementById('toastSuccessSaveTemplate'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);

				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSaveTemplates'));
				toast.show();
			})
			.finally(() => {
				isSavingTemplate = false;
			});
	}

	let isDeletingTemplate = $state(false);
	function deleteTemplate() {
		if (selectedTemplate === null || selectedTemplate === '-1') return;

		if (isDeletingTemplate) return;
		isDeletingTemplate = true;

		// remove template from list
		$templates.splice(selectedTemplate, 1);

		axios
			.post(API_URL + '/logs/saveTemplates', {
				templates: $templates
			})
			.then((response) => {
				if (response.data.success) {
					getTemplates();

					// show toast
					const toast = new bootstrap.Toast(
						document.getElementById('toastSuccessDeletingTemplate')
					);
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorDeletingTemplate'));
				toast.show();
			})
			.finally(() => {
				isDeletingTemplate = false;
				confirmDeleteTemplate = false;
			});
	}

	function updateSelectedTemplate() {
		if (selectedTemplate === '-1' || selectedTemplate === null) {
			// new template
			templateName = '';
			templateText = '';
		} else {
			// existing template
			templateName = $templates[selectedTemplate].name;
			templateText = $templates[selectedTemplate].text;
		}
		oldTemplateName = templateName;
		oldTemplateText = templateText;

		confirmDeleteTemplate = false;
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
				openSettingsModal();
			});
	}

	$effect(() => {
		if ($autoLoadImagesThisDevice === null || $autoLoadImagesThisDevice === undefined) {
			return;
		}

		localStorage.setItem('autoLoadImagesThisDevice', $autoLoadImagesThisDevice);
	});
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
								<div id="appearance">
									<h3 class="text-primary">Aussehen</h3>
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
								</div>

								<div id="functions">
									<h3 class="text-primary">Funktionen</h3>

									<div id="autoLoadImages">
										{#if $tempSettings.setAutoloadImagesPerDevice !== $settings.setAutoloadImagesPerDevice || $tempSettings.autoloadImagesByDefault !== $settings.autoloadImagesByDefault}
											<div class="unsaved-changes" transition:slide></div>
										{/if}

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
												bind:checked={$tempSettings.setAutoloadImagesPerDevice}
												type="checkbox"
												role="switch"
												id="setImageLoadingPerDeviceSwitch"
											/>
											<label class="form-check-label" for="setImageLoadingPerDeviceSwitch">
												Für jedes Gerät einzeln festlegen, ob die Bilder automatisch geladen werden
												sollen</label
											>
										</div>

										<div class="form-check form-switch ms-3">
											<input
												class="form-check-input"
												bind:checked={$autoLoadImagesThisDevice}
												type="checkbox"
												role="switch"
												id="loadImagesThisDeviceSwitch"
												disabled={!$tempSettings.setAutoloadImagesPerDevice}
											/>
											<label class="form-check-label" for="loadImagesThisDeviceSwitch">
												{#if $autoLoadImagesThisDevice}
													Bilder werden auf <b>diesem Gerät</b> automatisch geladen
												{:else}
													Bilder werden auf <b>diesem Gerät <u>nicht</u></b> automatisch geladen
												{/if}</label
											>
										</div>

										<div class="form-check form-switch mt-3">
											<input
												class="form-check-input"
												bind:checked={$tempSettings.autoloadImagesByDefault}
												type="checkbox"
												role="switch"
												id="autoLoadImagesSwitch"
												disabled={$tempSettings.setAutoloadImagesPerDevice}
											/>
											<label class="form-check-label" for="autoLoadImagesSwitch">
												{#if $tempSettings.autoloadImagesByDefault}
													Bilder werden (auf jedem Gerät) automatisch geladen
												{:else}
													Bilder werden (auf jedem Gerät) <b>nicht</b> automatisch geladen
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
										{#if $tempSettings.useOnThisDay !== $settings.useOnThisDay || JSON.stringify(onThisDayYears
													.trim()
													.split(',')
													.map( (year) => parseInt(year.trim()) )) !== JSON.stringify($settings.onThisDayYears)}
											<div class="unsaved-changes" transition:slide></div>
										{/if}

										<h5>Ein Blick zurück</h5>
										<ul>
											<li>
												Lege fest, aus welchen vergangenen Jahren Tagebucheinträge desselben
												Kalendertags angezeigt werden sollen.
											</li>
											<li>Gilt nur für den Schreibmodus.</li>
											<li>
												Beispiel: <code>1,5,10</code> sorgt dafür, dass du unter dem Textfeld noch die
												Einträge von vor 1 Jahr, vor 5 Jahren und vor 10 Jahren angezeigt bekommst (sofern
												vorhanden).
											</li>
										</ul>
										<div class="form-check form-switch">
											<input
												class="form-check-input"
												bind:checked={$tempSettings.useOnThisDay}
												type="checkbox"
												role="switch"
												id="useOnThisDaySwitch"
											/>
											<label class="form-check-label" for="useOnThisDaySwitch">
												{#if $tempSettings.useOnThisDay}
													Einträge desselben Tags aus der Vergangenheit anzeigen
												{:else}
													Einträge desselben Tags aus der Vergangenheit <b>nicht</b> anzeigen
												{/if}</label
											>
										</div>

										<div>
											<!-- <label for="useOnThisDayYears" class="form-label">Jahre</label> -->
											<input
												type="text"
												id="useOnThisDayYears"
												class="form-control {onThisDayYearsInvalid ? 'is-invalid' : ''}"
												aria-describedby="useOnThisDayHelpBlock"
												disabled={!$tempSettings.useOnThisDay}
												placeholder="Jahre, mit Komma getrennt"
												bind:value={onThisDayYears}
												invalid
											/>
											{#if onThisDayYearsInvalid}
												<div class="alert alert-danger mt-2" role="alert" transition:slide>
													Bitte nur Zahlen eingeben, die durch Kommas getrennt sind.
												</div>
											{/if}
											<div id="useOnThisDayHelpBlock" class="form-text">
												Trage hier alle vergangenen Jahre ein, die angezeigt werden sollen.
												Beispiel: <code>1,5,10</code>. Benutze Komma zur Trennung, verzichte auf
												Leerzeichen.
											</div>
										</div>
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

									<div id="templates">
										<h3 class="text-primary">Vorlagen</h3>
										<div>
											{#if oldTemplateName !== templateName || oldTemplateText !== templateText}
												<div class="unsaved-changes" transition:slide></div>
											{/if}

											<div class="d-flex flex-column">
												<select
													bind:value={selectedTemplate}
													class="form-select"
													aria-label="Select template"
													onchange={updateSelectedTemplate}
												>
													<option value="-1" selected={selectedTemplate === '-1'}>
														Neue Vorlage erstellen...
													</option>
													{#each $templates as template, index}
														<option value={index} selected={index === selectedTemplate}>
															{template.name}
														</option>
													{/each}
												</select>
											</div>

											<hr />

											{#if confirmDeleteTemplate}
												<div transition:slide class="d-flex flex-row align-items-center mb-2">
													<span
														>Vorlage <b>{$templates[selectedTemplate].name}</b> wirklich löschen?</span
													>
													<button
														type="button"
														class="btn btn-secondary ms-2"
														onclick={() => (confirmDeleteTemplate = false)}>Abbrechen</button
													>
													<button
														type="button"
														class="btn btn-danger ms-2"
														onclick={() => {
															deleteTemplate();
														}}
														disabled={isDeletingTemplate}
														>Löschen
														{#if isDeletingTemplate}
															<span
																class="spinner-border spinner-border-sm ms-2"
																role="status"
																aria-hidden="true"
															></span>
														{/if}
													</button>
												</div>
											{/if}
											<div class="d-flex flex-row">
												<input
													disabled={selectedTemplate === null}
													type="text"
													bind:value={templateName}
													class="form-control"
													placeholder="Name der Vorlage"
												/>
												<button
													disabled={selectedTemplate === '-1' || selectedTemplate === null}
													type="button"
													class="btn btn-outline-danger ms-5"
													onclick={() => {
														confirmDeleteTemplate = !confirmDeleteTemplate;
													}}><Fa fw icon={faTrash} /></button
												>
											</div>
											<textarea
												disabled={selectedTemplate === null}
												bind:value={templateText}
												class="form-control mt-2"
												rows="10"
												placeholder="Inhalt der Vorlage"
											>
											</textarea>
											<div class="d-flex justify-content-end">
												<button
													disabled={(oldTemplateName === templateName &&
														oldTemplateText === templateText) ||
														isSavingTemplate}
													type="button"
													class="btn btn-primary mt-2"
													onclick={saveTemplate}
												>
													Vorlage speichern
													{#if isSavingTemplate}
														<span
															class="spinner-border spinner-border-sm ms-2"
															role="status"
															aria-hidden="true"
														></span>
													{/if}
												</button>
											</div>
										</div>
									</div>

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
				</div>
				<div class="modal-footer">
					{#if settingsHaveChanged}
						<div class="footer-unsaved-changes" transition:fade={{ duration: 100 }}>
							Ungespeicherte Änderungen!
						</div>
					{/if}
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Abbrechen</button>
					<button
						type="button"
						class="btn btn-primary"
						onclick={saveUserSettings}
						disabled={isSaving || !settingsHaveChanged}
						>Speichern
						{#if isSaving}
							<span class="spinner-border spinner-border-sm ms-2" role="status" aria-hidden="true"
							></span>
						{/if}
					</button>
				</div>
				<!-- </div> -->
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

			<div
				id="toastSuccessSaveSettings"
				class="toast align-items-center text-bg-success"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Einstellungen gespeichert!</div>
				</div>
			</div>

			<div
				id="toastErrorSaveSettings"
				class="toast align-items-center text-bg-danger"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Fehler beim Speichern der Einstellungen!</div>
				</div>
			</div>

			<div
				id="toastErrorInvalidTemplateEmpty"
				class="toast align-items-center text-bg-danger"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Name oder Inhalt einer Vorlage dürfen nicht leer sein!</div>
				</div>
			</div>

			<div
				id="toastErrorInvalidTemplateDouble"
				class="toast align-items-center text-bg-danger"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Name der Vorlage existiert bereits</div>
				</div>
			</div>

			<div
				id="toastSuccessSaveTemplate"
				class="toast align-items-center text-bg-success"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Vorlage gespeichert</div>
				</div>
			</div>

			<div
				id="toastErrorDeletingTemplate"
				class="toast align-items-center text-bg-danger"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Fehler beim Löschen der Vorlage</div>
				</div>
			</div>

			<div
				id="toastSuccessDeletingTemplate"
				class="toast align-items-center text-bg-success"
				role="alert"
				aria-live="assertive"
				aria-atomic="true"
			>
				<div class="d-flex">
					<div class="toast-body">Vorlage gelöscht</div>
				</div>
			</div>
		</div>
	</div>
</main>

<style>
	.footer-unsaved-changes {
		background-color: orange;
		color: black;
		padding: 0.25rem 0.5rem;
		border-radius: 10px;
		margin-left: auto;
		margin-right: 2rem;
		font-style: italic;
	}

	div:has(> .unsaved-changes) {
		outline: 1px solid orange;
	}

	.unsaved-changes {
		background-color: orange;
		margin-top: -0.5rem;
		margin-left: -0.5rem;
		margin-right: -0.5rem;
		border-top-left-radius: 10px;
		border-top-right-radius: 10px;
		padding-left: 0.5rem;
		margin-bottom: 0.5rem;
	}

	.unsaved-changes::before {
		content: 'Ungespeicherte Änderungen';
	}

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

	.settings-content {
		padding: 0.5rem;
	}

	#settings-content > div > div {
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

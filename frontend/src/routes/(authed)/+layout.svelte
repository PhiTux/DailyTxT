<script>
	import * as bootstrap from 'bootstrap';
	import Fa from 'svelte-fa';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import {
		readingMode,
		settings,
		tempSettings,
		autoLoadImagesThisDevice
	} from '$lib/settingsStore.js';
	import { API_URL } from '$lib/APIurl.js';
	import { tags } from '$lib/tagStore.js';
	import TagModal from '$lib/TagModal.svelte';
	import { alwaysShowSidenav, generateNeonMesh, loadFlagEmoji } from '$lib/helpers.js';
	import { templates } from '$lib/templateStore';
	import {
		faRightFromBracket,
		faGlasses,
		faPencil,
		faSliders,
		faTriangleExclamation,
		faTrash,
		faCopy,
		faCheck,
		faSun,
		faMoon
	} from '@fortawesome/free-solid-svg-icons';
	import Tag from '$lib/Tag.svelte';
	import SelectTimezone from '$lib/SelectTimezone.svelte';
	import axios from 'axios';
	import { page } from '$app/state';
	import { blur, slide, fade } from 'svelte/transition';
	import { T, getTranslate, getTolgee } from '@tolgee/svelte';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	let { children } = $props();
	let inDuration = 150;
	let outDuration = 150;

	$effect(() => {
		if ($readingMode === true && page.url.pathname !== '/read') {
			goto('/read');
		} else if ($readingMode === false) {
			goto('/write');
		}
	});

	onMount(() => {
		getUserSettings();
		getTemplates();

		if (page.url.pathname === '/read') {
			$readingMode = true;
		} else if (page.url.pathname === '/write') {
			$readingMode = false;
		}

		const scrollSpy = new bootstrap.ScrollSpy(document.body, {
			target: '#settings-nav'
		});

		document.getElementById('settingsModal').addEventListener('shown.bs.modal', function () {
			const height = document.getElementById('modal-body').clientHeight;
			document.getElementById('settings-content').style.height = 'calc(' + height + 'px - 2rem)';
			document.getElementById('settings-nav').style.height = 'calc(' + height + 'px - 2rem)';
			document.getElementById('settings-content').style.overflowY = 'auto';

			setTimeout(() => {
				const dataSpyList = document.querySelectorAll('[data-bs-spy="scroll"]');
				dataSpyList.forEach((dataSpyEl) => {
					//bootstrap.ScrollSpy.getInstance(dataSpyEl).refresh();
					scrollSpy.refresh();
				});
			}, 400);
		});

		document.getElementById('settingsModal').addEventListener('hidden.bs.modal', function () {
			backupCodes = [];
		});
	});

	function logout(errorCode) {
		axios
			.get(API_URL + '/users/logout')
			.then((response) => {
				localStorage.removeItem('user');
				if (errorCode) {
					goto(`/login?error=${errorCode}`);
				} else {
					goto('/login');
				}
			})
			.catch((error) => {
				console.error(error);

				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLogout'));
				toast.show();
			});
	}

	let settingsModal;
	function openSettingsModal() {
		$tempSettings = JSON.parse(JSON.stringify($settings));
		aLookBackYears = $settings.aLookBackYears.toString();

		settingsModal = new bootstrap.Modal(document.getElementById('settingsModal'));
		settingsModal.show();
	}

	/* Important for development: convenient modal-handling with HMR */
	if (import.meta.hot) {
		import.meta.hot.dispose(() => {
			document.querySelectorAll('.modal-backdrop').forEach((el) => el.remove());
		});
	}

	let aLookBackYears = $state('');
	let isGettingUserSettings = $state(false);
	function getUserSettings() {
		if (isGettingUserSettings) return;
		isGettingUserSettings = true;

		axios
			.get(API_URL + '/users/getUserSettings')
			.then((response) => {
				$settings = response.data;
				$tempSettings = JSON.parse(JSON.stringify($settings));
				aLookBackYears = $settings.aLookBackYears.toString();
				updateLanguage();

				// set background
				setBackground();
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				if ($autoLoadImagesThisDevice === null || $autoLoadImagesThisDevice === undefined) {
					$autoLoadImagesThisDevice = $settings.autoloadImagesByDefault;
				}
				isGettingUserSettings = false;
			});
	}

	let aLookBackYearsInvalid = $state(false);
	// check if aLookBackYears is valid
	$effect(() => {
		aLookBackYearsInvalid = false;
		if ($tempSettings.useALookBack === false) {
			return;
		}

		//regex: years may only contain numbers and commas
		if (aLookBackYears.match(/[^0-9,]/)) {
			aLookBackYearsInvalid = true;
			return;
		}

		aLookBackYears
			.trim()
			.split(',')
			.forEach((year) => {
				if (!Number.isInteger(parseInt(year.trim()))) {
					aLookBackYearsInvalid = true;
				}
				return year;
			});
	});

	// check if settings have changed (special parsing of aLookBackYears needed)
	let settingsHaveChanged = $derived(
		JSON.stringify($settings) !== JSON.stringify($tempSettings) ||
			JSON.stringify($settings.aLookBackYears) !==
				JSON.stringify(
					aLookBackYears
						.trim()
						.split(',')
						.map((year) => parseInt(year.trim()))
				)
	);

	function updateLanguage() {
		if ($settings.useBrowserLanguage) {
			let browserLanguage = tolgeesMatchForBrowserLanguage();
			$tolgee.changeLanguage(
				browserLanguage === '' ? $tolgee.getInitialOptions().defaultLanguage : browserLanguage
			);
		} else {
			$tolgee.changeLanguage($settings.language);
		}
	}

	// Check if Tolgee contains the browser language
	// returns "" if the browser language is not available
	// return the language code if it is available
	function tolgeesMatchForBrowserLanguage() {
		const browserLanguage = window.navigator.language;
		const availableLanguages = $tolgee
			.getInitialOptions()
			.availableLanguages.map((lang) => lang.toLowerCase());

		// check if tolgee contains an exact match for the browser language OR a match for the first two characters (e.g., 'en' for 'en-US')
		if (availableLanguages.includes(browserLanguage.toLowerCase())) {
			return browserLanguage;
		}
		if (browserLanguage.length > 2) {
			const shortBrowserLanguage = browserLanguage.slice(0, 2);
			if (availableLanguages.includes(shortBrowserLanguage.toLowerCase())) {
				return shortBrowserLanguage;
			}
		}

		return '';
	}

	function setBackground() {
		if ($settings.background === 'monochrome') {
			document.querySelector('.background').style.background = '';
			document.body.style.backgroundColor = $settings.monochromeBackgroundColor;
		} else if ($settings.background === 'gradient') {
			document.body.style.backgroundColor = '';
			generateNeonMesh();
		}
	}

	let isSaving = $state(false);
	function saveUserSettings() {
		if (isSaving) return;
		isSaving = true;

		$tempSettings.aLookBackYears = aLookBackYears
			.trim()
			.split(',')
			.map((year) => parseInt(year.trim()));

		axios
			.post(API_URL + '/users/saveUserSettings', $tempSettings)
			.then((response) => {
				if (response.data.success) {
					$settings = $tempSettings;

					// update language
					updateLanguage();

					// set background
					setBackground();

					// show toast
					const toast = new bootstrap.Toast(document.getElementById('toastSuccessSaveSettings'));
					toast.show();

					settingsModal.hide();
				} else {
					console.error('Error saving settings');
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
		if (selectedTemplate === '-1' || selectedTemplate === null || $templates.length === 0) {
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

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmNewPassword = $state('');
	let changePasswordNotEqual = $state(false);
	let isChangingPassword = $state(false);
	let changingPasswordSuccess = $state(false);
	let changingPasswordError = $state(false);
	let changingPasswordIncorrect = $state(false);

	function changePassword() {
		changePasswordNotEqual = false;
		changingPasswordSuccess = false;
		changingPasswordError = false;
		changingPasswordIncorrect = false;

		if (newPassword !== confirmNewPassword) {
			changePasswordNotEqual = true;
			return;
		}

		if (isChangingPassword) return;
		isChangingPassword = true;

		axios
			.post(API_URL + '/users/changePassword', {
				old_password: currentPassword,
				new_password: newPassword
			})
			.then((response) => {
				if (response.data.success) {
					changingPasswordSuccess = true;
				} else {
					changingPasswordError = true;
					console.error('Error changing password');
					if (response.data.password_incorrect) {
						changingPasswordIncorrect = true;
					}
				}
			})
			.catch((error) => {
				console.error(error);
				console.log('Error on Changing password:', error.response.data.message);
				changingPasswordError = true;
			})
			.finally(() => {
				isChangingPassword = false;
			});
	}

	let showConfirmDeleteAccount = $state(false);
	let deleteAccountPassword = $state('');
	let isDeletingAccount = $state(false);
	let deleteAccountPasswordIncorrect = $state(false);
	let showDeleteAccountSuccess = $state(false);

	function deleteAccount() {
		if (isDeletingAccount) return;
		isDeletingAccount = true;

		axios
			.post(API_URL + '/users/deleteAccount', {
				password: deleteAccountPassword
			})
			.then((response) => {
				if (response.data.success) {
					showDeleteAccountSuccess = true;

					// close modal
					settingsModal.hide();

					logout(410); // HTTP 410 Gone => Account deleted
				} else if (response.data.password_incorrect) {
					deleteAccountPasswordIncorrect = true;
				} else {
					console.error('Error deleting account');
					console.error(response.data);
				}
			})
			.catch((error) => {
				console.error(error);
				deleteAccountPasswordIncorrect = true;
			})
			.finally(() => {
				isDeletingAccount = false;
				showConfirmDeleteAccount = false;
				deleteAccountPassword = '';
			});
	}

	let backupCodesPassword = $state('');
	let isGeneratingBackupCodes = $state(false);
	let backupCodes = $state([]);
	let showBackupCodesPasswordIncorrect = $state(false);
	let codesCopiedSuccess = $state(false);
	let showBackupCodesError = $state(false);

	function createBackupCodes() {
		if (isGeneratingBackupCodes) return;
		isGeneratingBackupCodes = true;

		showBackupCodesPasswordIncorrect = false;
		showBackupCodesError = false;
		backupCodes = [];

		axios
			.post(API_URL + '/users/createBackupCodes', {
				password: backupCodesPassword
			})
			.then((response) => {
				if (response.data.success) {
					backupCodes = response.data.backup_codes;
				} else {
					console.error('Error creating backup codes');
					console.error(response.data);
					showBackupCodesError = true;
				}
			})
			.catch((error) => {
				console.error(error);
				const toast = new bootstrap.Toast(document.getElementById('toastErrorCreateBackupCodes'));
				toast.show();
			})
			.finally(() => {
				isGeneratingBackupCodes = false;
			});
	}

	function copyBackupCodes() {
		if (backupCodes.length === 0) return;

		const codesText = backupCodes.join('\n');
		navigator.clipboard.writeText(codesText).then(
			() => {
				// Show success checkmark for 3 seconds
				codesCopiedSuccess = true;
				setTimeout(() => {
					codesCopiedSuccess = false;
				}, 3000);
			},
			(err) => {
				console.error('Failed to copy backup codes: ', err);
			}
		);
	}

	let exportPeriod = $state('periodAll');
	let exportStartDate = $state('');
	let exportEndDate = $state('');
	let exportImagesInHTML = $state(true);
	let exportSplit = $state('aio');
	let exportTagsInHTML = $state(true);
	let isExporting = $state(false);

	function exportData() {
		if (isExporting) return;
		isExporting = true;

		axios
			.get(API_URL + '/logs/exportData', {
				params: {
					period: exportPeriod,
					startDate: exportStartDate,
					endDate: exportEndDate,
					imagesInHTML: exportImagesInHTML,
					split: exportSplit,
					tagsInHTML: exportTagsInHTML
				},
				responseType: 'blob' // Expect a binary response
			})
			.then((response) => {
				const blob = new Blob([response.data], { type: 'application/zip' });
				const url = window.URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;

				const contentDisposition = response.headers['content-disposition'];
				let filename = 'DailyTxT_Export.zip';
				if (contentDisposition) {
					const filenameMatch = contentDisposition.match(/filename="(.+)"/);
					if (filenameMatch) {
						filename = filenameMatch[1];
					}
				}

				a.download = filename;
				document.body.appendChild(a);
				a.click();
				a.remove();
				window.URL.revokeObjectURL(url);
			})
			.catch((error) => {
				console.error(error);

				// show toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorExportData'));
				toast.show();
			})
			.finally(() => {
				isExporting = false;
			});
	}
</script>

<div class="d-flex flex-column h-100">
	<nav class="navbar navbar-expand-lg glass">
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
				<button class="btn btn-outline-secondary" onclick={() => logout(null)}
					><Fa icon={faRightFromBracket} /></button
				>
			</div>
		</div>
	</nav>

	{#key page.data}
		<div
			class="transition-wrapper overflow-y-auto"
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
		<div class="modal-content shadow-lg glass">
			<div class="modal-header">
				<h1>{$t('settings.title')}</h1>
				<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
			</div>
			<div class="modal-body" id="modal-body">
				<div class="row">
					<div class="col-4 overflow-y-auto">
						<nav class="flex-column align-items-stretch" id="settings-nav">
							<nav class="nav nav-pills flex-column">
								<a class="nav-link mb-1" href="#appearance">Aussehen</a>
								<a class="nav-link mb-1" href="#functions">Funktionen</a>

								<a class="nav-link mb-1" href="#tags">Tags</a>
								<a class="nav-link mb-1" href="#templates">Vorlagen</a>
								<a class="nav-link mb-1" href="#data">Daten</a>
								<a class="nav-link mb-1" href="#security">Sicherheit</a>
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
								<h3 class="text-primary">🎨 Aussehen</h3>
								<div id="lightdark">
									{#if $tempSettings.darkModeAutoDetect !== $settings.darkModeAutoDetect || $tempSettings.useDarkMode !== $settings.useDarkMode}
										<div class="unsaved-changes" transition:slide></div>
									{/if}
									<h5>Light-/Dark-Mode</h5>
									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="darkMode"
											id="darkModeAutoTrue"
											value={true}
											bind:group={$tempSettings.darkModeAutoDetect}
										/>
										<label class="form-check-label" for="darkModeAutoTrue">
											Light-/Dark-Mode automatisch erkennen (aktuell:
											{#if window.matchMedia('(prefers-color-scheme: dark)').matches}
												<b>Dark <Fa icon={faMoon} /></b>
											{:else}
												<b>Light <Fa icon={faSun} /></b>
											{/if})
										</label>
									</div>
									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="darkMode"
											id="darkModeAutoFalse"
											value={false}
											bind:group={$tempSettings.darkModeAutoDetect}
										/>
										<label class="form-check-label" for="darkModeAutoFalse">
											Light-/Dark-Mode manuell festlegen
										</label>
										{#if $tempSettings.darkModeAutoDetect === false}
											<div class="form-check form-switch d-flex flex-row ps-0" transition:slide>
												<label class="form-check-label me-5" for="darkModeSwitch">
													<Fa icon={faSun} />
												</label>
												<input
													class="form-check-input"
													bind:checked={$tempSettings.useDarkMode}
													type="checkbox"
													role="switch"
													id="darkModeSwitch"
												/>
												<label class="form-check-label ms-2" for="darkModeSwitch">
													<Fa icon={faMoon} />
												</label>
											</div>
										{/if}
									</div>
								</div>
								<div id="background">
									{#if $tempSettings.background !== $settings.background || $tempSettings.monochromeBackgroundColor !== $settings.monochromeBackgroundColor}
										<div class="unsaved-changes" transition:slide></div>
									{/if}

									<h5>Hintergrund</h5>
									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="background"
											id="background_gradient"
											value={'gradient'}
											bind:group={$tempSettings.background}
										/>
										<label class="form-check-label" for="background_gradient">
											Farbverlauf (wird bei jedem Seitenaufruf neu generiert)
										</label>
									</div>
									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="background"
											id="background_monochrome"
											value={'monochrome'}
											bind:group={$tempSettings.background}
										/>
										<label class="form-check-label" for="background_monochrome"> Einfarbig </label>
										{#if $tempSettings.background === 'monochrome'}
											<input
												transition:slide
												class="form-control form-control-color"
												bind:value={$tempSettings.monochromeBackgroundColor}
												type="color"
											/>
										{/if}
									</div>
								</div>
							</div>

							<div id="functions">
								<h3 class="text-primary">🛠️ Funktionen</h3>

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
									{#if $tempSettings.useBrowserLanguage !== $settings.useBrowserLanguage || $tempSettings.language !== $settings.language}
										<div class="unsaved-changes" transition:slide></div>
									{/if}
									<h5>🌐 Sprache</h5>
									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="language"
											id="language_auto"
											value={true}
											bind:group={$tempSettings.useBrowserLanguage}
										/>
										<label class="form-check-label" for="language_auto">
											Sprache anhand des Browsers ermitteln (aktuell: <code
												>{window.navigator.language}</code
											>
											{#if tolgeesMatchForBrowserLanguage() !== '' && tolgeesMatchForBrowserLanguage() !== window.navigator.language}
												➔ <code>{tolgeesMatchForBrowserLanguage()}</code> wird verwendet
											{/if}
											)
										</label>
										{#if $tempSettings.useBrowserLanguage && tolgeesMatchForBrowserLanguage() === ''}
											<div
												transition:slide
												disabled={!$settings.useBrowserLanguage}
												class="alert alert-danger"
												role="alert"
											>
												Die Sprache <code>{window.navigator.language}</code> ist nicht verfügbar. Es
												wird die Standardsprache
												<code>{$tolgee.getInitialOptions().defaultLanguage}</code> verwendet.
											</div>
										{/if}
									</div>
									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="language"
											id="language_manual"
											value={false}
											bind:group={$tempSettings.useBrowserLanguage}
										/>
										<label class="form-check-label" for="language_manual">
											Sprache dauerhaft festlegen
											{#if !$tempSettings.useBrowserLanguage}
												<select
													transition:slide
													class="form-select"
													bind:value={$tempSettings.language}
													disabled={$tempSettings.useBrowserLanguage}
												>
													{#each $tolgee.getInitialOptions().availableLanguages as lang}
														<option value={lang}>{loadFlagEmoji(lang)} {lang}</option>
													{/each}
												</select>
											{/if}
										</label>
									</div>
								</div>
								<div id="timezone">
									{#if $tempSettings.useBrowserTimezone !== $settings.useBrowserTimezone || ($tempSettings.timezone !== undefined && $tempSettings.timezone?.value !== $settings.timezone?.value)}
										<div class="unsaved-changes" transition:slide></div>
									{/if}
									<h5>Zeitzone</h5>
									Stelle die Zeitzone ein, die für den Timestamp ("Geschrieben am") genutzt werden soll.

									<div class="form-check mt-2">
										<input
											class="form-check-input"
											type="radio"
											name="timezone"
											id="timezone1"
											value={true}
											bind:group={$tempSettings.useBrowserTimezone}
										/>
										<label class="form-check-label" for="timezone1">
											Zeitzone automatisch anhand des Browsers ermitteln.
										</label>
										<br />
										Aktuell: <code>{new Intl.DateTimeFormat().resolvedOptions().timeZone}</code>
									</div>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="timezone"
											id="timezone2"
											value={false}
											bind:group={$tempSettings.useBrowserTimezone}
										/>
										<label class="form-check-label" for="timezone2">
											Für diesen Account immer die folgende Zeitzone verwenden:
										</label>
										<br />
										<SelectTimezone />
										{#if !$tempSettings.useBrowserTimezone}
											<span transition:fade>
												Ausgewählt: <code>{$tempSettings.timezone}</code>
											</span>
										{/if}
									</div>

									<div class="form-text mt-3">
										Wenn man auf Reisen ist, kann es sinnvoll sein, die Zeitzone anhand des Browsers
										zu ermitteln. Dann werden Datum und Uhrzeit am Zielort vorraussichtlich besser
										erkannt.<br />
										Wenn man hingegen zuhause im privaten Browser teils andere Zeitzonen (z. B. immer
										UTC) verwendet, kann es sinnvoll sein, hier eine bestimmte Zeitzone festzulegen.
									</div>
								</div>

								<div id="aLookBack">
									{#if $tempSettings.useALookBack !== $settings.useALookBack || JSON.stringify(aLookBackYears
												.trim()
												.split(',')
												.map( (year) => parseInt(year.trim()) )) !== JSON.stringify($settings.aLookBackYears)}
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
											bind:checked={$tempSettings.useALookBack}
											type="checkbox"
											role="switch"
											id="useALookBackSwitch"
										/>
										<label class="form-check-label" for="useALookBackSwitch">
											{#if $tempSettings.useALookBack}
												Einträge desselben Tags aus der Vergangenheit anzeigen
											{:else}
												Einträge desselben Tags aus der Vergangenheit <b>nicht</b> anzeigen
											{/if}</label
										>
									</div>

									<div>
										<input
											type="text"
											id="useALookBackYears"
											class="form-control {aLookBackYearsInvalid ? 'is-invalid' : ''}"
											aria-describedby="useALookBackHelpBlock"
											disabled={!$tempSettings.useALookBack}
											placeholder="Jahre, mit Komma getrennt"
											bind:value={aLookBackYears}
											invalid
										/>
										{#if aLookBackYearsInvalid}
											<div class="alert alert-danger mt-2" role="alert" transition:slide>
												Bitte nur Zahlen eingeben, die durch Kommas getrennt sind.
											</div>
										{/if}
										<div id="useALookBackHelpBlock" class="form-text">
											Trage hier alle vergangenen Jahre ein, die angezeigt werden sollen. Beispiel: <code
												>1,5,10</code
											>. Benutze Komma zur Trennung, verzichte auf Leerzeichen.
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
							</div>

							<div id="tags">
								<h3 class="text-primary">#️⃣ Tags</h3>
								<div>
									Hier können Tags bearbeitet oder auch vollständig aus DailyTxT gelöscht werden.
									{#if $tags.length === 0}
										<div class="alert alert-info my-2" role="alert">
											Es sind noch keine Tags vorhanden. Erstelle einen neuen Tag im Schreibmodus.
										</div>
									{/if}
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
												<div
													class="alert alert-danger align-items-center"
													role="alert"
													transition:slide
												>
													<div>
														<Fa icon={faTriangleExclamation} fw /> <b>Tag dauerhaft löschen?</b>
														Dies kann einen Moment dauern, da jeder Eintrag nach potenziellen Verlinkungen
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
							</div>

							<div id="templates">
								<h3 class="text-primary">📝 Vorlagen</h3>
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
												>Vorlage <b>{$templates[selectedTemplate]?.name}</b> wirklich löschen?</span
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
								<h3 class="text-primary">📁 Daten</h3>
								<div>
									<h5>Export</h5>
									Exportiere deine Einträge in einer formatierten HTML-Datei. Bilder werden wahlweise
									in der HTML eingebunden. Alle Dateien werden außerdem in einer Zip-Datei bereitgestellt.

									<h6>Zeitraum</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="period"
											value="periodAll"
											id="periodAll"
											bind:group={exportPeriod}
										/>
										<label class="form-check-label" for="periodAll">Gesamter Zeitraum</label>
									</div>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="period"
											value="periodVariable"
											id="periodVariable"
											bind:group={exportPeriod}
										/>
										<label class="form-check-label" for="periodVariable">Variabler Zeitraum</label>
										{#if exportPeriod === 'periodVariable'}
											<div class="d-flex flex-row" transition:slide>
												<div class="me-2">
													<label for="exportStartDate">Von:</label>
													<input
														type="date"
														class="form-control me-2"
														id="exportStartDate"
														bind:value={exportStartDate}
													/>
												</div>
												<div>
													<label for="exportEndDate">Bis:</label>
													<input
														type="date"
														class="form-control"
														id="exportEndDate"
														bind:value={exportEndDate}
													/>
												</div>
											</div>
											{#if exportStartDate !== '' && exportEndDate !== '' && exportStartDate > exportEndDate}
												<div class="alert alert-danger mt-2" role="alert" transition:slide>
													Das Startdatum muss vor dem Enddatum liegen!
												</div>
											{/if}
										{/if}
									</div>

									<h6>Anzahl der HTML-Dokumente</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="split"
											value="aio"
											id="splitAIO"
											bind:group={exportSplit}
										/>
										<label class="form-check-label" for="splitAIO">Eine einzige HTML</label>
									</div>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="split"
											value="year"
											id="splitYear"
											bind:group={exportSplit}
										/>
										<label class="form-check-label" for="splitYear">Eine HTML pro Jahr</label>
									</div>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="split"
											value="month"
											id="splitMonth"
											bind:group={exportSplit}
										/>
										<label class="form-check-label" for="splitMonth">Eine HTML pro Monat</label>
									</div>

									<h6>Bilder in HTML anzeigen</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="checkbox"
											name="images"
											id="exportImagesInHTML"
											bind:checked={exportImagesInHTML}
										/>
										<label class="form-check-label" for="exportImagesInHTML">
											Bilder direkt unter dem Text anzeigen <em
												>(werden zudem immer als Link bereitgestellt)</em
											>
										</label>
									</div>

									<h6>Tags drucken</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="checkbox"
											id="exportTagsInHTML"
											bind:checked={exportTagsInHTML}
										/>
										<label class="form-check-label" for="exportTagsInHTML"
											>Tags in der HTML anzeigen</label
										>
									</div>

									<div class="form-text">
										<u>Hinweise:</u>
										<ul>
											<li>Die HTML wird keinen Verlauf der einzelnen Tage enthalten.</li>
											<li>
												Ein Re-Import ist nicht möglich. Diese Funktion dient nicht dem Backup,
												sondern rein dem Export, um eine einfach lesbare HTML-Datei zu erhalten.
											</li>
										</ul>
									</div>
									<button
										class="btn btn-primary mt-3"
										onclick={exportData}
										data-sveltekit-noscroll
										disabled={isExporting ||
											(exportPeriod === 'periodVariable' &&
												(exportStartDate === '' || exportEndDate === ''))}
									>
										Exportieren
										{#if isExporting}
											<div class="spinner-border spinner-border-sm ms-2" role="status">
												<span class="visually-hidden">Loading...</span>
											</div>
										{/if}
									</button>
								</div>
								<div><h5>Import</h5></div>
							</div>

							<div id="security">
								<h3 class="text-primary">🔒 Sicherheit</h3>
								<div>
									<h5>Password ändern</h5>
									<form onsubmit={changePassword}>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="currentPassword"
												placeholder="Aktuelles Passwort"
												bind:value={currentPassword}
											/>
											<label for="currentPassword">Aktuelles Passwort</label>
										</div>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="newPassword"
												placeholder="Neues Passwort"
												bind:value={newPassword}
											/>
											<label for="newPassword">Neues Passwort</label>
										</div>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="confirmNewPassword"
												placeholder="Neues Passwort bestätigen"
												bind:value={confirmNewPassword}
											/>
											<label for="confirmNewPassword">Neues Passwort bestätigen</label>
										</div>
										<button class="btn btn-primary" onclick={changePassword}>
											{#if isChangingPassword}
												<!-- svelte-ignore a11y_no_static_element_interactions -->
												<div class="spinner-border" role="status">
													<span class="visually-hidden">Loading...</span>
												</div>
											{/if}
											Passwort ändern
										</button>
									</form>
									{#if changePasswordNotEqual}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											Die neuen Passwörter stimmen nicht überein!
										</div>
									{/if}
									{#if changingPasswordSuccess}
										<div class="alert alert-success mt-2" role="alert" transition:slide>
											Das Passwort wurde erfolgreich geändert!<br />
											Backup-Codes wurden ungültig gemacht (sofern vorhanden), und müssen neu erstellt
											werden.
										</div>
									{/if}
									{#if changingPasswordIncorrect}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											Das aktuelle Passwort ist falsch!
										</div>
									{:else if changingPasswordError}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											Fehler beim Ändern des Passworts!
										</div>
									{/if}
								</div>
								<div>
									<h5>Backup-Codes</h5>
									<ul>
										<li>
											Backup-Codes funktionieren wie Einmal-Passwörter. Sie können immer anstelle
											des Passworts verwendet werden, allerdings sind sie jeweils nur einmal gültig
											und werden anschließend gelöscht.
										</li>
										<li>
											Es werden immer 6 Codes generiert, welche die vorherigen Codes (sofern
											vorhanden) ersetzen.
										</li>
										<li>
											Du musst dir die Codes nach der Erstellung direkt notieren, sie können nicht
											erneut angezeigt werden!
										</li>
									</ul>

									<form onsubmit={createBackupCodes}>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="currentPassword"
												placeholder="Aktuelles Passwort"
												bind:value={backupCodesPassword}
											/>
											<label for="currentPassword">Passwort bestätigen</label>
										</div>
										<button
											class="btn btn-primary"
											onclick={createBackupCodes}
											data-sveltekit-noscroll
										>
											Backup-Codes generieren
											{#if isGeneratingBackupCodes}
												<div class="spinner-border spinner-border-sm" role="status">
													<span class="visually-hidden">Loading...</span>
												</div>
											{/if}
										</button>
									</form>
									{#if backupCodes.length > 0}
										<div class="alert alert-success alert-dismissible mt-3" transition:slide>
											<h6>Deine Backup-Codes:</h6>
											<p>
												Notiere dir die Codes, sie können nach dem Schließen dieses Fenstern nicht
												erneut angezeigt werden!
											</p>
											<button class="btn btn-secondary my-2" onclick={copyBackupCodes}>
												<Fa icon={codesCopiedSuccess ? faCheck : faCopy} />
												Codes kopieren
											</button>
											<ul class="list-group">
												{#each backupCodes as code}
													<li class="list-group-item backupCode">
														<code>{code}</code>
													</li>
												{/each}
											</ul>
										</div>
									{/if}
									{#if showBackupCodesError}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											Fehler beim Erstellen der Backup-Codes! Vielleicht stimmt das Passwort nicht?
										</div>
									{/if}
								</div>
								<div><h5>Username ändern</h5></div>
								<div>
									<h5>Konto löschen</h5>
									<p>
										Dies löscht dein Konto und alle damit verbundenen Daten. Dies kann nicht
										rückgängig gemacht werden!
									</p>
									<form
										onsubmit={() => {
											showConfirmDeleteAccount = true;
										}}
									>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="currentPassword"
												placeholder="Aktuelles Passwort"
												bind:value={deleteAccountPassword}
											/>
											<label for="currentPassword">Passwort bestätigen</label>
										</div>
										<button
											class="btn btn-danger"
											onclick={() => {
												showConfirmDeleteAccount = true;
											}}
											data-sveltekit-noscroll
										>
											Konto löschen
											{#if isDeletingAccount}
												<!-- svelte-ignore a11y_no_static_element_interactions -->
												<div class="spinner-border" role="status">
													<span class="visually-hidden">Loading...</span>
												</div>
											{/if}
										</button>
									</form>
									{#if showDeleteAccountSuccess}
										<div class="alert alert-success mt-2" role="alert" transition:slide>
											Dein Konto wurde erfolgreich gelöscht!<br />
											Du solltest jetzt eigentlich automatisch ausgeloggt werden. Falls nicht, dann logge
											dich bitte sebst aus.
										</div>
									{/if}
									{#if deleteAccountPasswordIncorrect}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											Das eingegebene Passwort ist falsch!
										</div>
									{/if}
									{#if showConfirmDeleteAccount}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											Bist du dir sicher, dass du dein Konto löschen möchtest? Dies kann nicht
											rückgängig gemacht werden!
											<div class="d-flex flex-row mt-2">
												<button
													class="btn btn-secondary"
													onclick={() => {
														showConfirmDeleteAccount = false;
														deleteAccountPassword = '';
													}}>Abbrechen</button
												>
												<button
													class="btn btn-danger ms-3"
													onclick={deleteAccount}
													disabled={isDeletingAccount}
													>Löschen bestätigen
													{#if isDeletingAccount}
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
								</div>
							</div>

							<div id="about">
								<h3 class="text-primary">💡 About</h3>
								Version:<br />
								Changelog: <br />
								Link zu github
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

	<div
		id="toastErrorLogout"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">Fehler beim Logout</div>
		</div>
	</div>

	<div
		id="toastErrorExportData"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">Fehler beim Exportieren!</div>
		</div>
	</div>
</div>

<style>
	h5,
	h6 {
		font-weight: 600;
		text-decoration: underline;
		text-decoration-color: #0d6efd;
	}

	h6 {
		margin-top: 0.7rem;
	}

	.backupCode {
		font-size: 15pt;
	}

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

	.settings-content > div {
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
		position: sticky;
		top: 0;
		backdrop-filter: blur(10px) saturate(150%);
		background-color: rgba(240, 240, 240, 0.9);
		padding: 4px;
		border-radius: 5px;
	}

	.modal-body {
		overflow-y: hidden;
	}

	.modal-header {
		border-bottom: 1px solid rgba(255, 255, 255, 0.2);
	}

	.modal-footer {
		border-top: 1px solid rgba(255, 255, 255, 0.2);
	}
</style>

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
	let scrollSpy;
	function openSettingsModal() {
		$tempSettings = JSON.parse(JSON.stringify($settings));
		aLookBackYears = $settings.aLookBackYears.toString();

		settingsModal = new bootstrap.Modal(document.getElementById('settingsModal'));
		settingsModal.show();

		// initialize ScrollSpy
		document.getElementById('settingsModal').addEventListener('shown.bs.modal', function onShown() {
			// Remove the event listener to prevent multiple executions
			document.getElementById('settingsModal').removeEventListener('shown.bs.modal', onShown);

			const height = document.getElementById('modal-body').clientHeight;
			document.getElementById('settings-content').style.height = 'calc(' + height + 'px - 2rem)';
			document.getElementById('settings-nav').style.height = 'calc(' + height + 'px - 2rem)';
			document.getElementById('settings-content').style.overflowY = 'auto';

			// Wait a little longer for all transitions to complete
			setTimeout(() => {
				// Apply ScrollSpy to the actual scroll area
				const settingsContent = document.getElementById('settings-content');
				console.log(settingsContent);
				if (scrollSpy) {
					scrollSpy.dispose(); // Remove old ScrollSpy
				}
				scrollSpy = new bootstrap.ScrollSpy(settingsContent, {
					target: '#settings-nav'
				});
				console.log(scrollSpy);
			}, 400);
		});
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

	const exportTranslations = {
		weekdays: [
			$t('weekdays.sunday'),
			$t('weekdays.monday'),
			$t('weekdays.tuesday'),
			$t('weekdays.wednesday'),
			$t('weekdays.thursday'),
			$t('weekdays.friday'),
			$t('weekdays.saturday')
		],
		dateFormat: $t('export.dateFormat'),
		uiElements: {
			exportTitle: $t('export.title'),
			user: $t('export.user'),
			exportedOn: $t('export.exportedOn'),
			exportedOnFormat: $t('export.exportedOnFormat'),
			entriesCount: $t('export.entriesCount'),
			images: $t('export.images'),
			files: $t('export.files'),
			tags: $t('export.tags')
		}
	};

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
					tagsInHTML: exportTagsInHTML,
					translations: JSON.stringify(exportTranslations)
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

{#snippet unsavedChanges()}
	<div class="unsaved-changes" data-content={$t('settings.unsaved_changes')} transition:slide></div>
{/snippet}

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
								<a class="nav-link mb-1" href="#appearance">{$t('settings.appearance')}</a>
								<a class="nav-link mb-1" href="#functions">{$t('settings.functions')}</a>

								<a class="nav-link mb-1" href="#tags">{$t('settings.tags')}</a>
								<a class="nav-link mb-1" href="#templates">{$t('settings.templates')}</a>
								<a class="nav-link mb-1" href="#data">{$t('settings.data')}</a>
								<a class="nav-link mb-1" href="#security">{$t('settings.security')}</a>
								<a class="nav-link mb-1" href="#about">{$t('settings.about')}</a>
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
								<h3 class="text-primary">🎨 {$t('settings.appearance')}</h3>
								<div id="lightdark">
									{#if $tempSettings.darkModeAutoDetect !== $settings.darkModeAutoDetect || $tempSettings.useDarkMode !== $settings.useDarkMode}
										{@render unsavedChanges()}
									{/if}
									<h5>{$t('settings.light_dark_mode')}</h5>
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
											{$t('settings.light_dark_auto_detect')}
											{#if window.matchMedia('(prefers-color-scheme: dark)').matches}
												<b><u>{$t('settings.dark_mode')} <Fa icon={faMoon} /></u></b>
											{:else}
												<b><u>{$t('settings.light_mode')} <Fa icon={faSun} /></u></b>
											{/if}
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
											{$t('settings.light_dark_manual')}
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
										{@render unsavedChanges()}
									{/if}

									<h5>{$t('settings.background')}</h5>
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
											{$t('settings.background_gradient')}
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
										<label class="form-check-label" for="background_monochrome">
											{$t('settings.background_monochrome')}
										</label>
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
								<h3 class="text-primary">🛠️ {$t('settings.functions')}</h3>

								<div id="autoLoadImages">
									{#if $tempSettings.setAutoloadImagesPerDevice !== $settings.setAutoloadImagesPerDevice || $tempSettings.autoloadImagesByDefault !== $settings.autoloadImagesByDefault}
										{@render unsavedChanges()}
									{/if}

									<h5>{$t('settings.images_title')}</h5>
									{@html $t('settings.images_description')}

									<div class="form-check form-switch">
										<input
											class="form-check-input"
											bind:checked={$tempSettings.setAutoloadImagesPerDevice}
											type="checkbox"
											role="switch"
											id="setImageLoadingPerDeviceSwitch"
										/>
										<label class="form-check-label" for="setImageLoadingPerDeviceSwitch">
											{$t('settings.images_loading_per_device')}
										</label>
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
											{@html $t('settings.images_loading_this_device')}
										</label>
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
											{$t('settings.images_loading_default')}
										</label>
									</div>
								</div>

								<div id="language">
									{#if $tempSettings.useBrowserLanguage !== $settings.useBrowserLanguage || $tempSettings.language !== $settings.language}
										{@render unsavedChanges()}
									{/if}
									<h5>🌐 {$t('settings.language')}</h5>
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
											{$t('settings.language_auto_detect')}
											<code>{window.navigator.language}</code>
											{#if tolgeesMatchForBrowserLanguage() !== '' && tolgeesMatchForBrowserLanguage() !== window.navigator.language}
												➔ <code>{tolgeesMatchForBrowserLanguage()}</code>
												{$t('settings.language_X_used')}
											{/if}
										</label>
										{#if $tempSettings.useBrowserLanguage && tolgeesMatchForBrowserLanguage() === ''}
											<div
												transition:slide
												disabled={!$settings.useBrowserLanguage}
												class="alert alert-danger"
												role="alert"
											>
												{@html $t('settings.language_not_available', {
													browserLanguage: window.navigator.language,
													defaultLanguage: $tolgee.getInitialOptions().defaultLanguage
												})}
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
											{$t('settings.set_language_manually')}
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
									<div class="form-text">
										{$t('settings.language.reload_info')}
									</div>
								</div>
								<div id="timezone">
									{#if $tempSettings.useBrowserTimezone !== $settings.useBrowserTimezone || ($tempSettings.timezone !== undefined && $tempSettings.timezone?.value !== $settings.timezone?.value)}
										{@render unsavedChanges()}
									{/if}
									<h5>{$t('settings.timezone')}</h5>
									{$t('settings.timezone.description', { written_on: $t('log.written_on') })}

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
											{@html $t('settings.timezone.auto_detect')}
											<code>{new Intl.DateTimeFormat().resolvedOptions().timeZone}</code>
										</label>
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
											{$t('settings.timezone.manual')}
										</label>
										<br />
										<SelectTimezone />
										{#if !$tempSettings.useBrowserTimezone}
											<span transition:fade>
												{$t('settings.timezone.selected')} <code>{$tempSettings.timezone}</code>
											</span>
										{/if}
									</div>

									<div class="form-text mt-3">
										{@html $t('settings.timezone.help_text')}
									</div>
								</div>

								<div id="aLookBack">
									{#if $tempSettings.useALookBack !== $settings.useALookBack || JSON.stringify(aLookBackYears
												.trim()
												.split(',')
												.map( (year) => parseInt(year.trim()) )) !== JSON.stringify($settings.aLookBackYears)}
										{@render unsavedChanges()}
									{/if}

									<h5>{$t('settings.aLookBack')}</h5>
									<ul>
										{@html $t('settings.aLookBack.description')}
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
											{$t('settings.ALookBack.useIt')}
										</label>
									</div>

									<div>
										<input
											type="text"
											id="useALookBackYears"
											class="form-control {aLookBackYearsInvalid ? 'is-invalid' : ''}"
											aria-describedby="useALookBackHelpBlock"
											disabled={!$tempSettings.useALookBack}
											placeholder={$t('settings.ALookBack.input_placeholder')}
											bind:value={aLookBackYears}
											invalid
										/>
										{#if aLookBackYearsInvalid}
											<div class="alert alert-danger mt-2" role="alert" transition:slide>
												{$t('settings.ALookBack.invalid_input')}
											</div>
										{/if}
										<div id="useALookBackHelpBlock" class="form-text">
											{@html $t('settings.ALookBack.help_text')}
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
								<h3 class="text-primary">#️⃣ {$t('settings.tags')}</h3>
								<div>
									{$t('settings.tags.description')}

									{#if $tags.length === 0}
										<div class="alert alert-info my-2" role="alert">
											{$t('settings.tags.no_tags')}
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
														<Fa icon={faTriangleExclamation} fw />
														{@html $t('settings.tags.delete_confirmation')}
													</div>
													<!-- svelte-ignore a11y_consider_explicit_label -->
													<div class="d-flex flex-row mt-2">
														<button class="btn btn-secondary" onclick={() => (deleteTagId = null)}
															>{$t('settings.abort')}
														</button>
														<button
															disabled={isDeletingTag}
															class="btn btn-danger ms-3"
															onclick={() => deleteTag(tag.id)}
															>{$t('settings.delete')}
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
								<h3 class="text-primary">📝 {$t('settings.templates')}</h3>
								<div>
									{#if oldTemplateName !== templateName || oldTemplateText !== templateText}
										{@render unsavedChanges()}
									{/if}

									<div class="d-flex flex-column">
										<select
											bind:value={selectedTemplate}
											class="form-select"
											aria-label="Select template"
											onchange={updateSelectedTemplate}
										>
											<option value="-1" selected={selectedTemplate === '-1'}>
												{$t('settings.templates.create_new')}
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
											<span>
												{@html $t('settings.templates.delete_confirmation', {
													template: $templates[selectedTemplate]?.name
												})}
											</span>
											<button
												type="button"
												class="btn btn-secondary ms-2"
												onclick={() => (confirmDeleteTemplate = false)}
												>{$t('settings.abort')}</button
											>
											<button
												type="button"
												class="btn btn-danger ms-2"
												onclick={() => {
													deleteTemplate();
												}}
												disabled={isDeletingTemplate}
												>{$t('settings.delete')}
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
											placeholder={$t('settings.template.name_of_template')}
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
										placeholder={$t('settings.template.content_of_template')}
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
											{$t('settings.template.save_template')}
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
								<h3 class="text-primary">📁 {$t('settings.data')}</h3>
								<div>
									<h5>{$t('settings.export')}</h5>
									{$t('settings.export.description')}

									<h6>{$t('settings.export.period')}</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="period"
											value="periodAll"
											id="periodAll"
											bind:group={exportPeriod}
										/>
										<label class="form-check-label" for="periodAll"
											>{$t('settings.export.period_all')}</label
										>
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
										<label class="form-check-label" for="periodVariable">
											{$t('settings.export.period_variable')}</label
										>
										{#if exportPeriod === 'periodVariable'}
											<div class="d-flex flex-row" transition:slide>
												<div class="me-2">
													<label for="exportStartDate">{$t('settings.export.start_date')}</label>
													<input
														type="date"
														class="form-control me-2"
														id="exportStartDate"
														bind:value={exportStartDate}
													/>
												</div>
												<div>
													<label for="exportEndDate">{$t('settings.export.end_date')}</label>
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
													{$t('settings.export.period_invalid')}
												</div>
											{/if}
										{/if}
									</div>

									<h6>{$t('settings.export.split')}</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="radio"
											name="split"
											value="aio"
											id="splitAIO"
											bind:group={exportSplit}
										/>
										<label class="form-check-label" for="splitAIO"
											>{$t('settings.export.split_aio')}
										</label>
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
										<label class="form-check-label" for="splitYear"
											>{$t('settings.export.split_year')}
										</label>
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
										<label class="form-check-label" for="splitMonth"
											>{$t('settings.export.split_month')}
										</label>
									</div>

									<h6>{$t('settings.export.show_images')}</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="checkbox"
											name="images"
											id="exportImagesInHTML"
											bind:checked={exportImagesInHTML}
										/>
										<label class="form-check-label" for="exportImagesInHTML">
											{@html $t('settings.export.show_images_description')}
										</label>
									</div>

									<h6>{$t('settings.export.show_tags')}</h6>
									<div class="form-check">
										<input
											class="form-check-input"
											type="checkbox"
											id="exportTagsInHTML"
											bind:checked={exportTagsInHTML}
										/>
										<label class="form-check-label" for="exportTagsInHTML">
											{$t('settings.export.show_tags_description')}
										</label>
									</div>

									<div class="form-text">
										{@html $t('settings.export.help_text')}
									</div>
									<button
										class="btn btn-primary mt-3"
										onclick={exportData}
										data-sveltekit-noscroll
										disabled={isExporting ||
											(exportPeriod === 'periodVariable' &&
												(exportStartDate === '' || exportEndDate === ''))}
									>
										{$t('settings.export.export_button')}
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
								<h3 class="text-primary">🔒 {$t('settings.security')}</h3>
								<div>
									<h5>{$t('settings.security.change_password')}</h5>
									<form onsubmit={changePassword}>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="currentPassword"
												placeholder={$t('settings.password.current_password')}
												bind:value={currentPassword}
											/>
											<label for="currentPassword">{$t('settings.password.current_password')}</label
											>
										</div>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="newPassword"
												placeholder={$t('settings.password.new_password')}
												bind:value={newPassword}
											/>
											<label for="newPassword">{$t('settings.password.new_password')}</label>
										</div>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="confirmNewPassword"
												placeholder={$t('settings.password.confirm_new_password')}
												bind:value={confirmNewPassword}
											/>
											<label for="confirmNewPassword"
												>{$t('settings.password.confirm_new_password')}</label
											>
										</div>
										<button class="btn btn-primary" onclick={changePassword}>
											{#if isChangingPassword}
												<!-- svelte-ignore a11y_no_static_element_interactions -->
												<div class="spinner-border" role="status">
													<span class="visually-hidden">Loading...</span>
												</div>
											{/if}
											{$t('settings.password.change_password_button')}
										</button>
									</form>
									{#if changePasswordNotEqual}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											{$t('settings.password.passwords_dont_match')}
										</div>
									{/if}
									{#if changingPasswordSuccess}
										<div class="alert alert-success mt-2" role="alert" transition:slide>
											{@html $t('settings.password.success')}
										</div>
									{/if}
									{#if changingPasswordIncorrect}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											{$t('settings.password.current_password_incorrect')}
										</div>
									{:else if changingPasswordError}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											{$t('settings.password.change_error')}
										</div>
									{/if}
								</div>
								<div>
									<h5>{$t('settings.backup_codes')}</h5>
									<ul>
										{@html $t('settings.backup_codes.description')}
									</ul>

									<form onsubmit={createBackupCodes}>
										<div class="form-floating mb-3">
											<input
												type="password"
												class="form-control"
												id="currentPassword"
												placeholder={$t('settings.password.current_password')}
												bind:value={backupCodesPassword}
											/>
											<label for="currentPassword">{$t('settings.password.confirm_password')}</label
											>
										</div>
										<button
											class="btn btn-primary"
											onclick={createBackupCodes}
											data-sveltekit-noscroll
										>
											{$t('settings.backup_codes.generate_button')}
											{#if isGeneratingBackupCodes}
												<div class="spinner-border spinner-border-sm" role="status">
													<span class="visually-hidden">Loading...</span>
												</div>
											{/if}
										</button>
									</form>
									{#if backupCodes.length > 0}
										<div class="alert alert-success alert-dismissible mt-3" transition:slide>
											{@html $t('settings.backup_codes.success')}

											<button class="btn btn-secondary my-2" onclick={copyBackupCodes}>
												<Fa icon={codesCopiedSuccess ? faCheck : faCopy} />
												{$t('settings.backup_codes.copy_button')}
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
											{$t('settings.backup_codes.error')}
										</div>
									{/if}
								</div>
								<div><h5>Username ändern</h5></div>
								<div>
									<h5>{$t('settings.delete_account')}</h5>
									<p>
										{$t('settings.delete_account.description')}
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
												placeholder={$t('settings.password.current_password')}
												bind:value={deleteAccountPassword}
											/>
											<label for="currentPassword">{$t('settings.password.confirm_password')}</label
											>
										</div>
										<button
											class="btn btn-danger"
											onclick={() => {
												showConfirmDeleteAccount = true;
											}}
											data-sveltekit-noscroll
										>
											{$t('settings.delete_account.delete_button')}
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
											{@html $t('settings.delete_account.success')}
										</div>
									{/if}
									{#if deleteAccountPasswordIncorrect}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											{$t('settings.delete_account.password_incorrect')}
										</div>
									{/if}
									{#if showConfirmDeleteAccount}
										<div class="alert alert-danger mt-2" role="alert" transition:slide>
											{$t('settings.delete_account.confirm')}

											<div class="d-flex flex-row mt-2">
												<button
													class="btn btn-secondary"
													onclick={() => {
														showConfirmDeleteAccount = false;
														deleteAccountPassword = '';
													}}>{$t('settings.abort')}</button
												>
												<button
													class="btn btn-danger ms-3"
													onclick={deleteAccount}
													disabled={isDeletingAccount}
													>{$t('settings.delete_account.confirm_button')}
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
								<h3 class="text-primary">💡 {$t('settings.about')}</h3>
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
						{$t('settings.unsaved_changes')}
					</div>
				{/if}
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal"
					>{$t('settings.abort')}</button
				>
				<button
					type="button"
					class="btn btn-primary"
					onclick={saveUserSettings}
					disabled={isSaving || !settingsHaveChanged}
					>{$t('settings.save')}
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
			<div class="toast-body">{$t('settings.toast.saved_edit_tag_success')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.saved_edit_tag_error')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.delete_tag_error')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.saved_settings_success')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.saved_settings_error')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.invalid_template_empty')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.invalid_template_double')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.saved_template_success')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.delete_template_error')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.delete_template_success')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.logout_error')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
			<div class="toast-body">{$t('settings.toast.export_data_error')}</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
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
		content: attr(data-content);
	}

	:global(.tagColumn > span) {
		width: min-content;
	}

	.tagColumn {
		gap: 0.5rem;
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
		position: relative;
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

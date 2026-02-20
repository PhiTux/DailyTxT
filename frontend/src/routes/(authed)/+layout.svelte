<script>
	import * as bootstrap from 'bootstrap';
	import Fa, { FaLayers } from 'svelte-fa';
	import { goto, invalidateAll } from '$app/navigation';
	import { onDestroy, onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import {
		readingMode,
		settings,
		tempSettings,
		autoLoadImagesThisDevice,
		darkMode,
		languageLoaded
	} from '$lib/settingsStore.js';
	import { API_URL } from '$lib/APIurl.js';
	import { tags, tagsLoaded } from '$lib/tagStore.js';
	import TagModal from '$lib/TagModal.svelte';
	import ChangelogModal from '$lib/ChangelogModal.svelte';
	import {
		alwaysShowSidenav,
		generateNeonMesh,
		needsReauthentication,
		isAuthenticated
	} from '$lib/helpers.js';
	import { templates } from '$lib/templateStore';
	import {
		faPersonRunning,
		faGlasses,
		faPencil,
		faSliders,
		faCircleUp,
		faBars,
		faCircle
	} from '@fortawesome/free-solid-svg-icons';
	import axios from 'axios';
	import { page } from '$app/state';
	import { blur, slide, fade } from 'svelte/transition';
	import Statistics from '$lib/settings/Statistics.svelte';
	import Admin from '$lib/settings/Admin.svelte';
	import Appearance from '$lib/settings/Appearance.svelte';
	import Features from '$lib/settings/Features.svelte';
	import Tags from '$lib/settings/Tags.svelte';
	import Templates from '$lib/settings/Templates.svelte';
	import Data from '$lib/settings/Data.svelte';
	import Security from '$lib/settings/Security.svelte';
	import Sharing from '$lib/settings/Sharing.svelte';
	import About from '$lib/settings/About.svelte';
	import Account from '$lib/settings/Account.svelte';
	import { getTranslate, getTolgee } from '@tolgee/svelte';
	import dailytxt from '$lib/assets/locked_heart_with_keyhole.svg';
	import { selectedDate } from '$lib/calendarStore';
	import DemoModeText from '$lib/DemoModeText.svelte';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	let { children } = $props();
	let inDuration = 150;
	let outDuration = 150;

	let current_version = $state('');
	let latest_stable_version = $state('');
	let latest_overall_version = $state('');
	let updateAvailable = $state(false);

	// Active sub-view of settings modal: 'settings' | 'stats' | 'admin'
	let activeSettingsView = $state('settings');

	// References for sliding indicator
	let settingsTabGroup;
	let settingsButton;
	let statsButton;
	let adminButton;

	// Calculate slide offset and width for the indicator
	function getSlideOffset(activeView) {
		if (!settingsTabGroup || !settingsButton) return 0;

		const container = settingsTabGroup;
		const containerRect = container.getBoundingClientRect();

		let activeButton;
		switch (activeView) {
			case 'settings':
				activeButton = settingsButton;
				break;
			case 'stats':
				activeButton = statsButton;
				break;
			case 'admin':
				activeButton = adminButton;
				break;
			default:
				activeButton = settingsButton;
		}

		if (!activeButton) return 0;

		const buttonRect = activeButton.getBoundingClientRect();
		// Add the container's scrollLeft to account for horizontal scrolling
		return buttonRect.left - containerRect.left + container.scrollLeft;
	}

	function getSlideWidth(activeView) {
		let activeButton;
		switch (activeView) {
			case 'settings':
				activeButton = settingsButton;
				break;
			case 'stats':
				activeButton = statsButton;
				break;
			case 'admin':
				activeButton = adminButton;
				break;
			default:
				activeButton = settingsButton;
		}

		return activeButton ? activeButton.offsetWidth : 0;
	}

	// Force indicator update when activeSettingsView changes or when modal is shown
	let indicatorNeedsUpdate = $state(0);

	// Function to compare version strings (semver-like)
	function compareVersions(v1, v2) {
		if (!v1 || !v2) return 0;

		const parseVersion = (version) => {
			const cleaned = version.replace(/^v/, '');
			const parts = cleaned.split('-')[0].split('.');
			return parts.map((part) => parseInt(part) || 0);
		};

		const version1 = parseVersion(v1);
		const version2 = parseVersion(v2);

		for (let i = 0; i < Math.max(version1.length, version2.length); i++) {
			const v1Part = version1[i] || 0;
			const v2Part = version2[i] || 0;

			if (v1Part > v2Part) return 1;
			if (v1Part < v2Part) return -1;
		}

		// if both have the same semver-number, check the testing-number (like 2.3.1-testing.3)
		// if one does not have anything on the right of "-", then this is the "stable" version
		const testingVersion1 = v1.split('-')[1] || '';
		const testingVersion2 = v2.split('-')[1] || '';

		if (!testingVersion1 && testingVersion2) return 1; // v1 is stable, v2 is testing -> v1 is newer
		if (testingVersion1 && !testingVersion2) return -1; // v1 is testing, v2 is stable -> v2 is newer

		return testingVersion1.localeCompare(testingVersion2) > 0;
	}

	// Function to check if updates are available
	function checkForUpdates() {
		if (!$settings.checkForUpdates) {
			updateAvailable = false;
			return;
		}

		const latestVersion = $settings.includeTestVersions
			? latest_overall_version
			: latest_stable_version;

		updateAvailable = compareVersions(latestVersion, current_version) > 0;
	}

	// React to changes in settings or version info
	$effect(() => {
		checkForUpdates();
	});

	$effect(() => {
		if ($readingMode === true && !page.url.pathname.endsWith('/read')) {
			goto(resolve('/read'));
		} else if ($readingMode === false) {
			goto(resolve('/write'));
		}
	});

	onDestroy(() => {
		$isAuthenticated = false;
	});

	onMount(() => {
		let needsReauth = needsReauthentication();

		// Check if re-authentication is needed FIRST
		if (!$isAuthenticated && needsReauth) {
			// Save current route for return after reauth
			localStorage.setItem('returnAfterReauth', window.location.pathname);
			goto(resolve('/reauth'));
			return; // Stop further initialization
		}

		$selectedDate = {
			year: new Date().getFullYear(),
			month: new Date().getMonth() + 1,
			day: new Date().getDate()
		};

		// Normal initialization only if authenticated
		getUserSettings();
		getTemplates();
		getVersionInfo();
		loadTags();
		loadShareTokenInfo();

		if (page.url.pathname.endsWith('/read')) {
			$readingMode = true;
		} else if (page.url.pathname.endsWith('/write')) {
			$readingMode = false;
		}

		document.getElementById('settingsModal').addEventListener('hidden.bs.modal', function () {
			backupCodes = [];
		});
	});

	let changelog = $state({});

	/**
	 * Triggers GET request to check if the user already logged in with the new version. If not, it returns the changelog.
	 * When set to force_show=true, it will show the changelog regardless of the version.
	 */
	function checkChangelog(force_show) {
		// close settings modal if force_show is true
		if (force_show) {
			const settingsModalEl = document.getElementById('settingsModal');
			const settingsModal = bootstrap.Modal.getInstance(settingsModalEl);
			if (settingsModal) {
				settingsModal.hide();
			}
		}

		axios
			.get(API_URL + '/users/checkChangelog', { params: { force_show: force_show } })
			.then((response) => {
				// don't show anything if showChangelogOnUpdate is false and force_show is not set
				if (!$settings.showChangelogOnUpdate && !force_show) {
					return;
				}
				if (response.data.changelog) {
					const sortedChangelog = {};
					Object.keys(response.data.changelog)
						.sort((a, b) => compareVersions(b, a))
						.forEach((key) => {
							sortedChangelog[key] = response.data.changelog[key];
						});
					changelog = sortedChangelog;
					// show changelog modal
					const changelogModal = new bootstrap.Modal(document.getElementById('changelogModal'));
					changelogModal.show();
				}
			})
			.catch((error) => {
				console.error(error);
			});
	}

	function loadTags() {
		axios
			.get(API_URL + '/logs/getTags')
			.then((response) => {
				$tags = response.data;
				$tagsLoaded = true;
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingTags'));
				toast.show();
			});
	}

	function logout(errorCode) {
		axios
			.get(API_URL + '/users/logout')
			.then(() => {
				localStorage.removeItem('user');
				if (errorCode) {
					goto(resolve(`/login?error=${errorCode}`));
				} else {
					goto(resolve('/login'));
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

	// Custom ScrollSpy state (manual implementation to avoid flicker)
	let activeSettingsSection = $state('appearance');
	let settingsSections = [];
	let removeScrollListener = null;
	let lastComputedSection = 'appearance';

	// Settings section metadata for mobile dropdown navigation
	const settingsSectionsMeta = [
		{ id: 'appearance', labelKey: 'settings.appearance' },
		{ id: 'features', labelKey: 'settings.features' },
		{ id: 'tags', labelKey: 'settings.tags' },
		{ id: 'templates', labelKey: 'settings.templates' },
		{ id: 'data', labelKey: 'settings.data' },
		{ id: 'security', labelKey: 'settings.security' },
		{ id: 'sharing', labelKey: 'settings.sharing' },
		{ id: 'account', labelKey: 'settings.account' },
		{ id: 'about', labelKey: 'settings.about' }
	];

	function scrollToSection(id) {
		const container = document.getElementById('settings-content');
		if (!container) return;
		const target = container.querySelector(':scope > #' + CSS.escape(id));
		if (!target) return;
		// Calculate dynamic offset (mobile dropdown height if visible)
		let offset = 4;
		const mobileBar = container.querySelector('.mobile-settings-dropdown');
		if (mobileBar && getComputedStyle(mobileBar).display !== 'none') {
			offset = mobileBar.getBoundingClientRect().height + 6; // add small gap
		}
		container.scrollTo({ top: target.offsetTop - offset, behavior: 'smooth' });
	}

	function computeActiveSection(container) {
		if (!container || settingsSections.length === 0) return;
		// Activation line: a bit below the top to give stability
		const activationY = container.scrollTop + container.clientHeight * 0.18; // 18% viewport height
		let current = settingsSections[0].id;
		for (let i = 0; i < settingsSections.length; i++) {
			const sec = settingsSections[i];
			if (sec.offsetTop <= activationY) {
				current = sec.id;
			} else {
				break;
			}
		}
		// Hysteresis: only update if different for stability
		if (current !== lastComputedSection) {
			lastComputedSection = current;
			activeSettingsSection = current;
		}
	}

	function initSettingsScrollSpy() {
		const container = document.getElementById('settings-content');
		if (!container) return;
		settingsSections = Array.from(container.querySelectorAll(':scope > div[id]'));
		if (settingsSections.length === 0) return;
		// Initial compute
		computeActiveSection(container);
		let raf = 0;
		const onScroll = () => {
			if (raf) cancelAnimationFrame(raf);
			raf = requestAnimationFrame(() => computeActiveSection(container));
		};
		container.addEventListener('scroll', onScroll, { passive: true });
		removeScrollListener = () => {
			container.removeEventListener('scroll', onScroll);
			if (raf) cancelAnimationFrame(raf);
		};
		// Recompute on resize for robustness
		const onResize = () => computeActiveSection(container);
		window.addEventListener('resize', onResize);
		const originalRemove = removeScrollListener;
		removeScrollListener = () => {
			originalRemove && originalRemove();
			window.removeEventListener('resize', onResize);
		};
	}

	function destroySettingsScrollSpy() {
		removeScrollListener && removeScrollListener();
		removeScrollListener = null;
		settingsSections = [];
	}

	function reinitializeSettingsScrollSpy() {
		// Destroy existing ScrollSpy first
		destroySettingsScrollSpy();

		// Re-setup the settings content scroll behavior
		const contentEl = document.getElementById('settings-content');
		const navEl = document.getElementById('settings-nav');
		const modalBody = document.getElementById('modal-body');

		if (contentEl && navEl && modalBody) {
			const height = modalBody.clientHeight;
			contentEl.style.height = 'calc(' + height + 'px - 2rem)';
			navEl.style.height = 'calc(' + height + 'px - 2rem)';
			contentEl.style.overflowY = 'auto';
			contentEl.scrollTop = 0;
			activeSettingsSection = 'appearance';
			// Short timeout to allow layout calculation before reading offsets
			setTimeout(initSettingsScrollSpy, 100);
		}
	}

	function switchToSettingsTab() {
		activeSettingsView = 'settings';
		// Reinitialize ScrollSpy when switching to settings tab
		setTimeout(reinitializeSettingsScrollSpy, 50);
	}

	function switchToStatsTab() {
		activeSettingsView = 'stats';
		// Destroy settings ScrollSpy when leaving settings tab
		destroySettingsScrollSpy();
	}

	function switchToAdminTab() {
		activeSettingsView = 'admin';
		// Destroy settings ScrollSpy when leaving settings tab
		destroySettingsScrollSpy();
	}

	function openSettingsModal() {
		activeSettingsView = 'settings';
		$tempSettings = JSON.parse(JSON.stringify($settings));
		aLookBackYears = $settings.aLookBackYears.toString();

		settingsModal = new bootstrap.Modal(document.getElementById('settingsModal'));
		settingsModal.show();

		// Initialize custom ScrollSpy
		const modalEl = document.getElementById('settingsModal');
		const onShown = () => {
			modalEl.removeEventListener('shown.bs.modal', onShown);
			const height = document.getElementById('modal-body').clientHeight;
			const contentEl = document.getElementById('settings-content');
			const navEl = document.getElementById('settings-nav');
			if (contentEl && navEl) {
				contentEl.style.height = 'calc(' + height + 'px - 2rem)';
				navEl.style.height = 'calc(' + height + 'px - 2rem)';
				contentEl.style.overflowY = 'auto';
				contentEl.scrollTop = 0;
				activeSettingsSection = 'appearance';
				// Short timeout to allow layout calculation before reading offsets
				setTimeout(initSettingsScrollSpy, 100);
				// Update indicator position after modal is fully shown
				setTimeout(() => {
					indicatorNeedsUpdate++;
				}, 50);
			}
		};
		modalEl.addEventListener('shown.bs.modal', onShown);
		modalEl.addEventListener('hidden.bs.modal', () => {
			destroySettingsScrollSpy();
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
			.then(async (response) => {
				$settings = response.data;
				$tempSettings = JSON.parse(JSON.stringify($settings));
				aLookBackYears = $settings.aLookBackYears.toString();

				// Save re-auth setting to localStorage for immediate availability
				localStorage.setItem(
					'requirePasswordOnPageLoad',
					$settings.requirePasswordOnPageLoad.toString()
				);

				await updateLanguage();

				// set background
				setBackground();

				// check if an update was made to show changelog
				checkChangelog(false);
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

	async function updateLanguage() {
		if ($settings.useBrowserLanguage) {
			let browserLanguage = tolgeesMatchForBrowserLanguage();
			await $tolgee.changeLanguage(
				browserLanguage === '' ? $tolgee.getInitialOptions().defaultLanguage : browserLanguage
			);
		} else {
			await $tolgee.changeLanguage($settings.language);
		}
		$languageLoaded = true;
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
			document
				.querySelector('meta[name="theme-color"]')
				.setAttribute('content', $settings.monochromeBackgroundColor);
		} else if ($settings.background === 'gradient') {
			document.body.style.backgroundColor = '';
			generateNeonMesh($darkMode);
			document
				.querySelector('meta[name="theme-color"]')
				.setAttribute('content', $darkMode ? 'rgba(83, 83, 83, 0.4)' : 'rgba(187, 187, 187, 0.3)');
		}
	}

	let isSaving = $state(false);
	function saveUserSettings() {
		if (isSaving) return;
		isSaving = true;
		let reloadRequired = false;

		$tempSettings.aLookBackYears = aLookBackYears
			.trim()
			.split(',')
			.map((year) => parseInt(year.trim()));

		axios
			.post(API_URL + '/users/saveUserSettings', $tempSettings)
			.then((response) => {
				if (response.data.success) {
					if (
						$settings.language !== $tempSettings.language ||
						$settings.useBrowserLanguage !== $tempSettings.useBrowserLanguage ||
						$settings.firstDayOfWeek !== $tempSettings.firstDayOfWeek
					) {
						reloadRequired = true;
					}
					$settings = $tempSettings;

					// Save re-auth setting to localStorage for immediate availability
					localStorage.setItem(
						'requirePasswordOnPageLoad',
						$tempSettings.requirePasswordOnPageLoad.toString()
					);

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
				if (reloadRequired) {
					invalidateAll();
					location.reload();
				}
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

	let newUsername = $state('');
	let changeUsernamePassword = $state('');
	let isChangingUsername = $state(false);
	let changeUsernameSuccess = $state(false);
	let changeUsernameError = $state('');
	let changeUsernamePasswordIncorrect = $state(false);

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

	let currentUser = $state(localStorage.getItem('user'));

	// Random smiley for greeting
	const smileys = [
		'😊',
		'😄',
		'👍',
		'🎉',
		'🙌',
		'🤗',
		'😎',
		'✨',
		'🌟',
		'🥳',
		'😇',
		'🥇',
		'🚀',
		'🌞'
	];
	function pickSmiley() {
		return smileys[Math.floor(Math.random() * smileys.length)];
	}

	function changeUsername() {
		changeUsernameSuccess = false;
		changeUsernameError = '';
		changeUsernamePasswordIncorrect = false;

		if (!newUsername.trim()) {
			changeUsernameError = $t('settings.change_username.empty_username');
			return;
		}

		if (isChangingUsername) return;
		isChangingUsername = true;

		axios
			.post(API_URL + '/users/changeUsername', {
				new_username: newUsername.trim(),
				password: changeUsernamePassword
			})
			.then((response) => {
				if (response.data.success) {
					changeUsernameSuccess = true;
					// Update localStorage with new username
					localStorage.setItem('user', newUsername.trim());
					currentUser = newUsername.trim();
					// Clear form
					newUsername = '';
					changeUsernamePassword = '';
				} else {
					if (response.data.password_incorrect) {
						changeUsernamePasswordIncorrect = true;
					} else if (response.data.username_taken) {
						changeUsernameError = $t('settings.change_username.username_taken');
					} else {
						changeUsernameError = $t('settings.change_username.error');
					}
				}
			})
			.catch((error) => {
				console.error(error);
				changeUsernameError = $t('settings.change_username.error');
			})
			.finally(() => {
				isChangingUsername = false;
			});
	}

	let backupCodesPassword = $state('');
	let isGeneratingBackupCodes = $state(false);
	let backupCodes = $state([]);
	let codesCopiedSuccess = $state(false);
	let showBackupCodesError = $state(false);

	// Share token state
	let hasShareToken = $state(false);
	let shareLink = $state('');
	let isGeneratingShareToken = $state(false);
	let isRevokingShareToken = $state(false);
	let linkCopiedSuccess = $state(false);
	let showShareTokenError = $state(false);
	let shareVerificationEmailsText = $state('');
	let isLoadingShareVerificationSettings = $state(false);
	let isSavingShareVerificationSettings = $state(false);
	let showShareVerificationSettingsError = $state(false);
	let showShareVerificationSettingsSuccess = $state(false);
	let shareVerificationSMTPConfigured = $state(false);
	let shareAccessLogs = $state([]);
	let isLoadingShareAccessLogs = $state(false);
	let shareSMTPHost = $state('');
	let shareSMTPPort = $state(587);
	let shareSMTPUsername = $state('');
	let shareSMTPPassword = $state('');
	let shareSMTPFrom = $state('');
	let shareSMTPTestRecipient = $state('');
	let isSavingShareSMTPSettings = $state(false);
	let isTestingShareSMTP = $state(false);
	let showShareSMTPSettingsError = $state(false);
	let showShareSMTPSettingsSuccess = $state(false);
	let showShareSMTPTestError = $state(false);
	let showShareSMTPTestSuccess = $state(false);
	let isClearingShareAccessLogs = $state(false);

	function loadShareTokenInfo() {
		axios
			.get(API_URL + '/users/getShareTokenInfo')
			.then((response) => {
				hasShareToken = response.data.has_token;
				loadShareVerificationSettings();
				loadShareSMTPSettings();
				loadShareAccessLogs();
			})
			.catch((error) => {
				console.error(error);
			});
	}

	function loadShareVerificationSettings() {
		isLoadingShareVerificationSettings = true;
		showShareVerificationSettingsError = false;

		axios
			.get(API_URL + '/users/getShareVerificationSettings')
			.then((response) => {
				const emails = response.data.emails || [];
				shareVerificationEmailsText = emails.join('\n');
				shareVerificationSMTPConfigured = response.data.smtp_configured === true;
			})
			.catch((error) => {
				console.error(error);
				showShareVerificationSettingsError = true;
			})
			.finally(() => {
				isLoadingShareVerificationSettings = false;
			});
	}

	function saveShareVerificationSettings() {
		if (isSavingShareVerificationSettings) return;
		isSavingShareVerificationSettings = true;
		showShareVerificationSettingsError = false;
		showShareVerificationSettingsSuccess = false;

		const emails = shareVerificationEmailsText
			.split(/[,\n]/)
			.map((value) => value.trim())
			.filter((value) => value !== '');

		axios
			.post(API_URL + '/users/saveShareVerificationSettings', { emails })
			.then((response) => {
				const savedEmails = response.data.emails || [];
				shareVerificationEmailsText = savedEmails.join('\n');
				showShareVerificationSettingsSuccess = true;
				setTimeout(() => {
					showShareVerificationSettingsSuccess = false;
				}, 3000);
			})
			.catch((error) => {
				console.error(error);
				showShareVerificationSettingsError = true;
			})
			.finally(() => {
				isSavingShareVerificationSettings = false;
			});
	}

	function loadShareAccessLogs() {
		isLoadingShareAccessLogs = true;

		axios
			.get(API_URL + '/users/getShareAccessLogs')
			.then((response) => {
				shareAccessLogs = response.data.logs || [];
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				isLoadingShareAccessLogs = false;
			});
	}

	function clearShareAccessLogs() {
		if (isClearingShareAccessLogs) return;
		isClearingShareAccessLogs = true;

		axios
			.post(API_URL + '/users/clearShareAccessLogs')
			.then(() => {
				shareAccessLogs = [];
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				isClearingShareAccessLogs = false;
			});
	}

	function loadShareSMTPSettings() {
		axios
			.get(API_URL + '/users/getShareSMTPSettings')
			.then((response) => {
				const settings = response.data.settings || {};
				shareSMTPHost = settings.host || '';
				shareSMTPPort = settings.port || 587;
				shareSMTPUsername = settings.username || '';
				shareSMTPPassword = settings.password || '';
				shareSMTPFrom = settings.from || '';
			})
			.catch((error) => {
				console.error(error);
			});
	}

	function saveShareSMTPSettings() {
		if (isSavingShareSMTPSettings) return;
		isSavingShareSMTPSettings = true;
		showShareSMTPSettingsError = false;
		showShareSMTPSettingsSuccess = false;

		axios
			.post(API_URL + '/users/saveShareSMTPSettings', {
				host: shareSMTPHost,
				port: Number(shareSMTPPort),
				username: shareSMTPUsername,
				password: shareSMTPPassword,
				from: shareSMTPFrom
			})
			.then((response) => {
				const settings = response.data.settings || {};
				shareSMTPHost = settings.host || '';
				shareSMTPPort = settings.port || 587;
				shareSMTPUsername = settings.username || '';
				shareSMTPPassword = settings.password || '';
				shareSMTPFrom = settings.from || '';
				showShareSMTPSettingsSuccess = true;
				setTimeout(() => {
					showShareSMTPSettingsSuccess = false;
				}, 3000);
			})
			.catch((error) => {
				console.error(error);
				showShareSMTPSettingsError = true;
			})
			.finally(() => {
				isSavingShareSMTPSettings = false;
			});
	}

	function testShareSMTP() {
		if (isTestingShareSMTP) return;
		isTestingShareSMTP = true;
		showShareSMTPTestError = false;
		showShareSMTPTestSuccess = false;

		axios
			.post(API_URL + '/users/testShareSMTP', {
				to_email: shareSMTPTestRecipient,
				host: shareSMTPHost,
				port: Number(shareSMTPPort),
				username: shareSMTPUsername,
				password: shareSMTPPassword,
				from: shareSMTPFrom
			})
			.then(() => {
				showShareSMTPTestSuccess = true;
				setTimeout(() => {
					showShareSMTPTestSuccess = false;
				}, 3000);
			})
			.catch((error) => {
				console.error(error);
				showShareSMTPTestError = true;
			})
			.finally(() => {
				isTestingShareSMTP = false;
			});
	}

	function generateShareToken() {
		if (isGeneratingShareToken) return;
		isGeneratingShareToken = true;
		showShareTokenError = false;

		axios
			.post(API_URL + '/users/generateShareToken')
			.then((response) => {
				if (response.data.success) {
					hasShareToken = true;
					shareLink = window.location.origin + resolve('/share/' + response.data.token);
					linkCopiedSuccess = false;
					loadShareAccessLogs();
				} else {
					showShareTokenError = true;
				}
			})
			.catch((error) => {
				console.error(error);
				showShareTokenError = true;
			})
			.finally(() => {
				isGeneratingShareToken = false;
			});
	}

	function revokeShareToken() {
		if (isRevokingShareToken) return;
		isRevokingShareToken = true;
		showShareTokenError = false;

		axios
			.get(API_URL + '/users/revokeShareToken')
			.then((response) => {
				if (response.data.success) {
					hasShareToken = false;
					shareLink = '';
					linkCopiedSuccess = false;
					loadShareAccessLogs();
				} else {
					showShareTokenError = true;
				}
			})
			.catch((error) => {
				console.error(error);
				showShareTokenError = true;
			})
			.finally(() => {
				isRevokingShareToken = false;
			});
	}

	function copyShareLink() {
		if (!shareLink) return;
		navigator.clipboard.writeText(shareLink).then(
			() => {
				linkCopiedSuccess = true;
				setTimeout(() => {
					linkCopiedSuccess = false;
				}, 3000);
			},
			(err) => {
				console.error('Failed to copy share link: ', err);
			}
		);
	}

	function createBackupCodes() {
		if (isGeneratingBackupCodes) return;
		isGeneratingBackupCodes = true;

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

	let importFile = $state();
	let importFileProgress = $state(0);
	let importPassword = $state('');
	let importEncrypted = $state('');
	let isImporting = $state(false);
	let showImportError = $state(false);
	let importErrorMessage = $state('');
	let showImportSuccess = $state(false);

	function importData() {
		if (isImporting) return;
		isImporting = true;

		showImportError = false;
		showImportSuccess = false;
		importErrorMessage = '';

		const formData = new FormData();
		formData.append('file', importFile[0]);
		formData.append('encrypted', importEncrypted);
		formData.append('password', importPassword);

		const config = {
			onUploadProgress: (progressEvent) => {
				importFileProgress = Math.round(progressEvent.progress * 100);
			}
		};

		axios
			.post(API_URL + '/logs/importData', formData, {
				headers: {
					'Content-Type': 'multipart/form-data'
				},
				...config
			})
			.then((response) => {
				if (response.data.success) {
					showImportSuccess = true;
				} else {
					console.error('Error importing data');
					console.error(response.data);
					importErrorMessage = response.data.message;
					showImportError = true;
				}
			})
			.catch((error) => {
				console.error(error);

				importErrorMessage = error.response.data;
				showImportError = true;
			})
			.finally(() => {
				isImporting = false;
			});
	}

	let backupPassword = $state('');
	let backupPeriod = $state('backupPeriodAll');
	let backupStartDate = $state('');
	let backupEndDate = $state('');
	let backupEncrypted = $state(false);
	let backupIncludeFiles = $state(true);
	let backupIncludeTemplates = $state(true);
	let backupIncludeTags = $state(true);
	let backupIncludeBookmarks = $state(true);
	let isBackingUp = $state(false);
	let showBackupError = $state(false);

	function backupData() {
		if (isBackingUp) return;
		isBackingUp = true;
		showBackupError = false;

		axios
			.post(
				API_URL + '/logs/backup',
				{
					password: backupPassword,
					period: backupPeriod,
					startDate: backupStartDate,
					endDate: backupEndDate,
					encrypted: backupEncrypted,
					includeFiles: backupIncludeFiles,
					includeTemplates: backupIncludeTemplates,
					includeTags: backupIncludeTags,
					includeBookmarks: backupIncludeBookmarks
				},
				{
					responseType: 'blob' // Expect a binary response
				}
			)
			.then((response) => {
				const blob = new Blob([response.data], { type: 'application/zip' });
				const url = window.URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;

				let filename = `DailyTxT_Backup_${localStorage.getItem('user')}_${backupEncrypted ? 'encrypted' : 'decrypted'}_${new Date().toISOString().split('T')[0]}.zip`;

				a.download = filename;
				document.body.appendChild(a);
				a.click();
				a.remove();
				window.URL.revokeObjectURL(url);
			})
			.catch((error) => {
				console.error(error);

				showBackupError = true;
			})
			.finally(() => {
				isBackingUp = false;
			});
	}

	let exportPeriod = $state('periodAll');
	let exportStartDate = $state('');
	let exportEndDate = $state('');
	let exportImagesInHTML = $state(true);
	let exportSplit = $state('aio');
	let exportTagsInHTML = $state(true);
	let exportExtendedFormatting = $state(false);
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
			// these will be overwritten when loading the correct language file
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

	async function exportData() {
		// get correct language file depending on tolgee current language
		const currentLang = $tolgee.getLanguage();
		// load translation file (await to ensure data is ready before exporting)
		const module = await import(`../../i18n/${currentLang}.json`);
		const translations = module.default;
		// Use the loaded translations (original assignments, no extra fallback logic)
		exportTranslations.dateFormat = translations.export.dateFormat;
		exportTranslations.uiElements.exportTitle = translations.export.title;
		exportTranslations.uiElements.user = translations.export.user;
		exportTranslations.uiElements.exportedOn = translations.export.exportedOn;
		exportTranslations.uiElements.exportedOnFormat = translations.export.exportedOnFormat;
		exportTranslations.uiElements.entriesCount = translations.export.entriesCount;
		exportTranslations.uiElements.images = translations.export.images;
		exportTranslations.uiElements.files = translations.export.files;
		exportTranslations.uiElements.tags = translations.export.tags;

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
					translations: JSON.stringify(exportTranslations),
					extendedFormatting: exportExtendedFormatting
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

	function getVersionInfo() {
		axios
			.get(API_URL + '/version')
			.then((response) => {
				current_version = response.data.current_version;
				latest_stable_version = response.data.latest_stable_version;
				latest_overall_version = response.data.latest_overall_version;
				// Trigger update check after loading version info
				checkForUpdates();
			})
			.catch((error) => {
				console.error('Error fetching version info:', error);
			});
	}

	let showInstallationHelp = $state(false);
	$effect(() => {
		if (window.matchMedia('(display-mode: standalone)').matches) {
			showInstallationHelp = false;
			console.log('DailyTxT is installed');
		} else {
			showInstallationHelp = true;
			console.log('DailyTxT is not installed');
		}
	});
</script>

<div class="d-flex flex-column h-100">
	<nav class="navbar navbar-expand-lg glass">
		<div class="row w-100">
			<div class="col-lg-4 col-sm-5 col d-flex flex-row justify-content-start align-items-center">
				{#if !$alwaysShowSidenav}
					<button
						class="btn d-xl-none ms-1"
						type="button"
						data-bs-toggle="offcanvas"
						data-bs-target="#sidenav"
						aria-controls="sidenav"><Fa icon={faBars} /></button
					>
				{/if}

				<div class="selectMode form-check form-switch d-flex flex-row align-items-center">
					<label class="me-3 modeSliderIcon" for="selectMode"
						><Fa icon={faPencil} size="1.5x" /></label
					>
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
					<label class="ms-2 modeSliderIcon" for="selectMode"
						><Fa icon={faGlasses} size="1.5x" /></label
					>
				</div>
			</div>

			<div class="col-lg-4 col-sm-2 col d-flex flex-row justify-content-center align-items-center">
				<div class="full-logo d-flex align-items-center">
					<img src={dailytxt} alt="" height="38px" class="user-select-none" />
					<span class="dailytxt ms-2 user-select-none">DailyTxT</span>
				</div>
			</div>

			<div class="col-lg-4 col-sm-5 col pe-0 d-flex flex-row justify-content-end">
				{#if window.DEMO_MODE === true}
					<div class="dropdown me-2" id="demoInNavbar">
						<button
							class="btn btn-info dropdown-toggle"
							type="button"
							data-bs-toggle="dropdown"
							aria-expanded="false"
						>
							DEMO
						</button>
						<div class="dropdown-menu dropdown-menu-end" id="demoModeDropdown">
							<DemoModeText />
						</div>
					</div>
				{/if}

				<div class="dropdown">
					<button
						type="button"
						class="btn btn-outline-secondary dropdown-toggle"
						data-bs-toggle="dropdown"
						aria-expanded="false"
					>
						<Fa icon={faSliders} />
						{#if updateAvailable}
							<FaLayers class="position-absolute top-0 start-100 translate-middle">
								<Fa icon={faCircle} size="1.2x" color="white" />
								<Fa icon={faCircleUp} size="1.2x" class="text-info" />
							</FaLayers>
						{/if}
					</button>
					<div class="dropdown-menu dropdown-menu-end glass-shadow p-4 greet-menu">
						<div class="d-flex flex-row justify-content-center mb-3">
							<h3 class="greeting">
								{@html $t('navbar.greeting', {
									user: `<span class="username">${currentUser}</span>`
								})}
								{pickSmiley()}
							</h3>
						</div>
						<div class="d-flex flex-row justify-content-center">
							<button
								class="btn btn-outline-secondary position-relative"
								onclick={openSettingsModal}
								data-bs-toggle="dropdown"
							>
								{$t('settings.title')}
								<Fa icon={faSliders} />
								{#if updateAvailable}
									<FaLayers class="position-absolute top-0 start-100 translate-middle">
										<Fa icon={faCircle} size="1.2x" color="white" />
										<Fa icon={faCircleUp} size="1.2x" class="text-info" />
									</FaLayers>
								{/if}
							</button>
						</div>
						<hr />
						<div class="d-flex flex-row">
							<button class="btn btn-outline-danger ms-auto" onclick={() => logout(null)}
								>{$t('nav.logout')} <Fa icon={faPersonRunning} /></button
							>
						</div>
					</div>
				</div>
			</div>
		</div>
	</nav>

	<div class="transition-stack flex-fill position-relative">
		{#key page.data}
			<div
				class="transition-wrapper overflow-y-auto position-absolute top-0 bottom-0 start-0 end-0"
				out:blur={{ duration: outDuration }}
				in:blur={{ duration: inDuration, delay: outDuration }}
			>
				{@render children()}
			</div>
		{/key}
	</div>
</div>

<TagModal
	bind:this={editTagModal}
	createTag={false}
	bind:editTag
	isSaving={isSavingEditedTag}
	{saveEditedTag}
/>

<ChangelogModal {changelog} {current_version} />

{#snippet unsavedChanges()}
	<div class="unsaved-changes" data-content={$t('settings.unsaved_changes')} transition:slide></div>
{/snippet}

<!-- Full screen modal -->
<div class="modal fade" data-bs-backdrop="static" id="settingsModal">
	<div
		class="modal-dialog modal-dialog-scrollable modal-dialog-centered modal-xl modal-fullscreen-lg-down"
	>
		<!--  -->
		<div class="modal-content shadow-lg">
			<div class="modal-header flex-wrap gap-2">
				<div class="d-flex w-100 align-items-center">
					<div
						class="btn-group flex-grow-1 overflow-auto position-relative"
						id="settingsTabGroup"
						role="group"
						aria-label="Settings views"
						style="scrollbar-width: none; -ms-overflow-style: none;"
						bind:this={settingsTabGroup}
					>
						<!-- Sliding indicator -->
						<div
							class="sliding-indicator"
							style="transform: translateX({indicatorNeedsUpdate &&
								getSlideOffset(activeSettingsView)}px); width: {indicatorNeedsUpdate &&
								getSlideWidth(activeSettingsView)}px;"
						></div>

						<button
							type="button"
							class="btn btn-outline-primary flex-shrink-0 {activeSettingsView === 'settings'
								? 'active'
								: ''}"
							onclick={switchToSettingsTab}
							bind:this={settingsButton}
						>
							{$t('settings.title')}
						</button>
						<button
							type="button"
							class="btn btn-outline-primary flex-shrink-0 {activeSettingsView === 'stats'
								? 'active'
								: ''}"
							onclick={switchToStatsTab}
							bind:this={statsButton}
						>
							{$t('settings.statistics.title')}
						</button>
						<button
							type="button"
							class="btn btn-outline-primary flex-shrink-0 {activeSettingsView === 'admin'
								? 'active'
								: ''}"
							onclick={switchToAdminTab}
							bind:this={adminButton}
						>
							{$t('settings.admin.title')}
						</button>
					</div>
					<button
						type="button"
						class="btn-close ms-3 flex-shrink-0"
						data-bs-dismiss="modal"
						aria-label="Close"
					></button>
				</div>
			</div>
			<div
				class="modal-body {activeSettingsView !== 'settings' ? 'modal-body-scrollable' : ''}"
				id="modal-body"
			>
				<div class="row">
					{#if activeSettingsView === 'settings'}
						<div class="col-4 overflow-y-auto d-none d-md-block">
							<nav class="flex-column align-items-stretch" id="settings-nav">
								<nav class="nav nav-pills flex-column custom-scrollspy-nav">
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'appearance'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('appearance')}
										>🎨 {$t('settings.appearance')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'features'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('features')}>🛠️ {$t('settings.features')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'tags'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('tags')}>#️⃣ {$t('settings.tags')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'templates'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('templates')}
										>📝 {$t('settings.templates')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'data'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('data')}>📁 {$t('settings.data')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'security'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('security')}>🔒 {$t('settings.security')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'sharing'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('sharing')}>🔗 {$t('settings.sharing')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'account'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('account')}>👤 {$t('settings.account')}</button
									>
									<button
										type="button"
										class="nav-link mb-1 text-start {activeSettingsSection === 'about'
											? 'active'
											: ''}"
										onclick={() => scrollToSection('about')}
									>
										💡 {$t('settings.about')}
										{#if updateAvailable}
											<FaLayers>
												<Fa icon={faCircle} size="1.2x" color="white" />
												<Fa icon={faCircleUp} size="1.2x" class="text-info" />
											</FaLayers>
										{/if}
									</button>
								</nav>
							</nav>
						</div>
						<div class="col-12 col-md-8">
							<div
								class="settings-content overflow-y-auto"
								data-bs-spy="scroll"
								data-bs-target="#settings-nav"
								data-bs-smooth-scroll="true"
								id="settings-content"
							>
								<!-- Mobile dropdown (visible on < md) -->
								<div class="d-md-none position-sticky top-0 p-1 mobile-settings-dropdown">
									<select
										id="settingsSectionSelect"
										class="form-select form-select-sm"
										bind:value={activeSettingsSection}
										onchange={() => scrollToSection(activeSettingsSection)}
									>
										{#each settingsSectionsMeta as sec}
											<option value={sec.id}>{$t(sec.labelKey)}</option>
										{/each}
									</select>
								</div>
								<div id="appearance">
									<Appearance {unsavedChanges} />
								</div>

								<div id="features">
									<Features
										{unsavedChanges}
										bind:aLookBackYears
										{aLookBackYearsInvalid}
										{tolgeesMatchForBrowserLanguage}
									/>
								</div>

								<div id="tags">
									<Tags
										{tags}
										{openTagModal}
										{deleteTagId}
										{askDeleteTag}
										{isDeletingTag}
										{deleteTag}
									/>
								</div>

								<div id="templates">
									<Templates
										{unsavedChanges}
										bind:selectedTemplate
										{updateSelectedTemplate}
										{confirmDeleteTemplate}
										bind:templateName
										bind:templateText
										{deleteTemplate}
										{oldTemplateName}
										{oldTemplateText}
										{isSavingTemplate}
										{isDeletingTemplate}
										{saveTemplate}
									/>
								</div>

								<div id="data">
									<Data
										{unsavedChanges}
										bind:importFile
										{importFileProgress}
										bind:importPassword
										bind:importEncrypted
										{isImporting}
										{showImportError}
										bind:importErrorMessage
										{showImportSuccess}
										{importData}
										bind:backupPassword
										bind:backupPeriod
										bind:backupStartDate
										bind:backupEndDate
										bind:backupEncrypted
										bind:backupIncludeFiles
										bind:backupIncludeTemplates
										bind:backupIncludeTags
										bind:backupIncludeBookmarks
										{isBackingUp}
										{showBackupError}
										{backupData}
										bind:exportPeriod
										bind:exportStartDate
										bind:exportEndDate
										bind:exportImagesInHTML
										bind:exportSplit
										bind:exportTagsInHTML
										bind:exportExtendedFormatting
										{isExporting}
										{exportData}
									/>
								</div>

								<div id="security">
									<Security
										{unsavedChanges}
										bind:currentPassword
										bind:newPassword
										bind:confirmNewPassword
										{isChangingPassword}
										{changePasswordNotEqual}
										{changingPasswordSuccess}
										{changingPasswordError}
										{changingPasswordIncorrect}
										{changePassword}
										bind:backupCodesPassword
										{isGeneratingBackupCodes}
										{backupCodes}
										{codesCopiedSuccess}
										{showBackupCodesError}
										{createBackupCodes}
										{copyBackupCodes}
									/>
								</div>

								<div id="sharing">
									<Sharing
										{hasShareToken}
										{shareLink}
										{isGeneratingShareToken}
										{isRevokingShareToken}
										{linkCopiedSuccess}
										{showShareTokenError}
										{generateShareToken}
										{revokeShareToken}
										{copyShareLink}
										bind:shareVerificationEmailsText
										{isLoadingShareVerificationSettings}
										{isSavingShareVerificationSettings}
										{showShareVerificationSettingsError}
										{showShareVerificationSettingsSuccess}
										{shareVerificationSMTPConfigured}
										{saveShareVerificationSettings}
										{shareAccessLogs}
										{isLoadingShareAccessLogs}
										{loadShareAccessLogs}
										{clearShareAccessLogs}
										{isClearingShareAccessLogs}
										bind:shareSMTPHost
										bind:shareSMTPPort
										bind:shareSMTPUsername
										bind:shareSMTPPassword
										bind:shareSMTPFrom
										bind:shareSMTPTestRecipient
										{saveShareSMTPSettings}
										{isSavingShareSMTPSettings}
										{showShareSMTPSettingsError}
										{showShareSMTPSettingsSuccess}
										{testShareSMTP}
										{isTestingShareSMTP}
										{showShareSMTPTestError}
										{showShareSMTPTestSuccess}
									/>
								</div>

								<div id="account">
									<Account
										{currentUser}
										bind:newUsername
										bind:changeUsernamePassword
										{isChangingUsername}
										{changeUsernameSuccess}
										{changeUsernameError}
										{changeUsernamePasswordIncorrect}
										{changeUsername}
										{showConfirmDeleteAccount}
										bind:deleteAccountPassword
										{isDeletingAccount}
										{deleteAccountPasswordIncorrect}
										{showDeleteAccountSuccess}
										{deleteAccount}
									/>
								</div>

								<div id="about">
									<About
										{current_version}
										{latest_stable_version}
										{latest_overall_version}
										{updateAvailable}
										{showInstallationHelp}
										{unsavedChanges}
										{checkChangelog}
									/>
								</div>
							</div>
						</div>
					{/if}
					{#if activeSettingsView === 'stats'}
						<div class="col-12">
							<Statistics />
						</div>
					{/if}
					{#if activeSettingsView === 'admin'}
						<div class="col-12">
							<Admin />
						</div>
					{/if}
				</div>
			</div>
			<div class="modal-footer">
				{#if activeSettingsView === 'settings'}
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
				{:else}
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">
						{$t('settings.close') || 'Close'}
					</button>
				{/if}
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
	#demoModeDropdown {
		width: min(500px, 90vw);
		padding: 1rem;
	}

	@media (max-width: 840px) {
		#demoInNavbar {
			display: none;
		}
	}

	.navbar {
		border-top: 0 !important;
		border-left: 0 !important;
		border-right: 0 !important;
		z-index: 15;
	}

	/* Limit settings dropdown width on viewport and keep greeting on one line when possible */
	.greet-menu {
		max-width: calc(100vw - 50px);
		width: max-content; /* shrink-to-fit to content up to max */
		min-width: 200px;
		border-radius: 10px;
	}
	.greeting {
		/* Let the dropdown grow up to its max-width and then wrap */
		white-space: normal;
		overflow-wrap: anywhere; /* allow breaking long words/usernames */
		word-break: break-word;
	}

	/* Allow the stacked absolute children to scroll without forcing the parent to expand */
	.transition-stack {
		min-height: 0;
	}
	:global(body[data-bs-theme='dark'] .multiselect) {
		background: #212529 !important;
		border: 1px solid #212529 !important;
	}

	:global(body[data-bs-theme='dark'] .multiselect > ul) {
		background: #212529 !important;
	}

	:global(.multiselect.disabled) {
		color: grey !important;
	}

	#settingsTabGroup > button {
		transition: text-decoration 0.3s ease;
	}
	:global(body[data-bs-theme='dark']) #settingsTabGroup > button {
		color: white;
	}
	:global(body[data-bs-theme='light']) #settingsTabGroup > button {
		color: black;
	}

	#settingsTabGroup > button.active {
		text-decoration-color: #f57c00;
		text-decoration-thickness: 3px;
	}

	:global(body[data-bs-theme='light']) #settingsTabGroup {
		background-color: #b8b8b8;
	}

	@media (max-width: 450px) {
		.modeSliderIcon {
			font-size: 0.8rem !important;
			margin: 0 !important;
		}
		#selectMode {
			transform: scale(1) !important;
			margin-left: -2.25rem !important;
		}
		.selectMode {
			padding-left: 1rem !important;
		}
	}

	@media (max-width: 600px) {
		.dailytxt {
			display: none;
		}
	}

	.dailytxt {
		color: #f57c00;
		font-size: 1.8rem;
		font-weight: 500;
		line-height: 1rem;
		position: relative;
		text-decoration: none;
		transition: color 0.3s ease;
	}

	.dailytxt::after {
		content: '';
		position: absolute;
		bottom: -9px;
		left: 50%;
		width: 0;
		height: 2px;
		background-color: #0d6efd;
		transform: translateX(-50%);
		transition: width 0.3s ease;
		z-index: -1;
	}

	.full-logo:hover > .dailytxt::after {
		width: 100%;
	}

	img {
		transition: 0.3s ease;
	}

	.full-logo:hover > img {
		transform: scale(1.15);
		filter: drop-shadow(0px 0px 4px #4891ff);
	}

	.modal-header > div > div > button {
		border: none;
		border-radius: 10px !important;
		text-decoration: underline;
		align-self: center;
	}

	.modal-header > div > .btn-group {
		background: #a8a8a83d;
	}

	:global(#settings-content h5, #settings-content h6) {
		font-weight: 600;
		text-decoration: underline;
		text-decoration-color: #0d6efd;
	}

	:global(h6) {
		margin-top: 0.7rem;
	}

	:global(.backupCode) {
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

	:global(:has(> .unsaved-changes)) {
		outline: 1px solid orange;
	}

	:global(.unsaved-changes) {
		background-color: orange;
		margin-top: -0.5rem;
		margin-left: -0.5rem;
		margin-right: -0.5rem;
		border-top-left-radius: 10px;
		border-top-right-radius: 10px;
		padding-left: 0.5rem;
		margin-bottom: 0.5rem;
		color: black;
	}

	:global(.unsaved-changes::before) {
		content: attr(data-content);
	}

	:global(.tagColumn > span) {
		width: min-content;
	}

	.selectMode > label {
		cursor: pointer;
	}

	.selectMode > .form-check,
	.selectMode {
		margin-bottom: 0 !important;
	}

	#selectMode:checked {
		border-color: #f57c00;
		background-color: #f57c00;
	}

	#selectMode:not(:checked) {
		background-color: #2196f3;
		background-image: url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='-4 -4 8 8'%3e%3ccircle r='3' fill='rgba(255, 255, 255, 1)'/></svg>");
	}

	.settings-content > div {
		padding: 0.5rem;
		position: relative;
	}

	:global(#settings-content > div > div) {
		padding: 0.5rem;
		border-radius: 10px;
		margin-bottom: 1rem;
	}
	:global(body[data-bs-theme='dark'] #settings-content > div > div) {
		background-color: #8080805d;
		box-shadow: 3px 3px 8px 4px #0000003f;
	}
	:global(body[data-bs-theme='light'] #settings-content > div > div) {
		background-color: #dfdfdf5d;
		box-shadow: 3px 3px 8px 4px #11111134;
	}

	:global(h3.text-primary) {
		font-weight: 700;
		position: sticky;
		top: 0;
		backdrop-filter: blur(10px) saturate(150%);
		background-color: rgba(240, 240, 240, 0.9);
		padding: 4px;
		border-radius: 5px;
		z-index: 10;
	}

	.modal-body {
		/* For settings tab, let internal elements handle scrolling */
		overflow-y: hidden;
	}

	.modal-body.modal-body-scrollable {
		/* For stats/admin tabs, let modal-body handle scrolling */
		overflow-y: auto;
	}

	.modal-header {
		border-bottom: none;
	}

	.modal-footer {
		border-top: none;
	}

	/* Custom ScrollSpy styles */
	.custom-scrollspy-nav .nav-link {
		border-left: 4px solid transparent;
		transition:
			background-color 0.18s ease,
			color 0.18s ease,
			border-color 0.25s ease;
		will-change: background-color, color, border-color;
	}
	:global(body[data-bs-theme='dark']) .custom-scrollspy-nav .nav-link {
		color: #9ac2ff;
	}
	:global(body[data-bs-theme='light']) .custom-scrollspy-nav .nav-link {
		color: #0c6dff;
	}

	.custom-scrollspy-nav .nav-link.active {
		font-weight: 600;
	}
	:global(body[data-bs-theme='dark']) .custom-scrollspy-nav .nav-link.active {
		background-color: rgba(116, 116, 116, 0.521);
		color: #62a1ff;
		border-left-color: #0d6efd;
	}
	:global(body[data-bs-theme='light']) .custom-scrollspy-nav .nav-link.active {
		background-color: rgba(13, 110, 253, 0.1);
		color: #0066ff;
		border-left-color: #0d6efd;
	}
	.custom-scrollspy-nav .nav-link:not(.active):hover {
		background-color: rgba(13, 110, 253, 0.05);
	}

	/* Mobile settings dropdown styling */
	.mobile-settings-dropdown {
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
		border-radius: 0.5rem;
		z-index: 20;
		backdrop-filter: blur(8px);
		background: rgba(255, 255, 255, 0.85);
	}
	@media (max-width: 767.98px) {
		/* Add a small spacer below dropdown to prevent content being fully hidden */
		#settings-content > .mobile-settings-dropdown + div {
			margin-top: 0.25rem;
		}
	}

	/* Hide scrollbar for tab buttons on small screens */
	.btn-group.overflow-auto::-webkit-scrollbar {
		display: none;
	}
	.btn-group.overflow-auto {
		scrollbar-width: none;
		-ms-overflow-style: none;
	}

	/* Sliding indicator for settings tabs */
	.sliding-indicator {
		position: absolute;
		top: 0;
		height: 100%;
		background-color: var(--bs-primary);
		border-radius: 0.375rem;
		transition:
			transform 0.3s cubic-bezier(0.4, 0, 0.2, 1),
			width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
		z-index: 0;
		pointer-events: none;
	}

	/* Ensure buttons are above the indicator */
	.btn-group .btn {
		position: relative;
		z-index: 1;
		background-color: transparent !important;
		border-color: transparent !important;
	}

	/* Active button styling - remove background since indicator handles it */
	.btn-group .btn.active {
		background-color: transparent !important;
		border-color: transparent !important;
		color: white !important;
	}

	/* Hover effect */
	.btn-group .btn:hover {
		background-color: rgba(13, 110, 253, 0.1) !important;
		border-color: transparent !important;
	}
</style>

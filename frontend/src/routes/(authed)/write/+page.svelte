<script>
	import '../../../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Sidenav from '$lib/Sidenav.svelte';
	import { selectedDate, cal, readingDate } from '$lib/calendarStore.js';
	import axios from 'axios';
	import { goto } from '$app/navigation';
	import { mount, onMount } from 'svelte';
	import { searchString, searchResults } from '$lib/searchStore.js';
	import * as TinyMDE from 'tiny-markdown-editor';
	import '../../../../node_modules/tiny-markdown-editor/dist/tiny-mde.css';
	import { API_URL } from '$lib/APIurl.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import {
		faCloudArrowUp,
		faCloudArrowDown,
		faSquarePlus,
		faQuestionCircle,
		faClockRotateLeft,
		faArrowLeft,
		faArrowRight,
		faTrash
	} from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import { v4 as uuidv4 } from 'uuid';
	import { slide, fade } from 'svelte/transition';
	import { settings, autoLoadImagesThisDevice } from '$lib/settingsStore';
	import { tags } from '$lib/tagStore';
	import Tag from '$lib/Tag.svelte';
	import TagModal from '$lib/TagModal.svelte';
	import FileList from '$lib/FileList.svelte';
	import { formatBytes, alwaysShowSidenav, sameDate } from '$lib/helpers.js';
	import ImageViewer from '$lib/ImageViewer.svelte';
	import TemplateDropdown from '$lib/TemplateDropdown.svelte';
	import { insertTemplate } from '$lib/templateStore';
	import ALookBack from '$lib/ALookBack.svelte';
	import { marked } from 'marked';
	import { T, getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	let cancelDownload = new AbortController();

	let tinyMDE;
	onMount(() => {
		// If we come from read mode, keep the last visible day as the active selected day
		if ($readingDate) {
			$selectedDate = $readingDate; // promote readingDate to selectedDate when switching to write mode
		}
		$readingDate = null; // no reading-highlighting when in write mode

		tinyMDE = new TinyMDE.Editor({ element: 'editor', content: '' });
		let commandBar = new TinyMDE.CommandBar({ element: 'toolbar', editor: tinyMDE });
		document.getElementsByClassName('TinyMDE')[0].classList.add('focus-ring');

		tinyMDE.addEventListener('change', (event) => {
			currentLog = event.content;
			handleInput();
		});

		mount(TemplateDropdown, {
			target: document.querySelector('.TMCommandBar')
		});

		getLog();

		// enable popovers
		const popoverTriggerList = document.querySelectorAll('[data-bs-toggle="popover"]');
		const popoverList = [...popoverTriggerList].map(
			(popoverTriggerEl) =>
				new bootstrap.Popover(popoverTriggerEl, { trigger: 'focus', html: true })
		);
	});

	$effect(() => {
		if ($insertTemplate) {
			currentLog = currentLog + $insertTemplate;
			tinyMDE.setContent(currentLog);

			$insertTemplate = '';
		}
	});

	$effect(() => {
		if (currentLog !== savedLog) {
			document.getElementsByClassName('TinyMDE')[0].classList.add('notSaved');
		} else {
			document.getElementsByClassName('TinyMDE')[0].classList.remove('notSaved');
		}
	});

	let lastSelectedDate = $state($selectedDate);
	let images = $state([]);
	let filesOfDay = $state([]);

	let loading = false;
	$effect(() => {
		if (!sameDate($selectedDate, lastSelectedDate)) {
			cancelDownload.abort();
			cancelDownload = new AbortController();

			if (loading) return;
			loading = true;

			images = [];
			filesOfDay = [];

			clearTimeout(timeout);
			const result = getLog();
			if (result) {
				lastSelectedDate = $selectedDate;
				$cal.currentYear = $selectedDate.year;
				$cal.currentMonth = $selectedDate.month - 1;
			} else {
				$selectedDate = lastSelectedDate;
			}
		}
		loading = false;
	});

	let altPressed = false;
	let ctrlPressed = false;
	function on_key_down(event) {
		if (event.key === 'Alt') {
			event.preventDefault();
			altPressed = true;
		}
		if (event.key === 'ArrowRight' && altPressed) {
			event.preventDefault();
			changeDay(+1);
		} else if (event.key === 'ArrowLeft' && altPressed) {
			event.preventDefault();
			changeDay(-1);
		}
		if (event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = true;
		}
		if (event.key === 'g' && ctrlPressed) {
			event.preventDefault();
			document.getElementById('tag-input').focus();
		}
	}

	function on_key_up(event) {
		if (event.key === 'Alt') {
			event.preventDefault();
			altPressed = false;
		}
		if (event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = false;
		}
	}

	function changeDay(increment) {
		$selectedDate = {
			day: $selectedDate.day + increment,
			month: $selectedDate.month,
			year: $selectedDate.year
		};
	}

	let currentLog = $state('');
	let savedLog = $state('');

	let logDateWritten = $state('');

	let timeout;

	function debounce(fn) {
		clearTimeout(timeout);
		timeout = setTimeout(() => fn(), 1000);
	}

	function handleInput() {
		debounce(() => {
			saveLog();
		});
	}

	let historyAvailable = $state(false);
	async function getLog() {
		if (savedLog !== currentLog) {
			const success = await saveLog();
			if (!success) {
				return false;
			}
		}

		try {
			const response = await axios.get(API_URL + '/logs/getLog', {
				params: {
					day: $selectedDate.day,
					month: $selectedDate.month,
					year: $selectedDate.year
				}
			});

			currentLog = response.data.text;
			filesOfDay = response.data.files;
			selectedTags = response.data.tags;
			historyAvailable = response.data.history_available;

			savedLog = currentLog;

			tinyMDE.setContent(currentLog);
			tinyMDE.setSelection({ row: 0, col: 0 });

			logDateWritten = response.data.date_written;

			getALookBack();

			return true;
		} catch (error) {
			console.error(error.response);
			// toast
			const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingLog'));
			toast.show();

			return false;
		}
	}

	let aLookBack = $state([]);

	function getALookBack() {
		// Skip if settings not loaded yet
		if (!$settings || $settings.useALookBack === undefined || !$settings.useALookBack) {
			aLookBack = [];
			return;
		}

		axios
			.get(API_URL + '/logs/getALookBack', {
				params: {
					day: $selectedDate.day,
					month: $selectedDate.month,
					year: $selectedDate.year,
					last_years: $settings.aLookBackYears.join(',')
				}
			})
			.then((response) => {
				aLookBack = response.data;
			})
			.catch((error) => {
				console.error(error);
			});
	}

	// Re-trigger aLookBack when settings are loaded/changed
	$effect(() => {
		if ($settings && $settings.useALookBack !== undefined) {
			getALookBack();
		}
	});

	const imageExtensions = ['jpeg', 'jpg', 'gif', 'png', 'webp'];
	//TODO: support svg? -> minsize is necessary...

	$effect(() => {
		if (filesOfDay) {
			// add all files to images if correct extension
			filesOfDay.forEach((file) => {
				// if image -> load it!
				if (
					imageExtensions.includes(file.filename.split('.').pop().toLowerCase()) &&
					!images.find((image) => image.uuid_filename === file.uuid_filename)
				) {
					images = [...images, file];

					if (autoLoadImages) {
						loadImage(file);
					}
				}
			});
		}
	});

	let autoLoadImages = $derived(
		($settings.setAutoloadImagesPerDevice && $autoLoadImagesThisDevice) ||
			(!$settings.setAutoloadImagesPerDevice && $settings.autoloadImagesByDefault)
	);

	function loadImage(file) {
		images.map((image) => {
			if (image.uuid_filename === file.uuid_filename) {
				image.loading = true;
			}
			return image;
		});

		axios
			.get(API_URL + '/logs/downloadFile', {
				params: { uuid: file.uuid_filename },
				responseType: 'blob',
				signal: cancelDownload.signal
			})
			.then((response) => {
				const url = URL.createObjectURL(new Blob([response.data]));
				images = images.map((image) => {
					if (image.uuid_filename === file.uuid_filename) {
						image.src = url;
						file.src = url;
						image.loading = false;
					}
					return image;
				});
			})
			.catch((error) => {
				if (error.name == 'CanceledError') {
					return;
				}

				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingFile'));
				toast.show();
			});
	}

	function loadImages() {
		images.forEach((image) => {
			if (!image.src) {
				loadImage(image);
			}
		});
	}

	async function saveLog() {
		if (currentLog === savedLog) {
			return true;
		}

		// axios to backend
		let timezone = $settings.useBrowserTimezone
			? Intl.DateTimeFormat().resolvedOptions().timeZone
			: $settings.timezone;
		let date_written = new Date().toLocaleString('de-DE', {
			timeZone: timezone,
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		});

		let dateOfSave = lastSelectedDate;
		try {
			const response = await axios.post(API_URL + '/logs/saveLog', {
				day: lastSelectedDate.day,
				month: lastSelectedDate.month,
				year: lastSelectedDate.year,
				text: currentLog,
				date_written: date_written
			});

			if (response.data.success) {
				savedLog = currentLog;
				logDateWritten = date_written;
				historyAvailable = response.data.history_available;

				// add to $cal.daysWithLogs
				if (!$cal.daysWithLogs.includes(lastSelectedDate.day)) {
					$cal.daysWithLogs = [...$cal.daysWithLogs, dateOfSave.day];
				}

				return true;
			} else {
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSavingLog'));
				toast.show();
				console.error('Log not saved');
				return false;
			}
		} catch (error) {
			// toast
			const toast = new bootstrap.Toast(document.getElementById('toastErrorSavingLog'));
			toast.show();
			console.error(error);
			return false;
		}
	}

	$effect(() => {
		if ($searchString === '') {
			$searchResults = [];
		}
	});

	function triggerFileInput() {
		document.getElementById('fileInput').click();
	}

	function onFileChange(event) {
		for (let i = 0; i < event.target.files.length; i++) {
			uploadFile(event.target.files[i]);
		}
	}

	let uploadingFiles = $state([]);
	let isDragOver = $state(false);
	let dragCounter = $state(0); // Track drag enter/leave events
	let draggedFileCount = $state(0); // Store file count during drag

	// Drag and drop handlers
	function handleDragEnter(event) {
		event.preventDefault();

		// Check if dragging files, not text or other content
		if (!hasFiles(event.dataTransfer)) {
			return;
		}

		dragCounter++;
		if (dragCounter === 1) {
			isDragOver = true;
			// Try to get file count
			extractDragInfo(event);
		}
	}

	function handleDragLeave(event) {
		event.preventDefault();

		// Only handle if we're actually dragging files
		if (!isDragOver) return;

		dragCounter--;
		if (dragCounter === 0) {
			isDragOver = false;
			draggedFileCount = 0;
		}
	}

	function handleDragOver(event) {
		event.preventDefault();

		// Check if dragging files, not text or other content
		if (!hasFiles(event.dataTransfer)) {
			return;
		}

		event.dataTransfer.dropEffect = 'copy';
		// Try again if we haven't got info yet
		if (draggedFileCount === 0) {
			extractDragInfo(event);
		}
	}

	function hasFiles(dataTransfer) {
		// Check if dataTransfer contains files
		return (
			dataTransfer.types &&
			(dataTransfer.types.includes('Files') ||
				dataTransfer.types.includes('application/x-moz-file') ||
				(dataTransfer.items && Array.from(dataTransfer.items).some((item) => item.kind === 'file')))
		);
	}

	function extractDragInfo(event) {
		// Double-check that we have files
		if (!hasFiles(event.dataTransfer)) {
			return;
		}

		const items = event.dataTransfer.items;
		if (items && items.length > 0) {
			let fileCount = 0;

			for (let i = 0; i < items.length; i++) {
				const item = items[i];
				if (item.kind === 'file') {
					fileCount++;
				}
			}

			if (fileCount > 0) {
				draggedFileCount = fileCount;
			}
		}
	}

	function handleDrop(event) {
		event.preventDefault();

		// Reset drag state
		isDragOver = false;
		dragCounter = 0;
		draggedFileCount = 0;

		// Check if we actually have files to upload
		const files = event.dataTransfer.files;
		if (files && files.length > 0) {
			for (let i = 0; i < files.length; i++) {
				uploadFile(files[i]);
			}
		}
	}

	function uploadFile(f) {
		let uuid = uuidv4();

		uploadingFiles = [...uploadingFiles, { name: f.name, progress: 0, size: f.size, uuid: uuid }];

		const config = {
			onUploadProgress: (progressEvent) => {
				uploadingFiles = uploadingFiles.map((file) => {
					if (file.uuid === uuid) {
						file.progress = Math.round(progressEvent.progress * 100);
					}
					return file;
				});
			}
		};

		const formData = new FormData();
		formData.append('day', $selectedDate.day);
		formData.append('month', $selectedDate.month);
		formData.append('year', $selectedDate.year);
		formData.append('file', f);
		formData.append('uuid', uuid);

		axios
			.post(API_URL + '/logs/uploadFile', formData, {
				...config
			})
			.then((response) => {
				// append to filesOfDay
				filesOfDay = [...filesOfDay, { filename: f.name, size: f.size, uuid_filename: uuid }];
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSavingFile'));
				toast.show();
			})
			.finally(() => {
				uploadingFiles = uploadingFiles.filter((file) => file.uuid !== uuid);
			});
	}

	function downloadFile(uuid) {
		// check if present in filesOfDay
		let file = filesOfDay.find((file) => file.uuid_filename === uuid);
		if (file?.src) {
			triggerAutomaticDownload(uuid);
			return;
		}

		// otherwise: download from server

		filesOfDay = filesOfDay.map((f) => {
			if (f.uuid_filename === uuid) {
				f.downloadProgress = 0;
			}
			return f;
		});

		const config = {
			params: { uuid: uuid },
			onDownloadProgress: (progressEvent) => {
				filesOfDay = filesOfDay.map((file) => {
					if (file.uuid_filename === uuid) {
						file.downloadProgress = Math.round((progressEvent.loaded / file.size) * 100);
					}
					return file;
				});
			},
			signal: cancelDownload.signal,
			responseType: 'blob'
		};

		axios
			.get(API_URL + '/logs/downloadFile', {
				...config
			})
			.then((response) => {
				const url = URL.createObjectURL(new Blob([response.data]));
				filesOfDay = filesOfDay.map((f) => {
					if (f.uuid_filename === uuid) {
						f.src = url;
					}
					return f;
				});
			})
			.catch((error) => {
				if (error.name == 'CanceledError') {
					return;
				}

				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingFile'));
				toast.show();
			})
			.finally(() => {
				// remove progress
				filesOfDay = filesOfDay.map((f) => {
					if (f.uuid_filename === uuid) {
						f.downloadProgress = -1;
					}
					return f;
				});

				triggerAutomaticDownload(uuid);
			});
	}

	function triggerAutomaticDownload(uuid) {
		let file;
		for (let i = 0; i < filesOfDay.length; i++) {
			if (filesOfDay[i].uuid_filename === uuid) {
				file = filesOfDay[i];
				break;
			}
		}

		const a = document.createElement('a');
		a.href = file.src;
		a.download = file.filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
	}

	let confirmDelete = $state({ uuid: '', filename: '' });
	function askDeleteFile(uuid, filename) {
		confirmDelete = { uuid: uuid, filename: filename };

		const modal = new bootstrap.Modal(document.getElementById('modalConfirmDeleteFile'));
		modal.show();
	}

	function deleteFile(uuid) {
		axios
			.get(API_URL + '/logs/deleteFile', {
				params: {
					uuid: uuid,
					year: $selectedDate.year,
					month: $selectedDate.month,
					day: $selectedDate.day
				}
			})
			.then((response) => {
				filesOfDay = filesOfDay.filter((file) => file.uuid_filename !== uuid);
				images = images.filter((image) => image.uuid_filename !== uuid);
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorDeletingFile'));
				toast.show();
			});
	}

	let searchTab = $state('');
	let showTagDropdown = $state(false);

	let filteredTags = $state([]);
	let selectedTags = $state([]);

	// Action: portal dropdown to <body> and position it under the input
	function portalDropdown(node, params) {
		let anchorEl;

		function getAnchor() {
			if (params?.anchor) {
				return typeof params.anchor === 'string'
					? document.querySelector(params.anchor)
					: params.anchor;
			}
			return document.getElementById('tag-input');
		}

		function position() {
			if (!anchorEl) return;
			const rect = anchorEl.getBoundingClientRect();
			node.style.position = 'fixed';
			node.style.top = rect.bottom + 'px';
			node.style.left = rect.left + 'px';
			/* node.style.width = rect.width + 'px'; */
			// keep within viewport horizontally (basic guard)
			const maxLeft = Math.max(8, Math.min(rect.left, window.innerWidth - node.offsetWidth - 8));
			node.style.left = maxLeft + 'px';
		}

		function attach() {
			// move element into body so it's not clipped by ancestors and backdrop-filter works as expected
			document.body.appendChild(node);
			position();
		}

		function onScroll() {
			position();
		}
		function onResize() {
			position();
		}

		anchorEl = getAnchor();
		attach();
		// use capture to react to scrolls on any ancestor
		window.addEventListener('scroll', onScroll, true);
		window.addEventListener('resize', onResize);

		return {
			update(newParams) {
				params = newParams;
				anchorEl = getAnchor();
				position();
			},
			destroy() {
				window.removeEventListener('scroll', onScroll, true);
				window.removeEventListener('resize', onResize);
				// Do not manually remove node; Svelte will detach it.
			}
		};
	}

	// show the correct tags in the dropdown
	$effect(() => {
		if ($tags.length === 0) {
			filteredTags = [];
			return;
		}

		// exclude already selected tags
		let tagsWithoutSelected = $tags.filter(
			(tag) => !selectedTags.find((selectedTag) => selectedTag === tag.id)
		);

		if (searchTab === '') {
			filteredTags = tagsWithoutSelected;
		} else {
			// remove trailing # if present
			let searchString = searchTab;
			if (searchString.startsWith('#')) {
				searchString = searchString.slice(1);
			}

			// filter tags for searchstring
			filteredTags = tagsWithoutSelected.filter((tag) =>
				tag.name.toLowerCase().includes(searchString.toLowerCase())
			);
		}

		selectedTagIndex = 0;
	});

	let selectedTagIndex = $state(0);
	// Handle Keyboard Navigation in Tag Dropdown
	function handleKeyDown(event) {
		if (!showTagDropdown || filteredTags.length === 0) return;

		switch (event.key) {
			case 'ArrowDown':
				event.preventDefault(); // Prevent cursor movement
				selectedTagIndex = Math.min(selectedTagIndex + 1, filteredTags.length - 1);
				ensureSelectedVisible();
				break;

			case 'ArrowUp':
				event.preventDefault(); // Prevent cursor movement
				selectedTagIndex = Math.max(selectedTagIndex - 1, 0);
				ensureSelectedVisible();
				break;

			case 'Enter':
				if (selectedTagIndex >= 0 && selectedTagIndex < filteredTags.length) {
					event.preventDefault();
					selectTag(filteredTags[selectedTagIndex].id);
				}
				document.activeElement.blur();
				break;

			case 'Escape':
				showTagDropdown = false;
				break;
		}
	}

	function ensureSelectedVisible() {
		setTimeout(() => {
			const dropdown = document.getElementById('tagDropdown');
			const selectedElement = dropdown?.querySelector('.tag-item.selected');

			if (dropdown && selectedElement) {
				const dropdownRect = dropdown.getBoundingClientRect();
				const selectedRect = selectedElement.getBoundingClientRect();

				if (selectedRect.top < dropdownRect.top) {
					dropdown.scrollTop -= dropdownRect.top - selectedRect.top;
				} else if (selectedRect.bottom > dropdownRect.bottom) {
					dropdown.scrollTop += selectedRect.bottom - dropdownRect.bottom;
				}
			}
		}, 40);
	}

	let showTagLoading = $state(false);

	function selectTag(id) {
		showTagLoading = true;

		axios
			.post(API_URL + '/logs/addTagToLog', {
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year,
				tag_id: id
			})
			.then((response) => {
				if (response.data.success) {
					selectedTags = [...selectedTags, id];
				} else {
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorAddingTagToDay'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorAddingTagToDay'));
				toast.show();
			})
			.finally(() => {
				showTagLoading = false;
			});

		searchTab = '';
	}

	function removeTag(id) {
		showTagLoading = true;

		axios
			.post(API_URL + '/logs/removeTagFromLog', {
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year,
				tag_id: id
			})
			.then((response) => {
				if (response.data.success) {
					selectedTags = selectedTags.filter((tag) => tag !== id);
				} else {
					// toast
					const toast = new bootstrap.Toast(
						document.getElementById('toastErrorRemovingTagFromDay')
					);
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorRemovingTagFromDay'));
				toast.show();
			})
			.finally(() => {
				showTagLoading = false;
			});
	}

	let newTag = $state({});
	let tagModal;

	function openTagModal() {
		newTag = {
			icon: '',
			name: '',
			color: '#f57c00'
		};

		tagModal.open();
	}

	let isSavingNewTag = $state(false);
	function saveNewTag() {
		isSavingNewTag = true;
		axios
			.post(API_URL + '/logs/saveNewTag', {
				icon: newTag.icon,
				name: newTag.name,
				color: newTag.color
			})
			.then((response) => {
				if (response.data.success) {
					loadTags();
					tagModal.close();
				} else {
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorSavingNewTag'));
					toast.show();
				}
			})
			.finally(() => {
				// close modal
				isSavingNewTag = false;
			});
	}

	let history = $state([]);
	let historySelected = $state(0);
	function getHistory() {
		axios
			.get(API_URL + '/logs/getHistory', {
				params: {
					day: $selectedDate.day,
					month: $selectedDate.month,
					year: $selectedDate.year
				}
			})
			.then((response) => {
				if (response.data.length === 0) {
					// no history
					return;
				}

				history = response.data.map((log) => {
					return {
						text: log.text,
						date_written: log.date_written
					};
				});
				historySelected = history.length - 1;

				// show history in a modal or something
				const modal = new bootstrap.Modal(document.getElementById('modalHistory'));
				modal.show();
			})
			.catch((error) => {
				console.error(error);
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingLog'));
				toast.show();
			});
	}

	function selectHistory() {
		if (historySelected < 0 || historySelected >= history.length) return;

		currentLog = history[historySelected].text;
		//logDateWritten = history[historySelected].date_written;

		tinyMDE.setContent(currentLog);
		tinyMDE.setSelection({ row: 0, col: 0 });
	}

	function showDeleteDayModal() {
		const modal = new bootstrap.Modal(document.getElementById('modalDeleteDay'));
		modal.show();
	}

	function deleteDay() {
		axios
			.get(API_URL + '/logs/deleteDay', {
				params: {
					day: $selectedDate.day,
					month: $selectedDate.month,
					year: $selectedDate.year
				}
			})
			.then((response) => {
				if (response.data.success) {
					currentLog = '';
					tinyMDE.setContent(currentLog);
					savedLog = '';
					logDateWritten = '';

					selectedTags = [];
					history = [];
					filesOfDay = [];
					images = [];
					$cal.daysBookmarked = $cal.daysBookmarked.filter((day) => day !== $selectedDate.day);
					$cal.daysWithFiles = $cal.daysWithFiles.filter((day) => day !== $selectedDate.day);
					$cal.daysWithLogs = $cal.daysWithLogs.filter((day) => day !== $selectedDate.day);
				} else {
					const toast = new bootstrap.Toast(document.getElementById('toastErrorDeletingDay'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				const toast = new bootstrap.Toast(document.getElementById('toastErrorDeletingDay'));
				toast.show();
			});
	}

	function renameFile(uuid_filename, new_filename) {
		// Validate filename
		if (!new_filename || new_filename.trim() === '') {
			const toast = new bootstrap.Toast(document.getElementById('toastErrorRenamingFile'));
			toast.show();
			return;
		}

		new_filename = new_filename.trim();

		axios
			.post(API_URL + '/logs/renameFile', {
				uuid: uuid_filename,
				new_filename: new_filename,
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year
			})
			.then((response) => {
				if (response.data.success) {
					// Update local file list
					filesOfDay = filesOfDay.map((file) => {
						if (file.uuid_filename === uuid_filename) {
							file.filename = new_filename;
						}
						return file;
					});

					// Update images list as well
					images = images.map((image) => {
						if (image.uuid_filename === uuid_filename) {
							image.filename = new_filename;
						}
						return image;
					});
				} else {
					const toast = new bootstrap.Toast(document.getElementById('toastErrorRenamingFile'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				const toast = new bootstrap.Toast(document.getElementById('toastErrorRenamingFile'));
				toast.show();
			});
	}

	function reorderFiles(newFileOrder) {
		// Create mapping of UUID to order
		const fileOrderMap = {};
		newFileOrder.forEach((file, index) => {
			fileOrderMap[file.uuid_filename] = index;
		});

		// Send to backend
		axios
			.post(API_URL + '/logs/reorderFiles', {
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year,
				file_order: fileOrderMap
			})
			.then((response) => {
				if (response.data.success) {
					// Update local state
					filesOfDay = newFileOrder;

					// Update images array - preserve existing properties like src, loading, etc.
					const newImagesOrder = [];
					newFileOrder.forEach((file) => {
						const existingImage = images.find((img) => img.uuid_filename === file.uuid_filename);
						if (existingImage) {
							// Preserve existing image properties (src, loading, etc.) and update with new file data
							newImagesOrder.push({
								...existingImage, // Keep existing image properties
								...file // Update with new file data (filename, etc.)
							});
						}
					});
					images = newImagesOrder;
				} else {
					const toast = new bootstrap.Toast(document.getElementById('toastErrorReorderingFiles'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				const toast = new bootstrap.Toast(document.getElementById('toastErrorReorderingFiles'));
				toast.show();
			});
	}
</script>

<DatepickerLogic />
<svelte:window
	onkeydown={on_key_down}
	onkeyup={on_key_up}
	ondragenter={handleDragEnter}
	ondragleave={handleDragLeave}
	ondragover={handleDragOver}
	ondrop={handleDrop}
/>

<!-- Drag and Drop Overlay -->
{#if isDragOver}
	<div class="drag-drop-overlay" transition:fade={{ duration: 150 }}>
		<div class="drag-drop-content">
			<Fa icon={faCloudArrowUp} size="3x" class="mb-3" />
			<h3>{$t('files.drop.title')}</h3>

			<div class="dragged-files-preview">
				<p class="files-count">
					🎯 {$t('files.drop.ready_to_upload', { count: draggedFileCount })}
				</p>
				<div class="file-drop-info">
					<p class="drop-instruction">{$t('files.drop.release_to_upload')}</p>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- shown on small Screen, when triggered -->
<div class="offcanvas offcanvas-start p-3" id="sidenav" tabindex="-1">
	<div class="offcanvas-header">
		<button
			type="button"
			class="btn-close"
			data-bs-dismiss="offcanvas"
			data-bs-target="#sidenav"
			aria-label="Close"
		></button>
	</div>
	<Sidenav />
</div>

<div class="d-flex flex-row justify-content-between h-100 main-row">
	<!-- shown on large Screen -->
	{#if $alwaysShowSidenav}
		<div class="sidenav p-3">
			<Sidenav />
		</div>
	{/if}

	<div class="d-flex flex-row middle-right flex-grow-1">
		<!-- Center -->
		<div class="d-flex flex-column pt-4 px-4 flex-grow-1" id="middle">
			<!-- Input-Area -->
			<div class="d-flex flex-row textAreaHeader glass">
				<div class="flex-fill textAreaDate">
					{new Date(
						Date.UTC($selectedDate.year, $selectedDate.month - 1, $selectedDate.day)
					).toLocaleDateString('locale', { weekday: 'long', timeZone: 'UTC' })}<br />
					{new Date(
						Date.UTC($selectedDate.year, $selectedDate.month - 1, $selectedDate.day)
					).toLocaleDateString('locale', {
						day: '2-digit',
						month: '2-digit',
						year: 'numeric',
						timeZone: 'UTC'
					})}
				</div>
				<div class="flex-fill textAreaWrittenAt">
					<div class={logDateWritten ? '' : 'opacity-50'}>{$t('log.written_on')}</div>
					{logDateWritten}
				</div>
				{#if historyAvailable}
					<div class="textAreaHistory d-flex flex-column justify-content-center">
						<button class="btn px-0 btn-hover" onclick={() => getHistory()}>
							<Fa icon={faClockRotateLeft} class="" size="1.5x" fw />
						</button>
					</div>
				{/if}
				<div class="textAreaDelete d-flex flex-column justify-content-center">
					<button class="btn px-0 btn-hover" onclick={() => showDeleteDayModal()}>
						<Fa icon={faTrash} class="" size="1.5x" fw />
					</button>
				</div>
			</div>
			<div id="log" class="focus-ring">
				<div id="toolbar"></div>
				<div id="editor"></div>
			</div>
			{#if images.length > 0}
				{#if !autoLoadImages && images.find((image) => !image.src && !image.loading)}
					<div class="d-flex flex-row">
						<button type="button" class="loadImageBtn" onclick={() => loadImages()}>
							<Fa icon={faCloudArrowDown} class="me-2" size="2x" fw /><br />
							{$t('log.load_images', { amount: images.length })}
							({formatBytes(
								images.filter((i) => !i.src).reduce((sum, image) => sum + (image.size || 0), 0)
							)})
						</button>
					</div>
				{:else}
					<ImageViewer {images} />
				{/if}
			{/if}

			{#if $settings.useALookBack && aLookBack.length > 0}
				<div class="mt-3 d-flex gap-2">
					{#each aLookBack as log}
						<ALookBack {log} />
					{/each}
				</div>
			{/if}
		</div>

		<div id="right" class="d-flex flex-column">
			<div class="tags glass">
				<div class="d-flex flex-row justify-content-between">
					<div class="d-flex flex-row">
						<h3>{$t('tags.tags')}</h3>
						{#if showTagLoading}
							<div class="spinner-border ms-3" role="status">
								<span class="visually-hidden">Loading...</span>
							</div>
						{/if}
					</div>
					<!-- svelte-ignore a11y_missing_attribute -->
					<a
						tabindex="-1"
						type="button"
						class="btn"
						data-bs-toggle="popover"
						data-bs-title={$t('tags.tags')}
						data-bs-content={$t('tags.description')}
					>
						<Fa icon={faQuestionCircle} fw /></a
					>
				</div>
				<div class="tagRow d-flex flex-row">
					<input
						bind:value={searchTab}
						onfocus={() => {
							showTagDropdown = true;
							selectedTagIndex = 0;
						}}
						onfocusout={() => {
							setTimeout(() => (showTagDropdown = false), 150);
						}}
						onkeydown={handleKeyDown}
						type="text"
						class="form-control"
						id="tag-input"
						placeholder={$t('tags.input')}
					/>
					<button class="newTagBtn btn btn-outline-secondary ms-2" onclick={openTagModal}>
						<Fa icon={faSquarePlus} fw />
						{$t('tags.new_tag')}
					</button>
				</div>
				{#if showTagDropdown}
					<div id="tagDropdown" use:portalDropdown>
						{#if filteredTags.length === 0}
							<em style="padding: 0.2rem;">{$t('tags.no_tags_found')}</em>
						{:else}
							{#each filteredTags as tag, index (tag.id)}
								<!-- svelte-ignore a11y_click_events_have_key_events -->
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<!-- svelte-ignore a11y_mouse_events_have_key_events -->
								<div
									role="button"
									tabindex="0"
									onclick={() => selectTag(tag.id)}
									onmouseover={() => (selectedTagIndex = index)}
									class="tag-item {index === selectedTagIndex ? 'selected' : ''}"
								>
									<Tag {tag} />
								</div>
							{/each}
						{/if}
					</div>
				{/if}
				<div class="selectedTags d-flex flex-row flex-wrap">
					{#if $tags.length !== 0}
						{#each selectedTags as tag_id (tag_id)}
							<div transition:slide={{ axis: 'x' }}>
								<Tag tag={$tags.find((tag) => tag.id === tag_id)} {removeTag} isRemovable="true" />
							</div>
						{/each}
					{/if}
				</div>
			</div>

			<div class="files d-flex flex-column glass">
				<button
					class="btn btn-secondary upload-btn {filesOfDay?.length > 0 ? 'mb-2' : ''}"
					id="uploadBtn"
					onclick={triggerFileInput}
					ondragenter={(e) => {
						e.preventDefault();
						e.currentTarget.classList.add('drag-hover');
					}}
					ondragleave={(e) => {
						e.preventDefault();
						e.currentTarget.classList.remove('drag-hover');
					}}
					ondragover={(e) => {
						e.preventDefault();
						e.dataTransfer.dropEffect = 'copy';
					}}
					ondrop={(e) => {
						e.preventDefault();
						e.currentTarget.classList.remove('drag-hover');
						const files = e.dataTransfer.files;
						if (files && files.length > 0) {
							for (let i = 0; i < files.length; i++) {
								uploadFile(files[i]);
							}
						}
					}}><Fa icon={faCloudArrowUp} class="me-2" id="uploadIcon" />{$t('files.upload')}</button
				>
				<input type="file" id="fileInput" multiple style="display: none;" onchange={onFileChange} />

				<FileList
					files={filesOfDay}
					{downloadFile}
					{askDeleteFile}
					{renameFile}
					{reorderFiles}
					editable
				/>
				{#each uploadingFiles as file}
					<div>
						{file.name}
						<div
							class="progress"
							role="progressbar"
							aria-label="Upload progress"
							aria-valuemin="0"
							aria-valuemax="100"
						>
							<div
								class="progress-bar {file.progress === 100
									? 'progress-bar-striped progress-bar-animated'
									: ''}"
								style:width={file.progress + '%'}
							>
								{#if file.progress !== 100}
									{file.progress}%
								{:else}
									{$t('files.encrypting')}
								{/if}
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>

	<div class="toast-container position-fixed bottom-0 end-0 p-3">
		<div
			id="toastErrorRemovingTagFromDay"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('tags.toast.error_removing')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorAddingTagToDay"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('tags.toast.error_adding')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorSavingNewTag"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('tags.toast.error_saving')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorSavingLog"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('log.toast.error_saving')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorLoadingLog"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('log.toast.error_loading')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorSavingFile"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('files.toast.error_saving')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorDeletingFile"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('files.toast.error_deleting')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorLoadingFile"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('files.toast.error_loading')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorDeletingDay"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('log.toast.error_deleting_day')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorRenamingFile"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">{$t('log.toast.error_renaming_file')}</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorReorderingFiles"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">{$t('log.toast.error_reordering_files')}</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>
	</div>

	<div class="modal fade" id="modalConfirmDeleteFile" tabindex="-1">
		<div class="modal-dialog modal-dialog-centered">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">{$t('modal.deleteFile.title')}</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"
					></button>
				</div>
				<div class="modal-body">
					<p>
						{@html $t('modal.deleteFile.body', { file: confirmDelete.filename })}
					</p>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal"
						>{$t('modal.close')}</button
					>
					<button
						onclick={() => deleteFile(confirmDelete.uuid)}
						type="button"
						class="btn btn-primary"
						data-bs-dismiss="modal">{$t('modal.deleteFile.delete')}</button
					>
				</div>
			</div>
		</div>
	</div>

	<div class="modal fade" id="modalHistory" tabindex="-1">
		<div class="modal-dialog modal-lg modal-fullscreen-lg-down modal-dialog-centered">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">{$t('modal.history.title')}</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"
					></button>
				</div>
				<div class="modal-body">
					<div class="d-flex flex-row justify-content-center">
						<button
							disabled={historySelected <= 0}
							class="btn btn-outline-secondary history-btn"
							onclick={() => {
								if (historySelected > 0) historySelected--;
							}}
						>
							<Fa icon={faArrowLeft} class="me-2" fw />
							{$t('modal.history.older')}
						</button>
						<select
							bind:value={historySelected}
							class="form-select mx-2"
							aria-label="Default select example"
						>
							{#each history as entry, index (index)}
								<option value={index}>{entry.date_written}</option>
							{/each}
						</select>
						<button
							disabled={historySelected >= history.length - 1}
							class="btn btn-outline-secondary history-btn"
							onclick={() => {
								if (historySelected < history.length - 1) historySelected++;
							}}
						>
							<Fa icon={faArrowRight} class="me-2" fw />
							{$t('modal.history.newer')}
						</button>
					</div>
					<div class="text mt-2">
						{@html marked.parse(history[historySelected]?.text || 'Error!')}
					</div>
				</div>
				<div class="modal-footer">
					<div class="d-flex flex-column">
						<div class="form-text">
							{@html $t('modal.history.description')}
						</div>
						<div class="d-flex flex-row justify-content-end mt-2">
							<button type="button" class="btn btn-secondary me-2" data-bs-dismiss="modal"
								>{$t('modal.close')}</button
							>
							<button
								onclick={() => selectHistory()}
								type="button"
								class="btn btn-primary"
								data-bs-dismiss="modal">{$t('modal.save')}</button
							>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div class="modal fade" id="modalDeleteDay" tabindex="-1">
		<div class="modal-dialog modal-dialog-centered">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">{$t('modal.deleteDay.title')}</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"
					></button>
				</div>
				<div class="modal-body">
					{@html $t('modal.deleteDay.description', {
						day: $selectedDate.day,
						month: $selectedDate.month,
						year: $selectedDate.year
					})}
					<br /><br />
					{$t('modal.deleteDay.thisIncludes')}
					<ul>
						{#snippet deleteDayBool(available, description)}
							<li class={available ? 'text-decoration-underline' : 'text-muted fst-italic'}>
								{#if available}✔️{:else}❌{/if}
								{description}
							</li>
						{/snippet}

						{#snippet deleteDayCount(item, description)}
							<li class={item.length > 0 ? 'text-decoration-underline' : 'text-muted fst-italic'}>
								{description}
							</li>
						{/snippet}

						{@render deleteDayBool(logDateWritten !== '', $t('modal.deleteDay.logEntry'))}
						{@render deleteDayBool(historyAvailable, $t('modal.deleteDay.history'))}

						{@render deleteDayCount(
							filesOfDay,
							$t('modal.deleteDay.files', { files: filesOfDay.length })
						)}
						{@render deleteDayCount(
							selectedTags,
							$t('modal.deleteDay.tags', { tags: selectedTags.length })
						)}
						{@render deleteDayBool(
							$cal.daysBookmarked.includes($selectedDate.day),
							$t('modal.deleteDay.bookmark')
						)}
					</ul>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal"
						>{$t('modal.deleteDay.button_close')}</button
					>
					<button
						onclick={() => deleteDay()}
						type="button"
						class="btn btn-danger"
						data-bs-dismiss="modal">{$t('modal.deleteDay.button_delete')}</button
					>
				</div>
			</div>
		</div>
	</div>

	<TagModal
		bind:this={tagModal}
		bind:editTag={newTag}
		createTag="true"
		isSaving={isSavingNewTag}
		{saveNewTag}
	/>
</div>

<style>
	#modalHistory > div > div {
		height: 80vh;
	}

	@media (max-width: 991px) {
		#modalHistory > div > div {
			height: 100vh;
		}
	}

	#modalHistory > div > div > .modal-body {
		height: 50%;
		display: flex;
		flex-direction: column;
	}

	#modalHistory > div > div > .modal-body > .text {
		flex: 1 1 auto;
		overflow-y: auto;
	}

	.text {
		border: 1px solid #ccc;
		border-radius: 15px;
		padding: 1rem;
		word-wrap: anywhere;
		white-space: break-spaces;
	}

	.history-btn {
		white-space: nowrap;
	}

	.btn-hover:hover {
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
		border: 1px solid #adadad77;
	}

	@media (max-width: 1150px) {
		.middle-right {
			flex-direction: column !important;
			align-items: center;
			justify-content: start !important;
		}

		#middle {
			flex: none !important;
		}

		#right {
			padding-right: 0 !important;
		}
	}

	@media (max-width: 500px) {
		#right {
			max-width: 100% !important;
			padding-left: 1rem !important;
			padding-right: 1rem !important;
		}
	}

	@media (min-width: 1400px) {
		#right {
			width: 500px !important;
		}
	}

	.main-row {
		max-width: 100vw;
	}

	.middle-right {
		justify-content: center;
		width: 100%;
	}

	#middle {
		width: 100%;
	}

	.tagRow {
		width: 100%;
	}

	.newTagBtn {
		white-space: nowrap;
	}

	.selectedTags {
		margin-top: 0.5rem;
		gap: 0.5rem;
	}

	#tag-input {
		width: inherit !important;
	}

	#tagDropdown {
		position: absolute;
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
		z-index: 1000;
		max-height: 200px;
		overflow-y: scroll;
		overflow-x: hidden;
		display: flex;
		flex-direction: column;
		backdrop-filter: blur(10px) saturate(150%);
		border-radius: 10px;
	}

	:global(body[data-bs-theme='dark']) #tagDropdown {
		background-color: rgba(87, 87, 87, 0.5);
	}

	:global(body[data-bs-theme='light']) #tagDropdown {
		background-color: rgba(196, 196, 196, 0.5);
	}

	.tag-item {
		cursor: pointer;
		padding: 5px;
	}

	:global(body[data-bs-theme='dark']) .tag-item.selected {
		background-color: #5f5f5f;
	}

	:global(body[data-bs-theme='light']) .tag-item.selected {
		background-color: #b9b9b9;
	}

	.tags {
		z-index: 10;
		padding: 0.5rem;
		margin-bottom: 2rem;
		border-radius: 10px;
	}

	.loadImageBtn {
		padding: 0.5rem 1rem;
		border: none;
		margin-top: 0.5rem;
		border-radius: 5px;
		transition: all ease 0.2s;
	}

	.modal-header {
		border-bottom: none;
	}

	.modal-footer {
		border-top: none;
	}

	.files {
		margin-bottom: 1rem;
		border-radius: 10px;
		padding: 1rem;
	}

	:global(#uploadIcon) {
		transition: all ease 0.3s;
	}

	:global(#uploadBtn:hover > #uploadIcon) {
		transform: scale(1.4);
	}

	:global(.TMCommandBar) {
		border-top: none;
		border-bottom: none;
		height: auto;
		flex-wrap: wrap;
		padding-top: 2px;
		padding-bottom: 3px;
	}

	:global(body[data-bs-theme='dark'] .TMCommandBar) {
		border-left: 1px solid #6a6a6a;
		border-right: 1px solid #6a6a6a;
	}

	:global(body[data-bs-theme='light'] .TMCommandBar) {
		border-left: 1px solid #cccccc;
		border-right: 1px solid #cccccc;
	}

	:global(body[data-bs-theme='dark'] .TMCommandBar) {
		background-color: rgba(70, 70, 70, 0.5);
	}

	:global(body[data-bs-theme='light'] .TMCommandBar) {
		background-color: rgba(202, 202, 202, 0.5);
	}

	:global(body[data-bs-theme='dark'] .TMCommandButton_Inactive) {
		background-color: transparent;
		fill: #f0f0f0;
	}

	:global(body[data-bs-theme='light'] .TMCommandButton_Inactive) {
		background-color: transparent;
		fill: #161616;
	}

	:global(body[data-bs-theme='dark'] .TMCommandButton_Inactive:hover) {
		background-color: rgba(180, 180, 180, 0.438);
	}

	:global(body[data-bs-theme='light'] .TMCommandButton_Inactive:hover) {
		background-color: rgba(180, 180, 180, 0.438);
	}

	:global(.TMCommandButton) {
		border-radius: 3px;
	}

	:global(body[data-bs-theme='dark'] .TinyMDE) {
		backdrop-filter: blur(8px) saturate(130%);
		background-color: rgba(50, 50, 50, 0.8);
		color: #f0f0f0;
	}

	:global(body[data-bs-theme='light'] .TinyMDE) {
		backdrop-filter: blur(8px) saturate(130%);
		background-color: rgba(255, 255, 255, 0.7);
		color: #1f1f1f;
	}

	#editor {
		height: 400px;
		word-break: break-word;
	}

	:global(.TinyMDE) {
		border: 1px solid lightgreen;

		border-bottom-left-radius: 5px;
		border-bottom-right-radius: 5px;
		overflow-y: auto;

		transition: all ease 0.2s;
	}

	:global(.TinyMDE:focus:not(.notSaved)) {
		box-shadow: 0 0 0 0.25rem #90ee9070;
	}

	:global(.TinyMDE:focus.notSaved) {
		box-shadow: 0 0 0 0.25rem #f57c0030;
	}

	:global(.TinyMDE.notSaved) {
		border-color: #f57c00;
	}

	.sidenav {
		width: 380px;
		min-width: 380px;
	}

	.textAreaHeader {
		border-left: 1px solid #6a6a6a;
		border-top: 1px solid #6a6a6a;
		border-right: 1px solid #6a6a6a;
		border-top-left-radius: 5px;
		border-top-right-radius: 5px;
	}

	.textAreaDate,
	.textAreaWrittenAt,
	.textAreaHistory {
		border-right: 1px solid #6a6a6a;
		padding: 0.25em;
	}

	.textAreaDelete {
		padding: 0.25em;
	}

	#log div:focus:not(.notSaved) {
		border-color: #90ee90;
		box-shadow: 0 0 0 0.25rem #90ee9070;
	}

	.textAreaDate {
		font-weight: 600;
	}

	#right {
		margin-top: 1.5rem !important;
		/* min-width: 300px;
		max-width: 400px; */
		width: 400px;
		padding-right: 2rem;
	}

	/* Drag and Drop Styles */
	.drag-drop-overlay {
		position: fixed;
		top: 0;
		left: 0;
		width: 100vw;
		height: 100vh;
		background-color: rgba(0, 0, 0, 0.7);
		backdrop-filter: blur(4px);
		z-index: 9999;
		display: flex;
		align-items: center;
		justify-content: center;
		pointer-events: none;
	}

	.drag-drop-content {
		text-align: center;
		color: white;
		background-color: rgba(255, 255, 255, 0.1);
		border: 2px dashed rgba(255, 255, 255, 0.5);
		border-radius: 20px;
		padding: 3rem;
		max-width: 600px;
		max-height: 80vh;
		overflow-y: auto;
	}

	.drag-drop-content h3 {
		margin-bottom: 1rem;
		font-size: 2rem;
	}

	.drag-drop-content p {
		font-size: 1.2rem;
		opacity: 0.9;
	}

	.dragged-files-preview {
		margin-top: 1.5rem;
		text-align: center;
	}

	.files-count {
		font-size: 1.3rem;
		margin-bottom: 1rem;
		font-weight: 500;
		color: #90ee90;
	}

	.file-drop-info {
		background-color: rgba(0, 0, 0, 0.2);
		border-radius: 10px;
		padding: 1rem;
		margin-top: 1rem;
	}

	.drop-instruction {
		font-size: 1.1rem;
		margin: 0;
		opacity: 0.9;
	}

	:global(.upload-btn.drag-hover) {
		background-color: #198754 !important;
		border-color: #198754 !important;
		transform: scale(1.05);
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
	}

	.upload-btn {
		transition: all 0.2s ease;
		border: 2px dashed transparent;
	}
</style>

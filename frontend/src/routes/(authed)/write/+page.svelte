<script>
	import '../../../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Sidenav from '$lib/Sidenav.svelte';
	import { selectedDate, cal, readingDate } from '$lib/calendarStore.js';
	import axios from 'axios';
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
		faTrash,
		faBars
	} from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import { v4 as uuidv4 } from 'uuid';
	import { slide, fade } from 'svelte/transition';
	import { settings, autoLoadImagesThisDevice } from '$lib/settingsStore';
	import { tags, tagsLoaded } from '$lib/tagStore';
	import Tag from '$lib/Tag.svelte';
	import TagModal from '$lib/TagModal.svelte';
	import FileList from '$lib/FileList.svelte';
	import { formatBytes, alwaysShowSidenav, sameDate } from '$lib/helpers.js';
	import ImageViewer from '$lib/ImageViewer.svelte';
	import TemplateDropdown from '$lib/TemplateDropdown.svelte';
	import { insertTemplate } from '$lib/templateStore';
	import ALookBack from '$lib/ALookBack.svelte';
	import { marked } from 'marked';
	import { getTranslate, getTolgee } from '@tolgee/svelte';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	let cancelDownload = new AbortController();

	let tinyMDE;
	let isMobile = false;
	onMount(() => {
		// If we come from read mode, keep the last visible day as the active selected day
		if ($readingDate) {
			$selectedDate = $readingDate; // promote readingDate to selectedDate when switching to write mode
		}
		$readingDate = null; // no reading-highlighting when in write mode

		// Detect mobile (simple heuristic: viewport width OR user agent)
		isMobile =
			typeof window !== 'undefined' &&
			(window.matchMedia('(max-width: 768px)').matches ||
				/Mobi|Android/i.test(navigator.userAgent));

		tinyMDE = new TinyMDE.Editor({ element: 'editor', content: '' });
		new TinyMDE.CommandBar({ element: 'toolbar', editor: tinyMDE });
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
		[...popoverTriggerList].map(
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
	let macAltPressed = $state(false);
	let MacCtrlPressed = $state(false);
	const isMac = navigator.platform.toUpperCase().includes('MAC');
	function on_key_down(event) {
		if (!isMac && event.key === 'Alt') {
			event.preventDefault();
			altPressed = true;
		}
		if (isMac && event.key === 'Alt') {
			event.preventDefault();
			macAltPressed = true;
		}
		if (isMac && event.key === 'Control') {
			event.preventDefault();
			MacCtrlPressed = true;
		}
		if (event.key === 'ArrowRight' && altPressed) {
			event.preventDefault();
			changeDay(+1);
		} else if (event.key === 'ArrowLeft' && altPressed) {
			event.preventDefault();
			changeDay(-1);
		}
		if (!isMac && event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = true;
		}
		if (event.key === 'g' && (ctrlPressed || MacCtrlPressed)) {
			event.preventDefault();
			document.getElementById('tag-input').focus();
		}
	}

	$effect(() => {
		if (isMac) {
			if (macAltPressed && MacCtrlPressed) {
				altPressed = true;
			} else {
				altPressed = false;
			}
		}
	});

	function on_key_up(event) {
		if (!isMac && event.key === 'Alt') {
			event.preventDefault();
			altPressed = false;
		}
		if (isMac && event.key === 'Alt') {
			event.preventDefault();
			macAltPressed = false;
		}
		if (isMac && event.key === 'Control') {
			event.preventDefault();
			MacCtrlPressed = false;
		}
		if (!isMac && event.key === 'Control') {
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

	// Swipe support (mobile): horizontal swipe on header changes day
	let touchStartX = 0;
	let touchStartY = 0;
	let touchStartTime = 0;
	const SWIPE_MIN_DISTANCE = 50; // px
	const SWIPE_MAX_OFF_AXIS = 70; // px vertical tolerance
	const SWIPE_MAX_DURATION = 800; // ms

	function onHeaderTouchStart(e) {
		if (e.touches.length !== 1) return;
		const t = e.touches[0];
		touchStartX = t.clientX;
		touchStartY = t.clientY;
		touchStartTime = Date.now();
	}

	function onHeaderTouchEnd(e) {
		const t = e.changedTouches && e.changedTouches[0];
		if (!t) return;
		const dx = t.clientX - touchStartX;
		const dy = t.clientY - touchStartY;
		const dt = Date.now() - touchStartTime;

		if (dt > SWIPE_MAX_DURATION) return;
		if (Math.abs(dx) < SWIPE_MIN_DISTANCE) return;
		if (Math.abs(dy) > SWIPE_MAX_OFF_AXIS) return;

		// valid horizontal swipe
		e.preventDefault();
		if (dx < 0) {
			changeDay(+1); // swipe left -> next day
		} else {
			changeDay(-1); // swipe right -> previous day
		}
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

			// Update editor content
			tinyMDE.setContent(currentLog);
			// Only auto-focus/select on non-mobile devices
			if (!isMobile) {
				try {
					tinyMDE.setSelection({ row: 0, col: 0 });
					// Ensure the underlying editable div gets focus
					const editorEl = document.querySelector(
						'#editor .TinyMDE textarea, #editor .TinyMDE [contenteditable="true"]'
					);
					if (editorEl) editorEl.focus();
				} catch {}
			}

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
	let initial_aLookBack = $state(false);
	$effect(() => {
		if (!initial_aLookBack && $settings && $settings.useALookBack) {
			getALookBack();
			initial_aLookBack = true;
		}
	});

	const imageExtensions = ['jpeg', 'jpg', 'gif', 'png', 'webp', 'bmp'];

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
		let date_written = new Date().toLocaleString($tolgee.getLanguage(), {
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
			.then(() => {
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
			.then(() => {
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

	// General touch device detection (iPad & others) for simplified touch-friendly tag selection
	let isTouchDevice = $state(false);
	let showTouchTagPanel = $state(false);
	onMount(() => {
		try {
			const ua = navigator.userAgent || '';
			const platform = navigator.platform || '';
			const maxTP = navigator.maxTouchPoints || 0;
			const coarse = window.matchMedia ? window.matchMedia('(pointer: coarse)').matches : false;
			const iPadLike = /iPad/.test(ua) || (/Mac/.test(platform) && maxTP > 1);
			isTouchDevice = maxTP > 0 || coarse || iPadLike || 'ontouchstart' in window;
		} catch (e) {
			isTouchDevice = false;
		}
	});

	let filteredTags = $state([]);
	let selectedTags = $state([]);

	// Action: portal dropdown to <body> and position it under the input (with iOS visualViewport handling)
	function portalDropdown(node, params) {
		let anchorEl;
		let frameRequested = false;
		let lastPlacement = 'bottom';
		const GAP = 4;

		function getAnchor() {
			if (params?.anchor) {
				return typeof params.anchor === 'string'
					? document.querySelector(params.anchor)
					: params.anchor;
			}
			return document.getElementById('tag-input');
		}

		function schedulePosition() {
			if (frameRequested) return;
			frameRequested = true;
			requestAnimationFrame(() => {
				frameRequested = false;
				_doPosition();
			});
		}

		function _doPosition() {
			if (!anchorEl || !node.isConnected) return;
			const rect = anchorEl.getBoundingClientRect();
			const vv = window.visualViewport;
			// Use fixed positioning + visual viewport offsets
			node.style.position = 'fixed';

			// Determine available space (visual viewport aware)
			const viewportHeight = vv ? vv.height : window.innerHeight;
			const viewportWidth = vv ? vv.width : window.innerWidth;
			const vOffsetTop = vv ? vv.offsetTop : 0; // when keyboard pushes visual viewport upward
			const vOffsetLeft = vv ? vv.offsetLeft : 0;
			// Safe-area bottom inset (iOS notch / home indicator) – cannot read env() directly here, fallback 0
			const safeBottomInset = 0;

			// First attempt: place below
			let top = rect.bottom + vOffsetTop + GAP;
			let left = rect.left + vOffsetLeft;

			// Temporarily set visibility hidden to measure height if not shown
			const prevVis = node.style.visibility;
			node.style.visibility = 'hidden';
			node.style.top = '0px';
			node.style.left = '0px';
			node.style.display = 'block';
			const menuH = node.offsetHeight || 180;
			const menuW = node.offsetWidth || 200;
			node.style.visibility = prevVis || '';

			const spaceBelow = viewportHeight + vOffsetTop - rect.bottom - GAP - safeBottomInset;
			const spaceAbove = rect.top - vOffsetTop - GAP;

			// Decide placement & dynamic max-height
			let desiredPlacement = 'bottom';
			// If below space is insufficient but above has more room, flip
			if (spaceBelow < Math.min(menuH, 220) && spaceAbove > spaceBelow) {
				desiredPlacement = 'top';
			}

			let available = desiredPlacement === 'bottom' ? spaceBelow : spaceAbove;
			// Cap maximal height to a reasonable viewport fraction
			const maxCap = Math.min(Math.max(available, 120), Math.floor(viewportHeight * 0.7));
			// Apply before final vertical positioning so scrollHeight reflects possible shrink
			node.style.maxHeight = maxCap + 'px';
			node.style.overflowY = 'auto';
			node.style.webkitOverflowScrolling = 'touch';

			if (desiredPlacement === 'top') {
				top = rect.top + vOffsetTop - GAP - maxCap;
				lastPlacement = 'top';
			} else {
				// keep below; if content smaller than available it will shrink naturally
				lastPlacement = 'bottom';
			}

			// Horizontal clamping
			if (left + menuW > viewportWidth + vOffsetLeft - 8) {
				left = viewportWidth + vOffsetLeft - menuW - 8;
			}
			left = Math.max(8 + vOffsetLeft, left);

			node.style.top = Math.round(top) + 'px';
			node.style.left = Math.round(left) + 'px';
			node.dataset.placement = lastPlacement;
		}

		function attach() {
			if (!node.isConnected) return;
			if (node.parentElement !== document.body) {
				document.body.appendChild(node);
			}
			// do an immediate position before RAF to avoid initial invisible state
			_doPosition();
			schedulePosition();
			// fallback in case rAF didn't fire yet or iOS delayed metrics
			setTimeout(() => {
				if (!node.dataset.placement) {
					_doPosition();
				}
			}, 48);
		}

		function onAny() {
			schedulePosition();
		}
		function onScrollCapture() {
			schedulePosition();
		}

		anchorEl = getAnchor();
		attach();

		// Global listeners
		window.addEventListener('scroll', onScrollCapture, true);
		window.addEventListener('resize', onAny);
		if (window.visualViewport) {
			window.visualViewport.addEventListener('resize', onAny);
			window.visualViewport.addEventListener('scroll', onAny);
		}

		return {
			update(newParams) {
				params = newParams;
				anchorEl = getAnchor();
				schedulePosition();
			},
			destroy() {
				window.removeEventListener('scroll', onScrollCapture, true);
				window.removeEventListener('resize', onAny);
				if (window.visualViewport) {
					window.visualViewport.removeEventListener('resize', onAny);
					window.visualViewport.removeEventListener('scroll', onAny);
				}
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

	function selectTag(id) {
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
				if (isTouchDevice) {
					showTouchTagPanel = false;
				}
			});

		searchTab = '';
	}

	function removeTag(id) {
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
			.catch((error) => {
				console.error(error.response);
				if (error.response.status == 400) {
					// name already exists
					// toast
					const toast = new bootstrap.Toast(
						document.getElementById('toastErrorSavingNewTagExists')
					);
					toast.show();
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
<div class="offcanvas offcanvas-start overflow-y-auto" id="sidenav" tabindex="-1">
	<div class="offcanvas-header sticky-top">
		<button
			type="button"
			class="btn-close btn-close-white"
			data-bs-dismiss="offcanvas"
			data-bs-target="#sidenav"
			aria-label="Close"
		></button>
	</div>
	<Sidenav />
</div>

<div class="d-flex flex-row justify-content-between main-row h-100">
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
			<div class="glass-shadow input-area">
				<div
					class="d-flex flex-row textAreaHeader glass"
					ontouchstart={onHeaderTouchStart}
					ontouchend={onHeaderTouchEnd}
				>
					<div class="flex-fill d-flex">
						<div class="w-50 textAreaDate">
							{new Date(
								Date.UTC($selectedDate.year, $selectedDate.month - 1, $selectedDate.day)
							).toLocaleDateString($tolgee.getLanguage(), { weekday: 'long', timeZone: 'UTC' })}<br
							/>
							{new Date(
								Date.UTC($selectedDate.year, $selectedDate.month - 1, $selectedDate.day)
							).toLocaleDateString($tolgee.getLanguage(), {
								day: '2-digit',
								month: '2-digit',
								year: 'numeric',
								timeZone: 'UTC'
							})}
						</div>
						<div class="w-50 textAreaWrittenAt">
							<div class={logDateWritten ? '' : 'opacity-50'}>{$t('log.written_on')}</div>
							{logDateWritten}
						</div>
					</div>
					<!-- Desktop buttons -->
					<div
						class="textAreaHistory header-btn-desktop d-flex flex-column justify-content-center {historyAvailable
							? ''
							: 'invisible'}"
					>
						<button class="btn px-0 btn-hover" onclick={() => getHistory()}>
							<Fa icon={faClockRotateLeft} size="1.5x" fw />
						</button>
					</div>
					<div class="textAreaDelete header-btn-desktop d-flex flex-column justify-content-center">
						<button class="btn px-0 btn-hover" onclick={() => showDeleteDayModal()}>
							<Fa icon={faTrash} size="1.5x" fw />
						</button>
					</div>
					<!-- Mobile dropdown -->
					<div
						class="dropdown header-actions-mobile d-flex flex-column justify-content-center ms-1"
					>
						<button
							class="btn px-2 btn-hover"
							type="button"
							data-bs-toggle="dropdown"
							aria-expanded="false"
						>
							<Fa icon={faBars} fw />
						</button>
						<ul class="dropdown-menu dropdown-menu-end">
							{#if historyAvailable}
								<li>
									<button type="button" class="dropdown-item" onclick={() => getHistory()}>
										<Fa icon={faClockRotateLeft} class="me-2" />{$t('log.dropdown.history')}
									</button>
								</li>
							{/if}
							<li>
								<button
									type="button"
									class="dropdown-item text-danger"
									onclick={() => showDeleteDayModal()}
								>
									<Fa icon={faTrash} class="me-2" />{$t('log.dropdown.deleteDay')}
								</button>
							</li>
						</ul>
					</div>
				</div>
				<div id="log" class="focus-ring">
					<div id="toolbar"></div>
					<div id="editor"></div>
				</div>
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
				<div class="mt-3 d-flex gap-3 overflow-x-auto mb-2 a-look-back">
					{#each aLookBack as log}
						<ALookBack {log} />
					{/each}
				</div>
			{/if}
		</div>

		<div id="right" class="d-flex flex-column">
			<div class="tags glass glass-shadow">
				<div class="d-flex flex-row justify-content-between">
					<div class="d-flex flex-row">
						<h3>{$t('tags.tags')}</h3>
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
					{#if isTouchDevice}
						<button
							id="tag-input"
							type="button"
							class="btn btn-outline-secondary flex-grow-1 text-start"
							onclick={() => (showTouchTagPanel = !showTouchTagPanel)}
						>
							{#if showTouchTagPanel}
								{$t('tags.hide_selector')}
							{:else}
								{$t('tags.input')}
							{/if}
						</button>
					{:else}
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
					{/if}
					<button class="newTagBtn btn btn-outline-secondary ms-2" onclick={openTagModal}>
						<Fa icon={faSquarePlus} fw />
						{$t('tags.new_tag')}
					</button>
				</div>
				{#if !isTouchDevice && showTagDropdown}
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
				{#if isTouchDevice && showTouchTagPanel}
					<div transition:slide>
						<div class="pt-2">
							<div class="touch-tag-panel">
								{#if $tags.length - selectedTags.length === 0}
									<em style="padding:0.2rem;">{$t('tags.no_tags_found')}</em>
								{:else}
									<div class="d-flex flex-row flex-wrap gap-1 selectTagTouchDevice">
										{#each $tags.filter((t) => !selectedTags.includes(t.id)) as tag (tag.id)}
											<button
												type="button"
												class="touch-tag-item btn btn-sm btn-outline-none"
												onclick={() => selectTag(tag.id)}
											>
												<Tag {tag} />
											</button>
										{/each}
									</div>
								{/if}
							</div>
						</div>
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

			<div class="files d-flex flex-column glass glass-shadow">
				<button
					class="btn btn-secondary upload-btn {filesOfDay?.length > 0 ? '' : ''}"
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
				{#if filesOfDay?.length > 0}
					<div transition:slide><div class="pt-3"></div></div>
				{/if}

				<div class="fileScroll">
					<FileList
						files={filesOfDay}
						{downloadFile}
						{askDeleteFile}
						{renameFile}
						{reorderFiles}
						editable
					/>
				</div>

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
			id="toastErrorSavingNewTagExists"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('tags.toast.error_saving_exists')}
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
						{@html marked.parse(history[historySelected]?.text || '')}
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
	.input-area {
		border-radius: 10px;
	}

	@media screen and (max-width: 567px) {
		:global(em-emoji-picker) {
			right: 0;
		}
	}

	:global(body[data-bs-theme='light']) .text-muted.fst-italic {
		color: rgba(99, 98, 98, 0.637) !important;
	}

	:global(body[data-bs-theme='dark']) .text-muted.fst-italic {
		color: rgba(228, 226, 230, 0.4) !important;
	}

	:global(body[data-bs-theme='light'] .loadImageBtn, body[data-bs-theme='light'] .fileBtn) {
		color: #000000;
	}

	#sidenav {
		padding-left: 1rem;
		padding-right: 1rem;
		padding-bottom: 1rem;
	}

	.offcanvas-header > button {
		background-color: #ccc;
		opacity: 0.8;
	}

	.selectTagTouchDevice {
		background-color: #9e9e9e65;
		padding: 0.5rem;
		border-radius: 10px;
	}

	.a-look-back {
		height: 110px;
		min-height: 110px;
	}

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
	}

	.history-btn {
		white-space: nowrap;
	}

	.btn-hover:hover {
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
		border: 1px solid #adadad77;
	}

	@media (max-width: 1200px) {
		.middle-right {
			flex-direction: column !important;
			align-items: center;
			justify-content: start !important;
		}

		#middle {
			flex: none !important;
		}

		#right {
			flex: 1 1 auto !important;
			width: 100% !important;
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

	@media (min-width: 1600px) {
		.sidenav {
			padding-left: 2rem !important;
			min-width: 430px !important;
		}

		#middle {
			padding-left: 3rem !important;
			padding-right: 3rem !important;
		}

		#right {
			min-width: 400px !important;
		}
	}

	.middle-right {
		justify-content: center;
		min-width: 0;
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
		border-bottom-left-radius: 10px;
		border-bottom-right-radius: 10px;
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
		min-height: 0;
	}

	.fileScroll {
		overflow-y: auto;
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
		background-color: rgba(50, 50, 50, 0.8);
		color: #f0f0f0;
	}

	:global(body[data-bs-theme='light'] .TinyMDE) {
		background-color: rgba(240, 240, 240, 0.6);
		color: #1f1f1f;
	}

	#editor {
		height: 400px;
		word-break: break-word;
	}

	@media screen and (max-height: 800px) {
		#editor {
			height: 350px;
		}
	}

	@media screen and (max-width: 600px) {
		#editor {
			height: 300px;
		}
	}

	:global(.TinyMDE) {
		border: 1px solid lightgreen;

		border-bottom-left-radius: 10px;
		border-bottom-right-radius: 10px;
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
		border-top-left-radius: 10px;
		border-top-right-radius: 10px;
	}

	.header-actions-mobile {
		display: none;
	}
	.header-btn-desktop {
		display: flex;
	}

	@media (max-width: 550px) {
		.header-actions-mobile {
			display: flex;
		}
		.header-btn-desktop {
			display: none !important;
		}
	}
	@media (min-width: 551px) {
		.header-actions-mobile {
			display: none !important;
		}
		.header-btn-desktop {
			display: flex !important;
		}
	}

	.textAreaDate {
		border-right: 1px solid #6a6a6a;
		padding: 0.25em;
	}

	@media (max-width: 500px) {
		.textAreaWrittenAt {
			font-size: 0.9rem;
		}
	}

	@media (max-width: 450px) {
		.textAreaWrittenAt {
			font-size: 0.8rem;
		}
		.textAreaDate {
			font-size: 0.9rem;
		}
	}

	@media (max-width: 400px) {
		.textAreaWrittenAt {
			font-size: 0.7rem;
		}
	}

	.textAreaWrittenAt,
	.textAreaHistory {
		padding: 0.25em;
		align-content: center;
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
		/* Keep a stable column so long lines in the editor don't steal its space */
		flex: 0 0 360px;
		width: 360px;
		min-width: 300px;
		max-width: 400px;
		padding-right: 2rem;
	}

	#middle {
		min-width: 0;
		width: 100%;
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

<script>
	import '../../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Sidenav from '$lib/Sidenav.svelte';
	import { selectedDate, cal, readingDate } from '$lib/calendarStore.js';
	import axios from 'axios';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { searchString, searchTag, searchResults, isSearching } from '$lib/searchStore.js';
	import * as TinyMDE from 'tiny-markdown-editor';
	import '../../../node_modules/tiny-markdown-editor/dist/tiny-mde.css';
	import { API_URL } from '$lib/APIurl.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import {
		faCloudArrowUp,
		faCloudArrowDown,
		faSquarePlus,
		faQuestionCircle
	} from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import { v4 as uuidv4 } from 'uuid';
	import { slide, fade } from 'svelte/transition';
	import { autoLoadImages } from '$lib/settingsStore';
	import { tags } from '$lib/tagStore';
	import Tag from '$lib/Tag.svelte';
	import TagModal from '$lib/TagModal.svelte';
	import FileList from '$lib/FileList.svelte';
	import { formatBytes } from '$lib/helpers.js';
	import ImageViewer from '$lib/ImageViewer.svelte';

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

	let cancelDownload = new AbortController();

	let tinyMDE;
	onMount(() => {
		$readingDate = null; // no reading-highlighting when in write mode

		tinyMDE = new TinyMDE.Editor({ element: 'editor', content: '' });
		let commandBar = new TinyMDE.CommandBar({ element: 'toolbar', editor: tinyMDE });
		document.getElementsByClassName('TinyMDE')[0].classList.add('focus-ring');

		tinyMDE.addEventListener('change', (event) => {
			currentLog = event.content;
			handleInput();
		});

		loadTags();

		getLog();

		// enable popovers
		const popoverTriggerList = document.querySelectorAll('[data-bs-toggle="popover"]');
		const popoverList = [...popoverTriggerList].map(
			(popoverTriggerEl) =>
				new bootstrap.Popover(popoverTriggerEl, { trigger: 'focus', html: true })
		);
	});

	function loadTags() {
		axios
			.get(API_URL + '/logs/getTags')
			.then((response) => {
				$tags = response.data;
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingTags'));
				toast.show();
			});
	}

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
		if ($selectedDate !== lastSelectedDate) {
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
				$cal.currentYear = $selectedDate.getFullYear();
				$cal.currentMonth = $selectedDate.getMonth();
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
		const newDate = new Date($selectedDate);
		newDate.setDate(newDate.getDate() + increment);
		$selectedDate = newDate;
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
					date: $selectedDate.toISOString()
				}
			});

			currentLog = response.data.text;
			filesOfDay = response.data.files;
			selectedTags = response.data.tags;

			savedLog = currentLog;

			tinyMDE.setContent(currentLog);
			tinyMDE.setSelection({ row: 0, col: 0 });

			logDateWritten = response.data.date_written;

			return true;
		} catch (error) {
			console.error(error.response);
			// toast
			const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingLog'));
			toast.show();

			return false;
		}
	}

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

					if ($autoLoadImages) {
						loadImage(file);
					}
				}
			});
		}
	});

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
		let date_written = new Date().toLocaleString('de-DE', {
			timeZone: 'Europe/Berlin',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		});

		let dateOfSave = lastSelectedDate;
		try {
			const response = await axios.post(API_URL + '/logs/saveLog', {
				date: lastSelectedDate.toISOString(),
				text: currentLog,
				date_written: date_written
			});

			if (response.data.success) {
				savedLog = currentLog;
				logDateWritten = date_written;

				// add to $cal.daysWithLogs
				if (!$cal.daysWithLogs.includes(lastSelectedDate.getDate())) {
					$cal.daysWithLogs = [...$cal.daysWithLogs, dateOfSave.getDate()];
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

	function searchForString() {
		if ($isSearching) {
			return;
		}
		$isSearching = true;

		axios
			.get(API_URL + '/logs/searchString', {
				params: {
					searchString: $searchString
				}
			})
			.then((response) => {
				$searchResults = [...response.data];
				$isSearching = false;
			})
			.catch((error) => {
				$searchResults = [];
				console.error(error);
				$isSearching = false;

				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSearching'));
				toast.show();
			});
	}

	function searchForTag() {
		$searchString = '';
		if ($isSearching) {
			return;
		}
		$isSearching = true;

		axios
			.get(API_URL + '/logs/searchTag', { params: { tag_id: $searchTag.id } })
			.then((response) => {
				$searchResults = [...response.data];
				$isSearching = false;
			})
			.catch((error) => {
				$isSearching = false;
				$searchResults = [];

				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSearching'));
				toast.show();
			});
	}

	function triggerFileInput() {
		document.getElementById('fileInput').click();
	}

	function onFileChange(event) {
		for (let i = 0; i < event.target.files.length; i++) {
			uploadFile(event.target.files[i]);
		}
	}

	let uploadingFiles = $state([]);

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
		formData.append('day', $selectedDate.getDate());
		formData.append('month', $selectedDate.getMonth() + 1);
		formData.append('year', $selectedDate.getFullYear());
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
					year: $selectedDate.getFullYear(),
					month: $selectedDate.getMonth() + 1,
					day: $selectedDate.getDate()
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
				day: $selectedDate.getDate(),
				month: $selectedDate.getMonth() + 1,
				year: $selectedDate.getFullYear(),
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
				day: $selectedDate.getDate(),
				month: $selectedDate.getMonth() + 1,
				year: $selectedDate.getFullYear(),
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
</script>

<DatepickerLogic />
<svelte:window onkeydown={on_key_down} onkeyup={on_key_up} />

<!-- shown on small Screen, when triggered -->
<div class="offcanvas-md d-md-none offcanvas-start p-3" id="sidenav" tabindex="-1">
	<div class="offcanvas-header">
		<button
			type="button"
			class="btn-close"
			data-bs-dismiss="offcanvas"
			data-bs-target="#sidenav"
			aria-label="Close"
		></button>
	</div>
	<Sidenav {searchForString} {searchForTag} />
</div>

<div class="d-flex flex-row justify-content-between h-100">
	<!-- shown on large Screen -->
	<div class="d-md-block d-none sidenav p-3">
		<Sidenav {searchForString} {searchForTag} />
	</div>

	<!-- Center -->
	<div class="d-flex flex-column mt-4 mx-4 flex-fill" id="middle">
		<!-- Input-Area -->
		<!-- <div class="d-flex flex-column"> -->
		<div class="d-flex flex-row textAreaHeader">
			<div class="flex-fill textAreaDate">
				{$selectedDate.toLocaleDateString('locale', { weekday: 'long' })}<br />
				{$selectedDate.toLocaleDateString('locale', {
					day: '2-digit',
					month: '2-digit',
					year: 'numeric'
				})}
			</div>
			<div class="flex-fill textAreaWrittenAt">
				<div class={logDateWritten ? '' : 'opacity-50'}>Geschrieben am:</div>
				{logDateWritten}
			</div>
			<div class="textAreaHistory">history</div>
			<div class="textAreaDelete">delete</div>
		</div>
		<div id="log" class="focus-ring">
			<div id="toolbar"></div>
			<div id="editor"></div>
		</div>
		{#if images.length > 0}
			{#if !$autoLoadImages && images.find((image) => !image.src && !image.loading)}
				<div class="d-flex flex-row">
					<button type="button" class="loadImageBtn" onclick={() => loadImages()}>
						<Fa icon={faCloudArrowDown} class="me-2" size="2x" fw /><br />
						{#if images.length === 1}
							1 Bild laden
						{:else}
							{images.length} Bilder laden
						{/if}
						({formatBytes(
							images.filter((i) => !i.src).reduce((sum, image) => sum + (image.size || 0), 0)
						)})
					</button>
				</div>
			{:else}
				<ImageViewer {images} />
				<!-- <div class="d-flex flex-row images mt-3">
					{#each images as image (image.uuid_filename)}
						<button
							type="button"
							onclick={() => {
								viewImage(image.uuid_filename);
							}}
							class="imageContainer d-flex align-items-center position-relative"
							transition:slide={{ axis: 'x' }}
						>
							{#if image.src}
								<img transition:fade class="image" alt={image.filename} src={image.src} />
							{:else}
								<div class="spinner-border" role="status">
									<span class="visually-hidden">Loading...</span>
								</div>
							{/if}
						</button>
					{/each}
				</div> -->
			{/if}
		{/if}
		{$selectedDate}<br />
		{lastSelectedDate}
		<!-- </div> -->
	</div>

	<div id="right" class="d-flex flex-column">
		<div class="tags">
			<div class="d-flex flex-row justify-content-between">
				<div class="d-flex flex-row">
					<h3>Tags</h3>
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
					data-bs-title="Tags"
					data-bs-content="Hier kannst du Tags zum ausgewählten Datum hinzufügen und entfernen, um deine Einträge zu kategorisieren. Ebenso kannst du hier neue Tags erstellen.<br/><br/>Um ein Tag zu ändern oder auch vollständig zu löschen, musst du in die Einstellungen wechseln."
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
					placeholder="Tag..."
				/>
				<button class="newTagBtn btn btn-outline-secondary ms-2" onclick={openTagModal}>
					<Fa icon={faSquarePlus} fw /> Neu
				</button>
			</div>
			{#if showTagDropdown}
				<div id="tagDropdown">
					{#if filteredTags.length === 0}
						<em style="padding: 0.2rem;">Kein Tag gefunden...</em>
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

		<div class="files d-flex flex-column">
			<button
				class="btn btn-secondary {filesOfDay?.length > 0 ? 'mb-2' : ''}"
				id="uploadBtn"
				onclick={triggerFileInput}
				><Fa icon={faCloudArrowUp} class="me-2" id="uploadIcon" />Upload</button
			>
			<input type="file" id="fileInput" multiple style="display: none;" onchange={onFileChange} />

			<FileList files={filesOfDay} {downloadFile} {askDeleteFile} deleteAllowed />
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
								Wird verschlüsselt...
							{/if}
						</div>
					</div>
				</div>
			{/each}
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
				<div class="toast-body">Fehler beim Enfternen des Tags!</div>
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
				<div class="toast-body">Fehler beim Hinzufügen des Tags zum ausgewählten Datum!</div>
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
				<div class="toast-body">Fehler beim Speichern des Tags!</div>
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
				<div class="toast-body">Fehler beim Speichern des Textes!</div>
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
				<div class="toast-body">Fehler beim Laden des Textes!</div>
			</div>
		</div>

		<div
			id="toastErrorSearching"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">Fehler beim Suchen!</div>
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
				<div class="toast-body">Fehler beim Speichern einer Datei!</div>
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
				<div class="toast-body">Fehler beim Löschen einer Datei!</div>
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
				<div class="toast-body">Fehler beim Download einer Datei!</div>
			</div>
		</div>
	</div>

	<div class="modal fade" id="modalConfirmDeleteFile" tabindex="-1">
		<div class="modal-dialog modal-dialog-centered">
			<div class="modal-content">
				<div class="modal-header">
					<h5 class="modal-title">Datei löschen?</h5>
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"
					></button>
				</div>
				<div class="modal-body">
					<p>
						Datei <u><span class="filenameWeight">{confirmDelete.filename}</span></u> wirklich löschen?
					</p>
				</div>
				<div class="modal-footer">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Schließen</button>
					<button
						onclick={() => deleteFile(confirmDelete.uuid)}
						type="button"
						class="btn btn-primary"
						data-bs-dismiss="modal">Löschen</button
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
	.tagRow {
		width: 100%;
	}

	.newTagBtn {
		white-space: nowrap;
	}

	.tag-item.selected {
		background-color: #b2b4b6;
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
		background-color: white;
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
		z-index: 1000;
		max-height: 200px;
		overflow-y: scroll;
		overflow-x: hidden;
		display: flex;
		flex-direction: column;
	}

	.tag-item {
		cursor: pointer;
		padding: 5px;
	}

	.tags {
		z-index: 10;
		padding: 0.5rem;
		margin-right: 2rem;
		margin-bottom: 2rem;
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
		border: 1px solid #ececec77;
		border-radius: 10px;
	}

	.loadImageBtn {
		padding: 0.5rem 1rem;
		border: none;
		margin-top: 0.5rem;
		border-radius: 5px;
		transition: all ease 0.2s;
	}

	:global(.modal.show) {
		background-color: rgba(80, 80, 80, 0.1) !important;
		backdrop-filter: blur(2px) saturate(150%);
	}

	.modal-content {
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
	}

	.filenameWeight {
		font-weight: 550;
	}

	.files {
		margin-right: 2rem;
		border-radius: 10px;
		padding: 1rem;
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
		border: 1px solid #ececec77;
	}

	:global(#uploadIcon) {
		transition: all ease 0.3s;
	}

	:global(#uploadBtn:hover > #uploadIcon) {
		transform: scale(1.4);
	}

	:global(.TMCommandBar) {
		border-top: 1px solid #ccc;
		border-left: 1px solid #ccc;
		border-right: 1px solid #ccc;
	}

	#editor {
		height: 400px;
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
		/* max-width: 430px; */
		width: 380px;
		min-width: 380px;
	}

	.textAreaHeader {
		border-left: 1px solid #ccc;
		border-top: 1px solid #ccc;
		border-right: 1px solid #ccc;
		border-top-left-radius: 5px;
		border-top-right-radius: 5px;
	}

	.textAreaDate,
	.textAreaWrittenAt,
	.textAreaHistory {
		border-right: 1px solid #ccc;
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
		min-width: 300px;
		max-width: 400px;
	}
</style>

<script>
	import '../../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Sidenav from '$lib/Sidenav.svelte';
	import { selectedDate, cal, readingDate } from '$lib/calendarStore.js';
	import axios from 'axios';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { searchString, searchResults } from '$lib/searchStore.js';
	import * as TinyMDE from 'tiny-markdown-editor';
	import '../../../node_modules/tiny-markdown-editor/dist/tiny-mde.css';
	import { API_URL } from '$lib/APIurl.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import { faCloudArrowUp, faTrash } from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import { v4 as uuidv4 } from 'uuid';
	import { slide, fade } from 'svelte/transition';

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

		getLog();
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
		if (loading) return;
		loading = true;

		if ($selectedDate !== lastSelectedDate) {
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
	}

	function on_key_up(event) {
		if (event.key === 'Alt') {
			event.preventDefault();
			altPressed = false;
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

	function base64ToArrayBuffer(base64) {
		var binaryString = atob(base64);
		var bytes = new Uint8Array(binaryString.length);
		for (var i = 0; i < binaryString.length; i++) {
			bytes[i] = binaryString.charCodeAt(i);
		}
		return bytes.buffer;
	}

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
					axios
						.get(API_URL + '/logs/downloadFile', { params: { uuid: file.uuid_filename } })
						.then((response) => {
							images = images.map((image) => {
								if (image.uuid_filename === file.uuid_filename) {
									image.src = response.data.file;
									file.src = response.data.file;
								}
								return image;
							});
						})
						.catch((error) => {
							console.error(error);
							// toast
							const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingFile'));
							toast.show();
						});
				}
			});
		}
	});

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

	let isSearching = $state(false);
	function search() {
		console.log($searchString);

		if (isSearching) {
			return;
		}
		isSearching = true;

		axios
			.get(API_URL + '/logs/search', {
				params: {
					searchString: $searchString
				}
			})
			.then((response) => {
				$searchResults = [...response.data];
				isSearching = false;
			})
			.catch((error) => {
				$searchResults = [];
				console.error(error);
				isSearching = false;

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

	function formatBytes(bytes) {
		if (!+bytes) return '0 Bytes';

		const k = 1024;
		//const dm = 2; // decimal places
		const sizes = ['B', 'KB', 'MB', 'GB'];

		const i = Math.floor(Math.log(bytes) / Math.log(k));

		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(0))} ${sizes[i]}`;
	}

	async function downloadFile(uuid) {
		// check if present in filesOfDay
		let file = filesOfDay.find((file) => file.uuid_filename === uuid);
		if (!file.src) {
			// download from server

			try {
				const response = await axios.get(API_URL + '/logs/downloadFile', {
					params: { uuid: uuid }
				});

				filesOfDay = filesOfDay.map((f) => {
					if (f.uuid_filename === uuid) {
						f.src = response.data.file;
					}
					return f;
				});
			} catch (error) {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingFile'));
				toast.show();
			}
		}

		for (let i = 0; i < filesOfDay.length; i++) {
			if (filesOfDay[i].uuid_filename === uuid) {
				file = filesOfDay[i];
				break;
			}
		}

		const blob = new Blob([base64ToArrayBuffer(file.src)], {
			type: 'application/octet-stream'
		});
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = file.filename;
		document.body.appendChild(a);
		a.click();
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

	let activeImage = $state('');
	function viewImage(uuid) {
		activeImage = uuid;

		const modal = new bootstrap.Modal(document.getElementById('modalImages'));
		modal.show();
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
	<Sidenav {search} />
</div>

<div class="d-flex flex-row justify-content-between h-100">
	<!-- shown on large Screen -->
	<div class="d-md-block d-none sidenav p-3">
		<Sidenav {search} />
	</div>

	<!-- Center -->
	<div class="d-flex flex-column mt-4 mx-4 flex-fill">
		<!-- Input-Area -->
		<div class="d-flex flex-column">
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
				<div class="d-flex flex-row images mt-3">
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
								<img
									transition:fade
									class="image"
									alt={image.filename}
									src={'data:image/jpg;base64,' + image.src}
								/>
							{:else}
								<div class="spinner-border" role="status">
									<span class="visually-hidden">Loading...</span>
								</div>
							{/if}
						</button>
					{/each}
				</div>
			{/if}
			{$selectedDate}<br />
			{lastSelectedDate}
		</div>
	</div>

	<div id="right" class="d-flex flex-column">
		<div>Tags</div>

		<div class="files d-flex flex-column">
			<button
				class="btn btn-secondary {filesOfDay?.length > 0 ? 'mb-2' : ''}"
				id="uploadBtn"
				onclick={triggerFileInput}
				><Fa icon={faCloudArrowUp} class="me-2" id="uploadIcon" />Upload</button
			>
			<input type="file" id="fileInput" multiple style="display: none;" onchange={onFileChange} />

			{#each filesOfDay as file (file.uuid_filename)}
				<div class="btn-group file mt-2" transition:slide>
					<button
						onclick={() => downloadFile(file.uuid_filename)}
						class="p-2 fileBtn d-flex flex-row align-items-center flex-fill"
						><div class="filename filenameWeight">{file.filename}</div>
						<span class="filesize">({formatBytes(file.size)})</span>
					</button>
					<button
						class="p-2 fileBtn deleteFileBtn"
						onclick={() => askDeleteFile(file.uuid_filename, file.filename)}
						><Fa icon={faTrash} id="uploadIcon" fw /></button
					>
				</div>
			{/each}
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

	<div
		class="modal fade"
		id="modalImages"
		tabindex="-1"
		aria-labelledby="modalImagesLabel"
		aria-hidden="true"
	>
		<div class="modal-dialog modal-xl modal-fullscreen-sm-down">
			<div class="modal-content">
				<div class="modal-header d-none d-sm-block">
					<!-- <h1 class="modal-title fs-5" id="exampleModalLabel"></h1> -->
					<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"
					></button>
				</div>

				<div class="modal-body">
					<div id="imageCarousel" class="carousel slide">
						<div class="carousel-indicators">
							{#each images as image, i (image.uuid_filename)}
								<button
									type="button"
									data-bs-target="#imageCarousel"
									data-bs-slide-to={i}
									aria-label="Slide {i}"
									class={image.uuid_filename === activeImage ? 'active' : ''}
								></button>
							{/each}
						</div>
						<div class="carousel-inner">
							{#each images as image}
								<div class="carousel-item {image.uuid_filename === activeImage ? 'active' : ''}">
									<img
										src={'data:image/' + image.filename.split('.').pop() + ';base64,' + image.src}
										class="d-block w-100"
										alt={image.filename}
									/>
									<div class="carousel-caption d-none d-md-block">
										<span class="imageLabelCarousel">{image.filename}</span>
										<button
											class="btn btn-primary"
											onclick={() => downloadFile(image.uuid_filename)}
										>
											Download
										</button>
									</div>
								</div>
							{/each}
						</div>
						<button
							class="carousel-control-prev"
							type="button"
							data-bs-target="#imageCarousel"
							data-bs-slide="prev"
						>
							<span class="carousel-control-prev-icon" aria-hidden="true"></span>
							<span class="visually-hidden">Previous</span>
						</button>
						<button
							class="carousel-control-next"
							type="button"
							data-bs-target="#imageCarousel"
							data-bs-slide="next"
						>
							<span class="carousel-control-next-icon" aria-hidden="true"></span>
							<span class="visually-hidden">Next</span>
						</button>
					</div>
				</div>
				<div class="modal-footer d-block d-sm-none">
					<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.imageLabelCarousel {
		font-size: 20px;
		transition: background-color ease 0.3s;
		padding: 5px;
		border-radius: 5px;
	}

	.carousel-caption:hover > .imageLabelCarousel {
		background-color: rgba(0, 0, 0, 0.4);
	}

	.image,
	.imageContainer {
		border-radius: 8px;
	}

	.imageContainer {
		min-height: 80px;
		padding: 0px;
		border: 0px;
		background-color: transparent;
		overflow: hidden;
	}

	.image:hover {
		transform: scale(1.1);
		box-shadow: 0 0 12px 3px rgba(0, 0, 0, 0.2);
	}

	.image {
		max-width: 250px;
		max-height: 150px;
		transition: all ease 0.3s;
	}

	.images {
		gap: 1rem;
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

	.filename {
		padding-right: 0.5rem;
		word-break: break-word;
	}

	.filesize {
		opacity: 0.7;
		font-size: 0.8rem;
		white-space: nowrap;
	}

	.fileBtn {
		border: 0;
		background-color: rgba(0, 0, 0, 0);
		transition: all ease 0.3s;
	}

	.fileBtn:hover {
		background-color: rgba(0, 0, 0, 0.1);
	}

	.deleteFileBtn {
		border-left: 1px solid rgba(92, 92, 92, 0.445);
	}

	.deleteFileBtn:hover {
		color: rgb(165, 0, 0);
	}

	.file {
		background-color: rgba(117, 117, 117, 0.45);
		border: 0px solid #ececec77;
		border-radius: 5px;
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
		width: 300px;
	}
</style>

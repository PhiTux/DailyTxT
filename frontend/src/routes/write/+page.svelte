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
	import { faCloudArrowUp } from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import { v4 as uuidv4 } from 'uuid';

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

	let loading = false;
	$effect(() => {
		if (loading) return;
		loading = true;

		if ($selectedDate !== lastSelectedDate) {
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
				console.log(response);
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				uploadingFiles = uploadingFiles.filter((file) => file.uuid !== uuid);
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
			{$selectedDate}<br />
			{lastSelectedDate}
		</div>
	</div>

	<div id="right" class="d-flex flex-column">
		<div>Tags</div>

		<div class="files">
			<button class="btn btn-secondary" id="uploadBtn" onclick={triggerFileInput}
				><Fa icon={faCloudArrowUp} class="me-2" id="uploadIcon" />Upload</button
			>
			<input type="file" id="fileInput" multiple style="display: none;" onchange={onFileChange} />

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
	</div>
</div>

<style>
	:global(#uploadIcon) {
		transition: all ease 0.3s;
	}

	:global(#uploadBtn:hover > #uploadIcon) {
		transform: scale(1.2);
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

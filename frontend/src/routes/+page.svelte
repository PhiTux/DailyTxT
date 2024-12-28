<script>
	import '../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Sidenav from './Sidenav.svelte';
	import { selectedDate } from '$lib/calendarStore.js';
	import axios from 'axios';
	import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	let API_URL = dev ? 'http://localhost:8000' : window.location.pathname.replace(/\/+$/, '');

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

	onMount(() => {
		getLog();
	});

	let lastSelectedDate = $state($selectedDate);

	$effect(() => {
		if ($selectedDate !== lastSelectedDate) {
			getLog();
			lastSelectedDate = $selectedDate;
		}
	});

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

	function getLog() {
		if (savedLog !== currentLog) {
			if (!saveLog()) {
				return;
			}
		}

		axios
			.get(API_URL + '/logs/getLog', {
				params: {
					date: $selectedDate.toISOString()
				}
			})
			.then((response) => {
				currentLog = response.data.text;
				savedLog = currentLog;
				logDateWritten = response.data.date_written;
			})
			.catch((error) => {
				console.error(error.response);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorLoadingLog'));
				toast.show();
			});
	}

	function saveLog() {
		// axios to backend
		let date_written = new Date().toLocaleString('de-DE', {
			timeZone: 'Europe/Berlin',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit'
		});

		axios
			.post(API_URL + '/logs/saveLog', {
				date: $selectedDate.toISOString(),
				text: currentLog,
				date_written: date_written
			})
			.then((response) => {
				if (response.data.success) {
					savedLog = currentLog;
					logDateWritten = date_written;
					return true;
				} else {
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorSavingLog'));
					toast.show();
					console.error('Log not saved');
					return false;
				}
			})
			.catch((error) => {
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSavingLog'));
				toast.show();
				console.error(error.response);
				return false;
			});
	}
</script>

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
	<Sidenav />
</div>

<div class="d-flex flex-row justify-content-between">
	<!-- shown on large Screen -->
	<div class="d-md-block d-none sidenav p-3">
		<Sidenav />
	</div>

	<!-- Center -->
	<div class="d-flex flex-column mt-4 mx-4 flex-fill">
		<!-- Input-Area -->
		<div class="d-flex flex-column">
			<div class="d-flex flex-row textAreaHeader">
				<div class="flex-fill textAreaDate">
					{$selectedDate.toLocaleDateString('locale', { weekday: 'long' })}<br />
					{$selectedDate.toLocaleDateString('locale')}
				</div>
				<div class="flex-fill textAreaWrittenAt">
					<div class={logDateWritten ? '' : 'opacity-50'}>Geschrieben am:</div>
					<!-- <br /> -->
					{logDateWritten}
				</div>
				<div class="textAreaHistory">history</div>
				<div class="textAreaDelete">delete</div>
			</div>
			<textarea
				bind:value={currentLog}
				oninput={handleInput}
				class="form-control {currentLog !== savedLog ? 'notSaved' : ''}"
				rows="10"
			></textarea>
		</div>
	</div>

	<div id="right">Right</div>

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
	</div>
</div>

<style>
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

	.notSaved {
		border-color: #f57c00;
		/* border-color: #ff9800; */
	}

	textarea:focus.notSaved {
		box-shadow: 0 0 0 0.25rem #f57c0030;
	}

	textarea:focus:not(.notSaved) {
		border-color: #90ee90;
		box-shadow: 0 0 0 0.25rem #90ee9070;
	}

	.textAreaDate {
		font-weight: 600;
	}

	textarea {
		resize: vertical;
		width: 100%;
		border-top-left-radius: 0;
		border-top-right-radius: 0;
		border-color: lightgreen;
		border-width: 1px;
	}

	#right {
		width: 300px;
	}
</style>

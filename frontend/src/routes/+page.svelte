<script>
	import '../scss/styles.scss';
	import * as bootstrap from 'bootstrap';
	import Sidenav from './Sidenav.svelte';
	import { selectedDate } from '$lib/calendarStore.js';
	import dayjs from 'dayjs';

	$effect(() => {
		if ($selectedDate) {
			console.log('hu');
		}
	});

	let currentLog = $state('');
	let savedLog = $state('');

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

	function saveLog() {
		// axios to backend
		console.log(dayjs().format('DD.MM.YYYY, HH:mm [Uhr]'));
		savedLog = currentLog;
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
					Geschrieben am:<br />
					TODO
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

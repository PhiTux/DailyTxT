<script>
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { cal } from '$lib/calendarStore.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import Sidenav from '$lib/Sidenav.svelte';
	import { onMount } from 'svelte';

	let readingData = $state([]);
	let search = $state('');

	onMount(() => {
		loadMonthForReading();
	});

	//#TODO Muss in die separate /read page (diese hier in /write umbenennen)
	let isLoadingMonthForReading = false;
	function loadMonthForReading() {
		if (isLoadingMonthForReading) {
			return;
		}
		isLoadingMonthForReading = true;

		axios
			.get(API_URL + '/logs/loadMonthForReading', {
				params: {
					month: $cal.currentMonth + 1,
					year: $cal.currentYear
				}
			})
			.then((response) => {
				readingData = response.data;
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				isLoadingMonthForReading = false;
			});
	}
</script>

<DatepickerLogic />

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
				<div class="flex-fill textAreaWrittenAt"></div>
				<div class="textAreaHistory">history</div>
				<div class="textAreaDelete">delete</div>
			</div>
			<div id="log" class="focus-ring">
				<div id="toolbar"></div>
				<div id="editor"></div>
			</div>
		</div>
	</div>

	<div id="right">Right</div>
</div>

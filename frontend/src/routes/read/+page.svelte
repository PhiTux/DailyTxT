<script>
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { cal, selectedDate, readingDate } from '$lib/calendarStore.js';
	import DatepickerLogic from '$lib/DatepickerLogic.svelte';
	import Sidenav from '$lib/Sidenav.svelte';
	import { onMount } from 'svelte';
	import { marked } from 'marked';

	marked.use({
		breaks: true,
		gfm: true
	});

	let logs = $state([]);
	let search = $state('');

	let observer;

	onMount(() => {
		loadMonthForReading();

		// Highlights automatically the day in the calendar, when the log is in the viewport
		observer = new IntersectionObserver(
			(entries) => {
				entries.forEach((entry) => {
					if (entry.isIntersecting) {
						$readingDate = new Date(
							$cal.currentYear,
							$cal.currentMonth,
							entry.target.getAttribute('data-log-day')
						);
					}
				});
			},
			{
				root: null,
				rootMargin: '0% 0px -70% 0px',
				threshold: 0.67
			}
		);
	});

	let currentMonth = $cal.currentMonth;
	let currentYear = $cal.currentYear;
	$effect(() => {
		if ($cal.currentMonth !== currentMonth || $cal.currentYear !== currentYear) {
			loadMonthForReading();
			currentMonth = $cal.currentMonth;
			currentYear = $cal.currentYear;
		}
	});

	$effect(() => {
		if ($selectedDate) {
			let el = document.querySelector(`.log[data-log-day="${$selectedDate.getDate()}"]`);
			if (el) {
				el.scrollIntoView({ behavior: 'smooth', block: 'start' });
			}
		}
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
				logs = response.data;
			})
			.catch((error) => {
				console.error(error);
			})
			.finally(() => {
				isLoadingMonthForReading = false;

				setTimeout(() => {
					document.querySelectorAll('.log').forEach((log) => {
						observer.observe(log);
					});
				}, 1000);
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
	<div class="d-flex flex-column my-4 mx-4 flex-fill overflow-y-auto" id="scrollArea">
		{#each logs as log}
			<!-- Log-Area -->
			<div class="log mb-3 p-3 d-flex flex-row" data-log-day={log.day}>
				<div class="date me-3 d-flex flex-column align-items-center">
					<p class="dateNumber">{log.day}</p>
					<p class="dateDay">
						<b>
							{new Date($cal.currentYear, $cal.currentMonth, log.day).toLocaleDateString('locale', {
								weekday: 'long'
							})}
						</b>
					</p>
				</div>
				<div>
					{@html marked.parse(log.text)}
				</div>
			</div>
		{/each}
	</div>

	<div id="right">Right</div>
</div>

<style>
	.log {
		backdrop-filter: blur(5px) saturate(150%);
		background-color: rgba(182, 183, 185, 0.75);
		border-radius: 15px;
	}

	.dateNumber {
		font-size: 3rem;
		font-weight: 600;
		font-style: italic;
		opacity: 0.5;
	}

	.dateDay {
		opacity: 0.7;
	}
</style>

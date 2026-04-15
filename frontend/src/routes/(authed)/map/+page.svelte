<script>
	import Map from '$lib/Map.svelte';
	import DayPreviewModal from '$lib/map/DayPreviewModal.svelte';
	import { onMount } from 'svelte';
	import axios from 'axios';
	import { API_URL } from '$lib/APIurl.js';
	import { slide } from 'svelte/transition';
	import { getTolgee, getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);
	import * as bootstrap from 'bootstrap';
	import { settings } from '$lib/settingsStore';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let allPins = $state([]);
	let pinStartDate = $state('');
	let pinEndDate = $state('');
	let showPreviewModal = $state(false);
	let previewDay = $state(null);
	let previewMonth = $state(null);
	let previewYear = $state(null);
	let showDateFilterMobile = $state(false);
	let smallScreen = $state(false);

	onMount(() => {
		loadAllPins();

		window.addEventListener('resize', () => {
			checkSmallScreen();
		});
		checkSmallScreen();
	});

	$effect(() => {
		if (Object.keys($settings).length > 0 && !$settings.useMap) {
			console.log($settings);
			goto(resolve('/write'));
		}
	});

	function checkSmallScreen() {
		smallScreen = window.innerWidth < 768;
	}

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	function loadAllPins() {
		axios
			.get(`${API_URL}/logs/allPins`)
			.then((response) => {
				allPins = response.data;
			})
			.catch((error) => {
				console.error('Error fetching pins:', error);

				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorFetchPins'));
				toast.show();
			});
	}

	function openPreview(day, month, year) {
		previewDay = day;
		previewMonth = month;
		previewYear = year;
		showPreviewModal = true;
	}

	function parseDateInput(dateString) {
		if (!dateString) return null;
		const [year, month, day] = dateString.split('-').map(Number);
		if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
			return null;
		}
		return { year, month, day };
	}

	function compareDateParts(a, b) {
		if (a.year !== b.year) return a.year - b.year;
		if (a.month !== b.month) return a.month - b.month;
		return a.day - b.day;
	}

	let selectedPins = $derived.by(() => {
		const startDate = parseDateInput(pinStartDate);
		const endDate = parseDateInput(pinEndDate);
		const filteredPins = [];

		for (const dayEntry of allPins) {
			const year = Number(dayEntry?.year);
			const month = Number(dayEntry?.month);
			const day = Number(dayEntry?.day);
			if (!Number.isFinite(year) || !Number.isFinite(month) || !Number.isFinite(day)) {
				continue;
			}

			const currentDate = { year, month, day };
			if (startDate && compareDateParts(currentDate, startDate) < 0) continue;
			if (endDate && compareDateParts(currentDate, endDate) > 0) continue;

			const pinsForDay = Array.isArray(dayEntry?.pins) ? dayEntry.pins : [];
			for (const pin of pinsForDay) {
				filteredPins.push({ ...pin, year, month, day });
			}
		}

		return filteredPins;
	});
</script>

<!-- aria-expanded={showDateFilterMobile} -->
{#if $settings.useMap}
	<div class="date-picker glass-shadow position-absolute top-0 end-0 m-3 p-3">
		<div class="d-flex flex-column">
			<button
				type="button"
				class="date-picker-toggle"
				onclick={() => (showDateFilterMobile = !showDateFilterMobile)}
			>
				<h6 class="align-self-center my-1">{$t('settings.export.period')}</h6>
				<span class="date-picker-chevron" aria-hidden="true">
					{showDateFilterMobile ? '▲' : '▼'}
				</span>
			</button>
			{#if showDateFilterMobile || !smallScreen}
				<div
					class="date-picker-content"
					transition:slide /* class:mobile-open={showDateFilterMobile} */
				>
					<div>
						<label for="pinStartDate">{$t('settings.export.start_date')}</label>
						<div class="date-input-row">
							<input type="date" class="form-control" id="pinStartDate" bind:value={pinStartDate} />
							{#if pinStartDate !== ''}
								<button
									type="button"
									class="btn-close clear-date-btn"
									aria-label="Delete start date"
									onclick={() => (pinStartDate = '')}
								></button>
							{/if}
						</div>
					</div>
					<div>
						<label for="pinEndDate">{$t('settings.export.end_date')}</label>
						<div class="date-input-row">
							<input type="date" class="form-control" id="pinEndDate" bind:value={pinEndDate} />
							{#if pinEndDate !== ''}
								<button
									type="button"
									class="btn-close clear-date-btn"
									aria-label="Delete end date"
									onclick={() => (pinEndDate = '')}
								></button>
							{/if}
						</div>
					</div>
					{#if pinStartDate !== '' && pinEndDate !== '' && pinStartDate > pinEndDate}
						<div transition:slide>
							<div class="pt-2"></div>
							<div class="alert alert-danger mb-0" role="alert">
								{$t('settings.export.period_invalid')}
							</div>
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>

	<Map fullScreen pins={selectedPins} {openPreview} />
{/if}

<DayPreviewModal
	bind:open={showPreviewModal}
	day={previewDay}
	month={previewMonth}
	year={previewYear}
	language={$tolgee.getLanguage()}
/>

<div class="toast-container position-fixed bottom-0 end-0 p-3">
	<div
		id="toastErrorFetchPins"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('map.toast.error_fetching_pins')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>
</div>

<style>
	h6 {
		font-size: 1.2rem;
		text-decoration: underline;
		text-decoration-color: #1565c0;
	}

	.date-picker {
		width: 220px;
		z-index: 10;
		border-radius: 10px !important;
		border-style: none !important;
		backdrop-filter: blur(7px) saturate(130%);
		background-color: rgba(51, 51, 51, 0.38);
	}

	.date-picker-toggle {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		padding: 0;
		background: transparent;
		border: 0;
		color: inherit;
	}

	.date-picker-chevron {
		position: absolute;
		right: 0;
		top: 50%;
		transform: translateY(-50%);
		display: none;
		font-size: 0.8rem;
		opacity: 0.8;
	}

	.date-picker-content {
		display: flex;
		flex-direction: column;
	}

	.date-input-row {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.clear-date-btn {
		opacity: 0.45;
		transform: scale(0.8);
		transition: opacity 0.15s ease;
		flex-shrink: 0;
	}

	.clear-date-btn:hover,
	.clear-date-btn:focus {
		opacity: 0.75;
	}

	@media (max-width: 768px) {
		.date-picker-toggle {
			cursor: pointer;
		}

		.date-picker-chevron {
			display: inline-block;
		}

		.date-picker-content {
			margin-top: 0.5rem;
		}
	}
</style>

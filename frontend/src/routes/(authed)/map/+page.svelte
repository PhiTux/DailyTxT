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

	let allPins = $state([]);
	let pinStartDate = $state('');
	let pinEndDate = $state('');
	let showPreviewModal = $state(false);
	let previewDay = $state(null);
	let previewMonth = $state(null);
	let previewYear = $state(null);

	onMount(() => {
		loadAllPins();
	});

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

<div class="date-picker glass-shadow position-absolute top-0 end-0 m-3 p-3">
	<div class="d-flex flex-column">
		<div>
			<label for="pinStartDate">{$t('settings.export.start_date')}</label>
			<div class="date-input-row">
				<input type="date" class="form-control" id="pinStartDate" bind:value={pinStartDate} />
				{#if pinStartDate !== ''}
					<button
						type="button"
						class="btn-close clear-date-btn"
						aria-label="Startdatum löschen"
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
						aria-label="Enddatum löschen"
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

	{#if pinStartDate === '' && pinEndDate === ''}
		Zeige alle Pins
	{:else if pinStartDate === ''}
		Zeige alle Pins bis {pinEndDate}
	{:else if pinEndDate === ''}
		Zeige alle Pins ab {pinStartDate}
	{:else}
		Zeige alle Pins von {pinStartDate} bis {pinEndDate}
	{/if}
</div>

<Map fullScreen pins={selectedPins} {openPreview} />

<DayPreviewModal
	bind:open={showPreviewModal}
	day={previewDay}
	month={previewMonth}
	year={previewYear}
	language={$tolgee.getLanguage()}
/>

<style>
	.date-picker {
		width: 220px;
		z-index: 10;
		border-radius: 10px !important;
		border-style: none !important;
		backdrop-filter: blur(7px) saturate(130%);
		background-color: rgba(51, 51, 51, 0.38);
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
</style>

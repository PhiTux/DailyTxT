<script>
	import { cal, selectedDate, readingDate } from '$lib/calendarStore.js';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import * as bootstrap from 'bootstrap';
	import { offcanvasIsOpen, sameDate } from '$lib/helpers.js';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	let { bookmarkDay } = $props();

	let days = $state([]);

	let animationDirection = $state(1); // swipe the dates left or right

	let lastMonth = $cal.currentMonth;
	let lastYear = $cal.currentYear;

	$effect(() => {
		if ($cal.currentMonth !== lastMonth || $cal.currentYear !== lastYear) {
			// set animation direction
			animationDirection = $cal.currentMonth > lastMonth ? 1 : -1;
			if ($cal.currentYear > lastYear) {
				animationDirection = 1;
			} else if ($cal.currentYear < lastYear) {
				animationDirection = -1;
			}

			days = updateCalendar();

			lastMonth = $cal.currentMonth;
			lastYear = $cal.currentYear;
		}
	});

	const updateCalendar = () => {
		const month = $cal.currentMonth;
		const year = $cal.currentYear;
		const firstDay = new Date(Date.UTC(year, month, 1));
		const lastDay = new Date(Date.UTC(year, month + 1, 0));

		let tempDays = [];
		// monday is first day
		let firstDayIndex = firstDay.getDay() - 1;
		if (firstDayIndex === -1) firstDayIndex = 6; // sunday gets 6

		for (let i = 0; i < firstDayIndex; i++) {
			tempDays.push(null); // Fill empty slots before the first day
		}

		for (let i = 1; i <= lastDay.getDate(); i++) {
			tempDays.push({ year: year, month: month + 1, day: i });
		}

		return tempDays;
	};

	const changeMonth = (increment) => {
		$cal.daysWithLogs = [];
		$cal.daysWithFiles = [];
		$cal.currentMonth += increment;
		if ($cal.currentMonth < 0) {
			$cal.currentMonth = 11;
			changeYear(-1);
		} else if ($cal.currentMonth > 11) {
			$cal.currentMonth = 0;
			changeYear(1);
		}
	};

	const changeYear = (increment) => {
		$cal.currentYear += increment;
	};

	let oc;

	const onDateClick = (date) => {
		$selectedDate = date;

		// close offcanvas/sidenav if open
		if (oc) {
			const bsOffcanvas = bootstrap.Offcanvas.getInstance(oc);
			if ($offcanvasIsOpen) {
				bsOffcanvas.hide();
			}
		}
	};

	$effect(() => {
		if (window.location.href) {
			setTimeout(() => {
				oc = document.querySelector('.offcanvas');
				oc.addEventListener('hidden.bs.offcanvas', () => {
					$offcanvasIsOpen = false;
				});
				oc.addEventListener('shown.bs.offcanvas', () => {
					$offcanvasIsOpen = true;
				});
			}, 500);
		}
	});

	onMount(() => {
		days = updateCalendar();

		oc = document.querySelector('.offcanvas');
		oc.addEventListener('hidden.bs.offcanvas', () => {
			$offcanvasIsOpen = false;
		});
		oc.addEventListener('shown.bs.offcanvas', () => {
			$offcanvasIsOpen = true;
		});
	});

	let months = Array.from({ length: 12 }, (_, i) =>
		new Date(2000, i).toLocaleString('default', { month: 'long' })
	);

	const onMonthSelect = (event) => {
		animationDirection = months.indexOf(event.target.value) > $cal.currentMonth ? 1 : -1;
		$cal.currentMonth = months.indexOf(event.target.value);
	};

	const onYearInput = (event) => {
		animationDirection = parseInt(event.target.value) > $cal.currentYear ? 1 : -1;
		const year = parseInt(event.target.value);
		if (year && !isNaN(year) && year >= 1) {
			$cal.currentYear = year;
		}
	};

	// weekdays
	const weekDays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
</script>

<div class="datepicker">
	<div class="datepicker-header">
		<button type="button" class="btn btnLeftRight" onclick={() => changeMonth(-1)}>&lt;</button>
		<div class="date-selectors">
			<select
				value={new Date(2000, $cal.currentMonth).toLocaleString('default', { month: 'long' })}
				onchange={onMonthSelect}
			>
				{#each months as month}
					<option value={month}>{month}</option>
				{/each}
			</select>
			<div class="year-input-group">
				<input
					type="number"
					value={$cal.currentYear}
					min="1"
					max="9999"
					class="year-input"
					oninput={onYearInput}
				/>
				<div class="year-controls">
					<button type="button" class="btn btn-year" onclick={() => changeYear(1)}>▲</button>
					<button type="button" class="btn btn-year" onclick={() => changeYear(-1)}>▼</button>
				</div>
			</div>
		</div>
		<button type="button" class="btn btnLeftRight" onclick={() => changeMonth(1)}>&gt;</button>
	</div>
	<div class="calendar-container">
		{#key days}
			<div
				class="datepicker-grid"
				in:fly={{ x: animationDirection > 0 ? 100 : -100, duration: 200 }}
				out:fly={{ x: animationDirection > 0 ? -100 : 100, duration: 200 }}
			>
				{#each weekDays as day}
					<div class="day-header">{day}</div>
				{/each}
				{#each days as day}
					{#if day}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div
							in:fly={{ y: 100, duration: 200 }}
							out:fly={{ y: -100, duration: 200 }}
							class="day
								{$cal.daysWithLogs.includes(day.day) ? 'mark-background' : ''} 
								{$cal.daysWithFiles.includes(day.day) ? 'mark-dot' : ''} 
								{$cal.daysBookmarked.includes(day.day) ? 'mark-circle' : ''}
								{(!$readingDate && sameDate($selectedDate, day)) || sameDate($readingDate, day) ? 'selected' : ''}"
							onclick={() => onDateClick(day)}
						>
							{day.day}
						</div>
					{:else}
						<div class="day empty-slot"></div>
					{/if}
				{/each}
			</div>
		{/key}
	</div>

	<div class="row mb-2">
		<div class="col-4"></div>
		<div class="col-4 d-flex justify-content-center">
			<button
				class="btn btn-primary"
				onclick={() => {
					$selectedDate = {
						day: new Date().getDate(),
						month: new Date().getMonth() + 1,
						year: new Date().getFullYear()
					};
				}}>{$t('calendar.button_today')}</button
			>
		</div>
		<div class="col-4 d-flex justify-content-end">
			<!-- svelte-ignore a11y_consider_explicit_label -->
			<button class="btn btn-secondary me-2" onclick={bookmarkDay}>
				<svg
					id="bookmark-icon"
					data-name="Layer 1"
					xmlns="http://www.w3.org/2000/svg"
					viewBox="0 0 91.5 122.88"
					width="18"
					><defs
						><style>
							.cls-1 {
								fill-rule: evenodd;
							}
						</style></defs
					><title>add-bookmark</title>
					{#if !$cal.daysBookmarked.includes($selectedDate.day)}
						<path
							class="cls-1"
							d="M62.42,0A29.08,29.08,0,1,1,33.34,29.08,29.08,29.08,0,0,1,62.42,0ZM3.18,19.65H24.73a38,38,0,0,0-1,6.36H6.35v86.75L37.11,86.12a3.19,3.19,0,0,1,4.18,0l31,26.69V66.68a39.26,39.26,0,0,0,6.35-2.27V119.7a3.17,3.17,0,0,1-5.42,2.24l-34-29.26-34,29.42a3.17,3.17,0,0,1-4.47-.33A3.11,3.11,0,0,1,0,119.7H0V22.83a3.18,3.18,0,0,1,3.18-3.18Zm55-2.79a4.1,4.1,0,0,1,.32-1.64l0-.06a4.33,4.33,0,0,1,3.9-2.59h0a4.23,4.23,0,0,1,1.63.32,4.3,4.3,0,0,1,1.39.93,4.15,4.15,0,0,1,.93,1.38l0,.07a4.23,4.23,0,0,1,.3,1.55v8.6h8.57a4.3,4.3,0,0,1,3,1.26,4.23,4.23,0,0,1,.92,1.38l0,.07a4.4,4.4,0,0,1,.31,1.49v.18a4.37,4.37,0,0,1-.32,1.55,4.45,4.45,0,0,1-.93,1.4,4.39,4.39,0,0,1-1.38.92l-.08,0a4.14,4.14,0,0,1-1.54.3H66.71v8.57a4.35,4.35,0,0,1-1.25,3l-.09.08a4.52,4.52,0,0,1-1.29.85l-.08,0a4.36,4.36,0,0,1-1.54.31h0a4.48,4.48,0,0,1-1.64-.32,4.3,4.3,0,0,1-1.39-.93,4.12,4.12,0,0,1-.92-1.38,4.3,4.3,0,0,1-.34-1.62V34H49.56a4.28,4.28,0,0,1-1.64-.32l-.07,0a4.32,4.32,0,0,1-2.25-2.28l0-.08a4.58,4.58,0,0,1-.3-1.54v0a4.39,4.39,0,0,1,.33-1.63,4.3,4.3,0,0,1,3.93-2.66h8.61V16.86Z"
						/>
					{:else}
						<path
							class="cls-1"
							d="M62.42,0A29.08,29.08,0,1,1,33.34,29.08,29.08,29.08,0,0,1,62.42,0ZM3.18,19.65H24.73a38,38,0,0,0-1,6.36H6.35v86.75L37.11,86.12a3.19,3.19,0,0,1,4.18,0l31,26.69V66.68a39.26,39.26,0,0,0,6.35-2.27V119.7a3.17,3.17,0,0,1-5.42,2.24l-34-29.26-34,29.42a3.17,3.17,0,0,1-4.47-.33A3.11,3.11,0,0,1,0,119.7H0V22.83a3.18,3.18,0,0,1,3.18-3.18Zm72.1,5.77a4.3,4.3,0,0,1,3,1.26,4.23,4.23,0,0,1,.92,1.38l0,.07a4.4,4.4,0,0,1,.31,1.49v.18a4.37,4.37,0,0,1-.32,1.55,4.45,4.45,0,0,1-.93,1.4,4.39,4.39,0,0,1-1.38.92l-.08,0a4.14,4.14,0,0,1-1.54.3H49.56a4.28,4.28,0,0,1-1.64-.32l-.07,0a4.32,4.32,0,0,1-2.25-2.28l0-.08a4.58,4.58,0,0,1-.3-1.54v0a4.39,4.39,0,0,1,.33-1.63,4.3,4.3,0,0,1,3.93-2.66Z"
						/>
					{/if}
				</svg>
			</button>
		</div>
	</div>
</div>

<style>
	button:has(#bookmark-icon) {
		background: #f57c00;
		border: none;
	}
	button:has(#bookmark-icon):hover {
		background: rgb(223, 111, 0);
	}

	.btnLeftRight {
		color: white;
	}
	.btnLeftRight:hover {
		background-color: #ffffff31;
	}
	.btnLeftRight:active {
		border-color: rgba(255, 255, 255, 0);
	}
	.datepicker {
		display: inline-block;
		font-family: Arial, sans-serif;
		border: 1px solid #ececec77;
		border-radius: 8px;
		/* overflow: hidden; */
		/* width: 300px; */
		box-sizing: border-box;
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
	}
	.datepicker-header {
		display: flex;
		justify-content: space-between;
		background: #007bff;
		color: white;
		padding: 8px 16px;
		font-size: 16px;
	}
	.calendar-container {
		position: relative;
		overflow: hidden;
		min-height: 282px;
	}
	.datepicker-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		padding: 8px;
		gap: 2px;
		text-align: center;
		position: absolute;
		width: 100%;
	}
	.datepicker-grid:not(.slide-left):not(.slide-right) {
		transform: translateX(0);
		opacity: 1;
	}
	.day-header {
		font-weight: bold;
		padding: 8px 0;
		font-size: 0.9em;
		color: #666;
	}
	.day {
		height: 32px;
		width: 32px;
		min-width: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		cursor: pointer;
		position: relative;
		border-radius: 50%;
		margin: 2px auto;
		user-select: none;
		--dot-color: rgb(250, 199, 58);
	}
	.day:hover {
		background: #f0f0f0;
	}
	.day.mark-background {
		background-color: #00ad00;
		color: white;
		aspect-ratio: 1;
	}

	.day.mark-circle {
		/* background-color: transparent;*/
		border: 3px solid #f57c00;
		/* color: #ff9224; */
	}

	.day.mark-dot::after {
		content: '';
		width: 6px;
		height: 6px;
		background-color: var(--dot-color);
		border-radius: 50%;
		position: absolute;
		bottom: 2px;
		aspect-ratio: 1;
	}
	.empty-slot {
		visibility: hidden;
	}
	.date-selectors {
		display: flex;
		gap: 4px;
		flex: 1;
		justify-content: center;
		align-items: center;
	}

	.date-selectors select {
		background: transparent;
		color: white;
		border: none;
		font-size: 16px;
		cursor: pointer;
		padding: 2px 12px;
		border-radius: 4px;
		max-width: 100px;
		text-align: center;
		text-align-last: center;
		-webkit-appearance: none;
		-moz-appearance: none;
		appearance: none;
	}

	.date-selectors select:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.date-selectors select option {
		background: white;
		color: black;
	}

	.date-selectors select:after {
		content: '▼';
		position: absolute;
		right: 5px;
	}

	.day.selected {
		background-color: #007bff;
		color: white;
	}

	.day.selected:hover {
		background-color: #0056b3;
	}

	.day.mark-dot:not(.selected)::after {
		content: '';
		width: 7px;
		height: 7px;
		background-color: var(--dot-color);
		border-radius: 50%;
		position: absolute;
		bottom: 1px;
	}

	.year-input {
		background: transparent;
		color: white;
		border: none;
		font-size: 16px;
		cursor: pointer;
		padding: 2px 12px;
		border-radius: 4px;
		max-width: 100px;
		text-align: center;
		-webkit-appearance: textfield;
		-moz-appearance: textfield;
		appearance: textfield;
		width: 60px; /* Angepasste Breite */
		padding: 2px 4px; /* Schmalere Paddings */
		padding-right: 20px; /* Platz für die Pfeile */
	}

	.year-input::-webkit-inner-spin-button,
	.year-input::-webkit-outer-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}

	.year-input:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.year-input-group {
		position: relative;
		display: inline-block;
	}

	.year-controls {
		position: absolute;
		right: 2px;
		top: 50%;
		transform: translateY(-50%);
		display: flex;
		flex-direction: column;
		gap: 0;
	}

	.btn-year {
		color: white;
		padding: 0;
		font-size: 12px;
		line-height: 12px;
		min-width: 16px;
		height: 16px;
		opacity: 0.1;
	}

	.year-input-group:hover .btn-year {
		opacity: 0.5;
	}

	.btn-year:hover {
		opacity: 1 !important;
		background-color: rgba(255, 255, 255, 0.1);
	}

	option {
		background-color: #007bff;
		color: white;
	}
</style>

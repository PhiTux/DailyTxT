<script>
	import { cal, selectedDate } from '$lib/calendarStore.js';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';

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
		const firstDay = new Date(year, month, 1);
		const lastDay = new Date(year, month + 1, 0);

		let tempDays = [];
		// monday is first day
		let firstDayIndex = firstDay.getDay() - 1;
		if (firstDayIndex === -1) firstDayIndex = 6; // sunday gets 6

		for (let i = 0; i < firstDayIndex; i++) {
			tempDays.push(null); // Fill empty slots before the first day
		}

		for (let i = 1; i <= lastDay.getDate(); i++) {
			const dayKey = `${year}-${(month + 1).toString().padStart(2, '0')}-${i
				.toString()
				.padStart(2, '0')}`;
			tempDays.push(new Date(Date.UTC(year, month, i)));
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

	const onDateClick = (date) => {
		$selectedDate = date;
	};

	onMount(() => {
		days = updateCalendar();
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
								{$cal.daysWithLogs.includes(day.getDate()) ? 'mark-background' : ''} 
								{$cal.daysWithFiles.includes(day.getDate()) ? 'mark-dot' : ''} 
								{$selectedDate.toDateString() === day.toDateString() ? 'selected' : ''}"
							onclick={() => onDateClick(day)}
						>
							{day.getDate()}
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
					$selectedDate = new Date();
				}}>Heute</button
			>
		</div>
		<div class="col-4 d-flex justify-content-end">
			<button class="btn btn-secondary me-2"> Mark </button>
		</div>
	</div>
</div>

<style>
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
		border: 1px solid #ccc;
		border-radius: 8px;
		/* overflow: hidden; */
		/* width: 300px; */
		box-sizing: border-box;
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
	}
	.day:hover {
		background: #f0f0f0;
	}
	.day.mark-background {
		background-color: #00ad00;
		color: white;
		aspect-ratio: 1;
	}
	.day.mark-dot::after {
		content: '';
		width: 6px;
		height: 6px;
		background-color: var(--color);
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
		width: 6px;
		height: 6px;
		background-color: var(--color);
		border-radius: 50%;
		position: absolute;
		bottom: 2px;
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

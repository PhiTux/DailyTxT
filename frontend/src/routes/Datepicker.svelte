<script>
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';

	let { currentlySelectedDate = new Date(), dateSelected } = $props();
	let shownDate = $state(currentlySelectedDate);
	let days = $state([]);
	let markedDays = {
		'2024-12-25': { type: 'background', color: '#28a745' }, // green instead of red
		'2024-12-31': { type: 'dot', color: '#28a745' } // green instead of blue
	};
	let currentMonth = $state(shownDate.toLocaleString('default', { month: 'long' }));
	let currentYear = $state(shownDate.getFullYear());

	let animationDirection = $state(1); // Für die Animationsrichtung

	const updateCalendar = (date) => {
		currentMonth = date.toLocaleString('default', { month: 'long' });
		currentYear = date.getFullYear();

		const month = date.getMonth();
		const year = date.getFullYear();
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
			tempDays.push({ date: new Date(year, month, i), mark: markedDays[dayKey] });
		}

		return tempDays;
	};

	const changeMonth = (increment) => {
		animationDirection = increment;
		shownDate.setMonth(shownDate.getMonth() + increment);
		days = updateCalendar(shownDate);
	};

	const onDateClick = (date) => {
		currentlySelectedDate = date;
		dateSelected(date);
	};

	onMount(() => {
		days = updateCalendar(shownDate);
	});

	let months = Array.from({ length: 12 }, (_, i) =>
		new Date(2000, i).toLocaleString('default', { month: 'long' })
	);

	let years = Array.from({ length: 201 }, (_, i) => 1900 + i);

	const onMonthSelect = (event) => {
		shownDate.setMonth(months.indexOf(event.target.value));
		days = updateCalendar(shownDate);
	};

	const onYearSelect = (event) => {
		shownDate.setFullYear(parseInt(event.target.value));
		days = updateCalendar(shownDate);
	};

	// weekdays
	const weekDays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
</script>

<div class="datepicker">
	<div class="datepicker-header">
		<button type="button" class="btn btnLeftRight" onclick={() => changeMonth(-1)}>&lt;</button>
		<div class="date-selectors">
			<select value={currentMonth} onchange={onMonthSelect}>
				{#each months as month}
					<option value={month}>{month}</option>
				{/each}
			</select>
			<select value={currentYear} onchange={onYearSelect}>
				{#each years as year}
					<option value={year}>{year}</option>
				{/each}
			</select>
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
				{#each days as day (day ? day.date : Math.random())}
					{#if day}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<div
							in:fly={{ y: 100, duration: 200 }}
							out:fly={{ y: -100, duration: 200 }}
							class="day
								{day.mark?.type === 'background' ? 'mark-background' : ''} 
								{day.mark?.type === 'dot' ? 'mark-dot' : ''} 
								{currentlySelectedDate.toDateString() === day.date.toDateString() ? 'selected' : ''}"
							style="--color: {day.mark?.color || 'transparent'}"
							onclick={() => onDateClick(day.date)}
						>
							{day.date.getDate()}
						</div>
					{:else}
						<div class="day empty-slot"></div>
					{/if}
				{/each}
			</div>
		{/key}
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
		overflow: hidden;
		width: 300px;
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
		min-height: 320px;
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
		background-color: var(--color);
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

	/* Ensure selected state takes precedence */
	.day.mark-background:not(.selected) {
		background-color: var(--color);
		color: white;
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
</style>

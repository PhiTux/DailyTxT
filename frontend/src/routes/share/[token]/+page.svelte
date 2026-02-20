<script>
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { onMount } from 'svelte';
	import { marked } from 'marked';
	import { page } from '$app/state';
	import { untrack } from 'svelte';

	marked.use({
		breaks: true,
		gfm: true
	});

	let token = $derived(page.params.token);

	// Calendar state
	let currentYear = $state(new Date().getFullYear());
	let currentMonth = $state(new Date().getMonth()); // 0-indexed

	let logs = $state([]);
	let isLoadingMonthForReading = $state(false);
	let isInvalidToken = $state(false);

	const monthNames = [
		'January', 'February', 'March', 'April', 'May', 'June',
		'July', 'August', 'September', 'October', 'November', 'December'
	];

	const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

	// Days that have at least one entry this month
	let markedDays = $state([]);

	function prevMonth() {
		if (currentMonth === 0) {
			currentMonth = 11;
			currentYear -= 1;
		} else {
			currentMonth -= 1;
		}
	}

	function nextMonth() {
		if (currentMonth === 11) {
			currentMonth = 0;
			currentYear += 1;
		} else {
			currentMonth += 1;
		}
	}

	function scrollToDay(day) {
		const el = document.querySelector(`.log[data-log-day="${day}"]`);
		if (el) {
			el.scrollIntoView({ behavior: 'smooth', block: 'start' });
		}
	}

	// Re-load whenever month/year changes
	$effect(() => {
		// track both reactive values
		const _year = currentYear;
		const _month = currentMonth;
		const _token = token;
		if (_token) {
			untrack(() => {
				loadMarkedDays(_year, _month);
				loadMonthForReading(_year, _month);
			});
		}
	});

	function loadMarkedDays(year, month) {
		axios
			.get(API_URL + '/share/getMarkedDays', {
				params: { token, year, month: month + 1 }
			})
			.then((response) => {
				markedDays = response.data.days_with_logs || [];
			})
			.catch((error) => {
				if (error.response?.status === 401) {
					isInvalidToken = true;
				}
				console.error(error);
			});
	}

	function loadMonthForReading(year, month) {
		if (isLoadingMonthForReading) return;
		isLoadingMonthForReading = true;
		logs = [];

		axios
			.get(API_URL + '/share/loadMonthForReading', {
				params: { token, year, month: month + 1 }
			})
			.then((response) => {
				logs = response.data.sort((a, b) => a.day - b.day);
			})
			.catch((error) => {
				if (error.response?.status === 401) {
					isInvalidToken = true;
				}
				console.error(error);
			})
			.finally(() => {
				isLoadingMonthForReading = false;
			});
	}

	const imageExtensions = ['jpeg', 'jpg', 'gif', 'png', 'webp', 'bmp'];

	function getImageSrc(uuid) {
		return API_URL + '/share/downloadFile?token=' + encodeURIComponent(token) + '&uuid=' + encodeURIComponent(uuid);
	}

	function downloadFile(uuid, filename) {
		const a = document.createElement('a');
		a.href = API_URL + '/share/downloadFile?token=' + encodeURIComponent(token) + '&uuid=' + encodeURIComponent(uuid);
		a.download = filename || uuid;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
	}

	function isImage(filename) {
		const ext = filename?.split('.').pop()?.toLowerCase();
		return imageExtensions.includes(ext);
	}

	// Build calendar grid for current month
	let calendarDays = $derived.by(() => {
		const year = currentYear;
		const month = currentMonth;
		const firstDow = new Date(year, month, 1).getDay(); // 0=Sun
		const daysInMonth = new Date(year, month + 1, 0).getDate();
		const cells = [];
		for (let i = 0; i < firstDow; i++) cells.push(null);
		for (let d = 1; d <= daysInMonth; d++) cells.push(d);
		return cells;
	});
</script>

<svelte:head>
	<title>DailyTxT – Shared Diary</title>
</svelte:head>

{#if isInvalidToken}
	<div class="d-flex align-items-center justify-content-center h-100">
		<div class="glass p-5 rounded-5 text-center">
			<h3>🔒 Invalid or expired share link</h3>
			<p class="text-muted mt-2">This share link is not valid or has been revoked.</p>
		</div>
	</div>
{:else}
	<div class="share-layout d-flex flex-column h-100">
		<!-- Minimal header -->
		<nav class="navbar glass px-3 py-2 d-flex flex-row align-items-center justify-content-between">
			<div class="d-flex align-items-center gap-2">
				<span class="dailytxt-brand">📖 DailyTxT</span>
			</div>
			<span class="badge bg-secondary">Read Only</span>
		</nav>

		<div class="share-content d-flex flex-row flex-fill overflow-hidden">
			<!-- Sidebar: mini calendar -->
			<div class="share-sidebar p-3 d-none d-md-flex flex-column">
				<div class="calendar-nav d-flex align-items-center justify-content-between mb-2">
					<button class="btn btn-sm btn-outline-secondary" onclick={prevMonth}>‹</button>
					<span class="fw-semibold">{monthNames[currentMonth]} {currentYear}</span>
					<button class="btn btn-sm btn-outline-secondary" onclick={nextMonth}>›</button>
				</div>
				<div class="mini-calendar">
					{#each dayNames as d}
						<div class="cal-header">{d}</div>
					{/each}
					{#each calendarDays as day}
						{#if day === null}
							<div></div>
						{:else}
							<button
								class="cal-day {markedDays.includes(day) ? 'has-entry' : ''}"
								onclick={() => scrollToDay(day)}
								title={markedDays.includes(day) ? 'Has entry' : ''}
							>{day}</button>
						{/if}
					{/each}
				</div>
			</div>

			<!-- Main log area -->
			<div class="flex-fill overflow-y-auto p-3" id="shareScrollArea">
				<!-- Mobile month nav -->
				<div class="d-md-none d-flex align-items-center justify-content-between mb-3">
					<button class="btn btn-sm btn-outline-secondary" onclick={prevMonth}>‹</button>
					<span class="fw-semibold">{monthNames[currentMonth]} {currentYear}</span>
					<button class="btn btn-sm btn-outline-secondary" onclick={nextMonth}>›</button>
				</div>

				{#if isLoadingMonthForReading}
					<div class="d-flex align-items-center justify-content-center h-100">
						<div class="spinner-border" role="status">
							<span class="visually-hidden">Loading…</span>
						</div>
					</div>
				{:else if logs.length === 0}
					<div class="d-flex align-items-center justify-content-center h-100">
						<div class="glass p-5 rounded-5 text-center">
							<span style="font-size:1.4rem;opacity:.7">No entries for this month</span>
						</div>
					</div>
				{:else}
					{#each logs as log (log.day)}
						{#if ('text' in log && log.text !== '') || log.tags?.length > 0 || log.files?.length > 0}
							<div class="log glass mb-3 p-3 d-flex flex-row" data-log-day={log.day}>
								<div class="date me-3 d-flex flex-column align-items-center">
									<p class="dateNumber">{log.day}</p>
									<p class="dateDay">
										<b>
											{new Date(currentYear, currentMonth, log.day).toLocaleDateString(undefined, { weekday: 'long' })}
										</b>
									</p>
									<p class="dateMonthYear">
										<i>{new Date(currentYear, currentMonth, log.day).toLocaleDateString(undefined, { year: 'numeric', month: 'long' })}</i>
									</p>
								</div>
								<div class="logContent flex-grow-1">
									{#if log.text && log.text !== ''}
										<div class="text">
											{@html marked.parse(log.text)}
										</div>
									{/if}
									{#if log.files && log.files.length > 0}
										<div class="mt-2 d-flex flex-column gap-1">
											{#each log.files as file}
												{#if isImage(file.filename)}
													<div class="shared-image-wrapper">
														<img
															src={getImageSrc(file.uuid_filename)}
															alt={file.filename}
															class="shared-image"
															loading="lazy"
														/>
													</div>
												{:else}
													<button
														class="btn btn-sm btn-outline-secondary text-start"
														onclick={() => downloadFile(file.uuid_filename, file.filename)}
													>
														📎 {file.filename}
													</button>
												{/if}
											{/each}
										</div>
									{/if}
								</div>
							</div>
						{/if}
					{/each}
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	:global(body) {
		height: 100vh;
		overflow: hidden;
	}

	.share-layout {
		height: 100vh;
	}

	.dailytxt-brand {
		font-size: 1.2rem;
		font-weight: 600;
	}

	.share-sidebar {
		width: 280px;
		min-width: 280px;
		border-right: 1px solid rgba(128, 128, 128, 0.2);
	}

	.mini-calendar {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: 2px;
	}

	.cal-header {
		text-align: center;
		font-size: 0.7rem;
		font-weight: 600;
		opacity: 0.6;
		padding: 2px 0;
	}

	.cal-day {
		text-align: center;
		font-size: 0.8rem;
		padding: 4px 2px;
		border: none;
		background: transparent;
		border-radius: 4px;
		cursor: pointer;
		color: inherit;
	}

	.cal-day:hover {
		background: rgba(128, 128, 128, 0.2);
	}

	.cal-day.has-entry {
		background: rgba(99, 160, 255, 0.3);
		font-weight: 600;
	}

	.cal-day.has-entry:hover {
		background: rgba(99, 160, 255, 0.5);
	}

	.log {
		border-radius: 15px;
	}

	:global(body[data-bs-theme='dark']) .log {
		box-shadow: 3px 3px 8px 4px rgba(0, 0, 0, 0.3);
	}

	:global(body[data-bs-theme='light']) .log {
		box-shadow: 3px 3px 8px 4px rgba(0, 0, 0, 0.2);
	}

	:global(body[data-bs-theme='dark']) .glass {
		background-color: rgba(68, 68, 68, 0.6) !important;
	}

	:global(body[data-bs-theme='light']) .glass {
		background-color: rgba(122, 122, 122, 0.6) !important;
		color: rgb(19, 19, 19);
	}

	.dateNumber {
		font-size: 3rem;
		font-weight: 600;
		font-style: italic;
		opacity: 0.5;
	}

	.dateDay {
		opacity: 0.7;
		font-size: 1.2rem;
	}

	.text {
		word-wrap: anywhere;
	}

	.logContent {
		width: 100%;
	}

	.shared-image {
		max-width: 100%;
		max-height: 400px;
		border-radius: 8px;
		object-fit: contain;
	}

	.shared-image-wrapper {
		margin-top: 0.5rem;
	}

	#shareScrollArea {
		max-height: calc(100vh - 56px);
	}
</style>

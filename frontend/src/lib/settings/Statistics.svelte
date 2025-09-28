<script>
	import { API_URL } from '$lib/APIurl';
	import { getTranslate, getTolgee } from '@tolgee/svelte';
	import axios from 'axios';
	import { onMount } from 'svelte';
	import Tag from '$lib/Tag.svelte';
	import { tags } from '$lib/tagStore';
	import { get } from 'svelte/store';
	import * as bootstrap from 'bootstrap';
	import { mount } from 'svelte';
	import { goto } from '$app/navigation';
	import { selectedDate } from '$lib/calendarStore.js';
	import { formatBytes } from '$lib/helpers';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	// Raw day stats from backend
	let dayStats = $state([]);

	// Derived years list & selected year
	let years = $state([]);
	let selectedYear = $state(new Date().getFullYear());

	// Heatmap data (weeks -> days)
	let weeks = $state([]);
	let maxWordCountYear = 0;
	let minWordCountYear = 0; // smallest > 0 value
	// Filter stats for selected year
	// Iterate all days of the year
	let legendRanges = $state([]);
	// Bootstrap tooltip support
	let heatmapEl = $state(null);
	let dayMap = new Map(); // key -> day data

	const buildYearData = () => {
		if (!years.includes(selectedYear)) return;

		// Filter stats for selected year
		const yearDays = dayStats.filter((d) => d.year === selectedYear);
		const mapByKey = new Map();
		let localMax = 0;
		let localMin = Infinity;
		for (const d of yearDays) {
			const key = `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`;
			mapByKey.set(key, d);
			if (d.wordCount > localMax) localMax = d.wordCount;
			if (d.wordCount > 0 && d.wordCount < localMin) localMin = d.wordCount;
		}
		maxWordCountYear = localMax;
		minWordCountYear = localMin === Infinity ? 0 : localMin;

		buildLegendRanges();

		// Iterate all days of the year
		const first = new Date(selectedYear, 0, 1);
		const last = new Date(selectedYear, 11, 31);
		// Build sequential weeks starting Monday (GitHub uses Sunday start; we use Monday for EU style)
		// Pre-fill leading empty cells
		// If week complete, push and start a new one
		// Push trailing week if it contains any filled cells
		const thresholds = [0.15, 0.35, 0.65]; // thresholds between intensity levels
		// Map ratio ranges to integer word ranges
		const weekdayIndex = (d) => (d.getDay() + 6) % 7;
		let current = new Date(first);
		weeks = [];
		let currentWeek = new Array(7).fill(null);
		// Pre-fill leading empty cells
		for (let i = 0; i < weekdayIndex(current); i++) {
			currentWeek[i] = { empty: true };
		}
		while (current <= last) {
			const idx = weekdayIndex(current);
			const key = `${current.getFullYear()}-${String(current.getMonth() + 1).padStart(2, '0')}-${String(current.getDate()).padStart(2, '0')}`;
			const stat = mapByKey.get(key);
			const wordCount = stat ? stat.wordCount : 0;
			const isBookmarked = stat ? stat.isBookmarked : false;
			currentWeek[idx] = {
				date: new Date(current),
				wordCount,
				isBookmarked,
				tags: stat ? stat.tags : [],
				fileCount: stat ? stat.fileCount : 0
			};
			// If week complete, push and start new
			if (idx === 6) {
				weeks.push(currentWeek);
				currentWeek = new Array(7).fill(null);
			}
			// Advance to the next day
			current.setDate(current.getDate() + 1);
		}
		// Push trailing week if contains any filled cells
		if (currentWeek.some((c) => c !== null)) weeks.push(currentWeek);

		// Rebuild day map for tooltips
		dayMap.clear();
		for (let wi = 0; wi < weeks.length; wi++) {
			const w = weeks[wi];
			for (let di = 0; di < w.length; di++) {
				const d = w[di];
				if (d && !d.empty) dayMap.set(`${wi}-${di}`, d);
			}
		}

		// Initialize / refresh bootstrap tooltips after DOM update
		setTimeout(initTooltips, 0);
	};

	const thresholds = [0.15, 0.35, 0.65]; // thresholds between levels
	function buildLegendRanges() {
		legendRanges = [];
		if (maxWordCountYear === 0) {
			legendRanges = [
				{ level: 0, from: 0, to: 0 },
				{ level: 1, from: 0, to: 0 },
				{ level: 2, from: 0, to: 0 },
				{ level: 3, from: 0, to: 0 },
				{ level: 4, from: 0, to: 0 }
			];
			return;
		}
		// Map ratio ranges to integer word ranges
		const segments = [
			{ level: 0, rFrom: 0, rTo: 0 },
			{ level: 1, rFrom: 0.0000001, rTo: thresholds[0] },
			{ level: 2, rFrom: thresholds[0], rTo: thresholds[1] },
			{ level: 3, rFrom: thresholds[1], rTo: thresholds[2] },
			{ level: 4, rFrom: thresholds[2], rTo: 1 }
		];
		legendRanges = segments.map((s) => {
			const from =
				s.level === 0 ? 0 : Math.max(minWordCountYear, Math.floor(s.rFrom * maxWordCountYear) || 1);
			let to = Math.floor(s.rTo * maxWordCountYear);
			if (s.level === 4) to = maxWordCountYear; // last covers top
			if (to < from) to = from; // ensure non-inverted
			return { level: s.level, from, to };
		});

		// Fix overlapping ranges by making them exclusive/inclusive properly
		for (let i = 1; i < legendRanges.length; i++) {
			// Make sure current segment doesn't start where previous ends
			if (legendRanges[i].from === legendRanges[i - 1].to && legendRanges[i].from > 0) {
				legendRanges[i].from = legendRanges[i - 1].to + 1;
			}
		}
	}

	const colorLevel = (wc) => {
		if (wc <= 0) return 0;
		if (maxWordCountYear <= 0) return 0;
		const r = wc / maxWordCountYear;
		if (r < thresholds[0]) return 1;
		if (r < thresholds[1]) return 2;
		if (r < thresholds[2]) return 3;
		return 4;
	};

	function selectYear(year) {
		selectedYear = year;
		buildYearData();
	}

	function prevYear() {
		const idx = years.indexOf(selectedYear);
		if (idx > 0) {
			selectedYear = years[idx - 1];
			buildYearData();
		}
	}
	function nextYear() {
		const idx = years.indexOf(selectedYear);
		if (idx !== -1 && idx < years.length - 1) {
			selectedYear = years[idx + 1];
			buildYearData();
		}
	}

	const fmtDate = (d) =>
		d.toLocaleDateString($tolgee.getLanguage(), {
			weekday: 'long',
			year: 'numeric',
			month: '2-digit',
			day: '2-digit'
		});

	let errorOnLoading = $state(false);
	let isLoading = $state(false);
	onMount(async () => {
		try {
			isLoading = true;
			const resp = await axios.get(API_URL + '/users/statistics');
			isLoading = false;
			dayStats = resp.data;
			// Normalize key names to camelCase if backend sent lower-case
			dayStats = dayStats.map((d) => ({
				year: d.year ?? d.Year,
				month: d.month ?? d.Month,
				day: d.day ?? d.Day,
				wordCount: d.wordCount ?? d.WordCount ?? 0,
				fileCount: d.fileCount ?? d.FileCount ?? 0,
				fileSizeBytes: d.fileSizeBytes ?? d.FileSizeBytes ?? 0,
				tags: d.tags ?? d.Tags ?? [],
				isBookmarked: d.isBookmarked ?? d.IsBookmarked ?? false
			}));
			// Collect years
			years = Array.from(new Set(dayStats.map((d) => d.year))).sort((a, b) => a - b);
			if (!years.includes(selectedYear) && years.length) selectedYear = years[years.length - 1];
			buildYearData();
		} catch (e) {
			isLoading = false;
			console.error('Failed loading statistics', e);
			errorOnLoading = true;
		}
	});

	// Build HTML skeleton for popover (tags get injected after show)
	function tooltipHTML(day) {
		const html = `
			<div class="popover-day-content">
				<div class="tt-head"><b>${fmtDate(day.date)}</b></div>
				<div class="tt-line">${$t('settings.statistics.wordCount', { wordCount: day.wordCount })}</div>
				${day.fileCount ? `<div class='tt-line'>${$t('settings.statistics.fileCount', { fileCount: day.fileCount })}</div>` : ''}
				${day.isBookmarked ? `<div class='tt-line'>★ ${$t('settings.statistics.bookmarked')}</div>` : ''}
				<div class="tt-tags"></div>
				<div class="tt-footer">
					<button type="button" class="tt-open-btn" data-year="${day.date.getFullYear()}" data-month="${day.date.getMonth() + 1}" data-day="${day.date.getDate()}">
						📝 ${$t('settings.statistics.open')}
					</button>
				</div>
			</div>`;
		return html;
	}

	// Function to open a specific date in /write
	function openDate(year, month, day) {
		// Set the selected date
		$selectedDate = {
			year: parseInt(year),
			month: parseInt(month),
			day: parseInt(day)
		};

		// Close the settings modal (assuming it's a Bootstrap modal)
		const settingsModal = document.querySelector('#settingsModal');
		if (settingsModal) {
			const modalInstance = bootstrap.Modal.getInstance(settingsModal);
			if (modalInstance) {
				modalInstance.hide();
			}
		}

		// Navigate to write page
		goto('/write');
	}

	function initTooltips() {
		// Dispose previous instances for day cells
		document.querySelectorAll('.day-cell[data-bs-toggle="popover"]').forEach((el) => {
			const inst = bootstrap.Popover.getInstance(el);
			if (inst) inst.dispose();
		});

		// Dispose previous instances for legend cells (keep as tooltips)
		document.querySelectorAll('.legend-cell[data-bs-toggle="tooltip"]').forEach((el) => {
			const inst = bootstrap.Tooltip.getInstance(el);
			if (inst) inst.dispose();
		});

		// Initialize day cell popovers
		const cells = heatmapEl?.querySelectorAll('.day-cell[data-day-key]') || [];
		cells.forEach((el, index) => {
			const key = el.getAttribute('data-day-key');
			const day = dayMap.get(key);
			if (!day) return;

			const htmlContent = tooltipHTML(day);
			el.setAttribute('data-bs-content', htmlContent);
			el.setAttribute('data-bs-toggle', 'popover');

			const popover = new bootstrap.Popover(el, {
				html: true,
				placement: 'top',
				trigger: 'click',
				animation: false,
				container: 'body',
				sanitize: false // Disable sanitization to allow our HTML
			});

			// Close other popovers when this one is shown
			el.addEventListener('show.bs.popover', () => {
				// Close all other popovers
				document.querySelectorAll('.day-cell[data-bs-toggle="popover"]').forEach((otherEl) => {
					if (otherEl !== el) {
						const otherInst = bootstrap.Popover.getInstance(otherEl);
						if (otherInst) otherInst.hide();
					}
				});
			});

			// After popover is shown, mount Tag components and setup button handlers
			const populate = () => {
				const inst = bootstrap.Popover.getInstance(el);
				if (!inst) {
					console.log('No popover instance found');
					return;
				}
				const popoverEl =
					typeof inst.getTipElement === 'function' ? inst.getTipElement() : inst.tip;

				const tagContainer = popoverEl?.querySelector('.tt-tags');
				const openBtn = popoverEl?.querySelector('.tt-open-btn');

				if (!tagContainer || tagContainer.dataset.populated === '1') return;
				// Mark to avoid double work
				tagContainer.dataset.populated = '1';
				const allTagsNow = get(tags) || [];
				(day.tags || []).forEach((tid) => {
					const tagObj = allTagsNow.find((t) => t.id == tid);
					if (tagObj) {
						mount(Tag, { target: tagContainer, props: { tag: tagObj } });
					}
				});

				// Setup button handler
				if (openBtn && !openBtn.dataset.handlerAdded) {
					openBtn.dataset.handlerAdded = '1';
					openBtn.addEventListener('click', (e) => {
						e.preventDefault();
						e.stopPropagation();

						// Get attributes from the clicked element or fallback to currentTarget
						const target = e.target.closest('.tt-open-btn') || e.currentTarget;
						const year = target.getAttribute('data-year');
						const month = target.getAttribute('data-month');
						const day = target.getAttribute('data-day');

						if (year && month && day) {
							// Hide popover first
							const inst = bootstrap.Popover.getInstance(el);
							if (inst) inst.hide();

							openDate(year, month, day);
						}
					});
				}
			};

			// Listen for popover events
			el.addEventListener('inserted.bs.popover', populate);
			el.addEventListener('shown.bs.popover', populate);
		});

		// Initialize legend cell tooltips (keep as simple tooltips)
		const legendCells = document.querySelectorAll('.legend-cell[data-bs-toggle="tooltip"]');
		legendCells.forEach((el) => {
			new bootstrap.Tooltip(el, {
				placement: 'top',
				trigger: 'hover focus'
			});
		});

		// Close popovers when clicking outside
		document.addEventListener(
			'click',
			(e) => {
				// Check if click is outside any day-cell and outside any popover
				if (
					!e.target.closest('.day-cell[data-bs-toggle="popover"]') &&
					!e.target.closest('.popover')
				) {
					document.querySelectorAll('.day-cell[data-bs-toggle="popover"]').forEach((el) => {
						const inst = bootstrap.Popover.getInstance(el);
						if (inst) inst.hide();
					});
				}
			},
			{ capture: true }
		); // Use capture to ensure this runs before other handlers
	}
</script>

<div class="settings-stats">
	<h2 class=" mb-3">{$t('settings.statistics.title')}</h2>

	{#if errorOnLoading}
		<div class="text-center">
			<p class="text-danger">{$t('settings.statistics.error_loading_data')}</p>
		</div>
	{/if}

	{#if isLoading}
		<div class="text-center">
			<div class="spinner-border" role="status">
				<span class="visually-hidden">Loading...</span>
			</div>
			<p class="mt-2 text-muted">{$t('settings.statistics.loading_data')}</p>
		</div>
	{:else if years.length !== 0}
		<div class="year-selector d-flex align-items-center gap-2 mb-3 flex-wrap">
			<button
				class="btn btn-sm btn-secondary nav-button"
				onclick={prevYear}
				disabled={years.indexOf(selectedYear) === 0}
				aria-label="previous year">&lt;</button
			>
			<select
				class="form-select form-select-sm year-dropdown"
				bind:value={selectedYear}
				onchange={(e) => selectYear(+e.target.value)}
			>
				{#each years as y}
					<option value={y}>{y}</option>
				{/each}
			</select>
			<button
				class="btn btn-sm btn-secondary nav-button"
				onclick={nextYear}
				disabled={years.indexOf(selectedYear) === years.length - 1}
				aria-label="next year">&gt;</button
			>
			<div class="legend ms-auto d-flex align-items-center gap-1">
				<span class="legend-label small">{$t('settings.statistics.legend')}</span>
				<div class="legend-colors d-flex align-items-center gap-1">
					{#each legendRanges as seg}
						<span
							class="legend-cell level-{seg.level}"
							data-bs-toggle="tooltip"
							data-bs-title={`${seg.from} – ${seg.to} ${$t('settings.statistics.words')}`}
						></span>
					{/each}
				</div>
			</div>
		</div>

		<div class="heatmap" aria-label="Year-Heatmap" bind:this={heatmapEl}>
			<div class="weeks d-flex">
				{#each weeks as week, wi}
					<div class="week-column d-flex flex-column">
						{#each week as day, di}
							{#if day === null || day.empty}
								<div class="day-cell empty" aria-hidden="true"></div>
							{:else}
								<div
									class="day-cell level-{colorLevel(day.wordCount)}"
									role="button"
									tabindex="0"
									data-bs-toggle="popover"
									data-day-key={`${wi}-${di}`}
								>
									{#if day.isBookmarked}
										<span class="bookmark" aria-label="Bookmarked">★</span>
									{/if}
								</div>
							{/if}
						{/each}
					</div>
				{/each}
			</div>
		</div>

		<h4 class="headerTotal">{$t('settings.statistics.total')}</h4>
		<ul>
			<li>
				{@html $t('settings.statistics.daysWithActivity', {
					days: dayStats.length.toLocaleString($tolgee.getLanguage())
				})}
			</li>
			<li>
				{@html $t('settings.statistics.wordCountTotal', {
					wordCount: dayStats
						.reduce((sum, d) => sum + d.wordCount, 0)
						.toLocaleString($tolgee.getLanguage())
				})}
			</li>
			<li>
				{@html $t('settings.statistics.fileCountWithDiskUsage', {
					fileCount: dayStats
						.reduce((sum, d) => sum + d.fileCount, 0)
						.toLocaleString($tolgee.getLanguage()),
					diskUsage: formatBytes(dayStats.reduce((sum, d) => sum + d.fileSizeBytes, 0))
				})}
			</li>
			<li>
				{@html $t('settings.statistics.bookmarkedDays', {
					days: dayStats.filter((d) => d.isBookmarked).length.toLocaleString($tolgee.getLanguage())
				})}
			</li>
			{#if $tags.length > 0}
				<li>
					{$t('tags.tags')}:<br />
					{#each $tags as tag (tag.id)}
						<span class="d-inline-block me-3 mb-2">
							<Tag {tag} />

							{@html $t('settings.statistics.tagUsedCount', {
								count: dayStats
									.filter((d) => d.tags.includes(tag.id))
									.length.toLocaleString($tolgee.getLanguage())
							})}
						</span>
						<br />
					{/each}
				</li>
			{/if}
		</ul>

		<h4 class="headerTotal">{$t('settings.statistics.funFacts')}</h4>
		<ul>
			<li>
				📊 {$t('settings.statistics.averageWordsPerLog', {
					wordCount: Math.round(
						dayStats.reduce((sum, d) => sum + d.wordCount, 0) /
							dayStats.filter((d) => d.wordCount > 0).length || 0
					)
				})}
			</li>
			<li>
				🏆 {$t('settings.statistics.mostProductiveDay')}: {(() => {
					const best = dayStats.reduce((max, d) => (d.wordCount > max.wordCount ? d : max), {
						wordCount: 0
					});
					if (best.wordCount > 0) {
						const date = new Date(best.year, best.month - 1, best.day);
						const formattedDate = date.toLocaleDateString($tolgee.getLanguage());
						return `${formattedDate} (${$t('settings.statistics.wordCount', { wordCount: best.wordCount })})`;
					}
					return '🤷‍♂️';
				})()}
			</li>
			<li>
				🗓️ {$t('settings.statistics.longestWritingStreak')}: {(() => {
					if (dayStats.length === 0) return '0 Tage';

					let maxStreak = 0;
					let currentStreak = 0;
					let maxStreakStart = null;
					let maxStreakEnd = null;
					let currentStreakStart = null;

					const sortedDays = [...dayStats].sort(
						(a, b) => new Date(a.year, a.month - 1, a.day) - new Date(b.year, b.month - 1, b.day)
					);

					for (let i = 0; i < sortedDays.length; i++) {
						const currentDay = sortedDays[i];
						const prevDay = i > 0 ? sortedDays[i - 1] : null;

						// Check if current day is consecutive to previous day
						let isConsecutive = false;
						if (prevDay) {
							const currentDate = new Date(currentDay.year, currentDay.month - 1, currentDay.day);
							const prevDate = new Date(prevDay.year, prevDay.month - 1, prevDay.day);
							const dayDiff = Math.floor((currentDate - prevDate) / (1000 * 60 * 60 * 24));
							isConsecutive = dayDiff === 1;
						}

						if (isConsecutive) {
							currentStreak++;
						} else {
							currentStreak = 1;
							currentStreakStart = currentDay;
						}

						// Update max streak if current is longer
						if (currentStreak > maxStreak) {
							maxStreak = currentStreak;
							maxStreakStart = currentStreakStart;
							maxStreakEnd = currentDay;
						}
					}

					if (maxStreak > 1 && maxStreakStart && maxStreakEnd) {
						const startDate = new Date(
							maxStreakStart.year,
							maxStreakStart.month - 1,
							maxStreakStart.day
						);
						const endDate = new Date(maxStreakEnd.year, maxStreakEnd.month - 1, maxStreakEnd.day);
						const formattedStart = startDate.toLocaleDateString($tolgee.getLanguage());
						const formattedEnd = endDate.toLocaleDateString($tolgee.getLanguage());
						return `${$t('settings.statistics.dayCount', { dayCount: maxStreak })} (${formattedStart} - ${formattedEnd})`;
					}

					return $t('settings.statistics.dayCount', { dayCount: maxStreak });
				})()}
			</li>
			<li>
				🌍 {$t('settings.statistics.favoriteWritingDay')}: {(() => {
					const weekdays = [
						$t('weekdays.sunday'),
						$t('weekdays.monday'),
						$t('weekdays.tuesday'),
						$t('weekdays.wednesday'),
						$t('weekdays.thursday'),
						$t('weekdays.friday'),
						$t('weekdays.saturday')
					];
					const dayCount = new Array(7).fill(0);
					dayStats.forEach((d) => {
						const date = new Date(d.year, d.month - 1, d.day);
						dayCount[date.getDay()]++;
					});
					const maxIndex = dayCount.indexOf(Math.max(...dayCount));
					return dayCount[maxIndex] > 0 ? weekdays[maxIndex] : '🤷‍♂️';
				})()}
			</li>
			<li>
				🎯 {(() => {
					if (dayStats.length === 0) return '0%';
					const sortedDays = [...dayStats].sort(
						(a, b) => new Date(a.year, a.month - 1, a.day) - new Date(b.year, b.month - 1, b.day)
					);
					const firstDate = new Date(
						sortedDays[0].year,
						sortedDays[0].month - 1,
						sortedDays[0].day
					);
					const today = new Date();
					const daysSinceFirst = Math.floor((today - firstDate) / (1000 * 60 * 60 * 24)) + 1;
					const activityRate = Math.round((dayStats.length / daysSinceFirst) * 100);
					return $t('settings.statistics.activityRate', { percent: activityRate });
				})()}
			</li>
			<li>
				📖 {$t('settings.statistics.bookpages', {
					pages: Math.round(dayStats.reduce((sum, d) => sum + d.wordCount, 0) / 300)
				})}
			</li>
		</ul>
	{:else if years.length === 0}
		<p class="text-info">{$t('settings.statistics.no_data')}</p>
	{/if}
</div>

<style>
	:global(body[data-bs-theme='dark']) .nav-button {
		color: #bebebe;
	}

	.nav-button:disabled {
		opacity: 0.5;
	}

	.headerTotal {
		margin-top: 1rem;
		margin-bottom: 0.5rem;
	}

	.settings-stats {
		min-height: 65vh;
	}

	.year-selector .year-dropdown {
		width: auto;
	}
	.heatmap {
		overflow-x: auto;
		position: relative; /* ensure tooltip absolute coords are relative to heatmap */
		border: 1px solid var(--bs-border-color, #ddd);
		padding: 0.5rem;
		border-radius: 0.5rem;
		background: var(--bs-body-bg, #fff);
	}
	.week-column {
		gap: 3px;
	}
	.weeks {
		gap: 3px;
	}
	.day-cell {
		width: 14px;
		height: 14px;
		border-radius: 3px;
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.day-cell.empty {
		background: transparent;
	}
	/* Color scale (adjust to theme) */

	:global(body[data-bs-theme='light']) .level-0 {
		background: var(--heatmap-empty, #ebedf0);
	}

	:global(body[data-bs-theme='dark']) .level-0 {
		background: var(--heatmap-empty, #333333);
	}

	.level-1 {
		background: #c6e48b;
	}
	.level-2 {
		background: #7bc96f;
	}
	.level-3 {
		background: #239a3b;
	}
	.level-4 {
		background: #196127;
	}
	.day-cell:hover {
		outline: 1px solid #333;
		z-index: 2;
	}
	.legend-cell {
		width: 14px;
		height: 14px;
		border-radius: 3px;
		display: inline-block;
	}
	.legend-cell.level-0 {
		background: var(--heatmap-empty, #ebedf0);
	}
	.legend-cell.level-1 {
		background: #c6e48b;
	}
	.legend-cell.level-2 {
		background: #7bc96f;
	}
	.legend-cell.level-3 {
		background: #239a3b;
	}
	.legend-cell.level-4 {
		background: #196127;
	}
	.bookmark {
		font-size: 10px;
		line-height: 1;
		color: #000;
		text-shadow: none;
	}
	:global(body[data-bs-theme='dark']) .day-cell.level-0 .bookmark {
		color: #bbb;
	}
	:global(body[data-bs-theme='light']) .day-cell.level-0 .bookmark {
		color: #555;
	}
	:global(body[data-bs-theme='light']) .day-cell.level-4 .bookmark {
		color: #dddddd;
	}

	/* Popover styling (applies inside Bootstrap popover) */
	:global(.popover-day-content) {
		min-width: 200px;
	}
	:global(.popover-day-content .tt-head) {
		margin-bottom: 4px;
		font-size: 14px;
	}
	:global(.popover-day-content .tt-tags) {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}
	:global(.popover-day-content .tt-tags .badge) {
		font-size: 12px;
		margin-right: 4px;
	}
	:global(.popover-day-content .tt-footer) {
		text-align: center;
		padding-top: 8px;
	}
	:global(.popover-day-content .tt-open-btn) {
		font-size: 12px !important;
		padding: 6px 12px !important;
		background: #007bff;
		color: white;
		border-radius: 4px;
		cursor: pointer;
		border: none;
		transition: background-color 0.2s;
	}
	:global(.popover-day-content .tt-open-btn:hover) {
		background: #0056b3;
		color: white;
	}

	/* Old tooltip styling (now removed as we use popovers) */

	/* Desktop: Add pointer cursor for day cells */
	@media (pointer: fine) {
		.day-cell[data-day-key] {
			cursor: pointer;
		}
	}
	@media (max-width: 600px) {
		.day-cell,
		.legend-cell {
			width: 11px;
			height: 11px;
		}
	}
</style>

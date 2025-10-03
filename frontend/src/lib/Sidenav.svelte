<script>
	import Datepicker from './Datepicker.svelte';
	import { searchString, searchTag, searchResults, isSearching } from '$lib/searchStore.js';
	import { selectedDate } from '$lib/calendarStore.js';
	import { tags } from '$lib/tagStore.js';
	import { faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';
	import { onMount } from 'svelte';
	import * as bootstrap from 'bootstrap';
	import Tag from './Tag.svelte';
	import { isAuthenticated, offcanvasIsOpen, sameDate } from '$lib/helpers.js';
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { cal } from '$lib/calendarStore.js';
	import { getTranslate, getTolgee } from '@tolgee/svelte';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	let oc;

	onMount(() => {
		const popoverTriggerList = document.querySelectorAll('[data-bs-toggle="popover"]');
		const popoverList = [...popoverTriggerList].map(
			(popoverTriggerEl) =>
				new bootstrap.Popover(popoverTriggerEl, { html: true, trigger: 'focus' })
		);

		oc = document.querySelector('.offcanvas');
		oc.addEventListener('hidden.bs.offcanvas', () => {
			$offcanvasIsOpen = false;
		});
		oc.addEventListener('shown.bs.offcanvas', () => {
			$offcanvasIsOpen = true;
		});
	});

	let searchInput = $state(null);
	let ctrlPressed = false;
	function on_key_down(event) {
		if (event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = true;
		}
		if (event.key === 'f' && ctrlPressed) {
			event.preventDefault();
			$searchTag = {};
			setTimeout(() => {
				searchInput.focus();
			}, 100);
		}
	}

	function on_key_up(event) {
		if (event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = false;
		}
	}

	$effect(() => {
		if ($isAuthenticated && window.location.href) {
			setTimeout(() => {
				oc = document.querySelector('.offcanvas');
				oc.addEventListener('hidden.bs.offcanvas', () => {
					$offcanvasIsOpen = false;
				});
				oc.addEventListener('shown.bs.offcanvas', () => {
					$offcanvasIsOpen = true;
				});
			}, 1000);
		}
	});

	function searchForString() {
		if ($isSearching) {
			return;
		}
		$isSearching = true;

		axios
			.get(API_URL + '/logs/searchString', {
				params: {
					searchString: $searchString
				}
			})
			.then((response) => {
				$searchResults = [...response.data];
				$isSearching = false;
			})
			.catch((error) => {
				$searchResults = [];
				console.error(error.response.data);
				$isSearching = false;

				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSearching'));
				toast.show();
			});
	}

	function searchForTag() {
		$searchString = '';
		if ($isSearching) {
			return;
		}
		$isSearching = true;

		axios
			.get(API_URL + '/logs/searchTag', { params: { tag_id: $searchTag.id } })
			.then((response) => {
				$searchResults = [...response.data];
				$isSearching = false;
			})
			.catch((error) => {
				$isSearching = false;
				$searchResults = [];

				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorSearching'));
				toast.show();
			});
	}

	let showTagDropdown = $state(false);
	let filteredTags = $state([]);
	let selectedTagIndex = $state(0);
	// Touch-Geräte Erkennung für alternative Tag-Auswahl
	let isTouchDevice = $state(false);
	onMount(() => {
		try {
			const ua = navigator.userAgent || '';
			const platform = navigator.platform || '';
			const maxTP = navigator.maxTouchPoints || 0;
			const coarse = window.matchMedia ? window.matchMedia('(pointer: coarse)').matches : false;
			const iPadLike = /iPad/.test(ua) || (/Mac/.test(platform) && maxTP > 1);
			isTouchDevice = maxTP > 0 || coarse || iPadLike || 'ontouchstart' in window;
		} catch (e) {
			isTouchDevice = false;
		}
	});

	function handleKeyDown(event) {
		if (!showTagDropdown && event.key === 'Enter') {
			searchForString();
			return;
		}
		if (filteredTags.length === 0) return;

		switch (event.key) {
			case 'ArrowDown':
				event.preventDefault(); // Prevent cursor movement
				selectedTagIndex = Math.min(selectedTagIndex + 1, filteredTags.length - 1);
				ensureSelectedVisible();
				break;

			case 'ArrowUp':
				event.preventDefault(); // Prevent cursor movement
				selectedTagIndex = Math.max(selectedTagIndex - 1, 0);
				ensureSelectedVisible();
				break;

			case 'Enter':
				if (selectedTagIndex >= 0 && selectedTagIndex < filteredTags.length) {
					event.preventDefault();
					selectSearchTag(filteredTags[selectedTagIndex].id);
				}
				document.activeElement.blur();
				break;

			case 'Escape':
				showTagDropdown = false;
				break;
		}
	}

	function ensureSelectedVisible() {
		setTimeout(() => {
			for (let i = 0; i < 2; i++) {
				const dropdown = document.querySelectorAll('.searchTagDropdown')[i];
				const selectedElement = dropdown?.querySelector('.searchTag-item.selected');

				if (dropdown && selectedElement) {
					const dropdownRect = dropdown.getBoundingClientRect();
					const selectedRect = selectedElement.getBoundingClientRect();

					if (selectedRect.top < dropdownRect.top) {
						dropdown.scrollTop -= dropdownRect.top - selectedRect.top;
					} else if (selectedRect.bottom > dropdownRect.bottom) {
						dropdown.scrollTop += selectedRect.bottom - dropdownRect.bottom;
					}
				}
			}
		}, 40);
	}

	$effect(() => {
		let search = $searchString;
		if (search.startsWith('#')) {
			search = search.slice(1);
			showTagDropdown = true;
			filteredTags = $tags.filter((tag) => tag.name.toLowerCase().includes(search.toLowerCase()));
		} else {
			showTagDropdown = false;
		}
	});

	//let searchTag = $state({});
	function selectSearchTag(tagId) {
		showTagDropdown = false;
		const tag = $tags.find((tag) => tag.id === tagId);
		if (!tag) {
			return;
		}
		$searchTag = tag;
		//$searchResults = [];

		searchForTag();
	}

	function removeSearchTag() {
		$searchTag = {};
		$searchResults = [];
	}

	// selects a search result
	function selectDate(date) {
		$selectedDate = date;

		// close offcanvas/sidenav if open
		if (oc) {
			const bsOffcanvas = bootstrap.Offcanvas.getInstance(oc);
			if ($offcanvasIsOpen) {
				bsOffcanvas.hide();
			}
		}
	}

	function bookmarkDay() {
		axios
			.get(API_URL + '/logs/bookmarkDay', {
				params: {
					year: $selectedDate.year,
					month: $selectedDate.month,
					day: $selectedDate.day
				}
			})
			.then((response) => {
				if (response.data.success) {
					if (response.data.bookmarked) {
						$cal.daysBookmarked = [...$cal.daysBookmarked, $selectedDate.day];
					} else {
						$cal.daysBookmarked = $cal.daysBookmarked.filter((day) => day !== $selectedDate.day);
					}
				} else {
					console.log('Error highlighting day:', response.data);
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorHighlighting'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error(error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorHighlighting'));
				toast.show();
			});
	}
</script>

<svelte:window onkeydown={on_key_down} onkeyup={on_key_up} />

<div class="d-flex flex-column h-100">
	<Datepicker {bookmarkDay} />
	<br />

	<div class="search d-flex flex-column">
		<form onsubmit={searchForString} class="input-group">
			<button
				class="btnSearchPopover btn btn-outline-secondary glass"
				type="button"
				data-bs-toggle="popover"
				data-bs-title="Suche"
				data-bs-content={$t('search.description')}
				tabindex="0"
				aria-label={$t('search.description')}><Fa icon={faQuestionCircle} /></button
			>
			{#if $searchTag.id}
				<!-- If a tag is selected ... -->
				<div class="ms-1 align-content-center">
					<Tag tag={$searchTag} removeTag={removeSearchTag} isRemovable />
				</div>
				{#if $isSearching}
					<div class="ms-2 align-content-center">
						<div class="spinner-border spinner-border-sm" role="status">
							<span class="visually-hidden">Loading...</span>
						</div>
					</div>
				{/if}
			{:else}
				<input
					bind:value={$searchString}
					bind:this={searchInput}
					id="search-input"
					type="text"
					class="form-control"
					placeholder="Suche"
					aria-label={$t('search.search')}
					aria-describedby="search-button"
					onkeydown={handleKeyDown}
					autocomplete="off"
					onfocus={() => {
						selectedTagIndex = 0;
						if ($searchString.startsWith('#')) {
							showTagDropdown = true;
						}
					}}
					onfocusout={() => {
						setTimeout(() => (showTagDropdown = false), 150);
					}}
				/>
				<button class="btn btn-outline-secondary glass" type="submit" id="search-button">
					{#if $isSearching}
						<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
					{:else}
						{$t('search.search')}
					{/if}
				</button>
			{/if}
		</form>
		{#if showTagDropdown}
			{#if isTouchDevice}
				<div class="touch-search-tag-panel glass">
					{#if filteredTags.length === 0}
						<em style="padding: 0.2rem;">{$t('tags.no_tags_found')}</em>
					{:else}
						<div class="touch-tag-grid gap-1">
							{#each filteredTags as tag (tag.id)}
								<button
									type="button"
									class="touch-tag-item btn btn-sm"
									onclick={() => selectSearchTag(tag.id)}
								>
									<Tag {tag} />
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{:else}
				<div class="searchTagDropdown glass">
					{#if filteredTags.length === 0}
						<em style="padding: 0.2rem;">
							{$t('tags.no_tags_found')}
						</em>
					{:else}
						{#each filteredTags as tag, index (tag.id)}
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<!-- svelte-ignore a11y_mouse_events_have_key_events -->
							<div
								role="button"
								tabindex="0"
								onclick={() => selectSearchTag(tag.id)}
								onmouseover={() => (selectedTagIndex = index)}
								class="searchTag-item {index === selectedTagIndex ? 'selected' : ''}"
							>
								<Tag {tag} />
							</div>
						{/each}
					{/if}
				</div>
			{/if}
		{/if}
		<div class="list-group flex-grow-1 mb-2 glass">
			{#if $searchResults.length > 0}
				{#each $searchResults as result}
					<button
						type="button"
						onclick={() => {
							selectDate({
								year: parseInt(result.year),
								month: parseInt(result.month),
								day: result.day
							});
						}}
						class="list-group-item list-group-item-action {sameDate($selectedDate, {
							year: parseInt(result.year),
							month: parseInt(result.month),
							day: result.day
						})
							? 'active'
							: ''}"
					>
						<div class="search-result-content">
							<div class="date">
								{new Date(result.year, result.month - 1, result.day).toLocaleDateString(
									$tolgee.getLanguage(),
									{
										day: '2-digit',
										month: '2-digit',
										year: 'numeric'
									}
								)}
							</div>
							<!-- <div class="search-separator"></div> -->
							<div class="text">
								{@html result.text}
							</div>
						</div>
					</button>
				{/each}
			{:else}
				<span class="noResult">
					{$t('search.no_results')}
				</span>
			{/if}
		</div>
	</div>
</div>

<div class="toast-container position-fixed bottom-0 end-0 p-3">
	<div
		id="toastErrorSearching"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('search.toast.error')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>

	<div
		id="toastErrorBookmarking"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('calendar.toast.error_bookmarking')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>
</div>

<style>
	.btnSearchPopover {
		border-bottom-left-radius: 0px;
	}

	:global(.datepicker) {
		margin-bottom: 3rem;
	}

	@media (max-height: 800px) {
		:global(.datepicker) {
			margin-bottom: 1rem;
		}
	}

	.noResult {
		font-size: 25pt;
		font-weight: 750;
		text-align: center;
		padding: 1rem;
		user-select: none;
	}

	:global(body[data-bs-theme='dark']) .noResult {
		color: #757575;
	}

	:global(body[data-bs-theme='light']) .noResult {
		color: #cccccc;
	}

	:global(.popover-body > span) {
		font-family: monospace;
		border: 1px solid #ccc;
		border-radius: 3px;
		padding: 0 5px;
		background-color: #eee;
	}

	:global(.popover-body) {
		overflow-y: auto;
		max-height: 80vh;
	}

	.searchTagDropdown {
		position: absolute;
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
		backdrop-filter: blur(10px) saturate(150%);
		z-index: 1000;
		left: 74px;
		margin-top: 38px;
		max-height: 200px;
		overflow-y: auto;
		overflow-x: hidden;
		display: flex;
		flex-direction: column;
		border-bottom-left-radius: 10px;
		border-bottom-right-radius: 10px;
	}

	.touch-search-tag-panel {
		padding: 0.5rem 0.6rem 0.6rem;
		max-height: 35vh;
		background: rgba(255, 255, 255, 0.08);
		backdrop-filter: blur(6px);
		border: 1px solid rgba(255, 255, 255, 0.15);
	}

	.touch-tag-grid {
		display: flex;
		flex-wrap: wrap;
	}

	.touch-tag-item {
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	.touch-tag-item:active {
		transform: scale(0.96);
	}

	@media (max-width: 1599px) {
		.searchTagDropdown {
			left: 58px;
		}
	}

	:global(body[data-bs-theme='dark']) .searchTagDropdown {
		background-color: rgba(87, 87, 87, 0.5);
	}

	:global(body[data-bs-theme='light']) .searchTagDropdown {
		background-color: rgba(196, 196, 196, 0.5);
	}

	:global(body[data-bs-theme='dark']) .searchTag-item.selected {
		background-color: #5f5f5f;
	}

	:global(body[data-bs-theme='light']) .searchTag-item.selected {
		background-color: #b9b9b9;
	}

	.searchTag-item {
		cursor: pointer;
		padding: 5px;
	}

	.search-result-content {
		display: flex;
		align-items: center;
	}

	.date {
		text-align: left;
	}

	.text {
		flex-grow: 1;
		word-wrap: break-word;
		border-left: 1px solid #68a1da;
		margin-left: 1rem;
		padding-left: 1rem;
	}

	#search-input {
		border-bottom-left-radius: 0;
	}

	#search-button {
		border-bottom-right-radius: 0;
	}

	/* Dynamic search results panel: fill remaining space, but never below 250px */
	.search {
		flex: 1 1 auto; /* allow search area to grow */
		min-height: 0; /* allow inner flex children to compute height */
		display: flex;
		flex-direction: column;
	}

	.list-group {
		border-top-left-radius: 0;
		border-top-right-radius: 0;
		overflow-y: auto;
		min-height: 250px; /* minimum requirement */
		flex: 1 1 auto; /* take all remaining vertical space */
		max-height: none; /* remove hard cap */
	}

	.input-group {
		height: auto !important;
	}
</style>

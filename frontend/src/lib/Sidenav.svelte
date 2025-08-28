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
	import { offcanvasIsOpen, sameDate } from '$lib/helpers.js';
	import { API_URL } from '$lib/APIurl.js';
	import axios from 'axios';
	import { cal } from '$lib/calendarStore.js';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

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
		if (window.location.href) {
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
				console.error(error);
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
				class="btnSearchPopover btn btn-outline-secondary glassLight"
				data-bs-toggle="popover"
				data-bs-title="Suche"
				data-bs-content={$t('search.description')}
				onclick={(event) => event.preventDefault()}><Fa icon={faQuestionCircle} /></button
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
				<button class="btn btn-outline-secondary glassLight" type="submit" id="search-button">
					{#if $isSearching}
						<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
					{:else}
						{$t('search.search')}
					{/if}
				</button>
			{/if}
		</form>
		{#if showTagDropdown}
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
		<div class="list-group flex-grow-1 mb-2 glassLight">
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
								{new Date(result.year, result.month - 1, result.day).toLocaleDateString('locale', {
									day: '2-digit',
									month: '2-digit',
									year: 'numeric'
								})}
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

	.noResult {
		font-size: 25pt;
		font-weight: 750;
		color: #ccc;
		text-align: center;
		padding: 1rem;
		user-select: none;
	}

	:global(.popover-body > span) {
		font-family: monospace;
		border: 1px solid #ccc;
		border-radius: 3px;
		padding: 0 5px;
		background-color: #eee;
	}

	.searchTagDropdown {
		position: absolute;
		/* background-color: white; */
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
		z-index: 1000;
		left: 60px;
		margin-top: 38px;
		max-height: 200px;
		overflow-y: auto;
		overflow-x: hidden;
		display: flex;
		flex-direction: column;
	}

	.searchTag-item.selected {
		background-color: #b2b4b6;
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

	.list-group {
		border-top-left-radius: 0;
		border-top-right-radius: 0;
		overflow-y: auto;
		min-height: 250px;
		/* backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
		border: 1px solid #ececec77; */
	}

	.input-group {
		height: auto !important;
	}
</style>

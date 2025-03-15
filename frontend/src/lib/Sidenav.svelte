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

	let { searchForString, searchForTag } = $props();

	onMount(() => {
		const popoverTriggerList = document.querySelectorAll('[data-bs-toggle="popover"]');
		const popoverList = [...popoverTriggerList].map(
			(popoverTriggerEl) =>
				new bootstrap.Popover(popoverTriggerEl, { html: true, trigger: 'focus' })
		);
	});

	let searchInput;
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

	let showTagDropdown = $state(false);
	let filteredTags = $state([]);
	let selectedTagIndex = $state(0);

	function handleKeyDown(event) {
		if (!showTagDropdown && event.key === 'Enter') searchForString();
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
		const tag = $tags.find((tag) => tag.id === tagId);
		if (!tag) {
			return;
		}
		$searchTag = tag;
		//$searchResults = [];
		showTagDropdown = false;

		searchForTag();
	}

	function removeSearchTag() {
		$searchTag = {};
		$searchResults = [];
	}
</script>

<svelte:window onkeydown={on_key_down} onkeyup={on_key_up} />

<div class="d-flex flex-column h-100">
	<Datepicker />
	<br />

	<div class="search">
		<form onsubmit={searchForString} class="input-group mt-5">
			<button
				class="btn btn-outline-secondary"
				data-bs-toggle="popover"
				data-bs-title="Suche"
				data-bs-content="Hier kannst "
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
					aria-label="Suche"
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
				<button class="btn btn-outline-secondary" type="submit" id="search-button">
					{#if $isSearching}
						<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
					{:else}
						Suche
					{/if}
				</button>
			{/if}
		</form>
		{#if showTagDropdown}
			<div class="searchTagDropdown">
				{#if filteredTags.length === 0}
					<em style="padding: 0.2rem;">Kein Tag gefunden...</em>
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
		<div class="list-group flex-grow-1 mb-2">
			{#each $searchResults as result}
				<button
					type="button"
					onclick={() => {
						$selectedDate = new Date(Date.UTC(result.year, result.month - 1, result.day));
					}}
					class="list-group-item list-group-item-action {$selectedDate.toDateString() ===
					new Date(Date.UTC(result.year, result.month - 1, result.day)).toDateString()
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
		</div>
	</div>
</div>

<style>
	.searchTagDropdown {
		position: absolute;
		background-color: white;
		box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
		z-index: 1000;
		left: 60px;
		max-height: 150px;
		overflow-y: scroll;
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
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
		border: 1px solid #ececec77;
	}

	.input-group {
		height: auto !important;
	}
</style>

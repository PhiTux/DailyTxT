<script>
	import Datepicker from './Datepicker.svelte';
	import { searchString, searchResults } from '$lib/searchStore.js';
	import { selectedDate } from '$lib/calendarStore.js';

	export let search = () => {};

	let searchInput;
	let ctrlPressed = false;
	function on_key_down(event) {
		if (event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = true;
		}
		if (event.key === 'f' && ctrlPressed) {
			event.preventDefault();
			searchInput.focus();
		}
	}

	function on_key_up(event) {
		if (event.key === 'Control') {
			event.preventDefault();
			ctrlPressed = false;
		}
	}
</script>

<svelte:window onkeydown={on_key_down} onkeyup={on_key_up} />

<div class="d-flex flex-column h-100">
	<Datepicker />
	<br />

	<form onsubmit={search} class="input-group mt-5">
		<input
			bind:value={$searchString}
			bind:this={searchInput}
			id="search-input"
			type="text"
			class="form-control"
			placeholder="Suche"
			aria-label="Suche"
			aria-describedby="search-button"
		/>
		<button class="btn btn-outline-secondary" type="submit" id="search-button">Suche</button>
	</form>
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

<style>
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

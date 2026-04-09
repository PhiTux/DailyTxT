<script>
	import {
		faPencil,
		faBars,
		faTrash,
		faLocationCrosshairs
	} from '@fortawesome/free-solid-svg-icons';
	import { onMount } from 'svelte';
	import Fa from 'svelte-fa';
	import { API_URL } from '$lib/APIurl.js';
	import { selectedDate } from '$lib/calendarStore.js';
	import axios from 'axios';
	import { Tooltip } from 'bootstrap';

	let { text = '', id = null, deletePin = () => {}, movePin = () => {} } = $props();
	let isEditing = $state(false);

	let editedText = $state('');

	$effect(() => {
		if (!isEditing) {
			editedText = text || '';
		}
	});

	export function resetEditing() {
		isEditing = false;
		editedText = text || '';
	}

	onMount(() => {
		resetEditing();
		const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
		[...tooltipTriggerList].map((tooltipTriggerEl) => new Tooltip(tooltipTriggerEl));
	});

	function startEditing(event) {
		event.preventDefault();
		event.stopPropagation();
		isEditing = true;
		editedText = text || '';

		setTimeout(() => {
			document.querySelector('#editTextInput')?.focus();
		}, 50);
	}

	function cancelEdit(event) {
		event.preventDefault();
		event.stopPropagation();
		editedText = text || '';
		isEditing = false;
	}

	let confirmDelete = $state(false);
	function confirmDeletePin(event) {
		event.preventDefault();
		event.stopPropagation();
		confirmDelete = true;
	}

	function confirmDeletePinAbort(event) {
		event.preventDefault();
		event.stopPropagation();
		confirmDelete = false;
	}

	let isUpdatingText = $state(false);
	/**
	 * Makes an API call to update the text of a pin and updates the local state accordingly
	 */
	function updatePinText() {
		isUpdatingText = true;
		axios
			.post(`${API_URL}/logs/updatePinText`, {
				pinId: id,
				text: editedText,
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year
			})
			.then((response) => {
				if (response.data.success) {
					text = editedText;
				} else {
					console.error('Failed to update pin:', response.data.message);
				}
			})
			.catch((error) => {
				console.error('Error updating pin:', error);
			})
			.finally(() => {
				isEditing = false;
				isUpdatingText = false;
			});
	}
</script>

<div class="saved-pin-popup">
	{#if isEditing}
		<div class="input-group">
			<input
				type="text"
				class="form-control form-control-sm"
				id="editTextInput"
				bind:value={editedText}
				onkeydown={(event) => {
					if (event.key === 'Enter') {
						updatePinText();
					} else if (event.key === 'Escape') {
						cancelEdit(event);
					}
				}}
			/>
			<button type="button" class="btn btn-success" onclick={updatePinText}>
				{#if !isUpdatingText}
					✓
				{:else}
					<div class="spinner-border spinner-border-sm" role="status">
						<span class="visually-hidden">Loading...</span>
					</div>
				{/if}</button
			>
			<button type="button" class="btn btn-danger" onclick={cancelEdit}>✖</button>
		</div>
	{:else if confirmDelete}
		<div class="d-flex flex-column align-items-center gap-2">
			<div>Möchtest du diesen Pin wirklich löschen?</div>
			<div>
				<button class="btn btn-danger btn-sm me-2" onclick={deletePin}>Ja</button>
				<button class="btn btn-secondary btn-sm" onclick={confirmDeletePinAbort}>Nein</button>
			</div>
		</div>
	{:else}
		<div class="saved-pin-view">
			<div class="saved-pin-text">
				{#if text !== ''}
					{text}
				{:else}
					<em class="no-description">Keine Beschreibung</em>
				{/if}
			</div>
			<div class="dropdown">
				<button
					class="btn btn-sm btn-secondary dropdown-toggle float-end"
					type="button"
					data-bs-toggle="dropdown"
					aria-expanded="false"
				>
					<Fa icon={faBars} />
				</button>
				<ul class="dropdown-menu dropdown-menu-end">
					<li>
						<button
							class="dropdown-item btn btn-primary"
							onclick={startEditing}
							data-bs-toggle="tooltip"
							data-bs-placement="left"
							data-bs-delay="500"
							title="Text bearbeiten"
						>
							<Fa icon={faPencil} fw />
						</button>
					</li>
					<li>
						<button
							class="dropdown-item btn btn-primary"
							onclick={movePin}
							data-bs-toggle="tooltip"
							data-bs-placement="left"
							data-bs-delay="500"
							title="Verschieben"
						>
							<Fa icon={faLocationCrosshairs} fw />
						</button>
					</li>
					<li>
						<button
							class="dropdown-item btn btn-danger"
							onclick={confirmDeletePin}
							data-bs-toggle="tooltip"
							data-bs-placement="left"
							data-bs-delay="500"
							title="Pin löschen"
						>
							<Fa icon={faTrash} fw />
						</button>
					</li>
				</ul>
			</div>
		</div>
	{/if}
</div>

<style>
	.no-description {
		font-size: 0.9em;
	}

	.dropdown-menu {
		min-width: 0 !important;
	}

	.saved-pin-text {
		white-space: pre-wrap;
		word-break: break-word;
		line-height: 1.35;
	}

	.saved-pin-popup {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		min-width: 190px;
	}

	.saved-pin-view {
		display: flex;
		flex-direction: row;
		align-items: center;
		justify-content: space-between;
		gap: 0.45rem;
	}
</style>

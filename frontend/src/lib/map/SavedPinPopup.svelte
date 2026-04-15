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
	import * as bootstrap from 'bootstrap';

	let {
		text = $bindable(''),
		id = null,
		deletePin = () => {},
		movePin = () => {},
		openPreview = () => {},
		day = null,
		month = null,
		year = null,
		language,
		translate = (key) => key,
		readingMode = false
	} = $props();

	function tr(key) {
		return typeof translate === 'function' ? translate(key) : key;
	}

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
		console.log(isEditing);

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
				day: day ? day : $selectedDate.day,
				month: month ? month : $selectedDate.month,
				year: year ? year : $selectedDate.year
			})
			.then((response) => {
				if (response.data.success) {
					text = editedText;
				} else {
					console.error('Failed to update pin:', response.data.message);

					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorUpdatePinText'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error('Error updating pin:', error);

				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorUpdatePinText'));
				toast.show();
			})
			.finally(() => {
				isEditing = false;
				isUpdatingText = false;
			});
	}
</script>

<div class="saved-pin-popup">
	{isEditing}
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
			<div>{tr('map.confirm_delete_pin')}</div>
			<div>
				<button class="btn btn-danger btn-sm me-2" onclick={deletePin}
					>{tr('settings.delete')}</button
				>
				<button class="btn btn-secondary btn-sm" onclick={confirmDeletePinAbort}
					>{tr('settings.abort')}</button
				>
			</div>
		</div>
	{:else}
		<div class="saved-pin-view d-flex flex-row">
			<div class="d-flex flex-column align-items-center flex-grow-1">
				{#if day && month && year}
					<div class="saved-pin-date">
						{new Date(year, month - 1, day).toLocaleDateString(language, {
							year: 'numeric',
							month: '2-digit',
							day: '2-digit'
						})}
					</div>
				{/if}
				<div class="saved-pin-text">
					{#if text !== ''}
						{text}
					{:else}
						<em class="no-description">{tr('map.pin.no_description')}</em>
					{/if}
				</div>
				{#if day && month && year}
					<button
						class="btn btn-sm btn-primary p-1 mt-1"
						onclick={() => openPreview(day, month, year)}
					>
						{tr('map.pin.open_preview')}
					</button>
				{/if}
			</div>
			{#if !readingMode}
				<div class="dropdown ps-2 border-start border-secondary">
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
								title={tr('map.pin.edit')}
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
								title={tr('map.pin.move')}
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
								title={tr('map.pin.delete')}
							>
								<Fa icon={faTrash} fw />
							</button>
						</li>
					</ul>
				</div>
			{/if}
		</div>
	{/if}
</div>

<div class="toast-container position-fixed bottom-0 end-0 p-3">
	<div
		id="toastErrorUpdatePinText"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{tr('map.toast.error_updating_pin_text')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>
</div>

<style>
	.saved-pin-date {
		text-decoration: underline;
		text-decoration-color: #1565c0;
	}

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

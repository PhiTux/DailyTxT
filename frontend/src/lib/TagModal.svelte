<script>
	import * as bootstrap from 'bootstrap';
	import Tag from './Tag.svelte';
	import { Picker } from 'emoji-picker-element';
	import Fa from 'svelte-fa';
	import { faTrash } from '@fortawesome/free-solid-svg-icons';

	let { editTag = $bindable(), createTag = false, saveNewTag, isSaving = false } = $props();

	function open() {
		// hide tag picker
		document.querySelector('.tooltip').classList.remove('shown');

		let modal = new bootstrap.Modal(document.getElementById('modalTag'));
		modal.show();
	}

	function close() {
		let modal = bootstrap.Modal.getInstance(document.getElementById('modalTag'));
		modal.hide();
	}

	export { open, close };

	let pickerShown = $state(false);
	function togglePicker() {
		document.querySelector('.tooltip').classList.toggle('shown');
		pickerShown = document.querySelector('.tooltip').classList.contains('shown');
	}

	function emojiSelected(ev) {
		editTag.icon = ev.detail.unicode;
		togglePicker();
	}
</script>

<div class="modal fade" id="modalTag" tabindex="-1">
	<div class="modal-dialog">
		<div class="modal-content">
			<div class="modal-header">
				<h5 class="modal-title">
					{#if createTag}
						Neues Tag erstellen
					{:else}
						Tag bearbeiten
					{/if}
				</h5>
				<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
			</div>
			<div class="modal-body">
				<div class="row">
					<div class="col-4">
						<h5>Emoji</h5>
					</div>
					<div class="col-8">
						{#if editTag.icon}
							<span>{editTag.icon}</span>
							<button class="removeBtn" type="button" onclick={(editTag.icon = '')}
								><Fa icon={faTrash} fw /></button
							>
						{:else}
							<span><em>Kein Emoji ausgewählt...</em></span>
						{/if}
					</div>
				</div>
				<div class="row">
					<div class="col-4"></div>
					<div class="col-8">
						<button
							class="btn btn-outline-secondary mb-2 {pickerShown ? 'active' : ''}"
							onclick={() => togglePicker()}
						>
							{#if editTag.icon === ''}
								😀
							{:else}
								{editTag.icon}
							{/if} Emoji auswählen
						</button>
						<!-- <em>(freiwillig)</em> -->
						<div class="tooltip" role="tooltip">
							<emoji-picker class="emojiPicker" onemoji-click={(ev) => emojiSelected(ev)}
							></emoji-picker>
						</div>
					</div>
				</div>

				<div class="row">
					<div class="col-4"><h5>Name</h5></div>
					<div class="col-8">
						<input
							bind:value={editTag.name}
							type="text"
							class="form-control mb-2"
							placeholder="Name"
						/>
					</div>
				</div>

				<div class="row">
					<div class="col-4">
						<h5>Farbe</h5>
					</div>
					<div class="col-8">
						<input
							bind:value={editTag.color}
							type="color"
							class="form-control form-control-color colorInput"
						/>
					</div>
				</div>

				<hr />
				<div class="row">
					<div class="col-4"><h5>Vorschau</h5></div>
					<div class="col-8"><Tag tag={editTag} /></div>
				</div>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Schließen</button>
				<button
					onclick={saveNewTag}
					type="button"
					class="btn btn-primary"
					disabled={!editTag.name || isSaving}
					>Speichern
					{#if isSaving}
						<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>
					{/if}
				</button>
			</div>
		</div>
	</div>
</div>

<style>
	.colorInput {
		width: 50px;
		padding: 3px;
		cursor: pointer;
	}

	.tooltip:not(.shown) {
		display: none;
	}

	.tooltip {
		display: contents;
		position: absolute;
		z-index: 1000;
	}

	.emojiPicker {
		position: absolute;
		z-index: 1000;
	}

	.removeBtn {
		background-color: transparent;
		border: 1px solid #ccc;
		border-radius: 5px;
		color: #495057;
		cursor: pointer;
		font-size: 11pt;
		margin-left: 0.3rem;
		transition: all 0.3s ease;
	}

	.removeBtn:hover {
		color: #dc3545;
	}

	:global(.modal.show) {
		background-color: rgba(80, 80, 80, 0.1) !important;
		backdrop-filter: blur(2px) saturate(150%);
	}

	.modal-content {
		backdrop-filter: blur(8px) saturate(150%);
		background-color: rgba(219, 219, 219, 0.45);
	}
</style>

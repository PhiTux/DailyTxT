<script>
	import * as bootstrap from 'bootstrap';
	import Tag from './Tag.svelte';
	import Fa from 'svelte-fa';
	import { faTrash } from '@fortawesome/free-solid-svg-icons';
	import { getTranslate } from '@tolgee/svelte';
	import EmojiMart from './EmojiMart.svelte';

	const { t } = getTranslate();

	let {
		editTag = $bindable(),
		createTag = false,
		saveNewTag,
		saveEditedTag,
		isSaving = false
	} = $props();

	let modalElement;
	let tooltipElement;

	function open() {
		// hide tag picker
		if (tooltipElement) {
			tooltipElement.classList.remove('shown');
		}

		let modal = new bootstrap.Modal(modalElement);
		modal.show();
	}

	function close() {
		let modal = bootstrap.Modal.getInstance(modalElement);
		modal.hide();
	}

	export { open, close };

	let pickerShown = $state(false);
	function togglePicker() {
		if (tooltipElement) {
			tooltipElement.classList.toggle('shown');
			pickerShown = tooltipElement.classList.contains('shown');
		}
	}

	function emojiSelected(ev) {
		editTag.icon = ev.native;
		togglePicker();
	}
</script>

<div bind:this={modalElement} class="modal fade" id="modalTag" tabindex="-1">
	<div class="modal-dialog">
		<div class="modal-content">
			<div class="modal-header">
				<h5 class="modal-title">
					{#if createTag}
						{$t('modal.tag.title_new')}
					{:else}
						{$t('modal.tag.title_edit')}
					{/if}
				</h5>
				<button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
			</div>
			<div class="modal-body">
				<div class="row">
					<div class="col-4">
						<h5>{$t('modal.tag.emoji')}</h5>
					</div>
					<div class="col-8">
						{#if editTag.icon}
							<span>{editTag.icon}</span>
							<button
								class="removeBtn"
								type="button"
								onclick={() => {
									editTag.icon = '';
								}}><Fa icon={faTrash} fw /></button
							>
						{:else}
							<span><em>{$t('modal.tag.no_emoji')}</em></span>
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
							{/if}
							{$t('modal.tag.select_emoji')}
						</button>
						<div class="tooltip" role="tooltip" bind:this={tooltipElement}>
							<EmojiMart select={emojiSelected} />
						</div>
					</div>
				</div>

				<div class="row">
					<div class="col-4"><h5>{$t('modal.tag.name')}</h5></div>
					<div class="col-8">
						<input
							bind:value={editTag.name}
							type="text"
							class="form-control mb-2"
							placeholder={$t('modal.tag.name')}
							onkeydown={(e) => {
								if (e.key === 'Enter') {
									createTag ? saveNewTag() : saveEditedTag();
								}
							}}
						/>
					</div>
				</div>

				<div class="row">
					<div class="col-4">
						<h5>{$t('modal.tag.color')}</h5>
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
					<div class="col-4"><h5>{$t('modal.tag.preview')}</h5></div>
					<div class="col-8"><Tag tag={editTag} /></div>
				</div>
			</div>
			<div class="modal-footer">
				<button type="button" class="btn btn-secondary" data-bs-dismiss="modal"
					>{$t('modal.close')}</button
				>
				<button
					onclick={() => {
						createTag ? saveNewTag() : saveEditedTag();
					}}
					type="button"
					class="btn btn-primary"
					disabled={!editTag.name || isSaving}
					>{$t('modal.save')}
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

	:global(em-emoji-picker) {
		position: absolute;
		z-index: 1000;
	}

	.removeBtn {
		background-color: transparent;
		border: 1px solid #ccc;
		border-radius: 5px;

		cursor: pointer;
		font-size: 11pt;
		margin-left: 0.3rem;
		transition: all 0.3s ease;
	}

	:global(body[data-bs-theme='dark']) .removeBtn {
		color: #c2c2c2;
	}

	:global(body[data-bs-theme='light']) .removeBtn {
		color: #495057;
	}

	.removeBtn:hover {
		color: #dc3545 !important;
	}

	.modal-header {
		border-bottom: none;
	}

	.modal-footer {
		border-top: none;
	}
</style>

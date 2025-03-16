<script>
	import Fa from 'svelte-fa';
	import { faTrash, faPencil, faXmark } from '@fortawesome/free-solid-svg-icons';
	let { tag, removeTag, deleteTag, editTag, isEditable, isRemovable, isDeletable } = $props();

	let fontColor = $state('#111');
	$effect(() => {
		const r = parseInt(tag.color.slice(1, 3), 16);
		const g = parseInt(tag.color.slice(3, 5), 16);
		const b = parseInt(tag.color.slice(5, 7), 16);
		const brightness = r * 0.299 + g * 0.587 + b * 0.114;

		if (brightness > 140) {
			fontColor = '#111';
		} else {
			fontColor = '#eee';
		}
	});
</script>

<span class="badge rounded-pill" style="background-color: {tag.color}; color: {fontColor}">
	<div class="d-flex flex-row">
		<div>{tag.icon} #{tag.name}</div>
		{#if isEditable}
			<button onclick={() => editTag(tag.id)} class="button btnEdit">
				<Fa icon={faPencil} fw />
			</button>
		{/if}
		{#if isRemovable}
			<button onclick={() => removeTag(tag.id)} class="button btnRemove">
				<Fa icon={faXmark} fw />
			</button>
		{/if}
		{#if isDeletable}
			<button onclick={() => deleteTag(tag.id)} class="button btnRemove">
				<Fa icon={faTrash} fw />
			</button>
		{/if}
	</div>
</span>

<style>
	span {
		background-color: #f8f9fa;
		color: #495057;
		font-size: 11pt;
		font-weight: 600;
	}

	button {
		background-color: transparent;
		border: none;
		color: #495057;
		cursor: pointer;
		font-size: 11pt;
		margin-left: 0.3rem;
		transition: all 0.3s ease;
	}

	.btnRemove:hover {
		color: #dc3545;
	}

	.btnEdit:hover {
		color: #007bff;
	}
</style>

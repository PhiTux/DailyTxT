<script>
	import Fa from 'svelte-fa';
	import { faTrash, faPencil, faXmark } from '@fortawesome/free-solid-svg-icons';
	let { tag, removeTag, isEditable, isRemovable, isDeletable } = $props();
</script>

<span class="badge rounded-pill" style="background-color: {tag.color};">
	<div class="d-flex flex-row">
		<div>{tag.icon} #{tag.name}</div>
		{#if isEditable}
			<button class="button btnEdit">
				<Fa icon={faPencil} fw />
			</button>
		{/if}
		{#if isRemovable}
			<button onclick={() => removeTag(tag.id)} class="button btnRemove">
				<Fa icon={faXmark} fw />
			</button>
		{/if}
		{#if isDeletable}
			<button class="button btnRemove">
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

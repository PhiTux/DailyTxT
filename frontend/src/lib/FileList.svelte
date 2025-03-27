<script>
	import { Fa } from 'svelte-fa';
	import { faTrash } from '@fortawesome/free-solid-svg-icons';
	import { slide } from 'svelte/transition';
	import { formatBytes } from './helpers.js';

	let { files, downloadFile, askDeleteFile, deleteAllowed } = $props();
</script>

{#each files as file (file.uuid_filename)}
	<div class="btn-group file mt-2" transition:slide>
		<button
			onclick={() => downloadFile(file.uuid_filename)}
			class="p-2 fileBtn d-flex flex-column flex-fill"
		>
			<div class="d-flex flex-row align-items-center">
				<div class="filename filenameWeight">{file.filename}</div>
				<span class="filesize">({formatBytes(file.size)})</span>
			</div>
			{#if file.downloadProgress >= 0}
				<div
					class="progress"
					role="progressbar"
					aria-label="Download progress"
					aria-valuemin="0"
					aria-valuemax="100"
				>
					<div
						class="progress-bar overflow-visible bg-info {file.downloadProgress === 0
							? 'progress-bar-striped progress-bar-animated'
							: ''}"
						style:width={file.downloadProgress + '%'}
						aria-valuenow={file.downloadProgress}
						aria-valuemax="100"
					>
						{#if file.downloadProgress === 0}
							<span class="text-dark">Wird entschlüsselt...</span>
						{:else}
							<span class="text-dark">Download: {file.downloadProgress}%</span>
						{/if}
					</div>
				</div>
			{/if}
		</button>
		{#if deleteAllowed}
			<button
				class="p-2 fileBtn deleteFileBtn"
				onclick={() => askDeleteFile(file.uuid_filename, file.filename)}
				><Fa icon={faTrash} fw /></button
			>
		{/if}
	</div>
{/each}

<style>
	.filename {
		padding-right: 0.5rem;
		word-break: break-word;
	}

	.filesize {
		opacity: 0.7;
		font-size: 0.8rem;
		white-space: nowrap;
	}

	.fileBtn {
		border: 0;
		background-color: rgba(0, 0, 0, 0);
		transition: all ease 0.3s;
	}

	.fileBtn:hover {
		background-color: rgba(0, 0, 0, 0.1);
	}

	.deleteFileBtn {
		border-left: 1px solid rgba(92, 92, 92, 0.445);
	}

	.deleteFileBtn:hover {
		color: rgb(165, 0, 0);
	}

	.file {
		background-color: rgba(117, 117, 117, 0.45);
		border: 0px solid #ececec77;
		border-radius: 5px;
	}
</style>

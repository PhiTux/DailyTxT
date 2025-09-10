<script>
	import { Fa } from 'svelte-fa';
	import { faTrash, faWrench, faEdit, faSave, faTimes } from '@fortawesome/free-solid-svg-icons';
	import { slide } from 'svelte/transition';
	import { formatBytes } from './helpers.js';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	let { files, downloadFile, askDeleteFile, deleteAllowed, renameFile } = $props();

	let openOptionsMenu = $state(null); // UUID of file with open options menu
	let editingFilename = $state(null); // UUID of file being renamed
	let newFilename = $state('');
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
							<span class="text-dark">
								{$t('files.decrypting')}
							</span>
						{:else}
							<span class="text-dark">{$t('files.download')}: {file.downloadProgress}%</span>
						{/if}
					</div>
				</div>
			{/if}
		</button>
		{#if deleteAllowed}
			<button
				class="p-2 fileBtn optionsBtn"
				onclick={() => {
					if (openOptionsMenu === file.uuid_filename) {
						openOptionsMenu = null;
						editingFilename = null;
					} else {
						openOptionsMenu = file.uuid_filename;
						editingFilename = null;
					}
				}}
			>
				<Fa icon={faWrench} fw />
			</button>
		{/if}
	</div>

	{#if deleteAllowed && openOptionsMenu === file.uuid_filename}
		<div transition:slide>
			<div class="options-menu p-3 mt-1">
				<div class="mb-3">
					<!-- svelte-ignore a11y_label_has_associated_control -->
					<label class="form-label small fw-bold">{$t('fileList.change_filename')}:</label>
					<div class="d-flex gap-2">
						{#if editingFilename === file.uuid_filename}
							<input
								type="text"
								class="form-control form-control-sm"
								id="newFilename-{file.uuid_filename}"
								bind:value={newFilename}
								onkeydown={(e) => {
									if (e.key === 'Enter') {
										if (renameFile) {
											renameFile(file.uuid_filename, newFilename);
										}
										editingFilename = null;
										openOptionsMenu = null;
									} else if (e.key === 'Escape') {
										editingFilename = null;
									}
								}}
							/>
							<button
								class="btn btn-sm btn-success"
								onclick={() => {
									if (renameFile) {
										renameFile(file.uuid_filename, newFilename);
									}
									editingFilename = null;
									openOptionsMenu = null;
								}}
							>
								<Fa icon={faSave} fw />
							</button>
							<button
								class="btn btn-sm btn-secondary"
								onclick={() => {
									editingFilename = null;
								}}
							>
								<Fa icon={faTimes} fw />
							</button>
						{:else}
							<input
								type="text"
								class="form-control form-control-sm"
								value={file.filename}
								disabled
							/>
							<button
								class="btn btn-sm btn-primary"
								onclick={() => {
									editingFilename = file.uuid_filename;
									newFilename = file.filename;
								}}
							>
								<Fa icon={faEdit} fw />
							</button>
						{/if}
					</div>
				</div>

				<hr style="color: black;" />

				<div>
					<button
						class="btn btn-sm btn-danger w-100"
						onclick={() => {
							askDeleteFile(file.uuid_filename, file.filename);
							openOptionsMenu = null;
						}}
					>
						<Fa icon={faTrash} fw class="me-2" />
						{$t('fileList.delete_file')}
					</button>
				</div>
			</div>
		</div>
	{/if}
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

	.optionsBtn {
		border-left: 1px solid rgba(92, 92, 92, 0.445);
	}

	.optionsBtn:hover {
		color: rgb(0, 123, 255);
	}

	.file {
		background-color: rgba(117, 117, 117, 0.45);
		border: 0px solid #ececec77;
		border-radius: 5px;
	}

	.options-menu {
		background-color: rgba(248, 249, 250, 0.95);
		border: 1px solid rgba(0, 0, 0, 0.125);
		border-radius: 5px;
		backdrop-filter: blur(5px);
	}
</style>

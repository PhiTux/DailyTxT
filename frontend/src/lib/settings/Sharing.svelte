<script>
	import { slide } from 'svelte/transition';
	import { Fa } from 'svelte-fa';
	import { faCopy, faCheck, faLink, faTrash, faRotate } from '@fortawesome/free-solid-svg-icons';

	let {
		hasShareToken,
		shareLink,
		isGeneratingShareToken,
		isRevokingShareToken,
		linkCopiedSuccess,
		showShareTokenError,
		generateShareToken,
		revokeShareToken,
		copyShareLink
	} = $props();
</script>

<h3 class="text-primary">🔗 Sharing</h3>
<div>
	<p class="form-text mb-3">
		Generate a secret read-only link to share your diary entries with others. Anyone with this link
		can view your diary without needing to log in. Revoke the link at any time to disable access.
	</p>

	{#if shareLink}
		<div class="mb-3" transition:slide>
			<label for="shareLinkInput" class="form-label fw-semibold">Share Link</label>
			<div class="input-group">
				<input id="shareLinkInput" type="text" class="form-control font-monospace" value={shareLink} readonly />
				<button class="btn btn-outline-secondary" onclick={copyShareLink} title="Copy link">
					{#if linkCopiedSuccess}
						<Fa icon={faCheck} class="text-success" />
					{:else}
						<Fa icon={faCopy} />
					{/if}
				</button>
			</div>
			<div class="form-text">Keep this link private — it grants read access to your diary.</div>
		</div>
	{/if}

	<div class="d-flex flex-row gap-2 flex-wrap">
		{#if !hasShareToken}
			<button
				class="btn btn-primary"
				onclick={generateShareToken}
				disabled={isGeneratingShareToken}
			>
				{#if isGeneratingShareToken}
					<span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"
					></span>
				{:else}
					<Fa icon={faLink} class="me-2" />
				{/if}
				Generate Share Link
			</button>
		{:else}
			<button
				class="btn btn-outline-secondary"
				onclick={generateShareToken}
				disabled={isGeneratingShareToken}
				title="Generate a new link (invalidates the old one)"
			>
				{#if isGeneratingShareToken}
					<span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"
					></span>
				{:else}
					<Fa icon={faRotate} class="me-2" />
				{/if}
				Regenerate Link
			</button>
			<button
				class="btn btn-outline-danger"
				onclick={revokeShareToken}
				disabled={isRevokingShareToken}
			>
				{#if isRevokingShareToken}
					<span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"
					></span>
				{:else}
					<Fa icon={faTrash} class="me-2" />
				{/if}
				Revoke Link
			</button>
		{/if}
	</div>

	{#if showShareTokenError}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			An error occurred. Please try again.
		</div>
	{/if}
</div>

<script>
	import { slide } from 'svelte/transition';
	import Tag from '../Tag.svelte';
	import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';

	let { tags, openTagModal, deleteTagId, askDeleteTag, isDeletingTag, deleteTag } = $props();

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
</script>

<h3 class="text-primary">#️⃣ {$t('settings.tags')}</h3>
<div>
	{$t('settings.tags.description')}

	{#if $tags.length === 0}
		<div class="alert alert-info my-2" role="alert">
			{$t('settings.tags.no_tags')}
		</div>
	{/if}
	<div class="d-flex flex-column tagColumn mt-1">
		{#each $tags as tag}
			<div class="mt-2">
				<Tag {tag} isEditable editTag={openTagModal} isDeletable deleteTag={askDeleteTag} />
				{#if deleteTagId === tag.id}
					<div transition:slide style="padding-top: 0.5rem">
						<div class="alert alert-danger align-items-center tagAlert" role="alert">
							<div>
								<Fa icon={faTriangleExclamation} fw />
								{@html $t('settings.tags.delete_confirmation')}
							</div>
							<!-- svelte-ignore a11y_consider_explicit_label -->
							<div class="d-flex flex-row mt-2">
								<button class="btn btn-secondary" onclick={() => (deleteTagId = null)}
									>{$t('settings.abort')}
								</button>
								<button
									disabled={isDeletingTag}
									class="btn btn-danger ms-3"
									onclick={() => deleteTag(tag.id)}
									>{$t('settings.delete')}
									{#if isDeletingTag}
										<span
											class="spinner-border spinner-border-sm ms-2"
											role="status"
											aria-hidden="true"
										></span>
									{/if}
								</button>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{/each}
	</div>
</div>

<script>
	import { slide } from 'svelte/transition';
	import { faTrash, faPlus, faSquarePlus } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';
	import { templates } from '$lib/templateStore';

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();

	let {
		unsavedChanges,
		selectedTemplate = $bindable(),
		updateSelectedTemplate,
		confirmDeleteTemplate = $bindable(),
		templateName = $bindable(),
		deleteTemplate,
		templateText = $bindable(),
		templateIsDefault = $bindable(),
		oldTemplateName,
		oldTemplateText,
		oldTemplateIsDefault,
		isSavingTemplate,
		isDeletingTemplate,
		saveTemplate,
		startCreatingTemplate
	} = $props();
</script>

<h3 class="text-primary">📝 {$t('settings.templates')}</h3>
<div>
	{#if oldTemplateName !== templateName || oldTemplateText !== templateText || oldTemplateIsDefault !== templateIsDefault}
		{@render unsavedChanges()}
	{/if}

	{#if $templates.length === 0 && selectedTemplate !== 'new'}
		<p class="text-muted">{$t('settings.templates.no_templates')}</p>
		<button type="button" class="btn btn-primary" onclick={startCreatingTemplate}>
			<Fa icon={faPlus} fw class="me-1" />{$t('settings.templates.new')}
		</button>
	{:else}
		{#if $templates.length > 0}
			<div class="d-flex flex-row align-items-center">
				<select
					bind:value={selectedTemplate}
					class="form-select"
					aria-label="Select template"
					onchange={updateSelectedTemplate}
				>
					{#if selectedTemplate === 'new'}
						<option value="new" selected>{$t('settings.templates.new')}</option>
					{:else}
						<option value={null} disabled selected={selectedTemplate === null}>
							{$t('settings.templates.select_placeholder')}
						</option>
					{/if}
					{#each $templates as template, index (template.name)}
						<option value={index} selected={index === selectedTemplate}>
							{template.name}
						</option>
					{/each}
				</select>
				<button
					type="button"
					class="btn btn-outline-secondary ms-2 text-nowrap"
					onclick={startCreatingTemplate}
				>
					<Fa icon={faSquarePlus} fw />
					{$t('settings.templates.new')}
				</button>
			</div>
		{/if}

		{#if selectedTemplate !== null}
			{#if $templates.length > 0}
				<hr />
			{/if}

			{#if confirmDeleteTemplate}
				<div transition:slide>
					<div class="d-flex flex-row align-items-center pb-2">
						<span>
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html $t('settings.templates.delete_confirmation', {
								template: $templates[selectedTemplate]?.name
							})}
						</span>
						<button
							type="button"
							class="btn btn-secondary ms-2"
							onclick={() => (confirmDeleteTemplate = false)}>{$t('settings.abort')}</button
						>
						<button
							type="button"
							class="btn btn-danger ms-2"
							onclick={() => {
								deleteTemplate();
							}}
							disabled={isDeletingTemplate}
							>{$t('settings.delete')}
							{#if isDeletingTemplate}
								<span class="spinner-border spinner-border-sm ms-2" role="status" aria-hidden="true"
								></span>
							{/if}
						</button>
					</div>
				</div>
			{/if}
			<div class="d-flex flex-row">
				<input
					type="text"
					bind:value={templateName}
					class="form-control"
					placeholder={$t('settings.template.name_of_template')}
				/>
				{#if selectedTemplate !== 'new'}
					<button
						type="button"
						class="btn btn-outline-danger ms-5"
						onclick={() => {
							confirmDeleteTemplate = !confirmDeleteTemplate;
						}}><Fa fw icon={faTrash} /></button
					>
				{/if}
			</div>
			<textarea
				bind:value={templateText}
				class="form-control mt-2"
				rows="10"
				placeholder={$t('settings.template.content_of_template')}
			>
			</textarea>
			<div class="form-check mt-2">
				<input
					class="form-check-input"
					type="checkbox"
					bind:checked={templateIsDefault}
					id="defaultTemplateCheck"
				/>
				<label class="form-check-label" for="defaultTemplateCheck">
					{$t('settings.template.set_as_default')}
				</label>
			</div>
			<div class="d-flex justify-content-start">
				<button
					disabled={(oldTemplateName === templateName &&
						oldTemplateText === templateText &&
						oldTemplateIsDefault === templateIsDefault) ||
						templateName == '' ||
						templateText == '' ||
						isSavingTemplate}
					type="button"
					class="btn btn-primary mt-2"
					onclick={() => saveTemplate()}
				>
					{$t('settings.template.save_template')}
					{#if isSavingTemplate}
						<span class="spinner-border spinner-border-sm ms-2" role="status" aria-hidden="true"
						></span>
					{/if}
				</button>
			</div>
		{/if}
	{/if}
</div>

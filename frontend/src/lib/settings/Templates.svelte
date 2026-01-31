<script>
	import { slide } from 'svelte/transition';
	import { faTrash } from '@fortawesome/free-solid-svg-icons';
	import { Fa } from 'svelte-fa';
	import { templates } from '$lib/templateStore';

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();

	let {
		unsavedChanges,
		selectedTemplate = $bindable(),
		updateSelectedTemplate,
		confirmDeleteTemplate,
		templateName = $bindable(),
		deleteTemplate,
		templateText = $bindable(),
		oldTemplateName,
		oldTemplateText,
		isSavingTemplate,
		isDeletingTemplate,
		saveTemplate
	} = $props();
</script>

<h3 class="text-primary">📝 {$t('settings.templates')}</h3>
<div>
	{#if oldTemplateName !== templateName || oldTemplateText !== templateText}
		{@render unsavedChanges()}
	{/if}

	<div class="d-flex flex-column">
		<select
			bind:value={selectedTemplate}
			class="form-select"
			aria-label="Select template"
			onchange={updateSelectedTemplate}
		>
			<option value="-1" selected={selectedTemplate === '-1'}>
				{$t('settings.templates.create_new')}
			</option>
			{#each $templates as template, index}
				<option value={index} selected={index === selectedTemplate}>
					{template.name}
				</option>
			{/each}
		</select>
	</div>

	<hr />

	{#if confirmDeleteTemplate}
		<div transition:slide>
			<div class="d-flex flex-row align-items-center pb-2">
				<span>
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
			disabled={selectedTemplate === null}
			type="text"
			bind:value={templateName}
			class="form-control"
			placeholder={$t('settings.template.name_of_template')}
		/>
		<button
			disabled={selectedTemplate === '-1' || selectedTemplate === null}
			type="button"
			class="btn btn-outline-danger ms-5"
			onclick={() => {
				confirmDeleteTemplate = !confirmDeleteTemplate;
			}}><Fa fw icon={faTrash} /></button
		>
	</div>
	<textarea
		disabled={selectedTemplate === null}
		bind:value={templateText}
		class="form-control mt-2"
		rows="10"
		placeholder={$t('settings.template.content_of_template')}
	>
	</textarea>
	<div class="d-flex justify-content-start">
		<button
			disabled={(oldTemplateName === templateName && oldTemplateText === templateText) ||
				isSavingTemplate}
			type="button"
			class="btn btn-primary mt-2"
			onclick={() => saveTemplate()}
		>
			{$t('settings.template.save_template')}
			{#if isSavingTemplate}
				<span class="spinner-border spinner-border-sm ms-2" role="status" aria-hidden="true"></span>
			{/if}
		</button>
	</div>
</div>

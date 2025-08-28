<script>
	import { templates, insertTemplate } from '$lib/templateStore';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();
</script>

<select
	bind:value={$insertTemplate}
	class="form-select form-select-sm"
	aria-label="Select template"
>
	<option value="" disabled>{$t('template.dropdown_insert')}</option>
	{#each $templates as template}
		<option value={template.text}>
			{template.name}
		</option>
	{/each}
</select>

<style>
	.form-select {
		width: 160px;
		padding-top: 0;
		padding-bottom: 0;
		align-self: center;
		margin-left: 0.5rem;
	}
</style>

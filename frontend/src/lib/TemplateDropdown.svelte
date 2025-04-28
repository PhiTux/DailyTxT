<script>
	import { templates, insertTemplate } from '$lib/templateStore';
</script>

<select
	bind:value={$insertTemplate}
	class="form-select form-select-sm"
	aria-label="Select template"
>
	<option disabled selected>Vorlage einfügen...</option>
	{#each $templates as template}
		<option value={template.text} selected={template.name === $insertTemplate}>
			{template.name}
		</option>
	{/each}
</select>

<style>
	.form-select {
		width: inherit !important;
		padding-top: 0;
		padding-bottom: 0;
		align-self: center;
	}
</style>

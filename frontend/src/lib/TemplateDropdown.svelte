<script>
	import { templates, insertTemplate } from '$lib/templateStore';
</script>

<select
	bind:value={$insertTemplate}
	class="form-select form-select-sm"
	aria-label="Select template"
>
	<option value="" disabled>Vorlage einfügen...</option>
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

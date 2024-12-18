<script>
	import '../scss/styles.scss';
	//import * as bootstrap from 'bootstrap';
	import { Tooltip } from 'bootstrap';
	import { goto } from '$app/navigation';

	//on mount
	import { onMount } from 'svelte';
	onMount(() => {
		const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
		const tooltipList = [...tooltipTriggerList].map(
			(tooltipTriggerEl) => new Tooltip(tooltipTriggerEl)
		);
	});
</script>

<h1>Welcome to SvelteKit</h1>
<p>Visit <a href="https://svelte.dev/docs/kit">svelte.dev/docs/kit</a> to read the documentation</p>
<button
	type="button"
	class="btn btn-primary"
	data-bs-toggle="tooltip"
	data-bs-title="Default tooltip">Primary</button
>

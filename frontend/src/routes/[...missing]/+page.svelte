<script>
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { base } from '$app/paths';
	onMount(() => {
		// Catch-all: redirect to login (or change to `${base}/write` if desired)
		goto(`${base}/login`, { replaceState: true });
	});
</script>

<!-- Optionally show a brief message -->
<div style="display:none"></div>

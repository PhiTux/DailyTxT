<script>
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	onMount(() => {
		// Catch-all: redirect to login
		goto(resolve('/login'), { replaceState: true });
	});
</script>

<!-- Optionally show a brief message -->
<div style="display:none"></div>

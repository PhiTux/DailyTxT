<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	onMount(() => {
		// Redirect root to login, honoring the app base path (e.g., /dailytxt)
		goto(resolve('/login'), { replaceState: true });
	});
</script>

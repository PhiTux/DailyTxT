<script context="module">
	// main navigation for the application
	import Settings from '$lib/Settings.svelte';
	import Statistics from '$lib/Statistics.svelte';
	import Admin from '$lib/Admin.svelte';

	// these are our 'pages' to navigate through (in order for this export to work the script tag MUST include context="module")
	export const navOptions = [
		{ page: 'Settings', component: Settings },
		{ page: 'Statistics', component: Statistics },
		{ page: 'Admin', component: Admin }
	];
</script>

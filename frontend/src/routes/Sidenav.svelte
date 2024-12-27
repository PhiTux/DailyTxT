<script>
	import Datepicker from './Datepicker.svelte';
	let { sidenav } = {};
	let selectedDate = $state(new Date());
</script>

{sidenav}
<!-- {#if sidenav} -->

<!-- {/if} -->

<Datepicker
	currentlySelectedDate={new Date(2025, 0, 31)}
	dateSelected={(date) => (selectedDate = date)}
/>
<br />
{selectedDate.toLocaleDateString('de-DE')}
<br />
Search

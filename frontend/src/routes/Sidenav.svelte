<script>
	import Datepicker from './Datepicker.svelte';
	let { sidenav } = {};
	let selectedDate = $state(new Date());
</script>

{sidenav}
<!-- {#if sidenav} -->

<!-- {/if} -->

<Datepicker dateSelected={(date) => (selectedDate = date)} />
<br />
{selectedDate.toDateString()}
<br />
Search

<script>
	import '../scss/styles.scss';
	//import * as bootstrap from 'bootstrap';
	import { Tooltip } from 'bootstrap';
	import { goto } from '$app/navigation';
	import Sidenav from './Sidenav.svelte';

	//on mount
	import { onMount } from 'svelte';
	onMount(() => {
		const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
		const tooltipList = [...tooltipTriggerList].map(
			(tooltipTriggerEl) => new Tooltip(tooltipTriggerEl)
		);
	});
</script>

<!-- shown on small Screen, when triggered -->
<div class="offcanvas-md d-md-none offcanvas-start p-3" id="sidenav" tabindex="-1">
	<div class="offcanvas-header">
		<button
			type="button"
			class="btn-close"
			data-bs-dismiss="offcanvas"
			data-bs-target="#sidenav"
			aria-label="Close"
		></button>
	</div>
	<Sidenav sidenav="true" />
</div>

<div class="d-flex flex-row justify-content-between">
	<!-- shown on large Screen -->
	<div class="d-md-block d-none sidenav p-3">
		<Sidenav />
	</div>

	<!-- Center -->
	<div class="d-flex flex-column mt-4 mx-4 flex-fill">
		<!-- Input-Area -->
		<div class="d-flex flex-column">
			<div class="d-flex flex-row">
				<div class="flex-fill">Datum</div>
				<div class="flex-fill">written at</div>
				<div>history</div>
				<div>delete</div>
			</div>
			<textarea class="form-control" rows="10"></textarea>
		</div>
	</div>

	<div id="right">Right</div>
</div>

<style>
	textarea {
		resize: vertical;
		width: 100%;
	}

	#right {
		width: 300px;
	}
</style>

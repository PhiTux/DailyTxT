<script>
	import { fade, blur, slide } from 'svelte/transition';
	import axios from 'axios';
	import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	export let data;
	let inDuration = 150;
	let outDuration = 150;

	let API_URL = dev ? 'http://localhost:8000' : window.location.pathname.replace(/\/+$/, '');

	function logout() {
		axios
			.get(API_URL + '/users/logout')
			.then((response) => {
				localStorage.removeItem('user');
				goto('/login');
			})
			.catch((error) => {
				console.error(error);
			});
	}
</script>

<main class="d-flex flex-column">
	<nav class="navbar navbar-expand-lg bg-body-tertiary">
		<div class="container-fluid">
			<button
				class="btn d-md-none"
				type="button"
				data-bs-toggle="offcanvas"
				data-bs-target="#sidenav"
				aria-controls="sidenav">menü</button
			>
			<a class="nav-item" href="/">Navbar</a>
			<a class="nav-item" href="/login">Navbar</a>
			<div class="dropdown">
				<button
					class="btn btn-outline-secondary dropdown-toggle"
					type="button"
					data-bs-toggle="dropdown"
					aria-expanded="false"
				>
					Dropdown button
				</button>
				<ul class="dropdown-menu dropdown-menu-end">
					<li><a class="dropdown-item" href="/settings">Settings</a></li>
					<li><button class="dropdown-item" onclick={logout}>Logout</button></li>
				</ul>
			</div>
		</div>
	</nav>

	<div class="wrapper h-100">
		{#key data.pathname}
			<div
				class="transition-wrapper h-100"
				out:blur={{ duration: outDuration }}
				in:blur={{ duration: inDuration, delay: outDuration }}
			>
				<slot />
			</div>
		{/key}
	</div>
</main>

<!-- in:crossfade={{ duration: inDuration, delay: outDuration }}
				out:crossfade={{ duration: outDuration }} -->

<style>
	main {
		height: 100vh;
	}

	.wrapper {
		position: relative; /* Ensure the wrapper is the positioning context */
	}

	.transition-wrapper {
		position: absolute; /* Ensure the transition wrapper does not occupy space */
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
	}
</style>

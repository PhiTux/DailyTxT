<script>
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import axios from 'axios';
	import { API_URL } from '$lib/APIurl.js';
	import { generateNeonMesh, isAuthenticated } from '$lib/helpers';
	import { getTranslate } from '@tolgee/svelte';
	import logo from '$lib/assets/locked_heart_with_keyhole.svg';

	const { t } = getTranslate();

	let password = $state('');
	let isValidating = $state(false);
	let error = $state('');

	onMount(() => {
		// get browser default-light/dark-mode settings
		const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
		generateNeonMesh(prefersDark);

		// Focus password input
		const passwordInput = document.getElementById('reauth-password');
		if (passwordInput) {
			passwordInput.focus();
		}
	});

	async function validatePassword() {
		if (isValidating || !password.trim()) return;

		isValidating = true;
		error = '';

		try {
			const response = await axios.post(API_URL + '/users/validatePassword', {
				password: password
			});

			if (response.data.valid) {
				$isAuthenticated = true;

				// Authentication successful - return to original route
				const returnPath = localStorage.getItem('returnAfterReauth') || '/write';
				localStorage.removeItem('returnAfterReauth');
				goto(returnPath);
			} else {
				error = $t('reauth.wrong_password');
				password = '';

				// Focus password input
				const passwordInput = document.getElementById('reauth-password');
				if (passwordInput) {
					passwordInput.focus();
				}
			}
		} catch (err) {
			console.error('Password validation error:', err);
			error = $t('reauth.authentication_error');
		} finally {
			isValidating = false;
		}
	}

	function handleKeydown(event) {
		if (event.key === 'Enter') {
			validatePassword();
		}
	}
</script>

<div class="background container-fluid h-100 d-flex align-items-center justify-content-center">
	<div class="card shadow-lg" style="max-width: 400px; width: 100%;">
		<div class="card-header text-center bg-primary text-white">
			<div class="d-flex justify-content-center align-items-center mb-3">
				<img src={logo} alt="" width="50px" />
				<span class="dailytxt ms-2">DailyTxT</span>
			</div>
			<h3 class="mb-0">{$t('reauth.title')}</h3>
		</div>
		<div class="card-body p-4">
			<p class="text-muted text-center mb-4">
				{$t('reauth.description')}
			</p>

			<form onsubmit={validatePassword}>
				<div class="mb-3 position-relative">
					<input
						id="reauth-password"
						type="password"
						bind:value={password}
						onkeydown={handleKeydown}
						class="form-control form-control-lg pe-5"
						placeholder={$t('login.password')}
						disabled={isValidating}
						required
					/>
				</div>

				{#if error}
					<div class="alert alert-danger">{error}</div>
				{/if}

				<button
					type="submit"
					class="btn btn-primary btn-lg w-100"
					disabled={isValidating || !password.trim()}
				>
					{#if isValidating}
						<span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"
						></span>
					{/if}
					{$t('reauth.confirmButton')}
				</button>
			</form>
		</div>
	</div>
</div>

<style scoped>
	.dailytxt {
		color: #f57c00;
		font-size: 2rem;
		font-weight: 500;
	}

	.container-fluid {
		min-height: 100vh;
	}

	.card {
		/* backdrop-filter: blur(10px); */
		/* background: rgba(255, 255, 255, 0.5); */
		border: none;
		border-radius: 10px;
	}

	.card-header {
		border-top-left-radius: 10px;
		border-top-right-radius: 10px;
	}
</style>

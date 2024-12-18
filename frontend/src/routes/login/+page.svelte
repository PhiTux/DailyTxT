<script>
	import img from '$lib/assets/locked_heart_with_keyhole.svg';
	import { onMount } from 'svelte';
	import axios from 'axios';
	import { dev } from '$app/environment';
	import { goto } from '$app/navigation';

	let show_warning_empty_fields = $state(false);
	let show_warning_passwords_do_not_match = $state(false);

	let show_registration_success = $state(false);
	let show_registration_failed = $state(false);
	let show_registration_failed_with_message = $state(false);
	let registration_failed_message = $state('');

	let API_URL = dev ? 'http://localhost:8000' : window.location.pathname.replace(/\/+$/, '');

	function handleLogin(event) {
		event.preventDefault();
		const username = document.getElementById('loginUsername').value;
		const password = document.getElementById('loginPassword').value;

		axios
			.post(API_URL + '/users/login', { username, password })
			.then((response) => {
				localStorage.setItem('user', JSON.stringify(response.data.username));
				goto('/');
			})
			.catch((error) => {
				console.error(error);
			});

		console.log('Login attempt:', { username, password });
		// Add your login logic here
	}

	function handleRegister(event) {
		show_warning_empty_fields = false;
		show_warning_passwords_do_not_match = false;
		show_registration_success = false;
		show_registration_failed = false;
		show_registration_failed_with_message = false;

		event.preventDefault();
		const username = document.getElementById('registerUsername').value;
		const password = document.getElementById('registerPassword').value;
		const password2 = document.getElementById('registerPassword2').value;

		if (username === '' || password === '') {
			show_warning_empty_fields = true;
			console.error('Please fill out all fields');
			return;
		}

		if (password !== password2) {
			show_warning_passwords_do_not_match = true;
			console.error('Passwords do not match');
			return;
		}

		axios
			.post(API_URL + '/users/register', { username, password })
			.then((response) => {
				if (response.data.success) {
					show_registration_success = true;
					console.log('Registration successful');
				} else {
					show_registration_failed = true;
					console.error('Registration failed');
				}
			})
			.catch((error) => {
				console.error(error.response.data.detail);
				registration_failed_message = error.response.data.detail;
				show_registration_failed_with_message = true;
			});
	}
</script>

<div class="logo-login-flex d-flex justify-content-center align-items-center flex-row h-100">
	<div class="logo-wrapper d-flex flex-column align-items-center">
		<img id="largeLogo" src={img} alt="locked heart with keyhole" />
		<p>DailyTxT</p>
	</div>
	<div class="login-wrapper">
		<div class="accordion" id="loginAccordion">
			<div class="accordion-item">
				<h2 class="accordion-header">
					<button
						class="accordion-button"
						type="button"
						data-bs-toggle="collapse"
						data-bs-target="#collapseOne"
						aria-expanded="true"
						aria-controls="collapseOne"
					>
						Login
					</button>
				</h2>
				<div
					id="collapseOne"
					class="accordion-collapse collapse show"
					data-bs-parent="#loginAccordion"
				>
					<div class="accordion-body">
						<form onsubmit={handleLogin}>
							<div class="form-floating mb-3">
								<!-- svelte-ignore a11y_autofocus -->
								<input
									type="text"
									class="form-control"
									id="loginUsername"
									placeholder="Username"
									autofocus
								/>
								<label for="loginUsername">Username</label>
							</div>
							<div class="form-floating mb-3">
								<input
									type="password"
									class="form-control"
									id="loginPassword"
									placeholder="Password"
								/>
								<label for="loginPassword">Password</label>
							</div>
							<div class="d-flex justify-content-center">
								<button type="submit" class="btn btn-primary">Login</button>
							</div>
						</form>
					</div>
				</div>
			</div>
			<div class="accordion-item">
				<h2 class="accordion-header">
					<button
						class="accordion-button collapsed"
						type="button"
						data-bs-toggle="collapse"
						data-bs-target="#collapseTwo"
						aria-expanded="false"
						aria-controls="collapseTwo"
					>
						Registrierung
					</button>
				</h2>
				<div id="collapseTwo" class="accordion-collapse collapse" data-bs-parent="#loginAccordion">
					<div class="accordion-body">
						<form onsubmit={handleRegister}>
							<div class="form-floating mb-3">
								<input
									type="text"
									class="form-control"
									id="registerUsername"
									placeholder="Username"
								/>
								<label for="registerUsername">Username</label>
							</div>
							<div class="form-floating mb-3">
								<input
									type="password"
									class="form-control"
									id="registerPassword"
									placeholder="Password"
								/>
								<label for="registerPassword">Password</label>
							</div>
							<div class="form-floating mb-3">
								<input
									type="password"
									class="form-control"
									id="registerPassword2"
									placeholder="Password bestätigen"
								/>
								<label for="registerPassword2">Password bestätigen</label>
							</div>
							{#if show_registration_failed_with_message}
								<div class="alert alert-danger" role="alert">
									Registrierung fehlgeschlagen!<br />
									Fehlermeldung: <i>{registration_failed_message}</i>
								</div>
							{/if}
							{#if show_registration_failed}
								<div class="alert alert-danger" role="alert">
									Registrierung fehlgeschlagen - bitte Fehlermeldungen analysieren!
								</div>
							{/if}
							{#if show_registration_success}
								<div class="alert alert-success" role="alert">
									Registrierung erfolgreich - bitte einloggen!
								</div>
							{/if}
							{#if show_warning_empty_fields}
								<div class="alert alert-danger" role="alert">
									Eingabefelder dürfen nicht leer sein!
								</div>
							{/if}
							{#if show_warning_passwords_do_not_match}
								<div class="alert alert-danger" role="alert">Passwörter stimmen nicht überein!</div>
							{/if}
							<div class="d-flex justify-content-center">
								<button type="submit" class="btn btn-primary">Registrieren</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.logo-wrapper {
		width: 50%;
	}

	#largeLogo {
		width: 40%;
		min-height: 10%;
	}

	.login-wrapper {
		width: 50%;
	}

	#loginAccordion {
		width: 70%;
	}

	@media screen and (max-width: 768px) {
		.logo-login-flex {
			flex-direction: column !important;
		}

		.login-wrapper {
			min-width: 50%;
			max-width: 75%;
		}
	}
</style>

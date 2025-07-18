<script>
	import img from '$lib/assets/locked_heart_with_keyhole.svg';
	import * as bootstrap from 'bootstrap';
	import { onMount } from 'svelte';
	import axios from 'axios';
	import { goto } from '$app/navigation';
	import { API_URL } from '$lib/APIurl.js';

	let show_login_failed = $state(false);
	let show_login_warning_empty_fields = $state(false);
	let is_logging_in = $state(false);

	let show_registration_warning_empty_fields = $state(false);
	let show_warning_passwords_do_not_match = $state(false);
	let show_registration_success = $state(false);
	let show_registration_failed = $state(false);
	let show_registration_failed_with_message = $state(false);
	let registration_failed_message = $state('');
	let is_registering = $state(false);
	let is_migrating = $state(false);

	let registration_allowed = $state(true);

	let migration_phases = $state([
		'creating_new_user',
		'migrating_templates',
		'migrating_logs',
		'migrating_files',
		'completed'
	]);

	let migration_phase = $state('');
	let migration_progress_total = $state(0);
	let migration_progress = $state(0);
	let migration_error_count = $state(0);

	let active_phase = $derived(
		// find the current phase in migration_phases
		migration_phases.indexOf(migration_phase)
	);

	onMount(() => {
		// if params error=440 or error=401, show toast
		if (window.location.search.includes('error=440')) {
			const toast = new bootstrap.Toast(document.getElementById('toastLoginExpired'));
			toast.show();
		} else if (window.location.search.includes('error=401')) {
			const toast = new bootstrap.Toast(document.getElementById('toastLoginInvalid'));
			toast.show();
		}

		// check if registration is allowed
		checkRegistrationAllowed();
	});

	function checkRegistrationAllowed() {
		axios
			.get(API_URL + '/users/isRegistrationAllowed')
			.then((response) => {
				registration_allowed = response.data.registration_allowed;
			})
			.catch((error) => {
				console.error('Error checking registration allowed:', error);
				registration_allowed = false; // Default to false if there's an error
			});
	}

	function handleMigrationProgress(username) {
		// Poll the server for migration progress
		const interval = setInterval(() => {
			axios
				.get(API_URL + '/users/migrationProgress', { params: { username } })
				.then((response) => {
					const progress = response.data.progress;
					if (progress) {
						migration_phase = progress.phase;
						migration_progress_total = progress.total_items;
						migration_progress = progress.processed_items;

						// Stop polling when migration is complete
						if (progress.phase === 'completed') {
							console.log('Migration completed successfully');
							is_migrating = false;
							migration_error_count = progress.error_count;
							clearInterval(interval);
						}
					}

					if (
						!response.data.migration_in_progress &&
						!response.data.progress.phase === 'not_started'
					) {
						console.log('Migration stopped');
						is_migrating = false;
						clearInterval(interval);
					}
				})
				.catch((error) => {
					console.error('Error fetching migration progress:', error);
					clearInterval(interval); // Stop polling on error
				});
		}, 500); // Poll every 500ms
	}

	function handleLogin(event) {
		event.preventDefault();

		show_login_failed = false;
		show_login_warning_empty_fields = false;
		is_migrating = false;

		const username = document.getElementById('loginUsername').value;
		const password = document.getElementById('loginPassword').value;

		if (username === '' || password === '') {
			show_login_warning_empty_fields = true;
			console.error('Please fill out all fields');
			return;
		}

		is_logging_in = true;

		axios
			.post(API_URL + '/users/login', { username, password })
			.then((response) => {
				if (response.data.migration_started) {
					is_migrating = true;

					handleMigrationProgress(response.data.username);
				} else {
					localStorage.setItem('user', JSON.stringify(response.data.username));
					goto('/write');
				}
			})
			.catch((error) => {
				console.log(error);
				if (error.response.status === 404) {
					show_login_failed = true;
				}
			})
			.finally(() => {
				is_logging_in = false;
			});
	}

	function handleRegister(event) {
		show_registration_warning_empty_fields = false;
		show_warning_passwords_do_not_match = false;
		show_registration_success = false;
		show_registration_failed = false;
		show_registration_failed_with_message = false;

		event.preventDefault();
		const username = document.getElementById('registerUsername').value;
		const password = document.getElementById('registerPassword').value;
		const password2 = document.getElementById('registerPassword2').value;

		if (username === '' || password === '') {
			show_registration_warning_empty_fields = true;
			console.error('Please fill out all fields');
			return;
		}

		if (password !== password2) {
			show_warning_passwords_do_not_match = true;
			console.error('Passwords do not match');
			return;
		}

		is_registering = true;

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
				console.error(error.response.data);
				registration_failed_message = error.response.data;
				show_registration_failed_with_message = true;
			})
			.finally(() => {
				is_registering = false;
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
							{#if is_migrating || migration_phase == 'completed'}
								<div class="alert alert-info" role="alert">
									Daten-Migration wurde gestartet. Dies kann einige Momente dauern.<br />
									{#if migration_phase !== 'completed'}
										<div class="text-bg-danger p-2 my-2 rounded">
											Währenddessen die Seite nicht neu laden und nicht neu einloggen!
										</div>
									{/if}

									Fortschritt:
									<div class="progress-item {active_phase >= 0 ? 'active' : ''}">
										<div class="d-flex">
											<div class="emoji">
												{#if active_phase <= 0}
													➡️
												{:else}
													✅
												{/if}
											</div>
											Account anlegen
										</div>
									</div>
									<div class="progress-item {active_phase >= 1 ? 'active' : ''}">
										<div class="d-flex">
											<div class="emoji">
												{#if active_phase <= 1}
													➡️
												{:else}
													✅
												{/if}
											</div>
											Vorlagen migrieren
										</div>
									</div>
									<div class="progress-item {active_phase >= 2 ? 'active' : ''}">
										<div class="d-flex">
											<div class="emoji">
												{#if active_phase <= 2}
													➡️
												{:else}
													✅
												{/if}
											</div>
											Einträge migrieren
										</div>

										{#if active_phase === 2}
											<div
												class="progress"
												role="progressbar"
												aria-label="Progress"
												aria-valuenow="0"
												aria-valuemin="0"
												aria-valuemax="100"
											>
												<div
													class="progress-bar"
													style="width: {(migration_progress / migration_progress_total) * 100}%"
												>
													{migration_progress}/{migration_progress_total}
												</div>
											</div>
										{/if}
									</div>
									<div class="progress-item {active_phase >= 3 ? 'active' : ''}">
										<div class="d-flex">
											<div class="emoji">
												{#if active_phase <= 3}
													➡️
												{:else}
													✅
												{/if}
											</div>
											Dateien migrieren
										</div>
										{#if active_phase === 3}
											<div
												class="progress"
												role="progressbar"
												aria-label="Progress"
												aria-valuenow="0"
												aria-valuemin="0"
												aria-valuemax="100"
											>
												<div
													class="progress-bar"
													style="width: {(migration_progress / migration_progress_total) * 100}%"
												>
													{migration_progress}/{migration_progress_total}
												</div>
											</div>
										{/if}
									</div>
									{#if migration_phase === 'completed'}
										{#if migration_error_count == 0}
											<div class="text-bg-success p-2 my-2 rounded">
												Migration wurde ohne erkannte Fehler abgeschlossen! Bitte Login erneut
												starten. <br />
												Prüfe anschließend, ob alle Daten korrekt migriert wurden.
											</div>
										{:else}
											<div class="text-bg-warning p-2 my-2 rounded">
												Migration wurde mit {migration_error_count} erkannten Fehlern abgeschlossen!
												Prüfe die Server-Logs für Details!<br />
												Falls der Login nicht funktioniert, oder die Daten fehlerhaft sind, so müssen
												die migrierten Daten händisch entfernt werden.
											</div>
										{/if}
									{/if}
								</div>
							{/if}
							{#if show_login_failed}
								<div class="alert alert-danger" role="alert">
									Login fehlgeschlagen!<br />
									Bitte Eingabedaten überprüfen.
								</div>
							{/if}
							{#if show_login_warning_empty_fields}
								<div class="alert alert-danger" role="alert">
									Eingabefelder dürfen nicht leer sein!
								</div>
							{/if}
							<div class="d-flex justify-content-center">
								<button type="submit" class="btn btn-primary" disabled={is_logging_in}>
									{#if is_logging_in || is_migrating}
										<div class="spinner-border spinner-border-sm" role="status">
											<span class="visually-hidden">Loading...</span>
										</div>
									{/if}
									Login
								</button>
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
									disabled={!registration_allowed}
									type="text"
									class="form-control"
									id="registerUsername"
									placeholder="Username"
								/>
								<label for="registerUsername">Username</label>
							</div>
							<div class="form-floating mb-3">
								<input
									disabled={!registration_allowed}
									type="password"
									class="form-control"
									id="registerPassword"
									placeholder="Password"
								/>
								<label for="registerPassword">Password</label>
							</div>
							<div class="form-floating mb-3">
								<input
									disabled={!registration_allowed}
									type="password"
									class="form-control"
									id="registerPassword2"
									placeholder="Password bestätigen"
								/>
								<label for="registerPassword2">Password bestätigen</label>
							</div>
							{#if !registration_allowed}
								<div class="alert alert-danger" role="alert">
									Registrierung ist derzeit nicht erlaubt!
								</div>
							{/if}
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
							{#if show_registration_warning_empty_fields}
								<div class="alert alert-danger" role="alert">
									Eingabefelder dürfen nicht leer sein!
								</div>
							{/if}
							{#if show_warning_passwords_do_not_match}
								<div class="alert alert-danger" role="alert">Passwörter stimmen nicht überein!</div>
							{/if}
							<div class="d-flex justify-content-center">
								<button
									type="submit"
									class="btn btn-primary"
									disabled={is_registering || !registration_allowed}
								>
									{#if is_registering}
										<div class="spinner-border spinner-border-sm" role="status">
											<span class="visually-hidden">Loading...</span>
										</div>
									{/if}
									Registrieren
								</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		</div>
	</div>

	<div class="toast-container position-fixed bottom-0 end-0 p-3">
		<div
			id="toastLoginExpired"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">Der Login ist abgelaufen. Bitte neu anmelden.</div>
			</div>
		</div>

		<div
			id="toastLoginInvalid"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">Authentifizierung fehlgeschlagen. Bitte neu anmelden.</div>
			</div>
		</div>
	</div>
</div>

<style>
	.progress-item {
		opacity: 0.5;
	}

	.progress-item.active {
		opacity: 1;
	}

	.progress-item .emoji {
		visibility: hidden;
	}

	.progress-item.active .emoji {
		visibility: visible;
	}

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

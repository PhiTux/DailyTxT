<script>
	import { getTranslate, getTolgee } from '@tolgee/svelte';
	import { API_URL } from '$lib/APIurl';
	import axios from 'axios';
	import { onDestroy, onMount } from 'svelte';
	import { slide } from 'svelte/transition';
	import { formatBytes } from '$lib/helpers';
	import * as bootstrap from 'bootstrap';

	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	let adminPassword = $state('');
	let isAdminAuthenticated = $state(false);

	let adminPasswordInput = $state(null);
	let currentUser = $state('');

	// Admin authentication state
	let isCheckingAdminAuth = $state(false);
	let adminAuthError = $state('');

	// Admin data
	let freeSpace = $state(0);
	let oldData = $state({});
	let users = $state([]);
	let appSettings = $state({});
	let isLoadingUsers = $state(false);
	let deleteUserId = $state(null);
	let isDeletingUser = $state(false);

	let confirmDeleteOldData = $state(false);
	let isDeletingOldData = $state(false);

	// Registration override controls
	let isOpeningRegistration = $state(false);
	let registrationAllowed = $state(false);
	let registrationAllowedTemporary = $state(false);
	let registrationUntil = $state('');
	let regStatusError = $state('');
	let regOpenSuccess = $state(false);
	let regOpenError = $state('');

	onMount(() => {
		currentUser = localStorage.getItem('user');
		resetAdminState();
		adminPasswordInput.focus();
	});

	onDestroy(() => {
		resetAdminState();
	});

	// Admin login function
	async function loginAsAdmin() {
		if (isCheckingAdminAuth || !adminPassword.trim()) return;

		isCheckingAdminAuth = true;
		adminAuthError = '';

		try {
			const response = await axios.post(API_URL + '/admin/validate-password', {
				password: adminPassword
			});

			if (response.data.valid) {
				isAdminAuthenticated = true;
				loadUsers(); // Load users immediately after successful login
			} else {
				adminAuthError = $t('settings.admin.invalid_password');
			}
		} catch (error) {
			adminAuthError = $t('settings.admin.login_error');
		} finally {
			isCheckingAdminAuth = false;
		}
	}

	// Function to make API calls with admin password
	async function makeAdminApiCall(endpoint, data = {}) {
		return axios.post(API_URL + endpoint, {
			...data,
			admin_password: adminPassword
		});
	}

	// Load all users with disk usage
	async function loadUsers() {
		if (isLoadingUsers || !isAdminAuthenticated) return;
		isLoadingUsers = true;

		try {
			const response = await makeAdminApiCall('/admin/get-data');
			users = response.data.users || [];
			freeSpace = response.data.free_space;
			oldData = response.data.old_data;
			appSettings = response.data.app_settings || {};

			// Also check registration status
			await checkRegistrationAllowed();
		} catch (error) {
			console.error('Error loading users:', error);
			if (error.response?.status === 401) {
				// Admin password became invalid, reset state
				resetAdminState();
			}
		} finally {
			isLoadingUsers = false;
		}
	}

	async function checkRegistrationAllowed() {
		regStatusError = '';
		try {
			const resp = await axios.get(API_URL + '/users/isRegistrationAllowed');
			registrationAllowed = !!resp.data.registration_allowed;
			registrationAllowedTemporary = !!resp.data.temporary_allowed;
			registrationUntil = resp.data.until || '';
		} catch (e) {
			regStatusError = $t('settings.admin.registration_status_error');
		}
	}

	async function openRegistrationTemporary() {
		if (isOpeningRegistration) return;
		isOpeningRegistration = true;

		regOpenSuccess = false;
		regOpenError = '';
		registrationUntil = '';
		try {
			const resp = await makeAdminApiCall('/admin/open-registration', { seconds: 300 });
			if (resp.data?.success) {
				regOpenSuccess = true;
				registrationUntil = resp.data.until || '';
				await checkRegistrationAllowed();
			} else {
				regOpenError = $t('settings.admin.registration_open_error');
			}
		} catch (e) {
			regOpenError = e?.response?.data || $t('settings.admin.registration_open_error');
		} finally {
			isOpeningRegistration = false;
		}
	}

	// Delete user
	async function deleteUser(userId, username) {
		if (isDeletingUser) return;
		isDeletingUser = true;

		try {
			const response = await makeAdminApiCall('/admin/delete-user', { user_id: userId });
			if (response.data.success) {
				users = users.filter((user) => user.id !== userId);
				deleteUserId = null;

				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastSuccessUserDelete'));
				toast.show();
			} else {
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorUserDelete'));
				toast.show();
			}
		} catch (error) {
			console.error('Error deleting user:', error);
			if (error.response?.status === 401) {
				resetAdminState();
			}
		} finally {
			isDeletingUser = false;
		}
	}

	// Reset admin authentication
	function resetAdminState() {
		isAdminAuthenticated = false;
		adminPassword = '';
		adminAuthError = '';
		users = [];
		appSettings = {};
		deleteUserId = null;
	}

	function confirmDeleteUser(userId) {
		deleteUserId = deleteUserId === userId ? null : userId;
	}

	// Delete old data directory
	async function deleteOldData() {
		if (isDeletingOldData) return;
		isDeletingOldData = true;

		try {
			const response = await makeAdminApiCall('/admin/delete-old-data');
			if (response.data.success) {
				// Reset old data state
				oldData = { exists: false };
				confirmDeleteOldData = false;

				// Show success toast
				const toast = new bootstrap.Toast(document.getElementById('toastSuccessOldDataDelete'));
				toast.show();
			} else {
				// Show error toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorOldDataDelete'));
				toast.show();
			}
		} catch (error) {
			console.error('Error deleting old data:', error);
			if (error.response?.status === 401) {
				resetAdminState();
			} else {
				// Show error toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorOldDataDelete'));
				toast.show();
			}
		} finally {
			isDeletingOldData = false;
		}
	}

	function toggleDeleteOldDataConfirmation() {
		confirmDeleteOldData = !confirmDeleteOldData;
	}
</script>

<div class="settings-admin">
	{#if !isAdminAuthenticated}
		<!-- Admin Login Form -->
		<div
			class="d-flex flex-column align-items-center justify-content-center"
			style="min-height: 400px;"
		>
			<div class="card" style="width: 100%; max-width: 400px;">
				<div class="card-body">
					<h4 class="card-title text-center mb-4">🔒 {$t('settings.admin.login_required')}</h4>
					<p class="card-text text-center text-muted mb-4">
						{$t('settings.admin.login_description')}
					</p>

					<form onsubmit={loginAsAdmin}>
						<div class="form-floating mb-3">
							<input
								bind:this={adminPasswordInput}
								type="password"
								class="form-control"
								id="adminPassword"
								placeholder={$t('settings.admin.password')}
								bind:value={adminPassword}
								disabled={isCheckingAdminAuth}
							/>
							<label for="adminPassword">{$t('settings.admin.password')}</label>
						</div>

						{#if adminAuthError}
							<div class="alert alert-danger" transition:slide>
								{adminAuthError}
							</div>
						{/if}

						<button
							type="submit"
							class="btn btn-primary w-100"
							disabled={isCheckingAdminAuth || !adminPassword.trim()}
						>
							{#if isCheckingAdminAuth}
								<span class="spinner-border spinner-border-sm me-2"></span>
							{/if}
							{$t('settings.admin.check_password')}
						</button>
					</form>
				</div>
			</div>
		</div>
	{:else}
		<!-- Admin Panel Content -->
		<div class="admin-authenticated" transition:slide>
			<h2 class="mb-4">⚙️ {$t('settings.admin.title')}</h2>

			<!-- Admin status bar -->
			<div
				class="d-flex align-items-center mb-4 p-3 alert alert-success border border-success rounded-4"
			>
				<span class="text-success me-3">🔓 {$t('settings.admin.authorized')} </span>
				<button class="btn btn-outline-secondary btn-sm ms-2" onclick={resetAdminState}>
					{$t('settings.admin.logout')}
				</button>
			</div>

			<!-- Registration Override Card -->
			{#if !registrationAllowed}
				<div class="card mt-4">
					<div class="card-header">
						<h4 class="card-title mb-0">📝 {$t('settings.admin.registration')}</h4>
					</div>
					<div class="card-body">
						<p class="text-muted mb-3">
							{$t('settings.admin.registration_description')}
						</p>

						<div class="d-flex align-items-center gap-3 mb-3">
							<div>
								<strong>{$t('settings.admin.current_status')}: </strong>
								<span
									class="badge {registrationAllowed || registrationAllowedTemporary
										? 'bg-success'
										: 'bg-danger'}"
								>
									{registrationAllowed || registrationAllowedTemporary
										? $t('settings.admin.registration_allowed')
										: $t('settings.admin.registration_blocked')}
								</span>
							</div>
							<button
								class="btn btn-outline-primary"
								onclick={openRegistrationTemporary}
								disabled={isOpeningRegistration}
							>
								{#if isOpeningRegistration}
									<span class="spinner-border spinner-border-sm me-2"></span>
								{/if}
								{$t('settings.admin.button_open_5_minutes')}
							</button>
							<button class="btn btn-outline-secondary" onclick={checkRegistrationAllowed}>
								{$t('settings.admin.button_refresh_status')}
							</button>
						</div>

						{#if registrationAllowedTemporary && registrationUntil}
							<div class="alert alert-success rounded-4" transition:slide>
								{@html $t('settings.admin.registration_allowed_until', {
									date_and_time: new Date(registrationUntil).toLocaleString($tolgee.getLanguage(), {
										year: 'numeric',
										month: 'numeric',
										day: 'numeric',
										hour: 'numeric',
										minute: 'numeric',
										second: 'numeric'
									})
								})}
							</div>
						{/if}

						{#if regStatusError}
							<div class="alert alert-warning" transition:slide>{regStatusError}</div>
						{/if}
						{#if regOpenError}
							<div class="alert alert-danger" transition:slide>{regOpenError}</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- User management card -->
			<div class="card mt-4">
				<div class="card-header">
					<h4 class="card-title mb-0">👥 {$t('settings.admin.user_management')}</h4>
				</div>
				<div class="card-body">
					{#if isLoadingUsers}
						<div class="text-center p-4">
							<div class="spinner-border" role="status">
								<span class="visually-hidden">Loading...</span>
							</div>
							<p class="mt-2 text-muted">{$t('settings.admin.loading_users')}</p>
						</div>
					{:else if users.length === 0}
						<p class="text-muted">{$t('settings.admin.no_users')}</p>
					{:else}
						<div class="table-responsive">
							<table class="table table-striped">
								<thead>
									<tr>
										<th>{$t('settings.admin.id')}</th>
										<th>{$t('settings.admin.username')}</th>
										<th>{$t('settings.admin.disk_usage')}</th>
										<th>{$t('settings.admin.delete_account')}</th>
									</tr>
								</thead>
								<tbody>
									{#each users as user}
										<tr>
											<td>{user.id}</td>
											<td>
												<strong>{user.username}</strong>
												{#if user.username === currentUser}
													<span class="badge bg-info text-dark ms-2">
														{$t('settings.admin.me')}
													</span>
												{/if}
											</td>
											<td>{formatBytes(user.disk_usage || 0)}</td>
											<td>
												<button
													class="btn btn-danger btn-sm"
													onclick={() => confirmDeleteUser(user.id)}
												>
													🗑️ {$t('settings.admin.delete')}
												</button>
												{#if user.username === currentUser}
													<div class="form-text text-muted">
														{$t('settings.admin.warning_delete_self')}
													</div>
												{/if}

												{#if deleteUserId === user.id}
													<div transition:slide>
														<div class="pt-2">
															<div class="alert alert-danger mb-0">
																<p class="mb-2">
																	<strong>{$t('settings.admin.confirm_delete')}</strong><br />
																	{$t('settings.admin.delete_warning')}
																</p>
																<div class="d-flex gap-2">
																	<button
																		class="btn btn-secondary btn-sm"
																		onclick={() => (deleteUserId = null)}
																	>
																		{$t('settings.admin.cancel')}
																	</button>
																	<button
																		class="btn btn-danger btn-sm"
																		onclick={() => deleteUser(user.id, user.username)}
																		disabled={isDeletingUser}
																	>
																		{#if isDeletingUser}
																			<span class="spinner-border spinner-border-sm me-1"></span>
																		{/if}
																		{$t('settings.admin.delete')}
																	</button>
																</div>
															</div>
														</div>
													</div>
												{/if}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>

						<!-- Summary -->
						<div class="mt-3">
							<div class="row">
								<div class="col-md-6">
									<strong>{$t('settings.admin.total_users')}: </strong>
									{users.length}
								</div>
								<div class="col-md-6">
									<strong>{$t('settings.admin.total_disk_usage')}: </strong>
									{formatBytes(users.reduce((sum, user) => sum + (user.disk_usage || 0), 0))}
								</div>
							</div>
							<div class="row">
								<div class="col-md-6"></div>
								<div class="col-md-6">
									<strong>{$t('settings.admin.free_disk_space')}: </strong>
									{formatBytes(freeSpace)}
								</div>
							</div>
						</div>
					{/if}

					<div class="mt-4 d-flex justify-content-center">
						<button class="btn btn-outline-primary" onclick={loadUsers} disabled={isLoadingUsers}>
							{#if isLoadingUsers}
								<span class="spinner-border spinner-border-sm me-2"></span>
							{/if}
							{$t('settings.admin.refresh_users')}
						</button>
					</div>
				</div>
			</div>

			<!-- Old Data Card -->
			{#if oldData.exists}
				<div class="card mt-4">
					<div class="card-header">
						<h4 class="card-title mb-0">📦 {$t('settings.admin.old_data')}</h4>
					</div>
					<div class="card-body">
						<p class="text-muted mb-3">
							{@html $t('settings.admin.old_data_description')}
						</p>

						{#if oldData.usernames && oldData.usernames.length > 0}
							<h6>{$t('settings.admin.old_users')}:</h6>
							<div class="mb-3">
								{#each oldData.usernames as username, index}
									<span class="badge bg-secondary me-1">{username}</span>
								{/each}
							</div>
						{:else}
							<p class="text-warning">
								{$t('settings.admin.no_old_users_found')}
							</p>
						{/if}

						<div class="row">
							<div class="col-md-6">
								<strong>{@html $t('settings.admin.old_data_size')}: </strong>
								{formatBytes(oldData.total_size)}
							</div>
						</div>

						<!-- Delete old data button -->
						<div class="mt-3">
							<button
								class="btn btn-danger"
								onclick={toggleDeleteOldDataConfirmation}
								disabled={isDeletingOldData}
							>
								🗑️ {$t('settings.admin.delete_old_data')}
							</button>

							{#if confirmDeleteOldData}
								<div transition:slide class="">
									<div class="pt-3">
										<div class="alert alert-danger mb-0">
											<p class="mb-2">
												<strong>{$t('settings.admin.confirm_delete_old_data')}</strong><br />
												{@html $t('settings.admin.delete_old_data_warning')}
											</p>
											<div class="d-flex gap-2">
												<button
													class="btn btn-secondary btn-sm"
													onclick={toggleDeleteOldDataConfirmation}
												>
													{$t('settings.admin.cancel')}
												</button>
												<button
													class="btn btn-danger btn-sm"
													onclick={deleteOldData}
													disabled={isDeletingOldData}
												>
													{#if isDeletingOldData}
														<span class="spinner-border spinner-border-sm me-1"></span>
													{/if}
													{$t('settings.admin.delete')}
												</button>
											</div>
										</div>
									</div>
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/if}

			<!-- App Settings Card / Environment Variables -->
			<div class="card mt-4">
				<div class="card-header">
					<h4 class="card-title mb-0">⚙️ {$t('settings.admin.environment_variables')}</h4>
				</div>
				<div class="card-body">
					<p class="text-muted mb-3">
						{$t('settings.admin.environment_variables_description')}
					</p>

					{#if Object.keys(appSettings).length > 0}
						<div class="list-group list-group-flush">
							{#each Object.entries(appSettings) as [key, value]}
								<div class="list-group-item px-0 py-2">
									<div class="row">
										<div class="col-4">
											<span class="fw-bold text-muted">{key}:</span>
										</div>
										<div class="col-8">
											<span class="font-monospace">
												{#if Array.isArray(value)}
													{JSON.stringify(value)}
												{:else if typeof value === 'boolean'}
													<span class="badge {value ? 'bg-success' : 'bg-danger'}">
														{value ? 'true' : 'false'}
													</span>
												{:else if key === 'secret_token' && value === ''}
													<span class="text-muted fst-italic"
														>{$t('settings.admin.hidden_for_security')}</span
													>
												{:else}
													{value}
												{/if}
											</span>
										</div>
									</div>
								</div>
							{/each}
						</div>
					{:else}
						<p class="text-muted">{$t('settings.admin.no_environment_variables')}</p>
					{/if}
				</div>
			</div>
		</div>
	{/if}

	<div class="toast-container position-fixed bottom-0 end-0 p-3">
		<div
			id="toastSuccessUserDelete"
			class="toast align-items-center text-bg-success"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('settings.statistics.toast_success_user_delete')}
				</div>
			</div>
		</div>

		<div
			id="toastErrorUserDelete"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('settings.statistics.toast_error_user_delete')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastSuccessOldDataDelete"
			class="toast align-items-center text-bg-success"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('settings.admin.toast_success_old_data_delete')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>

		<div
			id="toastErrorOldDataDelete"
			class="toast align-items-center text-bg-danger"
			role="alert"
			aria-live="assertive"
			aria-atomic="true"
		>
			<div class="d-flex">
				<div class="toast-body">
					{$t('settings.admin.toast_error_old_data_delete')}
				</div>
				<button
					type="button"
					class="btn-close me-2 m-auto"
					data-bs-dismiss="toast"
					aria-label="Close"
				></button>
			</div>
		</div>
	</div>
</div>

<style>
	.settings-admin {
		min-height: 65vh;
	}

	.table th {
		background-color: rgba(13, 110, 253, 0.1);
	}

	.admin-authenticated {
		max-width: 100%;
	}

	.card-title {
		color: var(--bs-primary);
	}
</style>

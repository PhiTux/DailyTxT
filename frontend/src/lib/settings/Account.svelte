<script>
	import { slide } from 'svelte/transition';

	let {
		currentUser,
		newUsername = $bindable(),
		changeUsernamePassword = $bindable(),
		isChangingUsername,
		changeUsernameSuccess,
		changeUsernamePasswordIncorrect,
		changeUsernameError,
		deleteAccountPassword = $bindable(),
		isDeletingAccount,
		deleteAccountPasswordIncorrect,
		showDeleteAccountSuccess,
		showConfirmDeleteAccount,
		changeUsername,
		deleteAccount
	} = $props();

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
</script>

<h3 class="text-primary">👤 {$t('settings.account')}</h3>

<div>
	<h5>{$t('settings.change_username')}</h5>
	<div class="form-text">
		{@html $t('settings.change_username.description')}
	</div>
	<div class="mb-3">
		{$t('settings.change_username.current_username')}: {currentUser}
	</div>

	<form onsubmit={changeUsername}>
		<div class="form-floating mb-3">
			<input
				type="text"
				class="form-control"
				id="newUsername"
				placeholder={$t('settings.change_username.new_username')}
				bind:value={newUsername}
				disabled={isChangingUsername}
			/>
			<label for="newUsername">{$t('settings.change_username.new_username')}</label>
		</div>
		<div class="form-floating mb-3">
			<input
				type="password"
				class="form-control"
				id="changeUsernamePassword"
				placeholder={$t('settings.password.current_password')}
				bind:value={changeUsernamePassword}
				disabled={isChangingUsername}
			/>
			<label for="changeUsernamePassword">{$t('settings.password.current_password')}</label>
		</div>
		<button
			class="btn btn-primary"
			onclick={changeUsername}
			disabled={isChangingUsername || !newUsername.trim() || !changeUsernamePassword.trim()}
		>
			{#if isChangingUsername}
				<span class="spinner-border spinner-border-sm me-2"></span>
			{/if}
			{$t('settings.change_username.button')}
		</button>
	</form>

	{#if changeUsernameSuccess}
		<div class="alert alert-success mt-2" role="alert" transition:slide>
			{$t('settings.change_username.success')}
		</div>
	{/if}
	{#if changeUsernamePasswordIncorrect}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.password.current_password_incorrect')}
		</div>
	{:else if changeUsernameError}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{changeUsernameError}
		</div>
	{/if}
</div>

<div>
	<h5>{$t('settings.delete_account')}</h5>
	<p>
		{$t('settings.delete_account.description')}
	</p>
	<form
		onsubmit={() => {
			if (deleteAccountPassword.trim() === '') {
				return;
			} else {
				showConfirmDeleteAccount = true;
			}
		}}
	>
		<div class="form-floating mb-3">
			<input
				type="password"
				class="form-control"
				id="currentPassword"
				placeholder={$t('settings.password.current_password')}
				bind:value={deleteAccountPassword}
			/>
			<label for="currentPassword">{$t('settings.password.confirm_password')}</label>
		</div>
		<button
			class="btn btn-danger {deleteAccountPassword.trim() === '' ? 'disabled' : ''}"
			onclick={() => {
				if (deleteAccountPassword.trim() === '') {
					return;
				} else {
					showConfirmDeleteAccount = true;
				}
			}}
			data-sveltekit-noscroll
		>
			{$t('settings.delete_account.delete_button')}
			{#if isDeletingAccount}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="spinner-border" role="status">
					<span class="visually-hidden">Loading...</span>
				</div>
			{/if}
		</button>
	</form>
	{#if showDeleteAccountSuccess}
		<div class="alert alert-success mt-2" role="alert" transition:slide>
			{@html $t('settings.delete_account.success')}
		</div>
	{/if}
	{#if deleteAccountPasswordIncorrect}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.delete_account.password_incorrect')}
		</div>
	{/if}
	{#if showConfirmDeleteAccount}
		<div transition:slide>
			<div class="pt-2">
				<div class="alert alert-danger mb-0" role="alert">
					{$t('settings.delete_account.confirm')}

					<div class="d-flex flex-row mt-2">
						<button
							class="btn btn-secondary"
							onclick={() => {
								showConfirmDeleteAccount = false;
								deleteAccountPassword = '';
							}}>{$t('settings.abort')}</button
						>
						<button class="btn btn-danger ms-3" onclick={deleteAccount} disabled={isDeletingAccount}
							>{$t('settings.delete_account.confirm_button')}
							{#if isDeletingAccount}
								<span class="spinner-border spinner-border-sm ms-2" role="status" aria-hidden="true"
								></span>
							{/if}
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

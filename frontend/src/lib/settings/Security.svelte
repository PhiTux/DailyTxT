<script>
	import { slide } from 'svelte/transition';
	import { Fa } from 'svelte-fa';
	import { faCopy, faCheck } from '@fortawesome/free-solid-svg-icons';
	import { settings, tempSettings } from '$lib/settingsStore.js';

	let {
		unsavedChanges,
		currentPassword = $bindable(),
		newPassword = $bindable(),
		confirmNewPassword = $bindable(),
		isChangingPassword,
		changePasswordNotEqual,
		changingPasswordSuccess,
		changingPasswordIncorrect,
		changingPasswordError,
		backupCodesPassword = $bindable(),
		isGeneratingBackupCodes,
		backupCodes,
		showBackupCodesError,
		codesCopiedSuccess,
		changePassword,
		createBackupCodes,
		copyBackupCodes
	} = $props();

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
</script>

<h3 class="text-primary">🔒 {$t('settings.security')}</h3>
<div>
	<h5>{$t('settings.security.change_password')}</h5>
	<form onsubmit={changePassword}>
		<div class="form-floating mb-3">
			<input
				type="password"
				class="form-control"
				id="currentPassword"
				placeholder={$t('settings.password.current_password')}
				bind:value={currentPassword}
			/>
			<label for="currentPassword">{$t('settings.password.current_password')}</label>
		</div>
		<div class="form-floating mb-3">
			<input
				type="password"
				class="form-control"
				id="newPassword"
				placeholder={$t('settings.password.new_password')}
				bind:value={newPassword}
			/>
			<label for="newPassword">{$t('settings.password.new_password')}</label>
		</div>
		<div class="form-floating mb-3">
			<input
				type="password"
				class="form-control"
				id="confirmNewPassword"
				placeholder={$t('settings.password.confirm_new_password')}
				bind:value={confirmNewPassword}
			/>
			<label for="confirmNewPassword">{$t('settings.password.confirm_new_password')}</label>
		</div>
		<button
			class="btn btn-primary"
			disabled={!currentPassword || !newPassword || !confirmNewPassword}
			onclick={changePassword}
		>
			{#if isChangingPassword}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="spinner-border" role="status">
					<span class="visually-hidden">Loading...</span>
				</div>
			{/if}
			{$t('settings.password.change_password_button')}
		</button>
	</form>
	{#if changePasswordNotEqual}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.password.passwords_dont_match')}
		</div>
	{/if}
	{#if changingPasswordSuccess}
		<div class="alert alert-success mt-2" role="alert" transition:slide>
			{$t('settings.password.success')}
		</div>
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.password.success_backup_codes_warning')}
		</div>
	{/if}
	{#if changingPasswordIncorrect}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.password.current_password_incorrect')}
		</div>
	{:else if changingPasswordError}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.password.change_error')}
		</div>
	{/if}
</div>
<div>
	<h5>{$t('settings.backup_codes')}</h5>
	<ul>
		{@html $t('settings.backup_codes.description')}
	</ul>

	<form onsubmit={createBackupCodes}>
		<div class="form-floating mb-3">
			<input
				type="password"
				class="form-control"
				id="currentPassword"
				placeholder={$t('settings.password.current_password')}
				bind:value={backupCodesPassword}
			/>
			<label for="currentPassword">{$t('settings.password.confirm_password')}</label>
		</div>
		<button
			class="btn btn-primary"
			onclick={createBackupCodes}
			data-sveltekit-noscroll
			disabled={isGeneratingBackupCodes || !backupCodesPassword.trim()}
		>
			{$t('settings.backup_codes.generate_button')}
			{#if isGeneratingBackupCodes}
				<div class="spinner-border spinner-border-sm" role="status">
					<span class="visually-hidden">Loading...</span>
				</div>
			{/if}
		</button>
	</form>
	{#if backupCodes.length > 0}
		<div transition:slide>
			<div class="pt-3">
				<div class="alert alert-success alert-dismissible mb-0">
					{@html $t('settings.backup_codes.success')}

					<button class="btn btn-secondary my-2" onclick={copyBackupCodes}>
						<Fa icon={codesCopiedSuccess ? faCheck : faCopy} />
						{$t('settings.backup_codes.copy_button')}
					</button>
					<ul class="list-group">
						{#each backupCodes as code}
							<li class="list-group-item backupCode">
								<code>{code}</code>
							</li>
						{/each}
					</ul>
				</div>
			</div>
		</div>
	{/if}
	{#if showBackupCodesError}
		<div class="alert alert-danger mt-2" role="alert" transition:slide>
			{$t('settings.backup_codes.error')}
		</div>
	{/if}
</div>
<div id="loginonreload">
	{#if $tempSettings.requirePasswordOnPageLoad !== $settings.requirePasswordOnPageLoad}
		{@render unsavedChanges()}
	{/if}

	<h5>{$t('settings.reauth.title')}</h5>
	{$t('settings.reauth.description')}

	<div class="form-check form-switch mt-2">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.requirePasswordOnPageLoad}
			type="checkbox"
			role="switch"
			id="requirePasswordOnPageLoadSwitch"
		/>
		<label class="form-check-label" for="requirePasswordOnPageLoadSwitch">
			{$t('settings.reauth.label')}
		</label>
	</div>
</div>

<script>
	import donate from '$lib/assets/bmc-button.png';
	import github from '$lib/assets/GitHub-Logo.png';
	import { settings, tempSettings } from '$lib/settingsStore.js';
	import { Fa } from 'svelte-fa';
	import { faCircleUp } from '@fortawesome/free-solid-svg-icons';

	let {
		current_version,
		latest_stable_version,
		latest_overall_version,
		updateAvailable,
		checkChangelog,
		showInstallationHelp,
		unsavedChanges
	} = $props();

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();
</script>

<h3 class="text-primary">💡 {$t('settings.about')}</h3>

{#if showInstallationHelp}
	<span>
		<div class="alert alert-info rounded-4 mt-3">
			{@html $t('settings.installation_help')}
		</div>
	</span>
{/if}

<span class="d-table mx-auto"
	>{@html $t('settings.about.made_by', {
		creator:
			'<a class="link-light link-underline link-underline-opacity-0 link-underline-opacity-75-hover" href="https://github.com/PhiTux" target="_blank">PhiTux / Marco Kümmel</a>'
	})}</span
>
<hr />

<u>{$t('settings.about.current_version')}:</u>
<b>{current_version}</b><br />
<u>{$t('settings.about.latest_version')}:</u>
{#if !updateAvailable}
	<b>{$settings.includeTestVersions ? latest_overall_version : latest_stable_version}</b>
{:else}
	<a href="https://hub.docker.com/r/phitux/dailytxt/tags" target="_blank"
		><span class="badge text-bg-info fs-6"
			>{$settings.includeTestVersions ? latest_overall_version : latest_stable_version}</span
		></a
	>
{/if}

<br />

{#if updateAvailable}
	<p class="alert alert-info rounded-4 d-flex align-items-center mt-2 mb-2 p-2">
		<Fa icon={faCircleUp} size="2x" class="text-info me-2" />
		{$t('settings.about.update_available')}
	</p>
{/if}

<span class="form-text">{$t('settings.about.version_info')}</span><br />

<button type="button" class="btn btn-secondary" onclick={() => checkChangelog(true)}>
	{$t('settings.about.changelog')}
</button>

<div id="updateSettings" class="mt-2">
	{#if $tempSettings.checkForUpdates !== $settings.checkForUpdates || $tempSettings.includeTestVersions !== $settings.includeTestVersions}
		{@render unsavedChanges()}
	{/if}

	<h5>{$t('settings.about.update_notification')}</h5>
	<div class="form-check form-switch">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.checkForUpdates}
			type="checkbox"
			role="switch"
			id="checkForUpdatesSwitch"
		/>
		<label class="form-check-label" for="checkForUpdatesSwitch">
			{$t('settings.updates.check_for_updates')}
		</label>
	</div>

	<div class="form-check form-switch ms-3">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.includeTestVersions}
			type="checkbox"
			role="switch"
			id="includeTestVersionsSwitch"
			disabled={!$tempSettings.checkForUpdates}
		/>
		<label class="form-check-label" for="includeTestVersionsSwitch">
			{$t('settings.updates.include_test_versions')}
		</label>
	</div>
</div>

<hr />

<a
	class="btn btn-secondary mx-auto d-table"
	href="https://github.com/PhiTux/DailyTxT"
	target="_blank"
>
	{$t('settings.about.source_code')}: <img src={github} alt="" width="100px" />
</a>

<hr />

<span class="d-table mx-auto text-center">{@html $t('settings.about.donate')}</span>
<a
	class="d-block mx-auto mt-2"
	href="https://www.buymeacoffee.com/PhiTux"
	target="_blank"
	style="width: 200px;"
>
	<img src={donate} alt="" width="200px" />
</a>

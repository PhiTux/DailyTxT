<script>
	import { settings, tempSettings, autoLoadImagesThisDevice } from '$lib/settingsStore.js';
	import { loadFlagEmoji } from '$lib/helpers.js';
	import SelectTimezone from '$lib/SelectTimezone.svelte';
	import { slide } from 'svelte/transition';

	import { getTranslate, getTolgee } from '@tolgee/svelte';
	const { t } = getTranslate();
	const tolgee = getTolgee(['language']);

	let {
		unsavedChanges,
		aLookBackYears = $bindable(),
		tolgeesMatchForBrowserLanguage,
		aLookBackYearsInvalid
	} = $props();
</script>

<h3 class="text-primary">🛠️ {$t('settings.features')}</h3>

<div id="autoLoadImages">
	{#if $tempSettings.setAutoloadImagesPerDevice !== $settings.setAutoloadImagesPerDevice || $tempSettings.autoloadImagesByDefault !== $settings.autoloadImagesByDefault}
		{@render unsavedChanges()}
	{/if}

	<h5>{$t('settings.images_title')}</h5>
	{@html $t('settings.images_description')}

	<div class="form-check form-switch">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.setAutoloadImagesPerDevice}
			type="checkbox"
			role="switch"
			id="setImageLoadingPerDeviceSwitch"
		/>
		<label class="form-check-label" for="setImageLoadingPerDeviceSwitch">
			{$t('settings.images_loading_per_device')}
		</label>
	</div>

	<div class="form-check form-switch ms-3">
		<input
			class="form-check-input"
			bind:checked={$autoLoadImagesThisDevice}
			type="checkbox"
			role="switch"
			id="loadImagesThisDeviceSwitch"
			disabled={!$tempSettings.setAutoloadImagesPerDevice}
		/>
		<label class="form-check-label" for="loadImagesThisDeviceSwitch">
			{@html $t('settings.images_loading_this_device')}
		</label>
	</div>

	<div class="form-check form-switch mt-3">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.autoloadImagesByDefault}
			type="checkbox"
			role="switch"
			id="autoLoadImagesSwitch"
			disabled={$tempSettings.setAutoloadImagesPerDevice}
		/>
		<label class="form-check-label" for="autoLoadImagesSwitch">
			{$t('settings.images_loading_default')}
		</label>
	</div>
</div>

<div id="language">
	{#if $tempSettings.useBrowserLanguage !== $settings.useBrowserLanguage || $tempSettings.language !== $settings.language}
		{@render unsavedChanges()}
	{/if}
	<h5>🌐 {$t('settings.language')}</h5>
	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="language"
			id="language_auto"
			value={true}
			bind:group={$tempSettings.useBrowserLanguage}
		/>
		<label class="form-check-label" for="language_auto">
			{$t('settings.language_auto_detect')}
			<code>{window.navigator.language}</code>
			{#if tolgeesMatchForBrowserLanguage() !== '' && tolgeesMatchForBrowserLanguage() !== window.navigator.language}
				➔ <code>{tolgeesMatchForBrowserLanguage()}</code>
				{$t('settings.language_X_used')}
			{/if}
		</label>
		{#if $tempSettings.useBrowserLanguage && tolgeesMatchForBrowserLanguage() === ''}
			<div
				transition:slide
				disabled={!$settings.useBrowserLanguage}
				class="alert alert-danger"
				role="alert"
			>
				{@html $t('settings.language_not_available', {
					browserLanguage: window.navigator.language,
					defaultLanguage: $tolgee.getInitialOptions().defaultLanguage
				})}
			</div>
		{/if}
	</div>
	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="language"
			id="language_manual"
			value={false}
			bind:group={$tempSettings.useBrowserLanguage}
		/>
		<label class="form-check-label" for="language_manual">
			{$t('settings.set_language_manually')}
			{#if !$tempSettings.useBrowserLanguage}
				<div transition:slide>
					<select
						class="form-select"
						bind:value={$tempSettings.language}
						disabled={$tempSettings.useBrowserLanguage}
					>
						{#each $tolgee.getInitialOptions().availableLanguages as lang}
							{#if lang !== 'no'}
								<option value={lang}>{loadFlagEmoji(lang)} {lang}</option>
							{/if}
						{/each}
					</select>
				</div>
			{/if}
		</label>
	</div>
	<div class="form-text">
		{$t('settings.language.reload_info')}
	</div>
	<div class="alert alert-info rounded-4 mt-2" role="alert">
		{$t('settings.language.help_translate')}
		<a href="https://github.com/PhiTux/DailyTxT/blob/main/TRANSLATION.md" target="_blank">GitHub</a>
	</div>
</div>
<div id="timezone">
	{#if $tempSettings.useBrowserTimezone !== $settings.useBrowserTimezone || ($tempSettings.timezone !== undefined && $tempSettings.timezone?.value !== $settings.timezone?.value)}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.timezone')}</h5>
	{$t('settings.timezone.description', { written_on: $t('log.written_on') })}

	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="timezone"
			id="timezone1"
			value={true}
			bind:group={$tempSettings.useBrowserTimezone}
		/>
		<label class="form-check-label" for="timezone1">
			{@html $t('settings.timezone.auto_detect')}
			<code>{new Intl.DateTimeFormat().resolvedOptions().timeZone}</code>
		</label>
	</div>
	<div class="form-check">
		<input
			class="form-check-input"
			type="radio"
			name="timezone"
			id="timezone2"
			value={false}
			bind:group={$tempSettings.useBrowserTimezone}
		/>
		<label class="form-check-label" for="timezone2">
			{$t('settings.timezone.manual')}
		</label>
		<br />
		<SelectTimezone />
		{#if !$tempSettings.useBrowserTimezone}
			<div transition:slide>
				<span>
					{$t('settings.timezone.selected')} <code>{$tempSettings.timezone}</code>
				</span>
			</div>
		{/if}
	</div>

	<div class="form-text mt-3">
		{@html $t('settings.timezone.help_text')}
	</div>
</div>

<div id="FirstDayOfWeek">
	{#if $tempSettings.firstDayOfWeek !== $settings.firstDayOfWeek}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.first_day_of_week')}</h5>
	<div class="form-text mt-2">
		{$t('settings.first_day_of_week.help_text')}
	</div>
	<select class="form-select w-auto" bind:value={$tempSettings.firstDayOfWeek}>
		<option value="sunday">{$t('weekdays.sunday')}</option>
		<option value="monday">{$t('weekdays.monday')}</option>
	</select>
</div>

<div id="writeDateFormat">
	{#if $tempSettings.writeDateFormat !== $settings.writeDateFormat}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.writeDateFormat')}</h5>
	{$t('settings.writeDateFormat.description')}<br />
	<u class="mt-2">{$t('settings.writeDateFormat.month')}:</u>
	<select class="form-select w-auto mt-2" bind:value={$tempSettings.writeDateFormat}>
		<option value="2-digit"
			>{$t('settings.writeDateFormat.2-digit', {
				example: new Date().toLocaleDateString($tolgee.getLanguage(), {
					day: '2-digit',
					month: '2-digit',
					year: 'numeric',
					timeZone: 'UTC'
				})
			})}</option
		>
		<option value="long">
			{$t('settings.writeDateFormat.long', {
				example: new Date().toLocaleDateString($tolgee.getLanguage(), {
					day: 'numeric',
					month: 'long',
					year: 'numeric',
					timeZone: 'UTC'
				})
			})}
		</option></select
	>
</div>

<div id="aLookBack">
	{#if $tempSettings.useALookBack !== $settings.useALookBack || JSON.stringify(aLookBackYears
				.trim()
				.split(',')
				.map((year) => parseInt(year.trim()))) !== JSON.stringify($settings.aLookBackYears)}
		{@render unsavedChanges()}
	{/if}

	<h5>{$t('settings.aLookBack')}</h5>
	<ul>
		{@html $t('settings.aLookBack.description')}
	</ul>
	<div class="form-check form-switch">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.useALookBack}
			type="checkbox"
			role="switch"
			id="useALookBackSwitch"
		/>
		<label class="form-check-label" for="useALookBackSwitch">
			{$t('settings.ALookBack.useIt')}
		</label>
	</div>

	<div>
		<input
			type="text"
			id="useALookBackYears"
			class="form-control {aLookBackYearsInvalid ? 'is-invalid' : ''}"
			aria-describedby="useALookBackHelpBlock"
			disabled={!$tempSettings.useALookBack}
			placeholder={$t('settings.ALookBack.input_placeholder')}
			bind:value={aLookBackYears}
			invalid
		/>
		{#if aLookBackYearsInvalid}
			<div class="alert alert-danger mt-2" role="alert" transition:slide>
				{$t('settings.ALookBack.invalid_input')}
			</div>
		{/if}
		<div id="useALookBackHelpBlock" class="form-text">
			{@html $t('settings.ALookBack.help_text')}
		</div>
	</div>
</div>

<div id="showChangelogOnUpdate">
	{#if $tempSettings.showChangelogOnUpdate !== $settings.showChangelogOnUpdate}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.show_changelog_on_update')}</h5>
	<div class="form-check form-switch">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.showChangelogOnUpdate}
			type="checkbox"
			role="switch"
			id="showChangelogOnUpdateSwitch"
		/>
		<label class="form-check-label" for="showChangelogOnUpdateSwitch">
			{$t('settings.show_changelog_on_update.description')}
		</label>
	</div>
</div>

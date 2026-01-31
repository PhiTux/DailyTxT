<script>
	import { slide } from 'svelte/transition';
	import { Fa } from 'svelte-fa';
	import { faSun, faMoon } from '@fortawesome/free-solid-svg-icons';
	import { settings, tempSettings } from '$lib/settingsStore.js';

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();

	let { unsavedChanges } = $props();
</script>

<h3 class="text-primary">🎨 {$t('settings.appearance')}</h3>
<div id="lightdark">
	{#if $tempSettings.darkModeAutoDetect !== $settings.darkModeAutoDetect || $tempSettings.useDarkMode !== $settings.useDarkMode}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.light_dark_mode')}</h5>
	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="darkMode"
			id="darkModeAutoTrue"
			value={true}
			bind:group={$tempSettings.darkModeAutoDetect}
		/>
		<label class="form-check-label" for="darkModeAutoTrue">
			{$t('settings.light_dark_auto_detect')}
			{#if window.matchMedia('(prefers-color-scheme: dark)').matches}
				<b><u>{$t('settings.dark_mode')} <Fa icon={faMoon} /></u></b>
			{:else}
				<b><u>{$t('settings.light_mode')} <Fa icon={faSun} /></u></b>
			{/if}
		</label>
	</div>
	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="darkMode"
			id="darkModeAutoFalse"
			value={false}
			bind:group={$tempSettings.darkModeAutoDetect}
		/>
		<label class="form-check-label" for="darkModeAutoFalse">
			{$t('settings.light_dark_manual')}
		</label>
		{#if $tempSettings.darkModeAutoDetect === false}
			<div class="form-check form-switch d-flex flex-row ps-0" transition:slide>
				<label class="form-check-label me-5" for="darkModeSwitch">
					<Fa icon={faSun} />
				</label>
				<input
					class="form-check-input"
					bind:checked={$tempSettings.useDarkMode}
					type="checkbox"
					role="switch"
					id="darkModeSwitch"
				/>
				<label class="form-check-label ms-2" for="darkModeSwitch">
					<Fa icon={faMoon} />
				</label>
			</div>
		{/if}
	</div>
</div>
<div id="background">
	{#if $tempSettings.background !== $settings.background || $tempSettings.monochromeBackgroundColor !== $settings.monochromeBackgroundColor}
		{@render unsavedChanges()}
	{/if}

	<h5>{$t('settings.background')}</h5>
	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="background"
			id="background_gradient"
			value={'gradient'}
			bind:group={$tempSettings.background}
		/>
		<label class="form-check-label" for="background_gradient">
			{$t('settings.background_gradient')}
		</label>
	</div>
	<div class="form-check mt-2">
		<input
			class="form-check-input"
			type="radio"
			name="background"
			id="background_monochrome"
			value={'monochrome'}
			bind:group={$tempSettings.background}
		/>
		<label class="form-check-label" for="background_monochrome">
			{$t('settings.background_monochrome')}
		</label>
		{#if $tempSettings.background === 'monochrome'}
			<input
				transition:slide
				class="form-control form-control-color"
				bind:value={$tempSettings.monochromeBackgroundColor}
				type="color"
			/>
		{/if}
	</div>
</div>

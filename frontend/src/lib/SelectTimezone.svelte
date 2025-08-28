<script>
	import { MultiSelect } from 'svelte-multiselect';
	import { tempSettings } from '$lib/settingsStore';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	let value = $state(null);

	function formatTimezone(tz) {
		return tz.replace(/_/g, ' ');
	}

	let timezones = Intl.supportedValuesOf('timeZone').map((tz) => ({
		label: formatTimezone(tz),
		value: tz
	}));

	$effect(() => {
		if (value !== null) {
			$tempSettings.timezone = value.value;
		}
	});
</script>

<MultiSelect
	bind:value
	options={timezones}
	maxSelect={1}
	placeholder={$t('settings.selectTimezone')}
	disabled={$tempSettings.useBrowserTimezone}
	id="selectTimezone"
/>

<style>
	:global(.multiselect) {
		background-color: white !important;
		border-radius: 0.375rem !important;
	}
</style>

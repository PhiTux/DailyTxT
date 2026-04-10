<script>
	import { settings, tempSettings } from '$lib/settingsStore.js';
	import Map from '$lib/Map.svelte';

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();

	let { unsavedChanges } = $props();
	let defaultMapPreview;

	export function externalInvalidateSize(adjustView = true) {
		defaultMapPreview?.externalInvalidateSize?.(adjustView);
	}
</script>

<h3 class="text-primary">🗺️ {$t('settings.map')}</h3>

<div id="useMap">
	{#if $tempSettings.useMap !== $settings.useMap}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.map.use_map')}</h5>

	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html $t('settings.map.use_map_description')}
	<!-- data-protection  -->

	<div class="form-check form-switch">
		<input
			class="form-check-input"
			bind:checked={$tempSettings.useMap}
			type="checkbox"
			role="switch"
			id="useMapSwitch"
		/>
		<label class="form-check-label" for="useMapSwitch">
			{$t('settings.map.use_map')}
		</label>
	</div>
</div>

<div id="defaultMap">
	{#if $tempSettings.defaultMap !== $settings.defaultMap}
		{@render unsavedChanges()}
	{/if}
	<h5>{$t('settings.map.default_map')}</h5>

	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html $t('settings.map.default_map_description')}

	<div class="d-flex flex-row align-items-center gap-3">
		<select class="form-select w-auto" bind:value={$tempSettings.defaultMap}>
			<option value="osm">OpenStreetMap (OSM)</option>
			<option value="esri">Satellite (Esri)</option>
			<option value="stadia">Satellite with Labels (Stadia)</option>
		</select>

		<div class="map-layer">
			<Map
				bind:this={defaultMapPreview}
				showMapSelection={false}
				showSearch={false}
				selectDefaultMap={$tempSettings.defaultMap}
				allowMouseZoom={false}
			/>
		</div>
	</div>
</div>

<style>
	.map-layer {
		width: 300px;
		height: 180px;
	}
</style>

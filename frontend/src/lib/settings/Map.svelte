<script>
	import {
		settings,
		tempSettings,
		mapViewBeforeMove,
		useGeolocationOnThisDevice
	} from '$lib/settingsStore.js';
	import Map from '$lib/Map.svelte';
	import { slide } from 'svelte/transition';

	import { getTranslate } from '@tolgee/svelte';
	const { t } = getTranslate();

	let { unsavedChanges } = $props();
	let defaultMapPreview = $state();

	export function externalInvalidateSize(adjustView = true) {
		defaultMapPreview?.externalInvalidateSize?.(adjustView);
	}

	let currentView = $state([]);
	let mapIsMovable = $state(false);
	function enableMap() {
		mapIsMovable = true;
		defaultMapPreview?.externalEnableMap?.();
		if (!$mapViewBeforeMove || $mapViewBeforeMove.length === 0) {
			$mapViewBeforeMove = defaultMapPreview?.externalGetView?.();
		}
	}

	function disableMap() {
		mapIsMovable = false;
		defaultMapPreview?.externalDisableMap?.();
		$tempSettings.defaultMapView = currentView;
	}

	function resetMap() {
		mapIsMovable = false;
		defaultMapPreview?.externalDisableMap?.();
		defaultMapPreview?.externalSetView?.(
			$mapViewBeforeMove[0],
			$mapViewBeforeMove[1],
			$mapViewBeforeMove[2]
		);
		$tempSettings.defaultMapView = $settings.defaultMapView;
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

{#if $tempSettings.useMap}
	<div id="defaultMap" transition:slide>
		{#if $tempSettings.defaultMap !== $settings.defaultMap || $tempSettings.defaultMapView?.some((value, index) => value !== $settings.defaultMapView[index])}
			{@render unsavedChanges()}
		{/if}
		<h5>{$t('settings.map.default_map')}</h5>

		<!-- eslint-disable-next-line svelte/no-at-html-tags -->
		{@html $t('settings.map.default_map_description')}

		<div class="d-flex flex-column align-items-start gap-3">
			<select class="form-select w-auto" bind:value={$tempSettings.defaultMap}>
				<option value="osm">{$t('map.osm')}</option>
				<option value="esri">{$t('map.satellite')}</option>
				<option value="esriAndMeta">{$t('map.satellite_and_meta')}</option>
			</select>

			<div class="map-layer disabled">
				<Map
					bind:this={defaultMapPreview}
					showMapSelection={false}
					showSearch={false}
					mapDisabled={true}
					selectDefaultMap={$tempSettings.defaultMap}
					bind:currentView
					allowMouseZoom={false}
				/>
			</div>

			<div class="d-flex flex-row justify-content-between w-100">
				<button class="btn btn-primary" disabled={mapIsMovable} onclick={enableMap}
					>{$t('map.pin.move')}</button
				>

				<div class="d-flex flex-row gap-2">
					{#if mapIsMovable}
						<button class="btn btn-primary" disabled={!mapIsMovable} onclick={disableMap}
							>{$t('settings.accept')}</button
						>
					{/if}
					{#if mapIsMovable || $tempSettings.defaultMapView?.some((value, index) => value !== $settings.defaultMapView[index])}
						<button
							class="btn btn-outline-danger"
							onclick={() => {
								resetMap();
							}}>{$t('settings.reset')}</button
						>
					{/if}
				</div>
			</div>
		</div>
	</div>

	<div id="useGeolocation" transition:slide>
		<h5>{$t('settings.map.use_geolocation')}</h5>

		<div class="form-check form-switch">
			<input
				class="form-check-input"
				bind:checked={$useGeolocationOnThisDevice}
				type="checkbox"
				role="switch"
				id="useGeolocationOnThisDeviceSwitch"
			/>
			<label class="form-check-label" for="useGeolocationOnThisDeviceSwitch">
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html $t('settings.map.use_geolocation_this_device')}
			</label>
		</div>
	</div>
{/if}

<style>
	.map-layer {
		width: 100%;
		height: 280px;
	}
</style>

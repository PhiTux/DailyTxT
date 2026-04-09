<script>
	import { onMount, mount, unmount } from 'svelte';
	import L from 'leaflet';
	import 'leaflet/dist/leaflet.css';
	import { getTolgee } from '@tolgee/svelte';
	import { faSearch } from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import lockedHeartPinUrl from '$lib/assets/locked_heart_with_keyhole.svg';
	import axios from 'axios';
	import { API_URL } from '$lib/APIurl.js';
	import { selectedDate } from '$lib/calendarStore.js';
	import SavedPinPopup from '$lib/map/SavedPinPopup.svelte';
	import NewPinPopup from '$lib/map/NewPinPopup.svelte';

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	const tolgee = getTolgee(['language']);

	let { pins } = $props();

	let mapElement;

	let map = null;
	let mapSearchOpen = $state(false);
	let mapSearchQuery = $state('');
	let mapSearchResults = $state([]);
	let mapSearchLoading = $state(false);
	let mapSearchError = $state('');
	let mapSearchDebounce;
	let mapSearchMarker = null;
	let mapClickPinMarker = null;
	let mapSearchAbortController = null;
	let customPinIcon = null;
	let addPinMarker = null;
	let mapClickPinName = $state('');
	let mapClickPinPopupApp = null;

	export function externalDrawAllPins() {
		drawAllPins(true);
	}

	function drawAllPins(adjustView) {
		if (!map || !customPinIcon) return;

		// remove existing pins (except search and click markers)
		map.eachLayer((layer) => {
			if (layer instanceof L.Marker && layer !== mapSearchMarker) {
				layer.remove();
			}
		});

		const pinLatLngs = [];

		pins.forEach((pin) => {
			const marker = L.marker([pin.lat, pin.lon], { icon: customPinIcon }).addTo(map);
			pinLatLngs.push([pin.lat, pin.lon]);

			const popupTarget = document.createElement('div');
			const popupApp = mount(SavedPinPopup, {
				target: popupTarget,
				props: {
					text: pin.text || '',
					id: pin.id,
					deletePin: () => {
						deletePin(pin.id);
						//marker.remove();
					}
				}
			});

			marker.bindPopup(popupTarget);
			marker.on('remove', () => {
				unmount(popupApp);
			});

			marker.on('popupopen', () => {
				popupApp.resetEditing?.();
			});

			marker.on('click', () => {
				if (mapClickPinMarker) {
					removeMapClickPin();
				}
			});
		});

		if (adjustView) {
			// adjust map view to fit all pins
			if (pinLatLngs.length > 0) {
				map.fitBounds(pinLatLngs, {
					padding: [30, 30],
					maxZoom: 15
				});
			}
		}
	}

	/**
	 * Makes an API call to delete a pin and updates the local state accordingly
	 */
	function deletePin(id) {
		axios
			.post(`${API_URL}/logs/deletePin`, {
				pinId: id,
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year
			})
			.then((response) => {
				if (response.data.success) {
					// remove the pin from the local state
					pins = pins.filter((pin) => pin.id !== id);
				} else {
					console.error('Failed to delete pin:', response.data.message);
				}
			})
			.catch((error) => {
				console.error('Error deleting pin:', error);
			})
			.finally(() => {
				drawAllPins(false);
			});
	}

	function createMapClickPopupContent() {
		const container = document.createElement('div');

		mapClickPinPopupApp = mount(NewPinPopup, {
			target: container,
			props: {
				initialValue: mapClickPinName,
				onChange: (value) => {
					mapClickPinName = value;
				},
				onSave: (value) => {
					addNewPinMarker(value);
				}
			}
		});

		return container;
	}

	function closeMapSearch() {
		mapSearchOpen = false;
		mapSearchResults = [];
		mapSearchError = '';
		mapSearchLoading = false;
		if (mapSearchAbortController) {
			mapSearchAbortController.abort();
			mapSearchAbortController = null;
		}
	}

	/**
	 * Saves the new pin via API and updates the local state
	 * @param text the text of the new pin, can be empty
	 */
	function addNewPinMarker(text) {
		// save the new pin (API call)
		axios
			.post(`${API_URL}/logs/addPin`, {
				lat: mapClickPinMarker.getLatLng().lat,
				lon: mapClickPinMarker.getLatLng().lng,
				day: $selectedDate.day,
				month: $selectedDate.month,
				year: $selectedDate.year,
				text: text
			})
			.then((response) => {
				if (response.data.success) {
					// add the new pin to the local state
					pins = [...pins, response.data.pin];
					drawAllPins(false);
				} else {
					console.error('Failed to add pin:', response.data.message);
				}
			})
			.catch((error) => {
				console.error('Error adding pin:', error);
			})
			.finally(() => {
				// at the end:
				removeMapClickPin();
			});
	}

	function handleMapBackgroundClick(event) {
		const canPlacePin = mapSearchResults.length === 0;

		// check if any popup is open and close it
		const hasOpenPopup = Boolean(map?.getContainer()?.querySelector('.leaflet-popup'));
		if (hasOpenPopup) {
			if (
				!map?.getContainer()?.querySelector('.leaflet-popup > div > div > div > .new-pin-popup')
			) {
				map?.getContainer()?.querySelector('.leaflet-popup')?.remove();
				return;
			}
		}

		if (mapSearchOpen) {
			closeMapSearch();
			return;
		}

		if (!canPlacePin || !map || !customPinIcon) return;

		if (addPinMarker) {
			addNewPinMarker();
			return;
		}

		if (!mapClickPinMarker) {
			mapClickPinMarker = L.marker(event.latlng, { icon: customPinIcon, opacity: 0.7 }).addTo(map);
			mapClickPinMarker.bindPopup(createMapClickPopupContent(), {
				offset: [0, 0]
			});
			mapClickPinMarker.openPopup();
			document
				.getElementsByClassName('leaflet-popup-close-button')[0]
				?.addEventListener('click', () => {
					removeMapClickPin();
				});
		} else {
			removeMapClickPin();
		}
	}

	function removeMapClickPin() {
		if (!mapClickPinMarker) return;

		const markerToRemove = mapClickPinMarker;
		mapClickPinMarker = null;
		mapClickPinName = '';
		if (mapClickPinPopupApp) {
			unmount(mapClickPinPopupApp);
			mapClickPinPopupApp = null;
		}
		markerToRemove.remove();
	}

	onMount(() => {
		customPinIcon = L.icon({
			iconUrl: lockedHeartPinUrl,
			iconSize: [34, 34],
			iconAnchor: [17, 34],
			popupAnchor: [0, -34]
		});

		// init map
		map = L.map(mapElement).setView([51.505, -0.09], 13);

		L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution: '&copy; <a href="http://www.openstreetmap.org/copyright">OpenStreetMap</a>'
		}).addTo(map);

		map.on('click', handleMapBackgroundClick);

		return () => {
			if (mapSearchAbortController) {
				mapSearchAbortController.abort();
			}
			clearTimeout(mapSearchDebounce);
			if (map) {
				map.remove();
				map = null;
			}
		};
	});

	function toggleMapSearch() {
		if (mapSearchOpen) {
			closeMapSearch();
			return;
		}

		mapSearchOpen = true;

		requestAnimationFrame(() => {
			document.getElementById('map-search-input')?.focus();
		});
	}

	function handleMapSearchInput(event) {
		mapSearchQuery = event.target.value;
		mapSearchError = '';

		clearTimeout(mapSearchDebounce);

		if (mapSearchQuery.trim().length < 2) {
			mapSearchResults = [];
			mapSearchLoading = false;
			if (mapSearchAbortController) {
				mapSearchAbortController.abort();
				mapSearchAbortController = null;
			}
			return;
		}

		mapSearchDebounce = setTimeout(() => {
			searchPhoton(mapSearchQuery);
		}, 250);
	}

	async function searchPhoton(query) {
		if (!query || query.trim().length < 2) {
			mapSearchResults = [];
			return;
		}

		if (mapSearchAbortController) {
			mapSearchAbortController.abort();
		}
		mapSearchAbortController = new AbortController();

		mapSearchLoading = true;
		mapSearchError = '';

		try {
			const lang = $tolgee.getLanguage() || 'en';
			const response = await fetch(
				`https://photon.komoot.io/api/?q=${encodeURIComponent(query.trim())}&limit=6&lang=${encodeURIComponent(lang)}`,
				{ signal: mapSearchAbortController.signal }
			);

			if (!response.ok) {
				throw new Error('Photon request failed');
			}

			const data = await response.json();
			mapSearchResults = (data.features || [])
				.map((feature) => {
					const coords = feature?.geometry?.coordinates;
					if (!Array.isArray(coords) || coords.length < 2) return null;

					const props = feature.properties || {};
					const nameParts = [
						props.name,
						props.street,
						props.city,
						props.state,
						props.country
					].filter(Boolean);

					return {
						label: nameParts.join(', '),
						lat: coords[1],
						lon: coords[0]
					};
				})
				.filter(Boolean);
		} catch (error) {
			if (error.name === 'AbortError') return;
			console.error(error);
			mapSearchError = 'Suche fehlgeschlagen';
			mapSearchResults = [];
		} finally {
			mapSearchLoading = false;
		}
	}

	function focusMapResult(result) {
		if (!map || !result) return;

		map.setView([result.lat, result.lon], 14);

		if (!mapSearchMarker) {
			mapSearchMarker = L.marker([result.lat, result.lon]).addTo(map);
		} else {
			mapSearchMarker.setLatLng([result.lat, result.lon]);
		}

		if (result.label) {
			mapSearchMarker.bindPopup(result.label).openPopup();
		}

		mapSearchQuery = result.label || mapSearchQuery;
		mapSearchResults = [];
	}

	function handleMapSearchKeydown(event) {
		if (event.key === 'Escape' && mapSearchOpen) {
			event.preventDefault();
			closeMapSearch();
		}
	}

	let hasSearchFeedback = $derived(
		mapSearchOpen &&
			(mapSearchLoading ||
				mapSearchError !== '' ||
				mapSearchResults.length > 0 ||
				mapSearchQuery.trim().length >= 2)
	);
</script>

<div class="map-wrapper mb-3">
	<div class="map" bind:this={mapElement}></div>

	<div class="map-search-dock {mapSearchOpen ? 'open' : ''}">
		<div class="map-search-group">
			<button
				type="button"
				class="map-search-toggle"
				onclick={toggleMapSearch}
				aria-label="Karte durchsuchen"
			>
				<Fa icon={faSearch} />
			</button>

			<div class="map-search-inline">
				<input
					id="map-search-input"
					type="text"
					class="map-search-input"
					placeholder="Ort suchen..."
					value={mapSearchQuery}
					oninput={handleMapSearchInput}
					onkeydown={handleMapSearchKeydown}
				/>
			</div>
		</div>

		<div class="map-search-feedback {hasSearchFeedback ? 'open' : ''}">
			{#if mapSearchLoading}
				<div class="map-search-status">Suche...</div>
			{:else if mapSearchError}
				<div class="map-search-status text-danger">{mapSearchError}</div>
			{:else if mapSearchResults.length > 0}
				<div class="map-search-results">
					{#each mapSearchResults as result, index (result.lat + '-' + result.lon + '-' + index)}
						<button type="button" class="map-search-result" onclick={() => focusMapResult(result)}>
							{result.label || `${result.lat.toFixed(5)}, ${result.lon.toFixed(5)}`}
						</button>
					{/each}
				</div>
			{:else if mapSearchQuery.trim().length >= 2}
				<div class="map-search-status">Keine Treffer</div>
			{/if}
		</div>
	</div>
</div>

<style>
	:global(.leaflet-marker-icon) {
		filter: drop-shadow(rgba(0, 0, 0, 0.8) 3px -2px 4px);
	}

	.map-wrapper {
		position: relative;
	}

	.map {
		height: 260px;
		border-radius: 10px;
	}

	.map-search-dock {
		--inline-width: min(280px, calc(100vw - 110px));
		position: absolute;
		left: 12px;
		bottom: 12px;
		z-index: 1200;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.45rem;
	}

	.map-search-group {
		display: flex;
		align-items: center;
		flex-wrap: nowrap;
		border-radius: 10px;
		overflow: hidden;
		background: rgba(255, 255, 255, 0.96);
		border: 1px solid rgba(0, 0, 0, 0.18);
		box-shadow: 0 2px 10px rgba(0, 0, 0, 0.15);
	}

	:global(body[data-bs-theme='dark']) .map-search-group {
		background: rgba(45, 45, 45, 0.96);
		border-color: rgba(255, 255, 255, 0.22);
	}

	.map-search-dock:not(.open) .map-search-toggle {
		border-right: none;
	}

	.map-search-toggle {
		width: 40px;
		height: 40px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		flex: 0 0 auto;
		border: 0;
		border-right: 1px solid rgba(0, 0, 0, 0.15);
		background: transparent;
		color: inherit;
	}

	:global(body[data-bs-theme='dark']) .map-search-toggle {
		border-right-color: rgba(255, 255, 255, 0.2);
	}

	.map-search-inline {
		width: 0;
		overflow: hidden;
		opacity: 0;
		transform: translateX(-8px);
		transition:
			width 180ms ease,
			opacity 140ms ease,
			transform 180ms ease;
	}

	.map-search-dock.open .map-search-inline {
		width: var(--inline-width);
		opacity: 1;
		transform: translateX(0);
	}

	.map-search-input {
		width: 100%;
		height: 40px;
		padding: 0 0.75rem;
		border: 0;
		background: transparent;
		color: #1f1f1f;
	}

	:global(body[data-bs-theme='dark']) .map-search-input {
		color: #f0f0f0;
	}

	.map-search-input:focus {
		outline: none;
	}

	.map-search-feedback {
		position: absolute;
		top: -8px;
		left: 0;
		transform: translateY(-100%);
		z-index: 1201;
		width: calc(40px + var(--inline-width));
		max-height: 0;
		overflow: hidden;
		opacity: 0;
		pointer-events: none;
		transition:
			max-height 180ms ease,
			opacity 140ms ease;
		padding: 0;
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.95);
		border: 1px solid rgba(0, 0, 0, 0.1);
	}

	.map-search-feedback.open {
		max-height: 240px;
		opacity: 1;
		pointer-events: auto;
	}

	:global(body[data-bs-theme='dark']) .map-search-feedback {
		background: rgba(45, 45, 45, 0.95);
		border-color: rgba(255, 255, 255, 0.15);
	}

	.map-search-status {
		font-size: 0.9rem;
		padding: 0.5rem 0.6rem;
	}

	.map-search-results {
		display: flex;
		flex-direction: column;
		max-height: 180px;
		overflow-y: auto;
		gap: 0.2rem;
		padding: 0.35rem;
	}

	.map-search-result {
		text-align: left;
		background: transparent;
		border: 1px solid rgba(0, 0, 0, 0.12);
		border-radius: 8px;
		padding: 0.35rem 0.5rem;
		font-size: 0.9rem;
	}

	:global(body[data-bs-theme='dark']) .map-search-result {
		border-color: rgba(255, 255, 255, 0.15);
		color: #f0f0f0;
	}

	.map-search-result:hover {
		background: rgba(0, 0, 0, 0.06);
	}

	:global(body[data-bs-theme='dark']) .map-search-result:hover {
		background: rgba(255, 255, 255, 0.09);
	}

	:global(.leaflet-popup-content .new-pin-popup) {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		min-width: 170px;
	}
</style>

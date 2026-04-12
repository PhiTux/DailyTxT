<script>
	import { onMount, mount, unmount } from 'svelte';
	import L from 'leaflet';
	import 'leaflet/dist/leaflet.css';
	import { getTolgee } from '@tolgee/svelte';
	import {
		faSearch,
		faUpRightAndDownLeftFromCenter,
		faMap
	} from '@fortawesome/free-solid-svg-icons';
	import Fa from 'svelte-fa';
	import lockedHeartPinUrl from '$lib/assets/locked_heart_with_keyhole.svg';
	import axios from 'axios';
	import { API_URL } from '$lib/APIurl.js';
	import { selectedDate } from '$lib/calendarStore.js';
	import SavedPinPopup from '$lib/map/SavedPinPopup.svelte';
	import NewPinPopup from '$lib/map/NewPinPopup.svelte';
	import { settings } from './settingsStore';
	import * as bootstrap from 'bootstrap';
	import { getTranslate } from '@tolgee/svelte';

	const { t } = getTranslate();

	axios.interceptors.request.use((config) => {
		config.withCredentials = true;
		return config;
	});

	const tolgee = getTolgee(['language']);

	let {
		pins = $bindable([]),
		openMapModal,
		showZoomButton = false,
		showMapSelection = true,
		showSearch = true,
		allowMouseZoom = true,
		selectDefaultMap,
		openPreview = () => {},
		mapDisabled = false,
		currentView = $bindable(),
		fullScreen = false
	} = $props();

	let mapElement;

	let map = null;
	let baseMapProvider = $state('osm');
	let mapSearchOpen = $state(false);
	let mapSearchQuery = $state('');
	let mapSearchResults = $state([]);
	let mapSearchLoading = $state(false);
	let mapSearchError = $state('');
	let mapSearchInputElement = $state(null);
	let mapSearchDebounce;
	let mapClickPinMarker = null;
	let mapSearchAbortController = null;
	let customPinIcon = null;
	let mapClickPinName = $state('');
	let mapClickPinPopupApp = null;
	let markerByPinKey = {};
	let movingPinID = null;
	let movingPinKey = null;
	let movingPinMarker = null;
	let movingPinOriginalLatLng = null;
	let movePinMouseMoveHandler = null;
	let movingPinIconElement = null;
	let osmTileLayer = null;
	let esriTileLayer = null;
	let stadiaTileLayer = null;
	let hasAppliedInitialDefaultMapView = false;
	let pinsSetForDate = $state();
	let viewSetForDate = $state();
	let lastPinsSignature = '';
	let movingPinDate = $state();
	let fullScreenInitialPinsDrawDone = false;

	function getValidDefaultMapView() {
		const view = $settings?.defaultMapView;
		if (!Array.isArray(view) || view.length < 3) return null;

		const lat = Number(view[0]);
		const lon = Number(view[1]);
		const zoom = Number(view[2]);

		if (!Number.isFinite(lat) || !Number.isFinite(lon) || !Number.isFinite(zoom)) {
			return null;
		}

		return [lat, lon, zoom];
	}

	function normalizeBaseMapProvider(provider) {
		return provider === 'osm' || provider === 'esri' || provider === 'stadia' ? provider : 'osm';
	}

	export function externalDrawAllPins() {
		drawAllPins(true);
	}

	export function externalInvalidateSize(adjustView = true) {
		if (!map) return;
		map.invalidateSize();
		if (adjustView) {
			drawAllPins(true);
		}
	}

	export function externalGetView() {
		if (!map) return null;
		const center = map.getCenter();
		const zoom = map.getZoom();
		return [center.lat, center.lng, zoom];
	}

	export function externalSetView(lat, lon, zoom) {
		if (!map) return;
		map.setView([lat, lon], zoom);
	}

	function setView(lat, lon, zoom) {
		if (
			!map ||
			sameDate(viewSetForDate, $selectedDate) ||
			!sameDate(pinsSetForDate, $selectedDate) ||
			(sameDate(pinsSetForDate, $selectedDate) && pins.length > 0)
		)
			return;
		viewSetForDate = $selectedDate;
		map.setView([lat, lon], zoom);
	}

	export function externalEnableMap() {
		if (!map) return;
		map.dragging.enable();
		map.doubleClickZoom.enable();
		map.scrollWheelZoom.enable();
		// set mouse enabled
		map.getContainer().style.cursor = '';
	}

	export function externalDisableMap() {
		if (!map) return;
		map.dragging.disable();
		map.doubleClickZoom.disable();
		map.scrollWheelZoom.disable();
		// set mouse disabled
		map.getContainer().style.cursor = 'not-allowed';
	}

	function sameDate(date1, date2) {
		return (
			date1 &&
			date2 &&
			date1.day === date2.day &&
			date1.month === date2.month &&
			date1.year === date2.year
		);
	}

	function getPinKey(id, day, month, year) {
		return `${id ?? ''}-${day ?? ''}-${month ?? ''}-${year ?? ''}`;
	}

	function onPinsChanged() {
		if (fullScreen) {
			const shouldAdjustView = !fullScreenInitialPinsDrawDone && pins.length > 0;
			drawAllPins(shouldAdjustView);
			if (shouldAdjustView) {
				fullScreenInitialPinsDrawDone = true;
			}
			return;
		}

		if (!sameDate(pinsSetForDate, $selectedDate)) {
			pinsSetForDate = $selectedDate;
			drawAllPins(true);
		} else {
			drawAllPins(false);
		}
	}

	$effect(() => {
		const pinsSignature = JSON.stringify(pins ?? []);
		if (pinsSignature !== lastPinsSignature) {
			lastPinsSignature = pinsSignature;
			onPinsChanged();
		}
	});

	$effect(() => {
		if (selectDefaultMap) {
			setBaseMap(normalizeBaseMapProvider(selectDefaultMap));
		}
	});

	$effect(() => {
		const defaultMapView = getValidDefaultMapView();
		if (!map || hasAppliedInitialDefaultMapView || !defaultMapView) return;

		map.setView([defaultMapView[0], defaultMapView[1]], defaultMapView[2]);
		hasAppliedInitialDefaultMapView = true;
	});

	$effect(() => {
		if ($selectedDate && map && $settings?.defaultMapView) {
			const defaultView = getValidDefaultMapView();
			setView(defaultView[0], defaultView[1], defaultView[2]);
		}
	});

	function updateMapView() {
		if (!map || !selectDefaultMap) return;
		const center = map.getCenter();
		const zoom = map.getZoom();
		const newView = [center.lat, center.lng, zoom];
		currentView = newView;
	}

	onMount(() => {
		customPinIcon = L.icon({
			iconUrl: lockedHeartPinUrl,
			iconSize: [34, 34],
			iconAnchor: [17, 34],
			popupAnchor: [0, -34]
		});

		const initialMapView = getValidDefaultMapView() || [51.505, -0.09, 13];

		// init map
		map = L.map(mapElement, {
			zoomControl: false,
			scrollWheelZoom: allowMouseZoom,
			dragging: !mapDisabled,
			doubleClickZoom: !mapDisabled
		}).setView([initialMapView[0], initialMapView[1]], initialMapView[2]);
		hasAppliedInitialDefaultMapView = getValidDefaultMapView() !== null;

		if (mapDisabled) {
			map.getContainer().style.cursor = 'not-allowed';
		}

		// set view-position after 100ms to ensure the map is properly displayed when inside a tab or modal
		setTimeout(() => {
			updateMapView(true);
		}, 100);

		osmTileLayer = L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
			maxZoom: 19,
			attribution:
				'&copy; <a href="http://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a>'
		});

		esriTileLayer = L.tileLayer(
			'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
			{
				maxZoom: 19,
				attribution:
					'Powered by <a href="https://www.esri.com" target="_blank">Esri</a> &mdash; Sources: Esri, DigitalGlobe, GeoEye, i-cubed, USDA FSA, USGS, AEX, Getmapping, Aerogrid, IGN, IGP, swisstopo, and the GIS User Community'
			}
		);

		stadiaTileLayer = L.tileLayer(
			'https://tiles.stadiamaps.com/tiles/alidade_satellite/{z}/{x}/{y}.jpg',
			{
				maxZoom: 19,

				attribution:
					'&copy; CNES, Distribution Airbus DS, &copy; Airbus DS, &copy; PlanetObserver (Contains Copernicus Data) | &copy; <a href="https://stadiamaps.com/" target="_blank">Stadia Maps</a> &copy; <a href="https://openmaptiles.org/" target="_blank">OpenMapTiles</a> &copy; <a href="https://www.openstreetmap.org/copyright" target="_blank">OpenStreetMap</a>'
			}
		);

		baseMapProvider = normalizeBaseMapProvider($settings.defaultMap);
		getActiveBaseLayer().addTo(map);

		map.on('click', handleMapBackgroundClick);
		map.on('popupopen', handleMapPopupOpen);
		map.on('moveend', () => updateMapView(false));
		window.addEventListener('keydown', handleGlobalKeydown);

		return () => {
			map?.off('popupopen', handleMapPopupOpen);
			map?.on('moveend', () => updateMapView(false));
			window.removeEventListener('keydown', handleGlobalKeydown);
			if (mapSearchAbortController) {
				mapSearchAbortController.abort();
			}
			clearTimeout(mapSearchDebounce);
			if (map) {
				map.remove();
				map = null;
			}
			osmTileLayer = null;
			esriTileLayer = null;
			stadiaTileLayer = null;
		};
	});

	function getActiveBaseLayer() {
		if (baseMapProvider === 'osm') {
			return osmTileLayer;
		} else if (baseMapProvider === 'esri') {
			return esriTileLayer;
		} else {
			return stadiaTileLayer;
		}
	}

	function setBaseMap(provider) {
		if (!map || !osmTileLayer || !esriTileLayer || !stadiaTileLayer) return;
		if (provider !== 'osm' && provider !== 'esri' && provider !== 'stadia') return;
		if (baseMapProvider === provider) return;

		const allBaseLayers = [osmTileLayer, esriTileLayer, stadiaTileLayer];
		allBaseLayers.forEach((layer) => {
			if (layer && map.hasLayer(layer)) {
				map.removeLayer(layer);
			}
		});

		baseMapProvider = provider;
		const nextLayer = getActiveBaseLayer();

		nextLayer.addTo(map);
	}

	function drawAllPins(adjustView) {
		if (!map || !customPinIcon) return;

		// remove existing pins (except the temporary new-pin marker)
		map.eachLayer((layer) => {
			if (layer instanceof L.Marker && layer !== mapClickPinMarker) {
				layer.remove();
			}
		});

		const pinLatLngs = [];
		markerByPinKey = {};

		pins.forEach((pin) => {
			const pinKey = getPinKey(pin.id, pin.day, pin.month, pin.year);
			const samePin = (candidate) => {
				if (candidate.id !== pin.id) return false;

				// In fullscreen map mode, pin IDs are only unique per date.
				if (pin.day && pin.month && pin.year) {
					return (
						candidate.day === pin.day &&
						candidate.month === pin.month &&
						candidate.year === pin.year
					);
				}

				return true;
			};

			const marker = L.marker([pin.lat, pin.lon], { icon: customPinIcon }).addTo(map);
			markerByPinKey[pinKey] = marker;
			pinLatLngs.push([pin.lat, pin.lon]);

			const popupTarget = document.createElement('div');
			const popupApp = mount(SavedPinPopup, {
				target: popupTarget,
				props: {
					get text() {
						const currentPin = pins.find((candidate) => samePin(candidate));
						return currentPin?.text || '';
					},
					set text(value) {
						pins = pins.map((currentPin) =>
							samePin(currentPin) ? { ...currentPin, text: value } : currentPin
						);
					},
					id: pin.id,
					deletePin: () => {
						deletePin(pin.id, pin.day, pin.month, pin.year);
					},
					movePin: () => {
						movePin(pin.id, pin.day, pin.month, pin.year);
					},
					openPreview: (day, month, year) => {
						openPreview(day, month, year);
					},
					day: pin.day || null,
					month: pin.month || null,
					year: pin.year || null,
					language: $tolgee.getLanguage(),
					translate: $t
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
				viewSetForDate = $selectedDate;
				map.fitBounds(pinLatLngs, {
					padding: [30, 30],
					maxZoom: 15
				});
			}
		}
	}

	function movePin(id, day, month, year) {
		const marker = markerByPinKey[getPinKey(id, day, month, year)];
		if (!marker) return;

		if (movePinMouseMoveHandler) {
			map.off('mousemove', movePinMouseMoveHandler);
			movePinMouseMoveHandler = null;
		}

		movingPinID = id;
		movingPinKey = getPinKey(id, day, month, year);
		movingPinDate = { day, month, year };
		movingPinMarker = marker;
		movingPinOriginalLatLng = marker.getLatLng();
		marker.closePopup();
		marker.setOpacity(0.5);

		const mapContainer = map.getContainer();
		mapContainer.classList.add('pin-moving');

		movingPinIconElement = marker.getElement();
		if (movingPinIconElement) {
			// Keep mouse events on the map while dragging so the cursor does not flicker.
			movingPinIconElement.style.pointerEvents = 'none';
		}

		movePinMouseMoveHandler = (event) => {
			if (!movingPinMarker) return;
			movingPinMarker.setLatLng(event.latlng);
		};

		map.on('mousemove', movePinMouseMoveHandler);
	}

	function cancelMovePin() {
		if (!movingPinMarker) return;

		if (movePinMouseMoveHandler) {
			map.off('mousemove', movePinMouseMoveHandler);
			movePinMouseMoveHandler = null;
		}

		if (movingPinOriginalLatLng) {
			movingPinMarker.setLatLng(movingPinOriginalLatLng);
		}

		if (movingPinIconElement) {
			movingPinIconElement.style.pointerEvents = '';
			movingPinIconElement = null;
		}

		movingPinMarker.setOpacity(1);
		map.getContainer().classList.remove('pin-moving');
		movingPinID = null;
		movingPinKey = null;
		movingPinDate = null;
		movingPinMarker = null;
		movingPinOriginalLatLng = null;
	}

	function handleGlobalKeydown(event) {
		if (event.key === 'Escape' && movingPinID !== null) {
			event.preventDefault();
			cancelMovePin();
		}
	}

	/**
	 * Saves a moved pin position. Backend call intentionally left empty for now.
	 */
	function updatePinPosition(pinID, lat, lon) {
		const targetPinKey =
			movingPinKey ||
			getPinKey(pinID, movingPinDate?.day, movingPinDate?.month, movingPinDate?.year);
		axios
			.post(`${API_URL}/logs/movePin`, {
				pinId: pinID,
				lat: lat,
				lon: lon,
				day: movingPinDate?.day || $selectedDate.day,
				month: movingPinDate?.month || $selectedDate.month,
				year: movingPinDate?.year || $selectedDate.year
			})
			.then((response) => {
				if (!response.data.success) {
					console.error('Failed to move pin:', response.data.message);
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorUpdatePinPosition'));
					toast.show();
				} else {
					pins = pins.map((pin) =>
						getPinKey(pin.id, pin.day, pin.month, pin.year) === targetPinKey
							? { ...pin, lat: lat, lon: lon }
							: pin
					);
				}
			})
			.catch((error) => {
				console.error('Error moving pin:', error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorUpdatePinPosition'));
				toast.show();
			})
			.finally(() => {
				drawAllPins(false);

				if (movingPinIconElement) {
					movingPinIconElement.style.pointerEvents = '';
					movingPinIconElement = null;
				}

				movingPinMarker.setOpacity(1);
				map.getContainer().classList.remove('pin-moving');
				movingPinID = null;
				movingPinKey = null;
				movingPinMarker = null;
				movingPinOriginalLatLng = null;
			});
	}

	/**
	 * Makes an API call to delete a pin and updates the local state accordingly
	 */
	function deletePin(id, day, month, year) {
		axios
			.post(`${API_URL}/logs/deletePin`, {
				pinId: id,
				day: day || $selectedDate.day,
				month: month || $selectedDate.month,
				year: year || $selectedDate.year
			})
			.then((response) => {
				if (response.data.success) {
					// remove the pin from the local state
					pins = pins.filter((pin) => pin.id !== id);
				} else {
					console.error('Failed to delete pin:', response.data.message);
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorDeletePin'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error('Error deleting pin:', error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorDeletePin'));
				toast.show();
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
				},
				fullScreen,
				translate: $t
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
					// toast
					const toast = new bootstrap.Toast(document.getElementById('toastErrorAddPin'));
					toast.show();
				}
			})
			.catch((error) => {
				console.error('Error adding pin:', error);
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorAddPin'));
				toast.show();
			})
			.finally(() => {
				// at the end:
				removeMapClickPin();
			});
	}

	function openNewPinPopupAt(latlng, suggestedName = '') {
		if (!map || !customPinIcon) return;

		if (mapClickPinMarker) {
			removeMapClickPin();
		}

		mapClickPinName = suggestedName || '';
		mapClickPinMarker = L.marker(latlng, { icon: customPinIcon, opacity: 0.7 }).addTo(map);
		mapClickPinMarker.bindPopup(createMapClickPopupContent(), {
			offset: [0, 0]
		});
		mapClickPinMarker.openPopup();

		document
			.getElementsByClassName('leaflet-popup-close-button')[0]
			?.addEventListener('click', () => {
				removeMapClickPin();
			});
	}

	function handleMapBackgroundClick(event) {
		if (mapDisabled) return;

		// moving a pin right now?
		if (movingPinID !== null && movingPinMarker) {
			const { lat, lng } = event.latlng;
			const pinID = movingPinID;

			movingPinMarker.setLatLng(event.latlng);

			if (movePinMouseMoveHandler) {
				map.off('mousemove', movePinMouseMoveHandler);
				movePinMouseMoveHandler = null;
			}

			updatePinPosition(pinID, lat, lng);
			return;
		}

		if (mapSearchOpen) {
			closeMapSearch();
			return;
		}

		// check if any popup is open and close it
		const hasOpenPopup = Boolean(map?.getContainer()?.querySelector('.leaflet-popup'));
		if (hasOpenPopup) {
			if (
				!map?.getContainer()?.querySelector('.leaflet-popup > div > div > div > .new-pin-popup')
			) {
				map?.closePopup();
				// Close existing saved-pin popup first; require a second click to place a new pin.
				return;
			}
		}

		const canPlacePin = mapSearchResults.length === 0;
		if (!canPlacePin || !map || !customPinIcon) return;

		if (!mapClickPinMarker && !fullScreen) {
			openNewPinPopupAt(event.latlng);
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

	function handleMapPopupOpen() {
		map
			?.getContainer()
			?.querySelectorAll('.leaflet-popup-content-wrapper')
			.forEach((popupWrapper) => popupWrapper.classList.add('popup-blur'));
	}

	function toggleMapSearch() {
		if (mapSearchOpen) {
			closeMapSearch();
			return;
		}

		mapSearchOpen = true;

		requestAnimationFrame(() => {
			mapSearchInputElement?.focus();
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
				// toast
				const toast = new bootstrap.Toast(document.getElementById('toastErrorPhotonSearch'));
				toast.show();

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
			mapSearchError = $t('map.toast.error_photon_search');
			mapSearchResults = [];

			// toast
			const toast = new bootstrap.Toast(document.getElementById('toastErrorPhotonSearch'));
			toast.show();
		} finally {
			mapSearchLoading = false;
		}
	}

	function focusMapResult(result) {
		if (!map || !result) return;

		map.setView([result.lat, result.lon], 14);

		closeMapSearch();
		openNewPinPopupAt([result.lat, result.lon], result.label || '');

		mapSearchQuery = result.label || mapSearchQuery;
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

<div class="map-wrapper">
	<div class="map" bind:this={mapElement}></div>

	{#if showMapSelection}
		<div class="map-basemap-menu" aria-label={$t('map.switch_map')}>
			<button type="button" class="map-basemap-trigger" title={$t('map.switch_map')}>
				<Fa icon={faMap} />
			</button>

			<div class="map-basemap-options" role="menu">
				<button
					type="button"
					role="menuitemradio"
					aria-checked={baseMapProvider === 'osm'}
					class:active={baseMapProvider === 'osm'}
					onclick={() => setBaseMap('osm')}
				>
					{$t('map.osm')}
				</button>
				<button
					type="button"
					role="menuitemradio"
					aria-checked={baseMapProvider === 'esri'}
					class:active={baseMapProvider === 'esri'}
					onclick={() => setBaseMap('esri')}
				>
					{$t('map.satellite')}
				</button>
				<button
					type="button"
					role="menuitemradio"
					aria-checked={baseMapProvider === 'stadia'}
					class:active={baseMapProvider === 'stadia'}
					onclick={() => setBaseMap('stadia')}
				>
					{$t('map.satellite_and_meta')}
				</button>
			</div>
		</div>
	{/if}

	{#if showZoomButton}
		<button
			type="button"
			class="map-top-right-action"
			aria-label={$t('map.open_modal')}
			title={$t('map.open_modal')}
			onclick={openMapModal}
		>
			<Fa icon={faUpRightAndDownLeftFromCenter} />
		</button>
	{/if}

	{#if showSearch}
		<div class="map-search-dock {mapSearchOpen ? 'open' : ''}">
			<div class="map-search-group">
				<button
					type="button"
					class="map-search-toggle"
					onclick={toggleMapSearch}
					aria-label={$t('map.search_place')}
				>
					<Fa icon={faSearch} />
				</button>

				<div class="map-search-inline">
					<input
						type="text"
						class="map-search-input"
						placeholder={$t('map.search_place')}
						bind:this={mapSearchInputElement}
						value={mapSearchQuery}
						oninput={handleMapSearchInput}
						onkeydown={handleMapSearchKeydown}
					/>
				</div>
			</div>

			<div class="map-search-feedback {hasSearchFeedback ? 'open' : ''}">
				{#if mapSearchLoading}
					<div class="map-search-status">{$t('search.searching')}</div>
				{:else if mapSearchError}
					<div class="map-search-status text-danger">{mapSearchError}</div>
				{:else if mapSearchResults.length > 0}
					<div class="map-search-results">
						{#each mapSearchResults as result, index (result.lat + '-' + result.lon + '-' + index)}
							<button
								type="button"
								class="map-search-result"
								onclick={() => focusMapResult(result)}
							>
								{result.label || `${result.lat.toFixed(5)}, ${result.lon.toFixed(5)}`}
							</button>
						{/each}
					</div>
				{:else if mapSearchQuery.trim().length >= 2}
					<div class="map-search-status">{$t('search.no_results')}</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<div class="toast-container position-fixed bottom-0 end-0 p-3">
	<div
		id="toastErrorUpdatePinPosition"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('map.toast.error_update_pin_position')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>

	<div
		id="toastErrorDeletePin"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('map.toast.error_delete_pin')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>

	<div
		id="toastErrorAddPin"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('map.toast.error_add_pin')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>

	<div
		id="toastErrorPhotonSearch"
		class="toast align-items-center text-bg-danger"
		role="alert"
		aria-live="assertive"
		aria-atomic="true"
	>
		<div class="d-flex">
			<div class="toast-body">
				{$t('map.toast.error_photon_search')}
			</div>
			<button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"
			></button>
		</div>
	</div>
</div>

<style>
	:global(.leaflet-marker-icon) {
		filter: drop-shadow(rgba(0, 0, 0, 0.8) 3px -2px 4px);
	}

	.map-wrapper {
		position: relative;
		height: 100%;
	}

	.map {
		height: 100%;
	}

	:global(.modal-body .map) {
		height: 100%;
	}

	.map,
	:global(.map-wrapper) {
		border-radius: 10px;
		z-index: 5;
	}

	.map-top-right-action {
		position: absolute;
		right: 12px;
		top: 12px;
		z-index: 500;
		width: 40px;
		height: 40px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: 10px;
		border: 1px solid rgba(0, 0, 0, 0.18);
		background: rgba(255, 255, 255, 0.96);
		box-shadow: 0 2px 10px rgba(0, 0, 0, 0.15);
		color: inherit;
	}

	.map-basemap-menu {
		position: absolute;
		left: 12px;
		top: 12px;
		z-index: 500;
	}

	.map-basemap-trigger {
		height: 40px;
		width: 40px;
		padding: 0 0.6rem;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: 10px;
		border: 1px solid rgba(0, 0, 0, 0.18);
		background: rgba(255, 255, 255, 0.96);
		box-shadow: 0 2px 10px rgba(0, 0, 0, 0.15);
		color: inherit;
		font-size: 0.78rem;
		font-weight: 700;
		letter-spacing: 0.04em;
	}

	.map-basemap-options {
		position: absolute;
		top: 0;
		left: 100%;
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		padding-left: 6px;
		opacity: 0;
		pointer-events: none;
		transform: translateX(-6px);
		transition:
			opacity 120ms ease,
			transform 120ms ease;
	}

	.map-basemap-menu:hover .map-basemap-options {
		opacity: 1;
		pointer-events: auto;
		transform: translateX(0);
	}

	.map-basemap-options button {
		height: 36px;
		min-width: 165px;
		padding: 0 0.55rem;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: 9px;
		border: 1px solid rgba(0, 0, 0, 0.18);
		background: rgba(255, 255, 255, 0.96);
		box-shadow: 0 2px 10px rgba(0, 0, 0, 0.15);
		color: inherit;
		font-size: 0.76rem;
		font-weight: 700;
		letter-spacing: 0.04em;
	}

	.map-basemap-options button.active {
		background: rgba(20, 122, 230, 0.75);
		color: white;
		border-color: rgba(20, 120, 230, 0.45);
	}

	:global(body[data-bs-theme='dark']) .map-top-right-action {
		background: rgba(45, 45, 45, 0.96);
		border-color: rgba(255, 255, 255, 0.22);
	}

	:global(body[data-bs-theme='dark']) .map-basemap-trigger,
	:global(body[data-bs-theme='dark']) .map-basemap-options button {
		background: rgba(45, 45, 45, 0.96);
		border-color: rgba(255, 255, 255, 0.22);
	}

	:global(body[data-bs-theme='dark']) .map-basemap-options button.active {
		background: rgba(74, 162, 255, 0.75);
		border-color: rgba(74, 162, 255, 0.5);
	}

	.map-search-dock {
		--inline-width: min(280px, calc(100vw - 110px));
		position: absolute;
		left: 12px;
		bottom: 12px;
		z-index: 500;
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
		z-index: 501;
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

	:global(.popup-blur) {
		backdrop-filter: blur(7px) saturate(130%);
		background-color: rgba(56, 56, 56, 0.38);
		color: white !important;
		font-size: 1.2em;
		border-bottom: 1px solid #1565c0;
	}

	:global(.leaflet-popup-tip) {
		backdrop-filter: blur(7px) saturate(130%);
		background-color: rgba(56, 56, 56, 0.38);
		border: 1px solid #1565c0;
	}

	:global(.leaflet-fade-anim .leaflet-popup) {
		transition: opacity 0.1s linear !important;
	}

	:global(.leaflet-control-attribution) {
		max-width: 180px;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
	}

	:global(.leaflet-control-attribution:hover),
	:global(.leaflet-control-attribution:focus-within) {
		max-width: min(78vw, 560px);
		overflow: visible;
		white-space: normal;
	}

	:global(.leaflet-container.pin-moving),
	:global(.leaflet-container.pin-moving *) {
		cursor: crosshair !important;
	}
</style>

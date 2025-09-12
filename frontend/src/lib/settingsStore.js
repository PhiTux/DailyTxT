import {writable, derived} from 'svelte/store';

export const readingMode = writable('');

export const settings = writable({});

export const tempSettings = writable({});

// should be separate, since it interacts with localStorage
export const autoLoadImagesThisDevice = writable(JSON.parse(localStorage.getItem('autoLoadImagesThisDevice')));

// Global darkMode derived from settings - available to all components
export const darkMode = derived(settings, ($settings) => {
	if (typeof window === 'undefined') return false; // SSR fallback
	
	if (!$settings || $settings.darkModeAutoDetect === undefined) {
		// Fallback to system preference if settings not loaded yet
		return window.matchMedia('(prefers-color-scheme: dark)').matches;
	}
	
	if ($settings.darkModeAutoDetect) {
		// Auto-detect: use system preference
		return window.matchMedia('(prefers-color-scheme: dark)').matches;
	} else {
		// Manual: use user choice
		return $settings.useDarkMode;
	}
});
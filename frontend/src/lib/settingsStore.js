import {writable} from 'svelte/store';

export const readingMode = writable('');

export const settings = writable({});

export const tempSettings = writable({});

// should be separate, since it interacts with localStorage
export const autoLoadImagesThisDevice = writable(JSON.parse(localStorage.getItem('autoLoadImagesThisDevice')));
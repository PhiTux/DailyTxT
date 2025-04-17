import {writable} from 'svelte/store';

export const readingMode = writable(false);

export const useTrianglify = writable(true);
export const trianglifyOpacity = writable(0.4);
export const trianglifyColor = writable('');
export const backgroundColor = writable('');

export const settings = writable({});

export const tempSettings = writable({});

export const autoLoadImagesThisDevice = writable(JSON.parse(localStorage.getItem('autoLoadImagesThisDevice')));
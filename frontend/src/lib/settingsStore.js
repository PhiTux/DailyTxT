import {writable} from 'svelte/store';

export const readingMode = writable('');

// old, to be deleted
export const useTrianglify = writable(true);
export const trianglifyOpacity = writable(0.4);
export const trianglifyColor = writable('');
export const backgroundColor = writable('');
//=========

export const settings = writable({});

export const tempSettings = writable({});

// should be separate, since it interacts with localStorage
export const autoLoadImagesThisDevice = writable(JSON.parse(localStorage.getItem('autoLoadImagesThisDevice')));
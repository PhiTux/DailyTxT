import {writable} from 'svelte/store';

export const readingMode = writable(false);

export const useTrianglify = writable(true);
export const trianglifyOpacity = writable(0.4);
export const trianglifyColor = writable('');
export const backgroundColor = writable('');
export const autoLoadImages = writable(true);

export const settings = writable({
  useTrianglify: true,
  trianglifyOpacity: 0.4,
  trianglifyColor: '',
  backgroundColor: '',
  autoloadImagesDefault: true,
  saveAutoloadImagesPerDevice: true,
});

export const tempSettings = writable({});
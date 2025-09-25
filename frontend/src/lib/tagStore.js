import {writable} from 'svelte/store';

export let tags = writable([]);
export let tagsLoaded = writable(false);
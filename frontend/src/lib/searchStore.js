import { writable } from "svelte/store";

export let searchString = writable("");
export let searchResults = writable([]);
export let searchTag = writable({});
export let isSearching = writable(false);
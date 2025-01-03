import { writable } from "svelte/store";

export let searchString = writable("");
export let searchResults = writable([]);
import {writable} from 'svelte/store';

function formatBytes(bytes) {
	if (!+bytes) return '0 Bytes';

	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB'];

	const i = Math.floor(Math.log(bytes) / Math.log(k));

	return `${parseFloat((bytes / Math.pow(k, i)).toFixed(0))} ${sizes[i]}`;
}

export { formatBytes };

export let alwaysShowSidenav = writable(true);

// check if offcanvas/sidenav is open
export let offcanvasIsOpen = writable(false);
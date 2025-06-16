import {writable} from 'svelte/store';

function formatBytes(bytes) {
	if (!+bytes) return '0 Bytes';

	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB'];

	const i = Math.floor(Math.log(bytes) / Math.log(k));

	return `${parseFloat((bytes / Math.pow(k, i)).toFixed(0))} ${sizes[i]}`;
}

function sameDate(date1, date2) {
	if (!date1 || !date2) return false;
	return (
		date1.day === date2.day &&
		date1.month === date2.month &&
		date1.year === date2.year
	);
}

export { formatBytes, sameDate };

export let alwaysShowSidenav = writable(true);

// check if offcanvas/sidenav is open
export let offcanvasIsOpen = writable(false);
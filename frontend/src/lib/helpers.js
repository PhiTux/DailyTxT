import {writable} from 'svelte/store';
import json from '../i18n/flags.json';

function formatBytes(bytes) {
	if (!+bytes) return '0 Bytes';

	const k = 1024;
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];

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

export const isAuthenticated = writable(false);

// Function to check if page load authentication is required
function needsReauthentication() {
	isAuthenticated.subscribe((value) => {
		if (value) return false;
	})

	if (typeof window === 'undefined') return false;

	// Check localStorage for re-auth requirement
	const requireReauth = localStorage.getItem('requirePasswordOnPageLoad');
	
	if (requireReauth !== 'true') {
		isAuthenticated.set(true);
	}

	return requireReauth === 'true';
}

function generateNeonMesh(dark) {
	const baseColors = ["#ff00ff", "#00ffff", "#ffea00", "#ff0080", "#00ff80", "#ff4500",
    "#ff1493", "#00ffcc", "#ff3333", "#66ff66", "#3399ff", "#ffcc00",
    "#ff6666", "#00ccff", "#cc33ff", "#33ffcc", "#ffff99", "#ff99ff",
    "#99ff99", "#66ccff", "#ff9900", "#ff0066", "#66ffcc", "#ff33cc",
    "#99ccff"]
	const numGradients = Math.floor(Math.random() * 3) + 3; // 3–5 Radial Gradients
	let gradients = [];

	for (let i = 0; i < numGradients; i++) {
		const baseColor = baseColors[Math.floor(Math.random() * baseColors.length)];
		const alpha = Math.random() * 0.4 + 0.1; // random alpha between 0.1 and 0.5

		// convert hex color to rgba
		const hex = baseColor.substring(1);
		const r = parseInt(hex.substring(0, 2), 16);
		const g = parseInt(hex.substring(2, 4), 16);
		const b = parseInt(hex.substring(4, 6), 16);
		const color = `rgba(${r}, ${g}, ${b}, ${alpha})`;

		const x = Math.floor(Math.random() * 100);
		const y = Math.floor(Math.random() * 100);
		const size = Math.floor(Math.random() * 30) + 30; // 30–60%
		gradients.push(`radial-gradient(circle at ${x}% ${y}%, ${color}, transparent ${size}%)`);
	}

	document.querySelector('.background').style.background = dark ? gradients.join(', ') + ', #111' : gradients;
	document.querySelector('.background').style.backgroundBlendMode = 'screen';
}

function loadFlagEmoji(language) {
	return json[language] || '';
}

export { formatBytes, sameDate, needsReauthentication, generateNeonMesh, loadFlagEmoji };

export let alwaysShowSidenav = writable(true);

// check if offcanvas/sidenav is open
export let offcanvasIsOpen = writable(false);

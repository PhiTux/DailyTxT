import { dev } from '$app/environment';

export let API_URL = dev
		? `${window.location.origin.replace(/:5173.*$/gm, '')}:8000`
		: window.location.pathname.replace(/\/+$/, '');
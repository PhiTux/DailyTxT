import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';

// Catch-all for unknown routes: redirect to /write (primary app surface)
// If /write itself handles auth, unauthenticated users will still be bounced to /login there.
// Adjust here if you later want a different fallback (e.g. redirect to /login when not authenticated).
export function load() {
	throw redirect(307, resolve('/write'));
}

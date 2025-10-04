import { dev, browser } from '$app/environment';
import { base } from '$app/paths';

// Base-aware API root:
// - Dev: talk to backend on port 8000 at same host
// - Prod: prefix with SvelteKit base (works for subpath deployments like /dailytxt)
const apiPath = `${base}`.replace(/\/$/, '') + '/api';

export const API_URL = browser
  ? (dev
      ? `${window.location.protocol}//${window.location.hostname}:8000/api`
      : apiPath)
  : apiPath;
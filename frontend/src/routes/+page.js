import { redirect } from "@sveltejs/kit";
import { resolve } from '$app/paths';

export function load() {
  // Redirect to the /write route
  throw redirect(307, resolve("/write"));
}
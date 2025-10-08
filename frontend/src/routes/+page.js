import { redirect } from "@sveltejs/kit";

export function load() {
  // Redirect to the /write route
  throw redirect(307, "/write");
}
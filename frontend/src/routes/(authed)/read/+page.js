import {redirect} from '@sveltejs/kit'
import { resolve } from '$app/paths';

export const load = () => {
  const user = localStorage.getItem('user');
		if (!user) {
			throw redirect(307, resolve('/login'));
		}
}
import {redirect} from '@sveltejs/kit'

export const load = () => {
  const user = localStorage.getItem('user');
		if (user) {
			throw redirect(307, '/write');
		}
}
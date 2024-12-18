import {redirect} from '@sveltejs/kit'

export const load = () => {
  const user = JSON.parse(localStorage.getItem('user'));
		if (user) {
			throw redirect(307, '/');
		}
}
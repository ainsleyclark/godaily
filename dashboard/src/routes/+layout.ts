import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';

export const ssr = false;
export const prerender = false;

export const load = ({ url }) => {
	if (!browser) return {};
	const secret = localStorage.getItem('godaily_api_secret');
	if (!secret && url.pathname !== '/login') {
		throw redirect(307, '/login');
	}
	if (secret && url.pathname === '/login') {
		throw redirect(307, '/');
	}
	return {};
};

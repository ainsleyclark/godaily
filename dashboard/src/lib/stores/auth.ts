import { writable, derived, get } from 'svelte/store';
import { browser } from '$app/environment';

const STORAGE_KEY = 'godaily_api_secret';

function createAuthStore() {
	const initial = browser ? (localStorage.getItem(STORAGE_KEY) ?? null) : null;
	const { subscribe, set } = writable<string | null>(initial);
	return {
		subscribe,
		setSecret(secret: string) {
			if (browser) localStorage.setItem(STORAGE_KEY, secret);
			set(secret);
		},
		clearSecret() {
			if (browser) localStorage.removeItem(STORAGE_KEY);
			set(null);
		}
	};
}

export const auth = createAuthStore();
export const isAuthenticated = derived(auth, ($auth) => !!$auth);

export function getSecret(): string | null {
	return get(auth);
}

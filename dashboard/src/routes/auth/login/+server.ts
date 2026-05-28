import { json, error } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import type { RequestHandler } from './$types';

// POST /auth/login — exchanges the dashboard password for the API secret.
// Runs server-side only, so DASHBOARD_PASSWORD and API_SECRET never reach the
// browser. When no password is configured (dev/CI) any request succeeds.
export const POST: RequestHandler = async ({ request }) => {
	let password = '';
	try {
		({ password } = await request.json());
	} catch {
		throw error(400, 'password is required');
	}

	const expected = env.DASHBOARD_PASSWORD ?? '';
	if (expected !== '' && password !== expected) {
		throw error(401, 'invalid password');
	}

	return json({ data: { token: env.API_SECRET ?? '' } });
};

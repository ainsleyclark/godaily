import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, '', '');
	const target = env.VITE_API_PROXY_TARGET || 'http://localhost:3000';
	return {
		plugins: [tailwindcss(), sveltekit()],
		server: {
			port: 5173,
			proxy: {
				'/api': {
					target,
					changeOrigin: true,
					secure: target.startsWith('https')
				}
			}
		}
	};
});

import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), '');
	const backendPort = env.BACKEND_PORT || env.PORT || '8084';
	const backendUrl = `http://127.0.0.1:${backendPort}`;

	return {
		plugins: [tailwindcss(), sveltekit({ adapter: adapter() })],
		server: {
			port: 3008,
			proxy: {
				'/api': {
					target: backendUrl,
					changeOrigin: true
				}
			}
		}
	};
});

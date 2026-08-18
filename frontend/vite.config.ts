import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, loadEnv } from 'vite';

const rootDir = join(dirname(fileURLToPath(import.meta.url)), '..');

function readAppVersion(): string {
	try {
		return readFileSync(join(rootDir, 'VERSION'), 'utf8').trim() || '1.0.0';
	} catch {
		return '1.0.0';
	}
}

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), '');
	const backendPort = env.BACKEND_PORT || env.PORT || '8084';
	const backendUrl = `http://127.0.0.1:${backendPort}`;
	const appVersion = env.PUBLIC_APP_VERSION || readAppVersion();

	return {
		plugins: [tailwindcss(), sveltekit({ adapter: adapter() })],
		define: {
			'import.meta.env.PUBLIC_APP_VERSION': JSON.stringify(appVersion)
		},
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

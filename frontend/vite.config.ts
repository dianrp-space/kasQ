import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, loadEnv, type Plugin } from 'vite';

const rootDir = join(dirname(fileURLToPath(import.meta.url)), '..');

function readAppVersion(): string {
	try {
		return readFileSync(join(rootDir, 'VERSION'), 'utf8').trim() || '1.0.0';
	} catch {
		return '1.0.0';
	}
}

function changelogPlugin(): Plugin {
	const virtualId = 'virtual:changelog';
	const resolvedId = '\0' + virtualId;
	const changelogPath = join(rootDir, 'CHANGELOG.md');
	return {
		name: 'kasq-changelog',
		resolveId(id) {
			if (id === virtualId) return resolvedId;
		},
		load(id) {
			if (id !== resolvedId) return;
			const raw = readFileSync(changelogPath, 'utf8');
			return `export default ${JSON.stringify(raw)};`;
		},
		handleHotUpdate({ file, server }) {
			if (file !== changelogPath) return;
			const mod = server.moduleGraph.getModuleById(resolvedId);
			if (mod) {
				server.moduleGraph.invalidateModule(mod);
				server.ws.send({ type: 'full-reload' });
			}
		}
	};
}

export default defineConfig(({ mode }) => {
	const env = loadEnv(mode, process.cwd(), '');
	const backendPort = env.BACKEND_PORT || env.PORT || '8084';
	const backendUrl = `http://127.0.0.1:${backendPort}`;
	const appVersion = env.PUBLIC_APP_VERSION || readAppVersion();

	return {
		plugins: [changelogPlugin(), tailwindcss(), sveltekit({ adapter: adapter() })],
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

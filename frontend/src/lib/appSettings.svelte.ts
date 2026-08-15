import { api } from '$lib/api';

export type AppSettings = {
	app_name: string;
	app_tagline: string;
	logo_url?: string;
	favicon_url?: string;
};

export const defaultAppSettings: AppSettings = {
	app_name: 'KasQ',
	app_tagline: 'Kas Ku — Pencatatan Keuangan Tim/Kas'
};

export const appSettings = $state<AppSettings>({ ...defaultAppSettings });

function applySettings(data: AppSettings) {
	appSettings.app_name = data.app_name;
	appSettings.app_tagline = data.app_tagline;
	appSettings.logo_url = data.logo_url;
	appSettings.favicon_url = data.favicon_url;
}

export async function loadAppSettings() {
	try {
		applySettings(await api.getAppSettings());
	} catch {
		applySettings({ ...defaultAppSettings });
	}
}

export function brandingUrl(path?: string) {
	if (!path) return undefined;
	if (path.startsWith('http')) return path;
	const base = import.meta.env.PUBLIC_API_URL || '';
	return `${base}${path}`;
}

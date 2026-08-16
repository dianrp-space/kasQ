const API_URL = import.meta.env.PUBLIC_API_URL || '';

export class ApiError extends Error {
	needsVerification = false;
	noTeam = false;
	constructor(message: string, needsVerification = false, noTeam = false) {
		super(message);
		this.needsVerification = needsVerification;
		this.noTeam = noTeam;
	}
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
	const res = await fetch(`${API_URL}${path}`, {
		...options,
		credentials: 'include',
		headers: {
			...(options.body instanceof FormData ? {} : { 'Content-Type': 'application/json' }),
			...options.headers
		}
	});
	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiError(err.error || 'Request failed', err.needs_verification === true, err.no_team === true);
	}
	return res.json();
}

export const api = {
	login: (email: string, password: string) =>
		request<{ user: import('./types').User; token: string }>('/api/auth/login', {
			method: 'POST',
			body: JSON.stringify({ email, password })
		}),
	logout: () => request<{ ok: boolean }>('/api/auth/logout', { method: 'POST' }),
	register: (data: { name: string; email: string; password: string }) =>
		request<{ message: string }>('/api/auth/register', {
			method: 'POST',
			body: JSON.stringify(data)
		}),
	verifyEmail: (token: string) =>
		request<{ message: string }>(`/api/auth/verify-email?token=${encodeURIComponent(token)}`),
	forgotPassword: (email: string) =>
		request<{ message: string }>('/api/auth/forgot-password', {
			method: 'POST',
			body: JSON.stringify({ email })
		}),
	resetPassword: (token: string, password: string) =>
		request<{ message: string }>('/api/auth/reset-password', {
			method: 'POST',
			body: JSON.stringify({ token, password })
		}),
	resendVerification: (email: string) =>
		request<{ message: string }>('/api/auth/resend-verification', {
			method: 'POST',
			body: JSON.stringify({ email })
		}),
	me: () => request<import('./types').User>('/api/me'),
	updateMe: (form: FormData) =>
		request<import('./types').User>('/api/me', { method: 'PUT', body: form }),
	getMyAvatar: async (): Promise<string | null> => {
		const res = await fetch(`${API_URL}/api/me/avatar`, { credentials: 'include' });
		if (!res.ok) return null;
		const data = await res.json();
		return data.url ?? null;
	},

	getAppSettings: () =>
		request<import('./types').AppSettings>('/api/app-settings'),
	updateAppSettings: (form: FormData) =>
		request<import('./types').AppSettings>('/api/admin/app-settings', {
			method: 'PUT',
			body: form
		}),

	getTeams: () => request<import('./types').Team[]>('/api/teams'),
	createTeam: (data: { name: string; slug?: string; initial_balance: number }) =>
		request<import('./types').Team>('/api/teams', { method: 'POST', body: JSON.stringify(data) }),
	updateTeam: (id: string, data: { name: string; slug: string; initial_balance: number }) =>
		request<import('./types').Team>(`/api/teams/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	deleteTeam: (id: string) => request<{ ok: boolean }>(`/api/teams/${id}`, { method: 'DELETE' }),

	getUsers: () => request<import('./types').User[]>('/api/users'),
	createUser: (data: {
		name: string;
		email: string;
		password: string;
		role: string;
		team_id?: string;
	}) => request<import('./types').User>('/api/users', { method: 'POST', body: JSON.stringify(data) }),
	updateUser: (
		id: string,
		data: { name: string; email: string; password?: string; role: string; team_id?: string }
	) => request<import('./types').User>(`/api/users/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	deleteUser: (id: string) => request<{ ok: boolean }>(`/api/users/${id}`, { method: 'DELETE' }),

	getBalance: (teamId: string, params?: Record<string, string>) => {
		const qs = params ? '?' + new URLSearchParams(params).toString() : '';
		return request<import('./types').Balance>(`/api/teams/${teamId}/balance${qs}`);
	},
	getTransactions: (teamId: string, params?: Record<string, string>) => {
		const qs = params ? '?' + new URLSearchParams(params).toString() : '';
		return request<import('./types').TransactionListResponse>(`/api/teams/${teamId}/transactions${qs}`);
	},
	createTransaction: (teamId: string, form: FormData) =>
		request<{ transaction: import('./types').Transaction; balance: import('./types').Balance }>(
			`/api/teams/${teamId}/transactions`,
			{ method: 'POST', body: form }
		),
	updateTransaction: (teamId: string, txId: string, form: FormData) =>
		request<{ transaction: import('./types').Transaction; balance: import('./types').Balance }>(
			`/api/teams/${teamId}/transactions/${txId}`,
			{ method: 'PUT', body: form }
		),
	deleteTransaction: (teamId: string, txId: string) =>
		request<{ ok: boolean; balance: import('./types').Balance }>(
			`/api/teams/${teamId}/transactions/${txId}`,
			{ method: 'DELETE' }
		),
	batchDeleteTransactions: (teamId: string, ids: string[]) =>
		request<{ ok: boolean; deleted: number; balance: import('./types').Balance }>(
			`/api/teams/${teamId}/transactions/batch-delete`,
			{ method: 'POST', body: JSON.stringify({ ids }) }
		),
	importTransactions: (teamId: string, form: FormData) =>
		request<import('./types').ImportResult>(`/api/teams/${teamId}/transactions/import`, {
			method: 'POST',
			body: form
		}),
	importTransactionsStream: async (
		teamId: string,
		form: FormData,
		onProgress: (ev: import('./types').ImportProgressEvent) => void
	): Promise<import('./types').ImportResult> => {
		form.append('stream', 'true');
		const res = await fetch(`${API_URL}/api/teams/${teamId}/transactions/import?stream=1`, {
			method: 'POST',
			body: form,
			credentials: 'include'
		});
		if (!res.ok) {
			if (res.status === 504) {
				throw new ApiError(
					'Gateway timeout (504). Sebagian transaksi mungkin sudah masuk — cek dashboard. Import ulang akan melewati duplikat. Untuk file besar, uncentang unduh nota dulu atau pecah per bulan.'
				);
			}
			const err = await res.json().catch(() => ({ error: res.statusText }));
			throw new ApiError(err.error || `Import gagal (HTTP ${res.status})`);
		}
		if (!res.body) {
			throw new ApiError('Import gagal: respons kosong');
		}
		const reader = res.body.getReader();
		const decoder = new TextDecoder();
		let buffer = '';
		let finalResult: import('./types').ImportResult | null = null;

		while (true) {
			const { done, value } = await reader.read();
			if (done) break;
			buffer += decoder.decode(value, { stream: true });
			const lines = buffer.split('\n');
			buffer = lines.pop() ?? '';
			for (const line of lines) {
				if (!line.trim()) continue;
				const ev = JSON.parse(line) as import('./types').ImportProgressEvent;
				if (ev.type === 'error') {
					throw new ApiError(ev.message || 'Import gagal');
				}
				if (ev.type === 'progress') {
					onProgress(ev);
				}
				if (ev.type === 'done' && ev.result) {
					finalResult = ev.result;
					onProgress(ev);
				}
			}
		}

		if (!finalResult) {
			throw new ApiError(
				'Koneksi terputus sebelum import selesai. Sebagian data mungkin sudah masuk — cek dashboard lalu import ulang (duplikat otomatis dilewati).'
			);
		}
		return finalResult;
	},
	downloadImportTemplate: async (teamId: string) => {
		const res = await fetch(`${API_URL}/api/teams/${teamId}/transactions/import/template`, {
			credentials: 'include'
		});
		if (!res.ok) {
			const err = await res.json().catch(() => ({ error: res.statusText }));
			throw new ApiError(err.error || 'Gagal unduh template');
		}
		const blob = await res.blob();
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = 'kasq-import-template.xlsx';
		a.click();
		URL.revokeObjectURL(url);
	},

	getIntegration: (teamId: string) =>
		request<import('./types').Integration>(`/api/teams/${teamId}/integrations`),
	updateWA: (teamId: string, enabled: boolean) =>
		request<{ ok: boolean }>(`/api/teams/${teamId}/integrations/wa`, {
			method: 'PUT',
			body: JSON.stringify({ enabled })
		}),
	getWAQR: (teamId: string) =>
		request<{
			status: string;
			qr: string;
			phone?: string;
			wa_name?: string;
			wa_picture_url?: string;
			pair_code?: string;
			qr_timeout_seconds?: number;
			pair_code_expires_seconds?: number;
			login_mode?: string;
		}>(`/api/teams/${teamId}/integrations/wa/qr`),
	startWAQRLogin: (teamId: string) =>
		request<{ ok: boolean }>(`/api/teams/${teamId}/integrations/wa/qr/start`, { method: 'POST' }),
	startWAPairLogin: (teamId: string, phone: string) =>
		request<{ pair_code: string; expires_seconds: number; status: string }>(
			`/api/teams/${teamId}/integrations/wa/pair`,
			{ method: 'POST', body: JSON.stringify({ phone }) }
		),
	updateTele: (
		teamId: string,
		data: { enabled: boolean; bot_token?: string; chat_id?: number | null }
	) =>
		request<{ ok: boolean }>(`/api/teams/${teamId}/integrations/tele`, {
			method: 'PUT',
			body: JSON.stringify(data)
		}),
	getTeleBotAvatar: async (teamId: string): Promise<string | null> => {
		const res = await fetch(`${API_URL}/api/teams/${teamId}/integrations/tele/avatar`, {
			credentials: 'include'
		});
		if (!res.ok) return null;
		const blob = await res.blob();
		return URL.createObjectURL(blob);
	},
	getWABotAvatar: async (teamId: string): Promise<string | null> => {
		const res = await fetch(`${API_URL}/api/teams/${teamId}/integrations/wa/avatar`, {
			credentials: 'include'
		});
		if (!res.ok) return null;
		const blob = await res.blob();
		return URL.createObjectURL(blob);
	},
	updateReportToken: (teamId: string, slug: string) =>
		request<{ token: string; report_url: string }>(`/api/teams/${teamId}/report-token`, {
			method: 'PUT',
			body: JSON.stringify({ slug })
		}),
	resetReportToken: (teamId: string) =>
		request<{ token: string; report_url: string }>(
			`/api/teams/${teamId}/report-token/reset`,
			{ method: 'POST' }
		),
	getNotaURL: (teamId: string, key: string, download = false) =>
		request<{ url: string }>(
			`/api/teams/${teamId}/nota-url?key=${encodeURIComponent(key)}&download=${download}`
		),

	getPublicReport: (token: string, params?: Record<string, string>) => {
		const qs = params ? '?' + new URLSearchParams(params).toString() : '';
		return request<import('./types').PublicReport>(`/api/public/report/${token}${qs}`);
	},
	getPublicNotaURL: async (token: string, key: string, download = false) => {
		const params = new URLSearchParams({ key });
		if (download) params.set('download', 'true');
		const res = await fetch(`${API_URL}/api/public/nota/${token}?${params}`);
		if (!res.ok) return null;
		const data = await res.json();
		return data.url ?? null;
	}
};

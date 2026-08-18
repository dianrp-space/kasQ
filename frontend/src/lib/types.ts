export interface User {
	id: string;
	name: string;
	email: string;
	role: 'admin' | 'ops';
	team_id?: string;
	has_avatar?: boolean;
}

export interface AppSettings {
	app_name: string;
	app_tagline: string;
	logo_url?: string;
	favicon_url?: string;
}

export interface Team {
	id: string;
	name: string;
	slug: string;
	initial_balance: number;
	created_at: string;
}

export interface Balance {
	initial_balance: number;
	opening_balance: number;
	total_in: number;
	total_out: number;
	current_balance: number;
	period_from?: string;
	period_to?: string;
}

export interface Transaction {
	id: string;
	team_id: string;
	created_by?: string;
	hari: string;
	tanggal: string;
	jenis: 'in' | 'out';
	deskripsi: string;
	total: number;
	nota_key?: string;
	nota_keys?: string[];
	keterangan?: string;
	source: 'web' | 'wa' | 'tele';
	created_at: string;
	creator_name?: string;
}

export interface TransactionListResponse {
	items: Transaction[];
	total: number;
	limit: number;
	page: number;
	offset: number;
}

export interface Integration {
	team_id: string;
	wa_enabled: boolean;
	wa_status: string;
	wa_phone?: string;
	wa_name?: string;
	wa_has_avatar?: boolean;
	wa_picture_url?: string;
	wa_allowed_phones?: string[];
	tele_enabled: boolean;
	tele_use_system_bot?: boolean;
	tele_bot_token?: string;
	tele_allowed_chat_id?: number;
	tele_bot_name?: string;
	tele_bot_username?: string;
	tele_bot_has_avatar?: boolean;
	has_tele_token: boolean;
	system_tele_bot_available?: boolean;
	system_tele_bot_name?: string;
	system_tele_bot_username?: string;
	system_tele_bot_has_avatar?: boolean;
	report_token?: string;
	report_url?: string;
	team_slug?: string;
	team_name?: string;
}

export interface PublicReport {
	team: Team;
	balance: Balance;
	transactions: Transaction[];
}

export interface ImportRowError {
	row: number;
	sheet?: string;
	message: string;
}

export interface ImportResult {
	imported: number;
	failed: number;
	skipped: number;
	duplicates: number;
	sheets_used: number;
	errors: ImportRowError[];
	balance: Balance;
}

export interface ImportProgressEvent {
	type: 'progress' | 'done' | 'error';
	phase?: 'prepare' | 'sheet' | 'row' | 'nota' | 'finish';
	message?: string;
	sheet?: string;
	row?: number;
	current?: number;
	total?: number;
	imported?: number;
	failed?: number;
	skipped?: number;
	duplicates?: number;
	result?: ImportResult;
}

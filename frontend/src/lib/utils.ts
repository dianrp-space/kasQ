export function formatRupiah(n: number): string {
	return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0 }).format(n);
}

export function formatDate(d: string): string {
	return new Date(d).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
}

export const HARI_LIST = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'];

export function getDayName(date: Date): string {
	return HARI_LIST[date.getDay() === 0 ? 6 : date.getDay() - 1];
}

export function toInputDate(d: Date = new Date()): string {
	const y = d.getFullYear();
	const m = String(d.getMonth() + 1).padStart(2, '0');
	const day = String(d.getDate()).padStart(2, '0');
	return `${y}-${m}-${day}`;
}

/** Rentang tanggal bulan kalender (local timezone). */
export function getCurrentMonthRange(date: Date = new Date()): { from: string; to: string } {
	const start = new Date(date.getFullYear(), date.getMonth(), 1);
	const end = new Date(date.getFullYear(), date.getMonth() + 1, 0);
	return { from: toInputDate(start), to: toInputDate(end) };
}

/** Nilai untuk input type="month" (YYYY-MM). */
export function toInputMonth(d: Date = new Date()): string {
	const y = d.getFullYear();
	const m = String(d.getMonth() + 1).padStart(2, '0');
	return `${y}-${m}`;
}

/** Rentang tanggal dari input bulan (YYYY-MM). */
export function getMonthRange(monthInput: string): { from: string; to: string } {
	const match = /^(\d{4})-(\d{2})$/.exec(monthInput);
	if (!match) return getCurrentMonthRange();
	const y = Number(match[1]);
	const m = Number(match[2]);
	if (m < 1 || m > 12) return getCurrentMonthRange();
	const start = new Date(y, m - 1, 1);
	const end = new Date(y, m, 0);
	return { from: toInputDate(start), to: toInputDate(end) };
}

export function formatMonthLabel(from: string): string {
	return new Date(from + 'T00:00:00').toLocaleDateString('id-ID', {
		month: 'long',
		year: 'numeric'
	});
}

/** Salam sesuai jam lokal (WIB / timezone browser). */
export function timeGreeting(date = new Date()): string {
	const h = date.getHours();
	if (h >= 4 && h < 11) return 'Selamat pagi';
	if (h >= 11 && h < 15) return 'Selamat siang';
	if (h >= 15 && h < 19) return 'Selamat sore';
	return 'Selamat malam';
}

export function greetingWithName(name: string, date = new Date()): string {
	const trimmed = name.trim();
	if (!trimmed) return timeGreeting(date);
	const first = trimmed.split(/\s+/)[0];
	return `${timeGreeting(date)}, ${first}`;
}

export function nameInitials(name: string): string {
	return (
		name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('') || '?'
	);
}

export function jenisLabel(j: 'in' | 'out'): string {
	return j === 'in' ? 'Masuk' : 'Keluar';
}

export function sourceLabel(s: 'web' | 'wa' | 'tele'): string {
	const map = { web: 'Web', wa: 'WhatsApp', tele: 'Telegram' };
	return map[s];
}

export const MAX_NOTA_FILES = 10;

export function txNotaKeys(tx: { nota_keys?: string[]; nota_key?: string }): string[] {
	if (tx.nota_keys && tx.nota_keys.length > 0) return tx.nota_keys;
	if (tx.nota_key) return [tx.nota_key];
	return [];
}

export function notaFilenameFromKey(key: string, index = 0): string {
	const base = key.split('/').pop()?.trim();
	if (base) return base;
	return `nota-${index + 1}.jpg`;
}

function uniqueArchiveNames(filenames: string[]): string[] {
	const seen = new Map<string, number>();
	return filenames.map((name) => {
		const count = seen.get(name) ?? 0;
		seen.set(name, count + 1);
		if (count === 0) return name;
		const dot = name.lastIndexOf('.');
		if (dot > 0) {
			return `${name.slice(0, dot)}-${count + 1}${name.slice(dot)}`;
		}
		return `${name}-${count + 1}`;
	});
}

export async function triggerDownloadUrl(url: string, filename: string): Promise<void> {
	const res = await fetch(url);
	if (!res.ok) throw new Error('download failed');
	const blob = await res.blob();
	const objectUrl = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = objectUrl;
	a.download = filename;
	a.click();
	URL.revokeObjectURL(objectUrl);
}

/** Unduh banyak file sebagai satu arsip ZIP (seperti Immich / Google Drive). */
export async function downloadFilesAsZip(
	items: { url: string; filename: string }[],
	zipName = 'nota.zip'
): Promise<void> {
	if (items.length === 0) return;
	if (items.length === 1) {
		await triggerDownloadUrl(items[0].url, items[0].filename);
		return;
	}

	const { zipSync } = await import('fflate');
	const names = uniqueArchiveNames(items.map((i) => i.filename));
	const archive: Record<string, Uint8Array> = {};

	for (let i = 0; i < items.length; i++) {
		const res = await fetch(items[i].url);
		if (!res.ok) throw new Error(`gagal unduh ${names[i]}`);
		archive[names[i]] = new Uint8Array(await res.arrayBuffer());
	}

	const zipped = zipSync(archive);
	const blob = new Blob([zipped], { type: 'application/zip' });
	const objectUrl = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = objectUrl;
	a.download = zipName.endsWith('.zip') ? zipName : `${zipName}.zip`;
	a.click();
	URL.revokeObjectURL(objectUrl);
}

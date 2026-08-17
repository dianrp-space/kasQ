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

export function jenisLabel(j: 'in' | 'out'): string {
	return j === 'in' ? 'Pemasukan' : 'Pengeluaran';
}

export function sourceLabel(s: 'web' | 'wa' | 'tele'): string {
	const map = { web: 'Web', wa: 'WhatsApp', tele: 'Telegram' };
	return map[s];
}

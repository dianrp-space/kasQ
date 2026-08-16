<script lang="ts">
	import { api, ApiError } from '$lib/api';
	import type { ImportProgressEvent, ImportResult } from '$lib/types';
	import { formatRupiah } from '$lib/utils';
	import { Alert, Button, Checkbox, Label, Modal, P, Progressbar, Spinner } from 'flowbite-svelte';

	let {
		open = $bindable(false),
		teamId = '',
		onImported
	}: {
		open: boolean;
		teamId: string;
		onImported?: (result: ImportResult) => void;
	} = $props();

	let file = $state<File | null>(null);
	let fetchNota = $state(true);
	let loading = $state(false);
	let downloading = $state(false);
	let error = $state('');
	let result = $state<ImportResult | null>(null);
	let progress = $state<ImportProgressEvent | null>(null);
	let progressLog = $state<string[]>([]);

	const progressPct = $derived.by(() => {
		if (!progress?.total || progress.total <= 0) return 0;
		return Math.min(100, Math.round(((progress.current ?? 0) / progress.total) * 100));
	});

	const phaseLabel = $derived.by(() => {
		if (!progress) return 'Memulai...';
		switch (progress.phase) {
			case 'prepare':
				return 'Persiapan';
			case 'sheet':
				return progress.sheet ? `Sheet: ${progress.sheet}` : 'Memproses sheet';
			case 'nota':
				return 'Mengunduh nota';
			case 'row':
				return 'Memproses baris';
			case 'finish':
				return 'Menyelesaikan';
			default:
				return progress.message || 'Memproses...';
		}
	});

	function pushLog(message: string) {
		if (!message) return;
		const next = [...progressLog, message];
		progressLog = next.length > 8 ? next.slice(-8) : next;
	}

	$effect(() => {
		if (open) {
			file = null;
			fetchNota = true;
			error = '';
			result = null;
			loading = false;
			progress = null;
			progressLog = [];
		}
	});

	async function downloadTemplate() {
		if (!teamId) return;
		downloading = true;
		error = '';
		try {
			await api.downloadImportTemplate(teamId);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal unduh template';
		} finally {
			downloading = false;
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		if (!teamId || !file) {
			error = 'Pilih file Excel (.xlsx) dulu';
			return;
		}
		loading = true;
		error = '';
		result = null;
		progress = { type: 'progress', phase: 'prepare', message: 'Mengunggah file...' };
		progressLog = [];
		try {
			const form = new FormData();
			form.append('file', file);
			form.append('fetch_nota', fetchNota ? 'true' : 'false');
			const res = await api.importTransactionsStream(teamId, form, (ev) => {
				if (ev.type === 'progress') {
					progress = ev;
					if (ev.message) pushLog(ev.message);
				}
			});
			result = res;
			onImported?.(res);
		} catch (err) {
			error = err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Import gagal';
		} finally {
			loading = false;
			progress = null;
		}
	}
</script>

<Modal bind:open title="Import dari Excel" size="lg" autoclose={false}>
	<form onsubmit={submit} class="space-y-4">
		<P class="text-sm text-slate-600 dark:text-slate-400">
			Semua sheet dalam file .xlsx dibaca sekaligus (mis. Februari, Maret, …). Setiap sheet butuh header:
			<strong>Hari</strong>, <strong>Tanggal</strong>, <strong>Jenis</strong>, <strong>Deskripsi</strong>,
			<strong>Total</strong>, <strong>Link Nota</strong> (opsional). Total minus otomatis dijadikan positif.
		</P>

		<div class="flex flex-wrap gap-2">
			<Button type="button" color="light" size="sm" disabled={downloading || !teamId} onclick={downloadTemplate}>
				{downloading ? 'Mengunduh...' : 'Unduh template .xlsx'}
			</Button>
		</div>

		<div>
			<Label for="import-file">File Excel</Label>
			<input
				id="import-file"
				type="file"
				accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
				class="block w-full cursor-pointer rounded-lg border border-gray-300 bg-gray-50 text-sm text-gray-900 focus:outline-none dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
				disabled={loading}
				onchange={(e) => {
					file = (e.target as HTMLInputElement).files?.[0] ?? null;
				}}
			/>
		</div>

		<Checkbox bind:checked={fetchNota} disabled={loading}>
			Unduh Link Nota ke MinIO (jika ada URL di kolom Link Nota / hyperlink "Lihat Gambar")
		</Checkbox>

		{#if loading && progress}
			<div class="rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-slate-700 dark:bg-slate-800/80">
				<div class="mb-2 flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200">
					<Spinner size="4" />
					<span>{phaseLabel}</span>
					{#if progress.total}
						<span class="text-slate-500 dark:text-slate-400">
							({progress.current ?? 0}/{progress.total})
						</span>
					{/if}
				</div>
				<Progressbar progress={progressPct} size="h-2.5" />
				<p class="mt-2 text-xs text-slate-500 dark:text-slate-400">{progressPct}% — {progress.message || 'Memproses...'}</p>
				<div class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-slate-600 dark:text-slate-300">
					<span>Berhasil: {progress.imported ?? 0}</span>
					<span>Gagal: {progress.failed ?? 0}</span>
					<span>Dilewati: {progress.skipped ?? 0}</span>
					<span>Duplikat: {progress.duplicates ?? 0}</span>
				</div>
				{#if fetchNota && progress.phase === 'nota'}
					<p class="mt-2 text-xs text-amber-700 dark:text-amber-400">
						Mengunduh nota per baris — ini yang paling lama di server produksi.
					</p>
				{/if}
				{#if progressLog.length > 0}
					<div class="mt-2 max-h-24 overflow-y-auto rounded border border-slate-200 bg-white p-2 font-mono text-[11px] text-slate-600 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-400">
						{#each progressLog as line}
							<p class="truncate">{line}</p>
						{/each}
					</div>
				{/if}
			</div>
		{/if}

		{#if error}
			<Alert color="red">{error}</Alert>
		{/if}

		{#if result}
			<Alert color={result.failed > 0 ? 'yellow' : 'green'}>
				<p>
					<strong>{result.sheets_used}</strong> sheet diproses —
					berhasil import <strong>{result.imported}</strong> baris.
					{#if result.duplicates > 0}
						Duplikat dilewati <strong>{result.duplicates}</strong>.
					{/if}
					{#if result.failed > 0}
						Gagal <strong>{result.failed}</strong>.
					{/if}
					{#if result.skipped > 0}
						Dilewati <strong>{result.skipped}</strong> (kosong/saldo/duplikat).
					{/if}
				</p>
				{#if result.balance}
					<p class="mt-1 text-sm">Saldo terkini: {formatRupiah(result.balance.current_balance)}</p>
				{/if}
			</Alert>
			{#if result.errors.length > 0}
				<div class="max-h-40 overflow-y-auto rounded-lg border border-slate-200 p-2 text-xs dark:border-slate-700">
					{#each result.errors as err}
						<p class="text-red-600 dark:text-red-400">
							{#if err.sheet}{err.sheet} — {/if}Baris {err.row}: {err.message}
						</p>
					{/each}
				</div>
			{/if}
		{/if}

		<div class="flex justify-end gap-2">
			<Button type="button" color="light" onclick={() => (open = false)} disabled={loading}>Tutup</Button>
			<Button type="submit" disabled={loading || !file}>
				{loading ? 'Mengimport...' : 'Import'}
			</Button>
		</div>
	</form>
</Modal>

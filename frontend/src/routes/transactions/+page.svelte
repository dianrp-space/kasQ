<script lang="ts">
	import { api, ApiError } from '$lib/api';
	import type { ImportResult, Team, User } from '$lib/types';
	import NoTeamBanner from '$lib/components/NoTeamBanner.svelte';
	import TxImportModal from '$lib/components/TxImportModal.svelte';
	import { toast } from '$lib/toast.svelte';
	import { getDayName, HARI_LIST, MAX_NOTA_FILES, toInputDate } from '$lib/utils';
	import { goto } from '$app/navigation';
	import { Alert, Button, Input, Label, Select, Spinner, Textarea } from 'flowbite-svelte';
	import { ImageOutline } from 'flowbite-svelte-icons';

	let user = $state<User | null>(null);
	let teams = $state<Team[]>([]);
	let selectedTeam = $state('');
	let importOpen = $state(false);
	let hari = $state(HARI_LIST[new Date().getDay() === 0 ? 6 : new Date().getDay() - 1]);
	let tanggal = $state(toInputDate());
	let jenis = $state<'in' | 'out'>('out');
	let deskripsi = $state('');
	let total = $state('');
	let keterangan = $state('');
	let notaFiles = $state<File[]>([]);
	let notaPreviews = $state<string[]>([]);
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	const needsTeam = $derived(user?.role === 'ops' && !user?.team_id);

	function onDateChange() {
		const d = new Date(tanggal + 'T00:00:00');
		hari = getDayName(d);
	}

	function clearNotaFiles() {
		for (const url of notaPreviews) URL.revokeObjectURL(url);
		notaFiles = [];
		notaPreviews = [];
	}

	function onNotaChange(e: Event) {
		const input = e.target as HTMLInputElement;
		const picked = [...(input.files ?? [])];
		input.value = '';
		if (!picked.length) return;
		const next = [...notaFiles, ...picked].slice(0, MAX_NOTA_FILES);
		for (const url of notaPreviews) URL.revokeObjectURL(url);
		notaFiles = next;
		notaPreviews = next.map((f) => URL.createObjectURL(f));
	}

	function removeNotaAt(i: number) {
		URL.revokeObjectURL(notaPreviews[i]);
		notaFiles = notaFiles.filter((_, idx) => idx !== i);
		notaPreviews = notaPreviews.filter((_, idx) => idx !== i);
	}

	async function load() {
		user = await api.me();
		if (user.role === 'admin') {
			goto('/admin');
			return;
		}
		teams = await api.getTeams();
		if (teams.length > 0) {
			selectedTeam = user.team_id || teams[0].id;
		}
	}

	async function submit(e: Event) {
		e.preventDefault();
		if (needsTeam || !selectedTeam) {
			error = 'Akun belum ditugaskan ke tim/kas. Minta admin menetapkan tim/kas kamu.';
			return;
		}
		loading = true;
		error = '';
		success = '';
		try {
			const form = new FormData();
			form.append('hari', hari);
			form.append('tanggal', tanggal);
			form.append('jenis', jenis);
			form.append('deskripsi', deskripsi);
			form.append('total', total);
			if (keterangan) form.append('keterangan', keterangan);
			for (const file of notaFiles) {
				form.append('nota', file);
			}

			const result = await api.createTransaction(selectedTeam, form);
			success = `Transaksi disimpan. Saldo terkini: Rp ${result.balance.current_balance.toLocaleString('id-ID')}`;
			toast.success('Transaksi disimpan');
			deskripsi = '';
			total = '';
			keterangan = '';
			clearNotaFiles();
		} catch (err) {
			if (err instanceof ApiError && err.noTeam) {
				error = err.message;
			} else {
				error = err instanceof Error ? err.message : 'Gagal simpan';
			}
			toast.error(error);
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		load();
	});

	function onImported(res: ImportResult) {
		if (res.imported > 0) {
			toast.success(`Import selesai: ${res.imported} transaksi`);
		}
	}
</script>

<svelte:head><title>Input Transaksi — KasQ</title></svelte:head>

<div class="page-head flex max-w-2xl flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
	<div>
		<h1>Input transaksi</h1>
		<p>Catat pemasukan atau pengeluaran, opsional dengan foto nota.</p>
	</div>
	<Button color="light" disabled={needsTeam || !selectedTeam} onclick={() => (importOpen = true)}>
		Import Excel
	</Button>
</div>

<TxImportModal bind:open={importOpen} teamId={selectedTeam} onImported={onImported} />

{#if needsTeam}
	<div class="mb-6 max-w-2xl">
		<NoTeamBanner />
	</div>
{/if}

<section class="app-panel max-w-2xl">
	<form onsubmit={submit} class="space-y-4">
		<div class="grid gap-4 sm:grid-cols-2">
			<div class="auth-field">
				<Label for="tanggal" class="text-sm font-medium text-slate-700 dark:text-slate-300">Tanggal</Label>
				<Input
					id="tanggal"
					type="date"
					bind:value={tanggal}
					onchange={onDateChange}
					required
					disabled={needsTeam}
				/>
			</div>
			<div class="auth-field">
				<Label for="hari" class="text-sm font-medium text-slate-700 dark:text-slate-300">Hari</Label>
				<Select id="hari" bind:value={hari} required disabled={needsTeam} placeholder="">
					{#each HARI_LIST as h}
						<option value={h}>{h}</option>
					{/each}
				</Select>
			</div>
		</div>

		<div class="auth-field">
			<p class="text-sm font-medium text-slate-700 dark:text-slate-300" id="jenis-label">Jenis</p>
			<div class="app-seg app-seg-jenis" role="group" aria-labelledby="jenis-label">
				<button
					type="button"
					data-tone="out"
					aria-pressed={jenis === 'out'}
					disabled={needsTeam}
					onclick={() => (jenis = 'out')}
				>
					Keluar
					<span class="seg-sub">Uang keluar dari kas</span>
				</button>
				<button
					type="button"
					data-tone="in"
					aria-pressed={jenis === 'in'}
					disabled={needsTeam}
					onclick={() => (jenis = 'in')}
				>
					Masuk
					<span class="seg-sub">Uang masuk ke kas</span>
				</button>
			</div>
		</div>

		<div class="auth-field">
			<Label for="deskripsi" class="text-sm font-medium text-slate-700 dark:text-slate-300">Deskripsi</Label>
			<Input
				id="deskripsi"
				type="text"
				bind:value={deskripsi}
				required
				placeholder="Contoh: Beli air minum galon"
				disabled={needsTeam}
			/>
		</div>

		<div class="auth-field">
			<Label for="total" class="text-sm font-medium text-slate-700 dark:text-slate-300">Total (Rp)</Label>
			<Input id="total" type="number" bind:value={total} required min="1" placeholder="12000" disabled={needsTeam} />
		</div>

		<div class="auth-field">
			<p class="text-sm font-medium text-slate-700 dark:text-slate-300">Nota</p>
			<label class="nota-drop {needsTeam ? 'pointer-events-none opacity-50' : ''}" for="nota">
				<ImageOutline class="h-6 w-6 text-slate-400" />
				<span class="text-sm font-medium text-slate-700 dark:text-slate-200">Tambah foto nota</span>
				<span class="text-xs text-slate-500 dark:text-slate-400">Opsional, maks. {MAX_NOTA_FILES} foto</span>
				<input
					id="nota"
					type="file"
					accept="image/*"
					multiple
					disabled={needsTeam}
					class="sr-only"
					onchange={onNotaChange}
				/>
			</label>
			{#if notaFiles.length > 0}
				<div class="mt-2 flex flex-wrap gap-2">
					{#each notaPreviews as src, i}
						<div class="relative">
							<img src={src} alt={notaFiles[i].name} class="h-16 w-16 rounded-lg object-cover" />
							<button
								type="button"
								class="absolute -right-1.5 -top-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-slate-800 text-[11px] leading-none text-white dark:bg-slate-200 dark:text-slate-900"
								onclick={() => removeNotaAt(i)}
								aria-label="Hapus foto {i + 1}"
							>
								×
							</button>
						</div>
					{/each}
				</div>
				<p class="auth-hint">{notaFiles.length} foto dipilih</p>
			{/if}
		</div>

		<div class="auth-field">
			<Label for="keterangan" class="text-sm font-medium text-slate-700 dark:text-slate-300">Keterangan</Label>
			<Textarea id="keterangan" rows={2} bind:value={keterangan} placeholder="Opsional" disabled={needsTeam} />
		</div>

		{#if error}<Alert color="red">{error}</Alert>{/if}
		{#if success}<Alert color="green">{success}</Alert>{/if}

		<div class="flex flex-col gap-2 sm:flex-row">
			<Button type="submit" class="auth-submit" disabled={loading || needsTeam}>
				{#if loading}
					<Spinner size="4" class="me-2" />
				{/if}
				{loading ? 'Menyimpan' : 'Simpan transaksi'}
			</Button>
			<Button href="/dashboard" color="light">Lihat dashboard</Button>
		</div>
	</form>
</section>

<section class="app-panel mt-5 max-w-2xl">
	<h2 class="mb-2 text-base font-semibold tracking-tight text-slate-900 dark:text-slate-50">Format WA / Telegram</h2>
	<pre class="app-code">out#Senin#100826#Beli air minum#12000#(Keterangan/opsional)
out#100826#Beli air minum#12000
in#Sabtu#010826#Refill kas Batam#2000000</pre>
	<p class="mt-3 text-xs leading-relaxed text-slate-500 dark:text-slate-400">
		Hari boleh dikosongkan — terisi otomatis dari tanggal. Keterangan opsional. Untuk pengeluaran dengan nota, kirim 1–10 foto. Telegram: caption format transaksi di foto mana pun (boleh album). WhatsApp: kirim beberapa foto berurutan, caption di foto terakhir (atau salah satu foto).
		Rekap Excel lama? Pakai Import Excel — kolom Hari, Tanggal, Jenis, Deskripsi, Total, Link Nota.
	</p>
</section>

<script lang="ts">
	import { api, ApiError } from '$lib/api';
	import type { ImportResult, Team, User } from '$lib/types';
	import NoTeamBanner from '$lib/components/NoTeamBanner.svelte';
	import TxImportModal from '$lib/components/TxImportModal.svelte';
	import { toast } from '$lib/toast.svelte';
	import { getDayName, HARI_LIST, toInputDate } from '$lib/utils';
	import { goto } from '$app/navigation';
	import { Alert, Button, Card, Heading, Input, Label, Select, Textarea } from 'flowbite-svelte';

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
	let notaFile = $state<File | null>(null);
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	const needsTeam = $derived(user?.role === 'ops' && !user?.team_id);

	function onDateChange() {
		const d = new Date(tanggal + 'T00:00:00');
		hari = getDayName(d);
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
			if (notaFile) form.append('nota', notaFile);

			const result = await api.createTransaction(selectedTeam, form);
			success = `Transaksi berhasil! Saldo terkini: Rp ${result.balance.current_balance.toLocaleString('id-ID')}`;
			toast.success('Transaksi disimpan');
			deskripsi = '';
			total = '';
			keterangan = '';
			notaFile = null;
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

<Heading tag="h1" class="mb-4 text-xl sm:mb-6 sm:text-2xl">Input Transaksi</Heading>

<div class="mb-4 flex flex-wrap gap-2">
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

<Card size="lg" shadow="sm" class="max-w-2xl p-3 sm:p-4">
	<form onsubmit={submit} class="space-y-4">
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<Label for="tanggal">Tanggal</Label>
				<Input
					id="tanggal"
					type="date"
					bind:value={tanggal}
					onchange={onDateChange}
					required
					disabled={needsTeam}
				/>
			</div>
			<div>
				<Label for="hari">Hari</Label>
				<Select id="hari" bind:value={hari} required disabled={needsTeam}>
					{#each HARI_LIST as h}
						<option value={h}>{h}</option>
					{/each}
				</Select>
			</div>
		</div>

		<div>
			<Label for="jenis">Jenis</Label>
			<Select id="jenis" bind:value={jenis} required disabled={needsTeam}>
				<option value="out">Pengeluaran</option>
				<option value="in">Pemasukan</option>
			</Select>
		</div>

		<div>
			<Label for="deskripsi">Deskripsi</Label>
			<Input
				id="deskripsi"
				type="text"
				bind:value={deskripsi}
				required
				placeholder="Contoh: Beli air minum galon"
				disabled={needsTeam}
			/>
		</div>

		<div>
			<Label for="total">Total (Rp)</Label>
			<Input id="total" type="number" bind:value={total} required min="1" placeholder="12000" disabled={needsTeam} />
		</div>

		<div>
			<Label for="nota">Nota (foto, opsional)</Label>
			<input
				id="nota"
				type="file"
				accept="image/*"
				disabled={needsTeam}
				class="block w-full cursor-pointer rounded-lg border border-gray-300 bg-gray-50 text-sm text-gray-900 focus:outline-none disabled:opacity-50"
				onchange={(e) => {
					notaFile = (e.target as HTMLInputElement).files?.[0] ?? null;
				}}
			/>
		</div>

		<div>
			<Label for="keterangan">Keterangan</Label>
			<Textarea id="keterangan" rows={2} bind:value={keterangan} placeholder="Opsional" disabled={needsTeam} />
		</div>

		{#if error}<Alert color="red">{error}</Alert>{/if}
		{#if success}<Alert color="green">{success}</Alert>{/if}

		<div class="flex flex-col gap-2 sm:flex-row">
			<Button type="submit" disabled={loading || needsTeam}>
				{loading ? 'Menyimpan...' : 'Simpan Transaksi'}
			</Button>
			<Button href="/dashboard" color="light">Lihat Dashboard</Button>
		</div>
	</form>
</Card>

<Card size="lg" shadow="sm" class="mt-4 max-w-2xl p-3 sm:p-4">
	<Heading tag="h2" class="mb-2 text-base">Format WA/Telegram</Heading>
	<pre class="overflow-x-auto rounded-lg bg-slate-100 p-3 text-xs text-slate-700 dark:bg-slate-800 dark:text-slate-300">out#Senin#100826#Beli air minum#12000#(Keterangan/opsional)
in#Sabtu#010826#Refill kas Batam#2000000</pre>
	<p class="mt-2 text-xs text-slate-500 dark:text-slate-400">
		Keterangan opsional. Untuk pengeluaran dengan nota, kirim foto dengan caption format di atas.
		Rekap Excel lama? Pakai <strong>Import Excel</strong> — kolom Hari, Tanggal, Jenis, Deskripsi, Total, Link Nota.
	</p>
</Card>

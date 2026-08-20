<script lang="ts">
	import { api, ApiError } from '$lib/api';
	import type { Transaction } from '$lib/types';
	import { getDayName, HARI_LIST, MAX_NOTA_FILES, toInputDate, txNotaKeys } from '$lib/utils';
	import { Alert, Button, Checkbox, Input, Label, Modal, Select, Textarea } from 'flowbite-svelte';

	let {
		open = $bindable(false),
		teamId = '',
		tx = null,
		onSaved
	}: {
		open: boolean;
		teamId: string;
		tx: Transaction | null;
		onSaved?: () => void;
	} = $props();

	let hari = $state('');
	let tanggal = $state('');
	let jenis = $state<'in' | 'out'>('out');
	let deskripsi = $state('');
	let total = $state('');
	let keterangan = $state('');
	let notaFiles = $state<File[]>([]);
	let removeNota = $state(false);
	let error = $state('');
	let loading = $state(false);

	$effect(() => {
		if (open && tx) {
			hari = tx.hari;
			tanggal = toInputDate(new Date(tx.tanggal));
			jenis = tx.jenis;
			deskripsi = tx.deskripsi;
			total = String(tx.total);
			keterangan = tx.keterangan || '';
			notaFiles = [];
			removeNota = false;
			error = '';
		}
	});

	function onDateChange() {
		const d = new Date(tanggal + 'T00:00:00');
		hari = getDayName(d);
	}

	async function submit(e: Event) {
		e.preventDefault();
		if (!tx || !teamId) return;
		loading = true;
		error = '';
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
			if (removeNota) form.append('remove_nota', 'true');

			await api.updateTransaction(teamId, tx.id, form);
			onSaved?.();
			open = false;
		} catch (err) {
			error = err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Gagal simpan';
		} finally {
			loading = false;
		}
	}
</script>

<Modal bind:open title="Edit Transaksi" size="lg" dismissable={!loading}>
	<form id="edit-tx-form" onsubmit={submit} class="space-y-4">
		<div class="grid gap-4 sm:grid-cols-2">
			<div>
				<Label for="edit-tanggal">Tanggal</Label>
				<Input id="edit-tanggal" type="date" bind:value={tanggal} onchange={onDateChange} required />
			</div>
			<div>
				<Label for="edit-hari">Hari</Label>
				<Select id="edit-hari" bind:value={hari} required>
					{#each HARI_LIST as h}
						<option value={h}>{h}</option>
					{/each}
				</Select>
			</div>
		</div>

		<div>
			<Label for="edit-jenis">Jenis</Label>
			<Select id="edit-jenis" bind:value={jenis} required>
				<option value="out">Keluar</option>
				<option value="in">Masuk</option>
			</Select>
		</div>

		<div>
			<Label for="edit-deskripsi">Deskripsi</Label>
			<Input id="edit-deskripsi" type="text" bind:value={deskripsi} required />
		</div>

		<div>
			<Label for="edit-total">Total (Rp)</Label>
			<Input id="edit-total" type="number" bind:value={total} required min="1" />
		</div>

		<div>
			<Label for="edit-keterangan">Keterangan</Label>
			<Textarea id="edit-keterangan" rows={2} bind:value={keterangan} />
		</div>

		<div>
			<Label for="edit-nota">Ganti nota (opsional, maks. {MAX_NOTA_FILES})</Label>
			<input
				id="edit-nota"
				type="file"
				accept="image/*"
				multiple
				class="block w-full cursor-pointer rounded-lg border border-gray-300 bg-gray-50 text-sm text-gray-900 focus:outline-none dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
				onchange={(e) => {
					const picked = [...((e.target as HTMLInputElement).files ?? [])].slice(0, MAX_NOTA_FILES);
					notaFiles = picked;
					if (notaFiles.length) removeNota = false;
				}}
			/>
			{#if notaFiles.length > 0}
				<p class="mt-1 text-xs text-slate-500 dark:text-slate-400">{notaFiles.length} foto akan mengganti nota lama</p>
			{/if}
			{#if txNotaKeys(tx ?? {}).length > 0}
				<Checkbox bind:checked={removeNota} disabled={notaFiles.length > 0} class="mt-2">
					Hapus {txNotaKeys(tx ?? {}).length > 1 ? `semua nota (${txNotaKeys(tx ?? {}).length} foto)` : 'nota yang ada'}
				</Checkbox>
			{/if}
		</div>

		{#if error}
			<Alert color="red">{error}</Alert>
		{/if}
	</form>
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			<Button color="alternative" onclick={() => (open = false)} disabled={loading}>Batal</Button>
			<Button type="submit" form="edit-tx-form" color="primary" disabled={loading || !tx}>
				{loading ? 'Menyimpan...' : 'Simpan Perubahan'}
			</Button>
		</div>
	{/snippet}
</Modal>

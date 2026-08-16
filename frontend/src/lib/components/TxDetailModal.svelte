<script lang="ts">
	import type { Transaction } from '$lib/types';
	import { formatDate, formatRupiah, jenisLabel, sourceLabel } from '$lib/utils';
	import { Badge, Button, Modal, P } from 'flowbite-svelte';

	let {
		open = $bindable(false),
		tx = null
	}: {
		open: boolean;
		tx: Transaction | null;
	} = $props();

	function sourceColor(source: Transaction['source']) {
		if (source === 'wa') return 'green';
		if (source === 'tele') return 'blue';
		return 'indigo';
	}
</script>

<Modal bind:open title="Detail Transaksi" size="md" autoclose={false}>
	{#if tx}
		<div class="space-y-4">
			<div class="flex flex-wrap items-center gap-2">
				<Badge color={tx.jenis === 'in' ? 'green' : 'red'}>{jenisLabel(tx.jenis)}</Badge>
				<Badge color={sourceColor(tx.source)}>{sourceLabel(tx.source)}</Badge>
				<span class="text-sm text-slate-500 dark:text-slate-400">
					{formatDate(tx.tanggal)} · {tx.hari}
				</span>
			</div>

			<div>
				<P class="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
					Total
				</P>
				<p class="text-lg font-semibold {tx.jenis === 'in' ? 'text-emerald-600' : 'text-red-600'}">
					{formatRupiah(tx.total)}
				</p>
			</div>

			<div>
				<P class="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
					Deskripsi
				</P>
				<p class="whitespace-pre-wrap break-words text-sm text-slate-900 dark:text-slate-100">
					{tx.deskripsi}
				</p>
			</div>

			{#if tx.keterangan}
				<div>
					<P class="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-slate-400">
						Keterangan
					</P>
					<p class="whitespace-pre-wrap break-words text-sm text-slate-700 dark:text-slate-300">
						{tx.keterangan}
					</p>
				</div>
			{/if}
		</div>
	{/if}
	{#snippet footer()}
		<div class="flex w-full justify-end">
			<Button color="alternative" onclick={() => (open = false)}>Tutup</Button>
		</div>
	{/snippet}
</Modal>

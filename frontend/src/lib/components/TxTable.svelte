<script lang="ts">
	import type { Transaction } from '$lib/types';
	import { formatDate, formatRupiah, jenisLabel, sourceLabel } from '$lib/utils';
	import { Badge, Button } from 'flowbite-svelte';
	import { EditOutline, EyeOutline, TrashBinOutline } from 'flowbite-svelte-icons';

	let {
		transactions,
		onViewNota,
		onDownloadNota,
		onEdit,
		onDelete
	}: {
		transactions: Transaction[];
		onViewNota?: (key: string) => void;
		onDownloadNota?: (key: string) => void;
		onEdit?: (tx: Transaction) => void;
		onDelete?: (tx: Transaction) => void;
	} = $props();

	const showActions = $derived(!!onEdit || !!onDelete);

	function sourceColor(source: Transaction['source']) {
		if (source === 'wa') return 'green';
		if (source === 'tele') return 'blue';
		return 'indigo';
	}
</script>

{#if transactions.length === 0}
	<p class="py-8 text-center text-sm text-slate-400 dark:text-slate-500">Belum ada transaksi</p>
{:else}
	<div class="space-y-3 md:hidden">
		{#each transactions as tx}
			<article class="rounded-lg border border-slate-200 bg-slate-50/80 p-3 dark:border-slate-700 dark:bg-slate-800/80">
				<div class="mb-2 flex items-start justify-between gap-2">
					<div class="min-w-0">
						<p class="font-medium text-slate-900 dark:text-slate-100">{tx.deskripsi}</p>
						<p class="text-xs text-slate-500 dark:text-slate-400">{formatDate(tx.tanggal)} · {tx.hari}</p>
					</div>
					<p class="shrink-0 text-sm font-semibold {tx.jenis === 'in' ? 'text-emerald-600' : 'text-red-600'}">
						{formatRupiah(tx.total)}
					</p>
				</div>
				<div class="mb-2 flex flex-wrap gap-1.5">
					<Badge color={tx.jenis === 'in' ? 'green' : 'red'}>{jenisLabel(tx.jenis)}</Badge>
					<Badge color={sourceColor(tx.source)}>{sourceLabel(tx.source)}</Badge>
				</div>
				{#if tx.keterangan}
					<p class="mb-2 text-xs text-slate-500 dark:text-slate-400">{tx.keterangan}</p>
				{/if}
				<div class="flex flex-wrap gap-2">
					{#if tx.nota_key && onViewNota}
						<Button size="xs" color="light" onclick={() => onViewNota(tx.nota_key!)}>
							<EyeOutline class="me-1 h-3 w-3" /> Nota
						</Button>
					{/if}
					{#if onEdit}
						<Button size="xs" color="light" onclick={() => onEdit(tx)}>
							<EditOutline class="me-1 h-3 w-3" /> Edit
						</Button>
					{/if}
					{#if onDelete}
						<Button size="xs" color="red" outline onclick={() => onDelete(tx)}>
							<TrashBinOutline class="me-1 h-3 w-3" /> Hapus
						</Button>
					{/if}
				</div>
			</article>
		{/each}
	</div>

	<div class="hidden overflow-x-auto md:block">
		<table class="w-full text-left text-sm">
			<thead class="border-b border-slate-200 text-slate-500 dark:border-slate-700 dark:text-slate-400">
				<tr>
					<th class="px-3 py-2">Tanggal</th>
					<th class="px-3 py-2">Hari</th>
					<th class="px-3 py-2">Jenis</th>
					<th class="px-3 py-2">Deskripsi</th>
					<th class="px-3 py-2 text-right">Total</th>
					<th class="px-3 py-2">Sumber</th>
					<th class="px-3 py-2">Keterangan</th>
					{#if onViewNota}
						<th class="px-3 py-2">Nota</th>
					{/if}
					{#if showActions}
						<th class="px-3 py-2">Aksi</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each transactions as tx}
					<tr class="border-b border-slate-100 hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-800/60">
						<td class="whitespace-nowrap px-3 py-2">{formatDate(tx.tanggal)}</td>
						<td class="px-3 py-2">{tx.hari}</td>
						<td class="px-3 py-2">
							<Badge color={tx.jenis === 'in' ? 'green' : 'red'}>{jenisLabel(tx.jenis)}</Badge>
						</td>
						<td class="max-w-xs truncate px-3 py-2">{tx.deskripsi}</td>
						<td class="whitespace-nowrap px-3 py-2 text-right font-medium">
							{formatRupiah(tx.total)}
						</td>
						<td class="px-3 py-2">
							<Badge color={sourceColor(tx.source)}>{sourceLabel(tx.source)}</Badge>
						</td>
						<td class="max-w-xs truncate px-3 py-2 text-slate-500 dark:text-slate-400">{tx.keterangan || '—'}</td>
						{#if onViewNota}
							<td class="px-3 py-2">
								{#if tx.nota_key}
									<div class="flex gap-1">
										<Button size="xs" color="light" onclick={() => onViewNota(tx.nota_key!)}>
											<EyeOutline class="h-4 w-4" />
										</Button>
									</div>
								{:else}
									—
								{/if}
							</td>
						{/if}
						{#if showActions}
							<td class="whitespace-nowrap px-3 py-2">
								<div class="flex gap-1">
									{#if onEdit}
										<Button size="xs" color="light" onclick={() => onEdit(tx)}>Edit</Button>
									{/if}
									{#if onDelete}
										<Button size="xs" color="red" outline onclick={() => onDelete(tx)}>Hapus</Button>
									{/if}
								</div>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

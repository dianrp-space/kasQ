<script lang="ts">
	import type { Balance } from '$lib/types';
	import { Card } from 'flowbite-svelte';
	import { formatRupiah } from '$lib/utils';

	let { balance }: { balance: Balance } = $props();

	const isPeriod = $derived(!!balance.period_from && !!balance.period_to);
</script>

<div class="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<p class="text-xs text-slate-500 dark:text-slate-400 sm:text-sm">{isPeriod ? 'Saldo Awal Periode' : 'Saldo Awal Kas'}</p>
		<p class="text-lg font-bold text-slate-800 dark:text-slate-100 sm:text-2xl">{formatRupiah(balance.opening_balance)}</p>
	</Card>
	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<p class="text-xs text-slate-500 dark:text-slate-400 sm:text-sm">{isPeriod ? 'Pemasukan Periode' : 'Total Pemasukan'}</p>
		<p class="text-lg font-bold text-emerald-600 sm:text-2xl">{formatRupiah(balance.total_in)}</p>
	</Card>
	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<p class="text-xs text-slate-500 dark:text-slate-400 sm:text-sm">{isPeriod ? 'Pengeluaran Periode' : 'Total Pengeluaran'}</p>
		<p class="text-lg font-bold text-red-600 sm:text-2xl">{formatRupiah(balance.total_out)}</p>
	</Card>
	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<p class="text-xs text-slate-500 dark:text-slate-400 sm:text-sm">{isPeriod ? 'Saldo Akhir (Tutup Buku)' : 'Saldo Terkini'}</p>
		<p class="text-lg font-bold text-emerald-700 dark:text-emerald-400 sm:text-2xl">{formatRupiah(balance.current_balance)}</p>
	</Card>
</div>

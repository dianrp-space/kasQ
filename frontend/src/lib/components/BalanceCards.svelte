<script lang="ts">
	import type { Balance } from '$lib/types';
	import { formatRupiah } from '$lib/utils';

	let { balance }: { balance: Balance } = $props();

	const isPeriod = $derived(!!balance.period_from && !!balance.period_to);
</script>

<div class="stat-grid">
	<div class="stat-tile">
		<p>{isPeriod ? 'Saldo awal periode' : 'Saldo awal kas'}</p>
		<p class="text-slate-800 dark:text-slate-100">{formatRupiah(balance.opening_balance)}</p>
	</div>
	<div class="stat-tile">
		<p>{isPeriod ? 'Pemasukan periode' : 'Total pemasukan'}</p>
		<p class="text-emerald-600 dark:text-emerald-400">{formatRupiah(balance.total_in)}</p>
	</div>
	<div class="stat-tile">
		<p>{isPeriod ? 'Pengeluaran periode' : 'Total pengeluaran'}</p>
		<p class="text-red-600 dark:text-red-400">{formatRupiah(balance.total_out)}</p>
	</div>
	<div class="stat-tile">
		<p>{isPeriod ? 'Saldo akhir periode' : 'Saldo terkini'}</p>
		<p class="text-primary-700 dark:text-primary-400">{formatRupiah(balance.current_balance)}</p>
	</div>
</div>

<script lang="ts">
	import { browser } from '$app/environment';
	import type { ApexOptions } from 'apexcharts';
	import { Chart } from '@flowbite-svelte-plugins/chart';
	import { Card, Heading } from 'flowbite-svelte';
	import type { Balance, Transaction } from '$lib/types';
	import { formatRupiah } from '$lib/utils';

	let { balance, transactions }: { balance: Balance; transactions: Transaction[] } = $props();
	let isDark = $state(browser && document.documentElement.classList.contains('dark'));

	$effect(() => {
		if (!browser) return;
		const root = document.documentElement;
		const update = () => {
			isDark = root.classList.contains('dark');
		};
		update();
		const observer = new MutationObserver(update);
		observer.observe(root, { attributes: true, attributeFilter: ['class'] });
		return () => observer.disconnect();
	});

	const chartTheme = $derived({
		mode: (isDark ? 'dark' : 'light') as 'dark' | 'light',
		labelColor: isDark ? '#94a3b8' : '#64748b',
		gridColor: isDark ? '#334155' : '#e2e8f0',
		tooltipTheme: (isDark ? 'dark' : 'light') as 'dark' | 'light'
	});

	const dailySeries = $derived.by(() => {
		const map = new Map<string, { in: number; out: number }>();
		for (const tx of transactions) {
			const key = tx.tanggal.slice(0, 10);
			const row = map.get(key) ?? { in: 0, out: 0 };
			if (tx.jenis === 'in') row.in += tx.total;
			else row.out += tx.total;
			map.set(key, row);
		}
		const labels = [...map.keys()].sort();
		return {
			labels,
			in: labels.map((d) => map.get(d)?.in ?? 0),
			out: labels.map((d) => map.get(d)?.out ?? 0)
		};
	});

	const donutOptions = $derived<ApexOptions>({
		theme: { mode: chartTheme.mode },
		chart: {
			type: 'donut',
			height: 220,
			fontFamily: 'inherit',
			background: 'transparent',
			toolbar: { show: false },
			foreColor: chartTheme.labelColor
		},
		series: [balance.total_in, balance.total_out],
		labels: ['Pemasukan', 'Pengeluaran'],
		colors: ['#059669', '#dc2626'],
		legend: { position: 'bottom', offsetY: 0, labels: { colors: chartTheme.labelColor } },
		dataLabels: { enabled: false },
		stroke: { colors: ['transparent'] },
		plotOptions: {
			pie: {
				donut: { size: '65%', background: 'transparent' }
			}
		},
		tooltip: {
			theme: chartTheme.tooltipTheme,
			y: {
				formatter: (val: number) => formatRupiah(val)
			}
		}
	});

	const barOptions = $derived<ApexOptions>({
		theme: { mode: chartTheme.mode },
		chart: {
			type: 'bar',
			height: 220,
			fontFamily: 'inherit',
			background: 'transparent',
			toolbar: { show: false },
			stacked: false,
			foreColor: chartTheme.labelColor
		},
		series: [
			{ name: 'Pemasukan', data: dailySeries.in },
			{ name: 'Pengeluaran', data: dailySeries.out }
		],
		xaxis: {
			categories: dailySeries.labels,
			labels: { rotate: -45, hideOverlappingLabels: true, style: { colors: chartTheme.labelColor } }
		},
		yaxis: { labels: { style: { colors: chartTheme.labelColor } } },
		grid: { borderColor: chartTheme.gridColor },
		colors: ['#059669', '#dc2626'],
		plotOptions: { bar: { borderRadius: 4, columnWidth: '55%' } },
		dataLabels: { enabled: false },
		tooltip: {
			theme: chartTheme.tooltipTheme,
			y: {
				formatter: (val: number) => formatRupiah(val)
			}
		}
	});
</script>

<div class="mb-4 grid gap-3 lg:grid-cols-2 lg:gap-4">
	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<Heading tag="h3" class="mb-2 text-sm font-semibold sm:text-base">Ringkasan Periode</Heading>
		{#key chartTheme.mode}
			<Chart options={donutOptions} class="dashboard-chart w-full min-h-0" />
		{/key}
	</Card>
	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<Heading tag="h3" class="mb-2 text-sm font-semibold sm:text-base">Transaksi Harian</Heading>
		{#if dailySeries.labels.length === 0}
			<p class="py-10 text-center text-sm text-slate-400 dark:text-slate-500">Belum ada data untuk grafik</p>
		{:else}
			{#key chartTheme.mode}
				<Chart options={barOptions} class="dashboard-chart w-full min-h-0" />
			{/key}
		{/if}
	</Card>
</div>

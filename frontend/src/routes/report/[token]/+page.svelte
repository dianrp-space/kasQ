<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { browser } from '$app/environment';
	import { appSettings, loadAppSettings } from '$lib/appSettings.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import type { PublicReport } from '$lib/types';
	import TxTable from '$lib/components/TxTable.svelte';
	import NotaPreviewModal from '$lib/components/NotaPreviewModal.svelte';
	import BalanceCards from '$lib/components/BalanceCards.svelte';
	import DashboardChart from '$lib/components/DashboardChart.svelte';
	import { formatMonthLabel, getMonthRange, notaFilenameFromKey, toInputMonth, downloadFilesAsZip } from '$lib/utils';
	import { Alert, Button, Card, Heading, Label, Select, Spinner } from 'flowbite-svelte';
	import MonthPeriodFilter from '$lib/components/MonthPeriodFilter.svelte';
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';

	let report = $state<PublicReport | null>(null);
	let error = $state('');
	let filterJenis = $state('');
	let filterMonth = $state(toInputMonth());
	let notaPreviewOpen = $state(false);
	let notaPreviewSrcs = $state<string[]>([]);
	let notaPreviewKeys = $state<string[]>([]);

	const token = $derived($page.params.token ?? '');
	const periodRange = $derived(getMonthRange(filterMonth));
	const periodLabel = $derived(formatMonthLabel(periodRange.from));

	async function load() {
		if (!token) return;
		try {
			const params: Record<string, string> = {
				date_from: periodRange.from,
				date_to: periodRange.to
			};
			if (filterJenis) params.jenis = filterJenis;
			report = await api.getPublicReport(token, params);
			error = '';
		} catch {
			error = 'Laporan tidak ditemukan atau token tidak valid';
		}
	}

	function applyFilter() {
		if (browser) {
			const url = new URL(window.location.href);
			url.searchParams.set('month', filterMonth);
			history.replaceState(history.state, '', url);
		}
		load();
	}

	async function viewNota(keys: string[]) {
		if (!keys.length) return;
		const urls = await Promise.all(keys.map((key) => api.getPublicNotaURL(token, key)));
		notaPreviewSrcs = urls.filter((u): u is string => !!u);
		notaPreviewKeys = keys;
		notaPreviewOpen = true;
	}

	async function downloadNota(key: string) {
		const url = await api.getPublicNotaURL(token, key, true);
		if (url) window.open(url, '_blank');
	}

	function downloadPreviewNota(index: number) {
		const key = notaPreviewKeys[index];
		if (key) downloadNota(key);
	}

	async function downloadAllPreviewNota() {
		if (!notaPreviewKeys.length) return;
		try {
			const items: { url: string; filename: string }[] = [];
			for (let i = 0; i < notaPreviewKeys.length; i++) {
				const key = notaPreviewKeys[i];
				const url = await api.getPublicNotaURL(token, key, true);
				if (url) items.push({ url, filename: notaFilenameFromKey(key, i) });
			}
			if (!items.length) throw new Error('no urls');
			const stamp = new Date().toISOString().slice(0, 10);
			await downloadFilesAsZip(items, `nota-${stamp}.zip`);
		} catch {
			// silent on public report
		}
	}

	$effect(() => {
		loadAppSettings();
	});

	onMount(() => {
		const qMonth = get(page).url.searchParams.get('month');
		if (qMonth && /^\d{4}-\d{2}$/.test(qMonth)) {
			filterMonth = qMonth;
		}
		load();
	});
</script>

<svelte:head><title>{appSettings.app_name} — Laporan Kas</title></svelte:head>

<div class="min-h-screen bg-slate-50 dark:bg-slate-900">
	<header class="border-b border-slate-200 bg-white px-4 py-4 dark:border-slate-700 dark:bg-slate-900">
		<div class="mx-auto max-w-6xl">
			<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
				<AppBrand size="sm" />
				<div class="flex items-center justify-between gap-3 sm:justify-end">
					{#if report}
						<p class="text-sm font-semibold text-primary-700 dark:text-primary-400 sm:text-right">
							{report.team.name}
						</p>
					{/if}
					<ThemeToggle class="shrink-0 bg-white shadow-sm dark:bg-slate-800" />
				</div>
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-6xl px-4 py-6">
		{#if error}
			<Alert color="red" class="text-center">{error}</Alert>
		{:else if report}
			<BalanceCards balance={report.balance} />
			<DashboardChart balance={report.balance} transactions={report.transactions} />

			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<div class="mb-4 flex flex-col gap-3 md:flex-row md:flex-wrap md:items-end">
					<div class="md:flex-1">
						<Heading tag="h2" class="text-lg">Transaksi</Heading>
						<p class="text-xs text-slate-500">Periode: {periodLabel}</p>
					</div>
					<div class="grid grid-cols-2 gap-2 md:contents">
						<div class="col-span-2 md:max-w-[180px]">
							<MonthPeriodFilter bind:value={filterMonth} />
						</div>
						<div class="col-span-2 md:max-w-[140px]">
							<Label for="filter-jenis" class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Jenis</Label>
							<Select id="filter-jenis" bind:value={filterJenis} placeholder="Semua jenis" class="w-full">
								<option value="">Semua jenis</option>
								<option value="in">Pemasukan</option>
								<option value="out">Pengeluaran</option>
							</Select>
						</div>
						<Button color="light" class="col-span-2 md:col-span-1" onclick={applyFilter}>Terapkan</Button>
					</div>
				</div>
				<TxTable transactions={report.transactions} onViewNota={viewNota} onDownloadNota={downloadNota} />
			</Card>
		{:else}
			<div class="flex items-center justify-center gap-2 py-12 text-slate-500">
				<Spinner size="6" />
				<span>Memuat laporan...</span>
			</div>
		{/if}
	</main>
</div>

<NotaPreviewModal bind:open={notaPreviewOpen} srcs={notaPreviewSrcs} onDownload={downloadPreviewNota} onDownloadAll={downloadAllPreviewNota} />

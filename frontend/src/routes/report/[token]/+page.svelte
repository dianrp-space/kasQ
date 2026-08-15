<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { appSettings, loadAppSettings } from '$lib/appSettings.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import type { PublicReport } from '$lib/types';
	import TxTable from '$lib/components/TxTable.svelte';
	import NotaPreviewModal from '$lib/components/NotaPreviewModal.svelte';
	import BalanceCards from '$lib/components/BalanceCards.svelte';
	import DashboardChart from '$lib/components/DashboardChart.svelte';
	import { formatMonthLabel, getCurrentMonthRange } from '$lib/utils';
	import { Alert, Button, Card, Heading, Input, Label, Select, Spinner } from 'flowbite-svelte';

	const defaultMonth = getCurrentMonthRange();

	let report = $state<PublicReport | null>(null);
	let error = $state('');
	let filterJenis = $state('');
	let filterFrom = $state(defaultMonth.from);
	let filterTo = $state(defaultMonth.to);
	let notaPreviewOpen = $state(false);
	let notaPreviewUrl = $state('');
	let notaDownloadKey = $state('');

	const token = $derived($page.params.token ?? '');
	const periodLabel = $derived(formatMonthLabel(filterFrom));

	async function load() {
		if (!token) return;
		try {
			const params: Record<string, string> = {};
			if (filterJenis) params.jenis = filterJenis;
			if (filterFrom) params.date_from = filterFrom;
			if (filterTo) params.date_to = filterTo;
			report = await api.getPublicReport(token, params);
			error = '';
		} catch {
			error = 'Laporan tidak ditemukan atau token tidak valid';
		}
	}

	async function viewNota(key: string) {
		const url = await api.getPublicNotaURL(token, key);
		if (!url) return;
		notaPreviewUrl = url;
		notaDownloadKey = key;
		notaPreviewOpen = true;
	}

	async function downloadNota(key: string) {
		const url = await api.getPublicNotaURL(token, key, true);
		if (url) window.open(url, '_blank');
	}

	function downloadPreviewNota() {
		if (notaDownloadKey) downloadNota(notaDownloadKey);
	}

	$effect(() => {
		loadAppSettings();
	});

	$effect(() => {
		load();
	});
</script>

<svelte:head><title>{appSettings.app_name} — Laporan Kas</title></svelte:head>

<div class="min-h-screen bg-slate-50 dark:bg-slate-900">
	<header class="border-b border-slate-200 bg-white px-4 py-4 pr-14 dark:border-slate-700 dark:bg-slate-900">
		<div class="mx-auto max-w-6xl">
			<div class="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
				<AppBrand size="sm" />
				{#if report}
					<p
						class="text-sm font-semibold text-primary-700 dark:text-primary-400 sm:max-w-[45%] sm:shrink-0 sm:text-right"
					>
						{report.team.name}
					</p>
				{/if}
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
						<div class="col-span-2 md:max-w-[140px]">
							<Label for="filter-jenis" class="sr-only">Jenis</Label>
							<Select id="filter-jenis" bind:value={filterJenis} placeholder="Semua jenis" class="w-full">
								<option value="">Semua jenis</option>
								<option value="in">Pemasukan</option>
								<option value="out">Pengeluaran</option>
							</Select>
						</div>
						<Input id="filter-from" type="date" bind:value={filterFrom} class="md:max-w-[160px]" />
						<Input id="filter-to" type="date" bind:value={filterTo} class="md:max-w-[160px]" />
						<Button color="light" class="col-span-2 md:col-span-1" onclick={load}>Filter</Button>
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

<NotaPreviewModal bind:open={notaPreviewOpen} src={notaPreviewUrl} onDownload={downloadPreviewNota} />

<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { browser } from '$app/environment';
	import { appSettings, loadAppSettings } from '$lib/appSettings.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import VersionLink from '$lib/components/VersionLink.svelte';
	import type { PublicReport } from '$lib/types';
	import TxTable from '$lib/components/TxTable.svelte';
	import NotaPreviewModal from '$lib/components/NotaPreviewModal.svelte';
	import BalanceCards from '$lib/components/BalanceCards.svelte';
	import DashboardChart from '$lib/components/DashboardChart.svelte';
	import { formatMonthLabel, getMonthRange, nameInitials, notaFilenameFromKey, toInputMonth, downloadFilesAsZip } from '$lib/utils';
	import { Alert, Button, Card, Heading, Label, Select, Spinner } from 'flowbite-svelte';
	import MonthPeriodFilter from '$lib/components/MonthPeriodFilter.svelte';
	import TxExportButtons from '$lib/components/TxExportButtons.svelte';
	import { toast } from '$lib/toast.svelte';
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';

	let report = $state<PublicReport | null>(null);
	let error = $state('');
	let filterJenis = $state('');
	let filterMonth = $state(toInputMonth());
	let notaPreviewOpen = $state(false);
	let notaPreviewSrcs = $state<string[]>([]);
	let notaPreviewKeys = $state<string[]>([]);
	let exportFormat = $state<'xlsx' | 'pdf' | null>(null);

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

	async function exportTx(format: 'xlsx' | 'pdf') {
		if (!token) return;
		exportFormat = format;
		try {
			const params: Record<string, string> = {
				date_from: periodRange.from,
				date_to: periodRange.to
			};
			if (filterJenis) params.jenis = filterJenis;
			await api.exportPublicReport(token, format, params);
			toast.success(format === 'pdf' ? 'PDF diunduh' : 'Excel diunduh');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Gagal export');
		} finally {
			exportFormat = null;
		}
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
			<div class="flex items-center justify-between gap-3">
				<AppBrand size="sm" />
				<ThemeToggle class="shrink-0 bg-white shadow-sm dark:bg-slate-800" />
			</div>
		</div>
	</header>

	<main class="mx-auto max-w-6xl px-4 py-6">
		{#if error}
			<Alert color="red" class="text-center">{error}</Alert>
		{:else if report}
			<div class="mb-5 flex items-center gap-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm sm:p-5 dark:border-slate-700 dark:bg-slate-900">
				{#if report.members && report.members.length > 0}
					<div class="flex shrink-0 -space-x-3">
						{#each report.members as member (member.id)}
							{#if member.has_avatar}
								<img
									src={api.getPublicMemberAvatar(token, member.id)}
									alt={member.name}
									title={member.name}
									class="h-12 w-12 rounded-full border-[3px] border-white object-cover shadow-md sm:h-14 sm:w-14 dark:border-slate-900"
								/>
							{:else}
								<div
									title={member.name}
									class="flex h-12 w-12 items-center justify-center rounded-full border-[3px] border-white bg-primary-600 text-sm font-semibold text-white shadow-md sm:h-14 sm:w-14 dark:border-slate-900"
								>
									{nameInitials(member.name)}
								</div>
							{/if}
						{/each}
					</div>
				{/if}
				<div class="min-w-0">
					<p class="truncate text-lg font-bold tracking-tight text-slate-900 sm:text-xl dark:text-slate-50">
						{report.team.name}
					</p>
					{#if report.members && report.members.length > 0}
						<p class="truncate text-sm text-slate-500 dark:text-slate-400">
							{report.members.map((m) => m.name).join(', ')}
						</p>
					{/if}
				</div>
				<div class="ms-auto flex shrink-0 flex-col items-end gap-1.5">
					<a
						href="/support"
						class="shrink-0 rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-primary-700 hover:bg-primary-50 dark:border-slate-700 dark:text-primary-400 dark:hover:bg-slate-800"
					>
						Tentang app
					</a>
					<VersionLink class="text-xs text-slate-400 hover:text-primary-700 dark:text-slate-500 dark:hover:text-primary-400" />
				</div>
			</div>
			<BalanceCards balance={report.balance} {periodLabel} />
			<DashboardChart balance={report.balance} transactions={report.transactions} />

			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<div class="mb-4 flex flex-col gap-3 md:flex-row md:flex-wrap md:items-end">
					<div class="md:flex-1">
						<Heading tag="h2" class="text-lg">Transaksi</Heading>
						<p class="text-xs text-slate-500">Periode: {periodLabel}</p>
					</div>
					<TxExportButtons busyFormat={exportFormat} onExport={exportTx} />
					<div class="grid grid-cols-2 gap-2 md:contents">
						<div class="col-span-2 md:max-w-[180px]">
							<MonthPeriodFilter bind:value={filterMonth} />
						</div>
						<div class="col-span-2 md:min-w-[11.5rem] md:max-w-[13rem]">
							<Label for="filter-jenis" class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Jenis</Label>
							<Select id="filter-jenis" bind:value={filterJenis} placeholder="" class="w-full">
								<option value="">Semua jenis</option>
								<option value="in">Masuk</option>
								<option value="out">Keluar</option>
							</Select>
						</div>
						<Button color="primary" class="col-span-2 bg-primary-600 text-white hover:bg-primary-700 md:col-span-1" onclick={applyFilter}>Terapkan</Button>
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

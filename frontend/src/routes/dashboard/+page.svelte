<script lang="ts">
	import { api } from '$lib/api';
	import type { Balance, Team, Transaction, User } from '$lib/types';
	import TxTable from '$lib/components/TxTable.svelte';
	import NoTeamBanner from '$lib/components/NoTeamBanner.svelte';
	import NotaPreviewModal from '$lib/components/NotaPreviewModal.svelte';
	import TxEditModal from '$lib/components/TxEditModal.svelte';
	import BalanceCards from '$lib/components/BalanceCards.svelte';
	import DashboardChart from '$lib/components/DashboardChart.svelte';
	import { confirm } from '$lib/confirm.svelte';
	import { toast } from '$lib/toast.svelte';
	import { formatMonthLabel, getMonthRange, greetingWithName, notaFilenameFromKey, toInputMonth, downloadFilesAsZip } from '$lib/utils';
	import { goto } from '$app/navigation';
	import { onDestroy, onMount } from 'svelte';
	import { Button, Card, Heading, Input, Label, Select } from 'flowbite-svelte';
	import MonthPeriodFilter from '$lib/components/MonthPeriodFilter.svelte';
	import { browser } from '$app/environment';
	import { SearchOutline } from 'flowbite-svelte-icons';

	const PAGE_SIZES = [20, 50, 100, 200] as const;
	const PAGE_SIZE_KEY = 'kasq.dashboard.pageSize';
	const CHART_TX_LIMIT = 5000;

	function readPageSize(): number {
		if (!browser) return 20;
		const n = Number(localStorage.getItem(PAGE_SIZE_KEY));
		return (PAGE_SIZES as readonly number[]).includes(n) ? n : 20;
	}

	let user = $state<User | null>(null);
	let teams = $state<Team[]>([]);
	let selectedTeam = $state('');
	let balance = $state<Balance | null>(null);
	let transactions = $state<Transaction[]>([]);
	let chartTransactions = $state<Transaction[]>([]);
	let txTotal = $state(0);
	let currentPage = $state(1);
	let pageSize = $state(20);
	let pageSizeValue = $state('20');
	let filterJenis = $state('');
	let filterMonth = $state(toInputMonth());
	let searchInput = $state('');
	let searchQuery = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let notaPreviewOpen = $state(false);
	let notaPreviewSrcs = $state<string[]>([]);
	let notaPreviewKeys = $state<string[]>([]);
	let editOpen = $state(false);
	let editingTx = $state<Transaction | null>(null);
	let selectedIds = $state<string[]>([]);

	const needsTeam = $derived(user?.role === 'ops' && !user?.team_id);
	const periodRange = $derived(getMonthRange(filterMonth));
	const periodLabel = $derived(formatMonthLabel(periodRange.from));
	const totalPages = $derived(Math.max(1, Math.ceil(txTotal / pageSize)));
	const rangeStart = $derived(txTotal === 0 ? 0 : (currentPage - 1) * pageSize + 1);
	const rangeEnd = $derived(Math.min(currentPage * pageSize, txTotal));
	const pageTitle = $derived(user?.name ? greetingWithName(user.name) : 'Dashboard');
	const emptyTxMessage = $derived(
		searchQuery.trim() ? 'Tidak ada transaksi yang cocok dengan pencarian' : 'Belum ada transaksi'
	);

	function buildParams(includeJenis = true) {
		const params: Record<string, string> = {};
		if (includeJenis && filterJenis) params.jenis = filterJenis;
		params.date_from = periodRange.from;
		params.date_to = periodRange.to;
		return params;
	}

	async function load() {
		user = await api.me();
		if (user.role === 'admin') {
			goto('/admin');
			return;
		}
		teams = await api.getTeams();
		if (!selectedTeam && teams.length > 0) {
			selectedTeam = user.team_id || teams[0].id;
		}
		if (selectedTeam && !needsTeam) {
			await loadTeamData();
		}
	}

	async function loadTeamData() {
		const balanceParams = buildParams(false);
		const tableParams: Record<string, string> = {
			...buildParams(true),
			page: String(currentPage),
			limit: String(pageSize)
		};
		if (searchQuery.trim()) tableParams.q = searchQuery.trim();
		const chartParams = {
			...buildParams(true),
			page: '1',
			limit: String(CHART_TX_LIMIT)
		};

		const [balanceRes, txRes, chartRes] = await Promise.all([
			api.getBalance(selectedTeam, balanceParams),
			api.getTransactions(selectedTeam, tableParams),
			api.getTransactions(selectedTeam, chartParams)
		]);

		balance = balanceRes;
		transactions = txRes.items;
		txTotal = txRes.total;
		chartTransactions = chartRes.items;

		const visible = new Set(transactions.map((tx) => tx.id));
		selectedIds = selectedIds.filter((id) => visible.has(id));

		const maxPage = Math.max(1, Math.ceil(txRes.total / pageSize));
		if (currentPage > maxPage) {
			currentPage = maxPage;
			await loadTeamData();
		}
	}

	function applyFilter() {
		currentPage = 1;
		selectedIds = [];
		searchQuery = searchInput.trim();
		loadTeamData();
	}

	function onSearchInput() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => {
			searchQuery = searchInput.trim();
			currentPage = 1;
			selectedIds = [];
			loadTeamData();
		}, 350);
	}

	function changePageSize(value: string) {
		const n = Number(value);
		if (!(PAGE_SIZES as readonly number[]).includes(n) || n === pageSize) return;
		pageSize = n;
		pageSizeValue = String(n);
		if (browser) localStorage.setItem(PAGE_SIZE_KEY, String(n));
		currentPage = 1;
		selectedIds = [];
		loadTeamData();
	}

	async function reorderTxs(ids: string[]) {
		try {
			await api.reorderTransactions(selectedTeam, ids);
		} catch {
			toast.error('Gagal simpan urutan');
			await loadTeamData();
		}
	}

	function goToPage(page: number) {
		const next = Math.min(Math.max(1, page), totalPages);
		if (next === currentPage) return;
		currentPage = next;
		selectedIds = [];
		loadTeamData();
	}

	async function viewNota(keys: string[]) {
		if (!keys.length) return;
		const urls: string[] = [];
		const loadedKeys: string[] = [];
		for (const key of keys) {
			try {
				const { url } = await api.getNotaURL(selectedTeam, key);
				if (url) {
					urls.push(url);
					loadedKeys.push(key);
				}
			} catch {}
		}
		if (!urls.length) {
			toast.error('Nota tidak bisa dibuka — cek akses baca MinIO (GetObject) dari jaringan dev');
			return;
		}
		notaPreviewSrcs = urls;
		notaPreviewKeys = loadedKeys;
		notaPreviewOpen = true;
	}

	async function downloadNota(key: string) {
		try {
			const { url } = await api.getNotaURL(selectedTeam, key, true);
			window.open(url, '_blank');
		} catch {
			toast.error('Gagal unduh nota');
		}
	}

	function downloadPreviewNota(index: number) {
		const key = notaPreviewKeys[index];
		if (key) downloadNota(key);
	}

	async function downloadAllPreviewNota() {
		if (!notaPreviewKeys.length) return;
		try {
			toast.info('Menyiapkan ZIP…');
			const items = await Promise.all(
				notaPreviewKeys.map(async (key, i) => {
					const { url } = await api.getNotaURL(selectedTeam, key, true);
					return { url, filename: notaFilenameFromKey(key, i) };
				})
			);
			const stamp = new Date().toISOString().slice(0, 10);
			await downloadFilesAsZip(items, `nota-${stamp}.zip`);
			toast.success(`ZIP ${items.length} nota siap diunduh`);
		} catch {
			toast.error('Gagal buat ZIP nota');
		}
	}

	function editTx(tx: Transaction) {
		editingTx = tx;
		editOpen = true;
	}

	async function deleteTx(tx: Transaction) {
		const ok = await confirm({
			title: 'Hapus transaksi?',
			message: `Hapus "${tx.deskripsi}" (${tx.jenis === 'in' ? 'Pemasukan' : 'Pengeluaran'})? Tindakan ini tidak bisa dibatalkan.`,
			confirmLabel: 'Hapus',
			color: 'red'
		});
		if (!ok) return;
		try {
			const result = await api.deleteTransaction(selectedTeam, tx.id);
			balance = result.balance;
			selectedIds = selectedIds.filter((id) => id !== tx.id);
			await loadTeamData();
			toast.success('Transaksi dihapus');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Gagal hapus transaksi');
		}
	}

	async function batchDelete() {
		if (selectedIds.length === 0) return;
		const ok = await confirm({
			title: 'Hapus transaksi terpilih?',
			message: `Hapus ${selectedIds.length} transaksi? Tindakan ini tidak bisa dibatalkan.`,
			confirmLabel: 'Hapus semua',
			color: 'red'
		});
		if (!ok) return;
		try {
			const result = await api.batchDeleteTransactions(selectedTeam, selectedIds);
			balance = result.balance;
			selectedIds = [];
			await loadTeamData();
			toast.success(`${result.deleted} transaksi dihapus`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Gagal hapus transaksi');
		}
	}

	async function onTxSaved() {
		await loadTeamData();
		const balanceParams = buildParams(false);
		balance = await api.getBalance(selectedTeam, balanceParams);
	}

	onMount(() => {
		pageSize = readPageSize();
		pageSizeValue = String(pageSize);
		load();
	});

	onDestroy(() => clearTimeout(searchTimer));
</script>

<svelte:head><title>{pageTitle} — KasQ</title></svelte:head>

<div class="mb-3 sm:mb-4">
	<Heading tag="h1" class="text-xl sm:text-2xl">{pageTitle}</Heading>
	<p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Dashboard</p>
</div>

{#if needsTeam}
	<div class="mb-6 max-w-2xl">
		<NoTeamBanner />
	</div>
{:else}
	{#if balance}
		<BalanceCards {balance} />
		<DashboardChart {balance} transactions={chartTransactions} />
	{/if}

	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<div class="mb-3 flex flex-col gap-2 md:flex-row md:flex-wrap md:items-end">
			<div class="md:flex-1">
				<Heading tag="h2" class="text-base sm:text-lg">Transaksi</Heading>
				<p class="text-xs text-slate-500">
					Periode: {periodLabel}
					{#if txTotal > 0}
						· {txTotal} transaksi
					{/if}
				</p>
			</div>
			<div class="grid grid-cols-2 gap-2 md:contents">
				<div class="col-span-2 md:max-w-[180px]">
					<MonthPeriodFilter bind:value={filterMonth} />
				</div>
				<div class="col-span-2 md:min-w-[11.5rem] md:max-w-[13rem]">
					<Label for="filter-jenis" class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Jenis</Label>
					<Select id="filter-jenis" bind:value={filterJenis} placeholder="" class="w-full">
						<option value="">Semua jenis</option>
						<option value="in">Pemasukan</option>
						<option value="out">Pengeluaran</option>
					</Select>
				</div>
				<div class="col-span-2 md:min-w-[14rem] md:max-w-xs md:flex-1">
					<Label for="filter-search" class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Cari</Label>
					<div class="relative">
						<SearchOutline class="pointer-events-none absolute start-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
						<Input
							id="filter-search"
							class="w-full ps-8"
							placeholder="Deskripsi, keterangan, total…"
							bind:value={searchInput}
							oninput={onSearchInput}
						/>
					</div>
				</div>
				<Button color="primary" class="col-span-2 bg-primary-600 text-white hover:bg-primary-700 md:col-span-1" onclick={applyFilter}>
					Terapkan
				</Button>
				{#if selectedIds.length > 0}
					<Button color="red" outline class="col-span-2 md:col-span-1" onclick={batchDelete}>
						Hapus ({selectedIds.length})
					</Button>
				{/if}
			</div>
		</div>
		<p class="mb-2 text-xs text-slate-500 dark:text-slate-400">
			Seret ikon garis (desktop) atau panah (ponsel) untuk menyusun ulang transaksi di tanggal yang sama.
		</p>
		<TxTable
			{transactions}
			selectable
			sortable
			emptyMessage={emptyTxMessage}
			bind:selectedIds
			onViewNota={viewNota}
			onDownloadNota={downloadNota}
			onEdit={editTx}
			onDelete={deleteTx}
			onReorder={reorderTxs}
		/>
		<div class="mt-4 flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 pt-4 dark:border-slate-700 sm:flex-row sm:items-center">
			<p class="text-sm text-slate-500 dark:text-slate-400">
				{#if txTotal > 0}
					Menampilkan {rangeStart}–{rangeEnd} dari {txTotal} transaksi
				{:else}
					Tidak ada transaksi
				{/if}
			</p>
			<div class="flex flex-wrap items-center gap-2">
				<div class="flex items-center gap-2">
					<Label for="page-size" class="mb-0 whitespace-nowrap text-xs text-slate-500 dark:text-slate-400">Per halaman</Label>
					<Select
						id="page-size"
						class="w-[5.5rem]"
						placeholder=""
						bind:value={pageSizeValue}
						onchange={() => changePageSize(pageSizeValue)}
					>
						{#each PAGE_SIZES as size}
							<option value={String(size)}>{size}</option>
						{/each}
					</Select>
				</div>
				{#if totalPages > 1}
					<Button size="sm" color="light" disabled={currentPage <= 1} onclick={() => goToPage(currentPage - 1)}>
						Sebelumnya
					</Button>
					<span class="min-w-[7rem] text-center text-sm text-slate-600 dark:text-slate-300">
						Halaman {currentPage} / {totalPages}
					</span>
					<Button
						size="sm"
						color="light"
						disabled={currentPage >= totalPages}
						onclick={() => goToPage(currentPage + 1)}
					>
						Selanjutnya
					</Button>
				{/if}
			</div>
		</div>
	</Card>
{/if}

<NotaPreviewModal bind:open={notaPreviewOpen} srcs={notaPreviewSrcs} onDownload={downloadPreviewNota} onDownloadAll={downloadAllPreviewNota} />
<TxEditModal bind:open={editOpen} teamId={selectedTeam} tx={editingTx} onSaved={onTxSaved} />

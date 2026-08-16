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
	import { formatMonthLabel, getMonthRange, toInputMonth } from '$lib/utils';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { Button, Card, Heading, Label, Select } from 'flowbite-svelte';
	import MonthPeriodFilter from '$lib/components/MonthPeriodFilter.svelte';

	const PAGE_SIZE = 20;
	const CHART_TX_LIMIT = 5000;

	let user = $state<User | null>(null);
	let teams = $state<Team[]>([]);
	let selectedTeam = $state('');
	let balance = $state<Balance | null>(null);
	let transactions = $state<Transaction[]>([]);
	let chartTransactions = $state<Transaction[]>([]);
	let txTotal = $state(0);
	let currentPage = $state(1);
	let filterJenis = $state('');
	let filterMonth = $state(toInputMonth());
	let notaPreviewOpen = $state(false);
	let notaPreviewUrl = $state('');
	let notaDownloadKey = $state('');
	let editOpen = $state(false);
	let editingTx = $state<Transaction | null>(null);
	let selectedIds = $state<string[]>([]);

	const needsTeam = $derived(user?.role === 'ops' && !user?.team_id);
	const periodRange = $derived(getMonthRange(filterMonth));
	const periodLabel = $derived(formatMonthLabel(periodRange.from));
	const totalPages = $derived(Math.max(1, Math.ceil(txTotal / PAGE_SIZE)));
	const rangeStart = $derived(txTotal === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1);
	const rangeEnd = $derived(Math.min(currentPage * PAGE_SIZE, txTotal));

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
		const tableParams = {
			...buildParams(true),
			page: String(currentPage),
			limit: String(PAGE_SIZE)
		};
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

		const maxPage = Math.max(1, Math.ceil(txRes.total / PAGE_SIZE));
		if (currentPage > maxPage) {
			currentPage = maxPage;
			await loadTeamData();
		}
	}

	function applyFilter() {
		currentPage = 1;
		selectedIds = [];
		loadTeamData();
	}

	function goToPage(page: number) {
		const next = Math.min(Math.max(1, page), totalPages);
		if (next === currentPage) return;
		currentPage = next;
		selectedIds = [];
		loadTeamData();
	}

	async function viewNota(key: string) {
		const { url } = await api.getNotaURL(selectedTeam, key);
		notaPreviewUrl = url;
		notaDownloadKey = key;
		notaPreviewOpen = true;
	}

	async function downloadNota(key: string) {
		const { url } = await api.getNotaURL(selectedTeam, key, true);
		window.open(url, '_blank');
	}

	function downloadPreviewNota() {
		if (notaDownloadKey) downloadNota(notaDownloadKey);
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
		load();
	});
</script>

<svelte:head><title>Dashboard — KasQ</title></svelte:head>

<Heading tag="h1" class="mb-3 text-xl sm:mb-4 sm:text-2xl">Dashboard</Heading>

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
				<div class="col-span-2 md:max-w-[140px]">
					<Label for="filter-jenis" class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Jenis</Label>
					<Select id="filter-jenis" bind:value={filterJenis} placeholder="Semua jenis" class="w-full">
						<option value="">Semua jenis</option>
						<option value="in">Pemasukan</option>
						<option value="out">Pengeluaran</option>
					</Select>
				</div>
				<Button color="light" class="col-span-2 md:col-span-1" onclick={applyFilter}>Terapkan</Button>
				{#if selectedIds.length > 0}
					<Button color="red" outline class="col-span-2 md:col-span-1" onclick={batchDelete}>
						Hapus ({selectedIds.length})
					</Button>
				{/if}
			</div>
		</div>
		<TxTable
			{transactions}
			selectable
			bind:selectedIds
			onViewNota={viewNota}
			onDownloadNota={downloadNota}
			onEdit={editTx}
			onDelete={deleteTx}
		/>
		{#if txTotal > PAGE_SIZE}
			<div class="mt-4 flex flex-col items-center justify-between gap-3 border-t border-slate-200 pt-4 dark:border-slate-700 sm:flex-row">
				<p class="text-sm text-slate-500 dark:text-slate-400">
					Menampilkan {rangeStart}–{rangeEnd} dari {txTotal} transaksi
				</p>
				<div class="flex items-center gap-2">
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
				</div>
			</div>
		{/if}
	</Card>
{/if}

<NotaPreviewModal bind:open={notaPreviewOpen} src={notaPreviewUrl} onDownload={downloadPreviewNota} />
<TxEditModal bind:open={editOpen} teamId={selectedTeam} tx={editingTx} onSaved={onTxSaved} />

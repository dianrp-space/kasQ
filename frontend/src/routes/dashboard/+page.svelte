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
	import { formatMonthLabel, getCurrentMonthRange } from '$lib/utils';
	import { goto } from '$app/navigation';
	import { Button, Card, Heading, Input, Label, Select } from 'flowbite-svelte';

	const defaultMonth = getCurrentMonthRange();

	let user = $state<User | null>(null);
	let teams = $state<Team[]>([]);
	let selectedTeam = $state('');
	let balance = $state<Balance | null>(null);
	let transactions = $state<Transaction[]>([]);
	let filterJenis = $state('');
	let filterFrom = $state(defaultMonth.from);
	let filterTo = $state(defaultMonth.to);
	let notaPreviewOpen = $state(false);
	let notaPreviewUrl = $state('');
	let notaDownloadKey = $state('');
	let editOpen = $state(false);
	let editingTx = $state<Transaction | null>(null);

	const needsTeam = $derived(user?.role === 'ops' && !user?.team_id);
	const periodLabel = $derived(formatMonthLabel(filterFrom));

	function buildParams(includeJenis = true) {
		const params: Record<string, string> = {};
		if (includeJenis && filterJenis) params.jenis = filterJenis;
		if (filterFrom) params.date_from = filterFrom;
		if (filterTo) params.date_to = filterTo;
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
		const txParams = buildParams(true);
		const balanceParams = buildParams(false);
		[balance, transactions] = await Promise.all([
			api.getBalance(selectedTeam, balanceParams),
			api.getTransactions(selectedTeam, txParams)
		]);
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
			await loadTeamData();
			toast.success('Transaksi dihapus');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Gagal hapus transaksi');
		}
	}

	async function onTxSaved() {
		await loadTeamData();
		const balanceParams = buildParams(false);
		balance = await api.getBalance(selectedTeam, balanceParams);
	}

	$effect(() => {
		load();
	});

	$effect(() => {
		if (selectedTeam && !needsTeam) loadTeamData();
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
		<DashboardChart {balance} {transactions} />
	{/if}

	<Card size="xl" shadow="sm" class="p-3 sm:p-4">
		<div class="mb-3 flex flex-col gap-2 md:flex-row md:flex-wrap md:items-end">
			<div class="md:flex-1">
				<Heading tag="h2" class="text-base sm:text-lg">Transaksi</Heading>
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
				<div>
					<Label for="filter-from" class="sr-only">Dari</Label>
					<Input id="filter-from" type="date" bind:value={filterFrom} class="md:max-w-[160px]" />
				</div>
				<div>
					<Label for="filter-to" class="sr-only">Sampai</Label>
					<Input id="filter-to" type="date" bind:value={filterTo} class="md:max-w-[160px]" />
				</div>
				<Button color="light" class="col-span-2 md:col-span-1" onclick={loadTeamData}>Filter</Button>
			</div>
		</div>
		<TxTable
			{transactions}
			onViewNota={viewNota}
			onDownloadNota={downloadNota}
			onEdit={editTx}
			onDelete={deleteTx}
		/>
	</Card>
{/if}

<NotaPreviewModal bind:open={notaPreviewOpen} src={notaPreviewUrl} onDownload={downloadPreviewNota} />
<TxEditModal bind:open={editOpen} teamId={selectedTeam} tx={editingTx} onSaved={onTxSaved} />

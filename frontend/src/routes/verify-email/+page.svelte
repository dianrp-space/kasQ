<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { onMount } from 'svelte';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { Alert, Button, Spinner } from 'flowbite-svelte';

	let status = $state<'loading' | 'success' | 'error'>('loading');
	let message = $state('');

	onMount(async () => {
		const token = $page.url.searchParams.get('token');
		if (!token) {
			status = 'error';
			message = 'Token verifikasi tidak ditemukan';
			return;
		}
		try {
			const res = await api.verifyEmail(token);
			status = 'success';
			message = res.message;
		} catch (err) {
			status = 'error';
			message = err instanceof Error ? err.message : 'Verifikasi gagal';
		}
	});
</script>

<svelte:head><title>Verifikasi email — KasQ</title></svelte:head>

<AuthShell
	title="Verifikasi email"
	subtitle={status === 'loading' ? 'Memeriksa tautan konfirmasi.' : ''}
>
	{#if status === 'loading'}
		<div class="flex items-center gap-3 py-4 text-slate-500 dark:text-slate-400">
			<Spinner size="6" />
			<span>Memverifikasi</span>
		</div>
	{:else if status === 'success'}
		<Alert color="green">{message}</Alert>
		<Button href="/login" class="auth-submit mt-4">Masuk</Button>
	{:else}
		<Alert color="red">{message}</Alert>
		<Button href="/login" color="light" class="mt-4 w-full">Kembali ke masuk</Button>
	{/if}

	{#snippet footer()}
		<a href="/login">Ke halaman masuk</a>
	{/snippet}
</AuthShell>

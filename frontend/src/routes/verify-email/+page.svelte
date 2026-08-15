<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { onMount } from 'svelte';
	import { Alert, Button, Card, Heading, Spinner } from 'flowbite-svelte';

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

<div class="auth-shell">
	<Card class="w-full max-w-md p-6 text-center">
		<Heading tag="h1" class="mb-4 text-xl text-primary-700">Verifikasi Email</Heading>
		{#if status === 'loading'}
			<div class="flex flex-col items-center gap-3">
				<Spinner />
				<p class="text-slate-500">Memverifikasi...</p>
			</div>
		{:else if status === 'success'}
			<Alert color="green" class="text-left">{message}</Alert>
			<Button href="/login" class="mt-4">Login</Button>
		{:else}
			<Alert color="red" class="text-left">{message}</Alert>
			<Button href="/login" color="light" class="mt-4">Kembali ke Login</Button>
		{/if}
	</Card>
</div>

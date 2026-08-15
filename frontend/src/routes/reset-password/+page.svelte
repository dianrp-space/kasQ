<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { goto } from '$app/navigation';
	import { Alert, Button, Card, Heading, Input, Label } from 'flowbite-svelte';

	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	const token = $derived($page.url.searchParams.get('token') ?? '');

	async function submit(e: Event) {
		e.preventDefault();
		if (!token) {
			error = 'Token reset tidak valid';
			return;
		}
		if (password !== confirmPassword) {
			error = 'Konfirmasi password tidak cocok';
			return;
		}
		loading = true;
		error = '';
		try {
			const res = await api.resetPassword(token, password);
			success = res.message;
			setTimeout(() => goto('/login'), 2000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Reset gagal';
		} finally {
			loading = false;
		}
	}
</script>

<div class="auth-shell">
	<Card class="w-full max-w-md p-6">
		<div class="mb-6 text-center">
			<Heading tag="h1" class="text-2xl text-primary-700">Reset Password</Heading>
			<p class="text-sm text-slate-500">Masukkan password baru</p>
		</div>

		{#if !token}
			<Alert color="red">Link reset tidak valid. Minta link baru dari halaman lupa password.</Alert>
			<Button href="/forgot-password" color="light" class="mt-4">Lupa Password</Button>
		{:else if success}
			<Alert color="green">
				<p>{success}</p>
				<p class="mt-1 text-sm">Mengalihkan ke login...</p>
			</Alert>
		{:else}
			<form onsubmit={submit} class="space-y-4">
				<div>
					<Label for="password">Password Baru</Label>
					<Input id="password" type="password" bind:value={password} required minlength={6} />
				</div>
				<div>
					<Label for="confirm">Konfirmasi Password</Label>
					<Input id="confirm" type="password" bind:value={confirmPassword} required minlength={6} />
				</div>
				{#if error}<Alert color="red">{error}</Alert>{/if}
				<Button type="submit" class="w-full" disabled={loading}>
					{loading ? 'Menyimpan...' : 'Simpan Password'}
				</Button>
			</form>
		{/if}
	</Card>
</div>

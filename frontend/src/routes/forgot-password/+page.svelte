<script lang="ts">
	import { api } from '$lib/api';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import { Alert, Button, Input, Label, Spinner } from 'flowbite-svelte';
	import { CheckCircleSolid } from 'flowbite-svelte-icons';

	let email = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	async function submit(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';
		success = '';
		try {
			const res = await api.forgotPassword(email);
			success = res.message;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal mengirim email';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Lupa password — KasQ</title></svelte:head>

<AuthShell title="Lupa password" subtitle="Masukkan email akun. Kami kirim tautan untuk mengatur password baru.">
	{#if success}
		<Alert color="green">
			{#snippet icon()}
				<CheckCircleSolid class="h-5 w-5" />
			{/snippet}
			<span class="font-medium">{success}</span>
			<p class="mt-2 text-sm">
				Jika <strong class="break-all">{email}</strong> terdaftar, cek inbox dan folder spam. Tautan reset biasanya berlaku terbatas.
			</p>
			<Button href="/login" class="auth-submit mt-4">Kembali ke masuk</Button>
		</Alert>
	{:else}
		<form onsubmit={submit} class="space-y-4">
			<div class="auth-field">
				<Label for="email" class="text-sm font-medium text-slate-700 dark:text-slate-300">Email</Label>
				<Input id="email" type="email" bind:value={email} required autocomplete="email" placeholder="nama@perusahaan.com" />
			</div>
			{#if error}<Alert color="red">{error}</Alert>{/if}
			<Button type="submit" class="auth-submit" disabled={loading}>
				{#if loading}
					<Spinner size="4" class="me-2" />
				{/if}
				{loading ? 'Mengirim' : 'Kirim tautan reset'}
			</Button>
		</form>
	{/if}

	{#snippet footer()}
		<a href="/login">Kembali ke masuk</a>
	{/snippet}
</AuthShell>

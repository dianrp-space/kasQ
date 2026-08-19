<script lang="ts">
	import { page } from '$app/stores';
	import { api } from '$lib/api';
	import { goto } from '$app/navigation';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import PasswordInput from '$lib/components/PasswordInput.svelte';
	import { Alert, Button, Label, Spinner } from 'flowbite-svelte';

	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	const token = $derived($page.url.searchParams.get('token') ?? '');
	const mismatch = $derived(confirmPassword.length > 0 && password !== confirmPassword);
	const tooShort = $derived(password.length > 0 && password.length < 6);

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

<svelte:head><title>Reset password — KasQ</title></svelte:head>

<AuthShell title="Password baru" subtitle="Masukkan password baru untuk akun ini.">
	{#if !token}
		<Alert color="red">Tautan reset tidak valid. Minta tautan baru dari halaman lupa password.</Alert>
		<Button href="/forgot-password" class="auth-submit mt-4">Lupa password</Button>
	{:else if success}
		<Alert color="green">
			<p>{success}</p>
			<p class="mt-1 text-sm">Mengalihkan ke halaman masuk.</p>
		</Alert>
	{:else}
		<form onsubmit={submit} class="space-y-4">
			<div class="auth-field">
				<Label for="password" class="text-sm font-medium text-slate-700 dark:text-slate-300">Password baru</Label>
				<PasswordInput id="password" bind:value={password} required minlength={6} autocomplete="new-password" />
				{#if tooShort}
					<p class="auth-hint-error">Minimal 6 karakter</p>
				{:else}
					<p class="auth-hint">Minimal 6 karakter</p>
				{/if}
			</div>
			<div class="auth-field">
				<Label for="confirm" class="text-sm font-medium text-slate-700 dark:text-slate-300">Konfirmasi password</Label>
				<PasswordInput id="confirm" bind:value={confirmPassword} required minlength={6} autocomplete="new-password" />
				{#if mismatch}
					<p class="auth-hint-error">Konfirmasi belum cocok</p>
				{/if}
			</div>
			{#if error}<Alert color="red">{error}</Alert>{/if}
			<Button type="submit" class="auth-submit" disabled={loading || mismatch}>
				{#if loading}
					<Spinner size="4" class="me-2" />
				{/if}
				{loading ? 'Menyimpan' : 'Simpan password'}
			</Button>
		</form>
	{/if}

	{#snippet footer()}
		<a href="/login">Kembali ke masuk</a>
	{/snippet}
</AuthShell>

<script lang="ts">
	import { api } from '$lib/api';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import PasswordInput from '$lib/components/PasswordInput.svelte';
	import { Alert, Button, Input, Label, Spinner } from 'flowbite-svelte';
	import { CheckCircleSolid } from 'flowbite-svelte-icons';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	const mismatch = $derived(confirmPassword.length > 0 && password !== confirmPassword);
	const tooShort = $derived(password.length > 0 && password.length < 6);

	async function register(e: Event) {
		e.preventDefault();
		error = '';
		success = '';
		if (password !== confirmPassword) {
			error = 'Konfirmasi password tidak cocok';
			return;
		}
		loading = true;
		try {
			const res = await api.register({ name, email, password });
			success = res.message;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Registrasi gagal';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Daftar — KasQ</title></svelte:head>

<AuthShell title="Buat akun" subtitle="Akun ops untuk mencatat transaksi. Admin akan menetapkan tim/kas setelah email terverifikasi.">
	{#if success}
		<Alert color="green">
			{#snippet icon()}
				<CheckCircleSolid class="h-5 w-5" />
			{/snippet}
			<span class="font-medium">{success}</span>
			<ul class="mt-2 list-inside list-disc space-y-1 text-sm">
				<li>Cek inbox <strong class="break-all">{email}</strong></li>
				<li>Cek folder spam / junk</li>
				<li>Tautan konfirmasi berlaku 24 jam</li>
				<li>Setelah verifikasi, admin menetapkan tim/kas sebelum kamu bisa input transaksi</li>
			</ul>
			<Button href="/login" class="auth-submit mt-4">Ke halaman masuk</Button>
		</Alert>
	{:else}
		<form onsubmit={register} class="space-y-4">
			<div class="auth-field">
				<Label for="name" class="text-sm font-medium text-slate-700 dark:text-slate-300">Nama</Label>
				<Input id="name" bind:value={name} required minlength={2} autocomplete="name" placeholder="Nama lengkap" />
			</div>
			<div class="auth-field">
				<Label for="email" class="text-sm font-medium text-slate-700 dark:text-slate-300">Email</Label>
				<Input id="email" type="email" bind:value={email} required autocomplete="email" placeholder="nama@perusahaan.com" />
			</div>
			<div class="auth-field">
				<Label for="password" class="text-sm font-medium text-slate-700 dark:text-slate-300">Password</Label>
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
				{loading ? 'Mendaftar' : 'Daftar'}
			</Button>
		</form>
	{/if}

	{#snippet footer()}
		Sudah punya akun?
		<a href="/login">Masuk</a>
	{/snippet}
</AuthShell>

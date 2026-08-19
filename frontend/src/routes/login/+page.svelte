<script lang="ts">
	import { api, ApiError } from '$lib/api';
	import { homePath } from '$lib/roles';
	import { goto } from '$app/navigation';
	import AuthShell from '$lib/components/AuthShell.svelte';
	import PasswordInput from '$lib/components/PasswordInput.svelte';
	import { Alert, Button, Input, Label, Spinner } from 'flowbite-svelte';
	import { EnvelopeSolid } from 'flowbite-svelte-icons';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let needsVerification = $state(false);
	let loading = $state(false);
	let resendLoading = $state(false);
	let resendSuccess = $state(false);
	let resendError = $state('');

	async function login(e: Event) {
		e.preventDefault();
		loading = true;
		error = '';
		needsVerification = false;
		resendSuccess = false;
		resendError = '';
		try {
			await api.login(email, password);
			const me = await api.me();
			goto(homePath(me.role));
		} catch (err) {
			if (err instanceof ApiError) {
				error = err.message;
				needsVerification = err.needsVerification;
			} else {
				error = 'Login gagal. Coba lagi.';
			}
		} finally {
			loading = false;
		}
	}

	async function resend() {
		if (!email) {
			resendError = 'Isi email terlebih dahulu';
			return;
		}
		resendLoading = true;
		resendSuccess = false;
		resendError = '';
		try {
			await api.resendVerification(email);
			resendSuccess = true;
		} catch (err) {
			resendError = err instanceof Error ? err.message : 'Gagal kirim ulang';
		} finally {
			resendLoading = false;
		}
	}
</script>

<svelte:head><title>Masuk — KasQ</title></svelte:head>

<AuthShell title="Masuk" subtitle="Gunakan email dan password akun ops.">
	<form onsubmit={login} class="space-y-4">
		<div class="auth-field">
			<Label for="email" class="text-sm font-medium text-slate-700 dark:text-slate-300">Email</Label>
			<Input id="email" type="email" bind:value={email} required autocomplete="email" placeholder="nama@perusahaan.com" />
		</div>
		<div class="auth-field">
			<div class="flex items-center justify-between gap-3">
				<Label for="password" class="text-sm font-medium text-slate-700 dark:text-slate-300">Password</Label>
				<a href="/forgot-password" class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300">Lupa password?</a>
			</div>
			<PasswordInput id="password" bind:value={password} required autocomplete="current-password" />
		</div>

		{#if error && !needsVerification}
			<Alert color="red">{error}</Alert>
		{/if}

		{#if needsVerification}
			<Alert color="yellow">
				{#snippet icon()}
					<EnvelopeSolid class="h-5 w-5" />
				{/snippet}
				<span class="font-medium">Email belum diverifikasi</span>
				<p class="mt-1 text-sm">
					Akun <strong class="break-all">{email}</strong> belum aktif. Buka tautan konfirmasi di inbox sebelum masuk.
				</p>
				{#if resendSuccess}
					<p class="mt-2 text-sm text-emerald-700 dark:text-emerald-400">Email verifikasi sudah dikirim. Cek inbox dan folder spam.</p>
				{/if}
				{#if resendError}
					<p class="mt-2 text-sm text-red-600 dark:text-red-400">{resendError}</p>
				{/if}
				<Button
					type="button"
					color="light"
					class="mt-3 w-full"
					disabled={resendLoading || !email}
					onclick={resend}
				>
					{#if resendLoading}
						<Spinner size="4" class="me-2" />
					{/if}
					{resendLoading ? 'Mengirim' : resendSuccess ? 'Kirim ulang lagi' : 'Kirim ulang email verifikasi'}
				</Button>
			</Alert>
		{/if}

		<Button type="submit" class="auth-submit" disabled={loading}>
			{#if loading}
				<Spinner size="4" class="me-2" />
			{/if}
			{loading ? 'Sedang masuk' : 'Masuk'}
		</Button>
	</form>

	{#snippet footer()}
		Belum punya akun?
		<a href="/register">Daftar</a>
	{/snippet}
</AuthShell>

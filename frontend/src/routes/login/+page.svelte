<script lang="ts">
	import { api, ApiError } from '$lib/api';
	import { loadAppSettings } from '$lib/appSettings.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import { homePath } from '$lib/roles';
	import { goto } from '$app/navigation';
	import { Alert, Button, Card, Input, Label } from 'flowbite-svelte';
	import { EnvelopeSolid } from 'flowbite-svelte-icons';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let needsVerification = $state(false);
	let loading = $state(false);
	let resendLoading = $state(false);
	let resendSuccess = $state(false);
	let resendError = $state('');

	$effect(() => {
		loadAppSettings();
	});

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
				error = 'Login gagal';
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

<div class="auth-shell">
	<Card class="w-full max-w-md p-6">
		<div class="mb-6 text-center">
			<AppBrand size="lg" centered />
		</div>

		<form onsubmit={login} class="space-y-4">
			<div>
				<Label for="email">Email</Label>
				<Input id="email" type="email" bind:value={email} required autocomplete="email" />
			</div>
			<div>
				<div class="mb-1 flex items-center justify-between">
					<Label for="password">Password</Label>
					<a href="/forgot-password" class="text-xs text-primary-600 hover:underline">Lupa password?</a>
				</div>
				<Input id="password" type="password" bind:value={password} required autocomplete="current-password" />
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
						Akun <strong class="break-all">{email}</strong> belum aktif. Klik link konfirmasi di email sebelum login.
					</p>
					{#if resendSuccess}
						<p class="mt-2 text-sm text-emerald-700">Email verifikasi telah dikirim — cek inbox dan spam.</p>
					{/if}
					{#if resendError}
						<p class="mt-2 text-sm text-red-600">{resendError}</p>
					{/if}
					<Button
						type="button"
						color="light"
						class="mt-3 w-full"
						disabled={resendLoading || !email}
						onclick={resend}
					>
						{resendLoading ? 'Mengirim...' : resendSuccess ? 'Kirim Ulang Lagi' : 'Kirim Ulang Email Verifikasi'}
					</Button>
				</Alert>
			{/if}

			<Button type="submit" class="w-full" disabled={loading}>
				{loading ? 'Masuk...' : 'Masuk'}
			</Button>
		</form>

		<p class="mt-4 text-center text-sm text-slate-500">
			Belum punya akun?
			<a href="/register" class="font-medium text-primary-600 hover:underline">Daftar</a>
		</p>
	</Card>
</div>

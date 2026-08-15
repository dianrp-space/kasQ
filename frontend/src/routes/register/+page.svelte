<script lang="ts">
	import { api } from '$lib/api';
	import { loadAppSettings } from '$lib/appSettings.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import { Alert, Button, Card, Heading, Input, Label } from 'flowbite-svelte';
	import { CheckCircleSolid } from 'flowbite-svelte-icons';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	$effect(() => {
		loadAppSettings();
	});

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

<div class="auth-shell">
	<Card class="w-full max-w-md p-6">
		<div class="mb-6 text-center">
			<AppBrand size="lg" centered />
			<p class="mt-2 text-sm text-slate-500">Buat akun ops untuk input transaksi</p>
		</div>

		{#if success}
			<Alert color="green">
				{#snippet icon()}
					<CheckCircleSolid class="h-5 w-5" />
				{/snippet}
				<span class="font-medium">{success}</span>
				<ul class="mt-2 list-inside list-disc space-y-1 text-sm">
					<li>Cek inbox <strong>{email}</strong></li>
					<li>Cek folder <strong>Spam / Junk</strong></li>
					<li>Link konfirmasi berlaku 24 jam</li>
					<li>Setelah verifikasi, admin akan menetapkan tim/kas sebelum kamu bisa input transaksi</li>
				</ul>
				<Button href="/login" class="mt-4 w-full">Ke halaman login</Button>
			</Alert>
		{:else}
			<form onsubmit={register} class="space-y-4">
				<div>
					<Label for="name">Nama</Label>
					<Input id="name" bind:value={name} required minlength={2} />
				</div>
				<div>
					<Label for="email">Email</Label>
					<Input id="email" type="email" bind:value={email} required />
				</div>
				<div>
					<Label for="password">Password</Label>
					<Input id="password" type="password" bind:value={password} required minlength={6} />
				</div>
				<div>
					<Label for="confirm">Konfirmasi Password</Label>
					<Input id="confirm" type="password" bind:value={confirmPassword} required minlength={6} />
				</div>
				{#if error}<Alert color="red">{error}</Alert>{/if}
				<Button type="submit" class="w-full" disabled={loading}>
					{loading ? 'Mendaftar...' : 'Daftar'}
				</Button>
			</form>
		{/if}

		<p class="mt-4 text-center text-sm text-slate-500">
			Sudah punya akun?
			<a href="/login" class="font-medium text-primary-600 hover:underline">Masuk</a>
		</p>
	</Card>
</div>

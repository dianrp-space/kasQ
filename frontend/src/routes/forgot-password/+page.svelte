<script lang="ts">
	import { api } from '$lib/api';
	import { Alert, Button, Card, Heading, Input, Label } from 'flowbite-svelte';

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

<div class="auth-shell">
	<Card class="w-full max-w-md p-6">
		<div class="mb-6 text-center">
			<Heading tag="h1" class="text-2xl text-primary-700">Lupa Password</Heading>
			<p class="text-sm text-slate-500">Masukkan email untuk menerima link reset</p>
		</div>

		<form onsubmit={submit} class="space-y-4">
			<div>
				<Label for="email">Email</Label>
				<Input id="email" type="email" bind:value={email} required />
			</div>
			{#if error}<Alert color="red">{error}</Alert>{/if}
			{#if success}<Alert color="green">{success}</Alert>{/if}
			<Button type="submit" class="w-full" disabled={loading}>
				{loading ? 'Mengirim...' : 'Kirim Link Reset'}
			</Button>
		</form>

		<p class="mt-4 text-center text-sm">
			<a href="/login" class="text-primary-600 hover:underline">Kembali ke login</a>
		</p>
	</Card>
</div>

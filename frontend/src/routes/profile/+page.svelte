<script lang="ts">
	import { api } from '$lib/api';
	import type { User } from '$lib/types';
	import { getContext } from 'svelte';
	import { toast } from '$lib/toast.svelte';
	import { Avatar, Button, Card, Heading, Input, Label } from 'flowbite-svelte';

	let user = $state<User | null>(null);
	let name = $state('');
	let avatarFile = $state<File | null>(null);
	let avatarPreview = $state<string | null>(null);
	let removeAvatar = $state(false);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');

	const refreshUser = getContext<() => Promise<void>>('refreshUser');

	const initials = $derived(
		(name || user?.name || '?')
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('') || '?'
	);

	async function load() {
		loading = true;
		error = '';
		try {
			user = await api.me();
			name = user.name;
			removeAvatar = false;
			avatarFile = null;
			if (avatarPreview) {
				URL.revokeObjectURL(avatarPreview);
				avatarPreview = null;
			}
			if (user.has_avatar) {
				const url = await api.getMyAvatar();
				if (url) avatarPreview = url;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal memuat profil';
		} finally {
			loading = false;
		}
	}

	function onAvatarChange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0] ?? null;
		if (avatarPreview && avatarPreview.startsWith('blob:')) {
			URL.revokeObjectURL(avatarPreview);
		}
		avatarFile = file;
		removeAvatar = false;
		avatarPreview = file ? URL.createObjectURL(file) : null;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!user) return;
		saving = true;
		error = '';
		try {
			const form = new FormData();
			form.append('name', name.trim());
			if (avatarFile) form.append('avatar', avatarFile);
			if (removeAvatar) form.append('remove_avatar', 'true');
			user = await api.updateMe(form);
			toast.success('Profil disimpan');
			await refreshUser?.();
			await load();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal simpan profil';
		} finally {
			saving = false;
		}
	}

	function clearAvatar() {
		if (avatarPreview && avatarPreview.startsWith('blob:')) {
			URL.revokeObjectURL(avatarPreview);
		}
		avatarFile = null;
		avatarPreview = null;
		removeAvatar = true;
	}

	$effect(() => {
		load();
		return () => {
			if (avatarPreview && avatarPreview.startsWith('blob:')) {
				URL.revokeObjectURL(avatarPreview);
			}
		};
	});
</script>

<svelte:head><title>Profil — KasQ</title></svelte:head>

<Heading tag="h1" class="mb-4 text-2xl">Profil</Heading>

{#if loading}
	<p class="text-slate-500 dark:text-slate-400">Memuat...</p>
{:else if user}
	<Card size="lg" shadow="sm" class="max-w-xl p-4 sm:p-6">
		<form onsubmit={save} class="space-y-5">
			<div class="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
				{#if avatarPreview && !removeAvatar}
					<Avatar src={avatarPreview} alt={name} size="xl" />
				{:else}
					<Avatar size="xl" class="bg-primary-100 text-primary-700 dark:bg-primary-900/50 dark:text-primary-300">
						<span class="text-lg font-semibold">{initials}</span>
					</Avatar>
				{/if}
				<div class="flex flex-col gap-2">
					<Label for="avatar">Foto profil</Label>
					<input
						id="avatar"
						type="file"
						accept="image/png,image/jpeg,image/webp"
						class="text-sm text-slate-600 file:mr-3 file:rounded-lg file:border-0 file:bg-primary-50 file:px-3 file:py-2 file:text-sm file:font-medium file:text-primary-700 dark:text-slate-300 dark:file:bg-primary-900/40 dark:file:text-primary-300"
						onchange={onAvatarChange}
					/>
					{#if user.has_avatar || avatarPreview}
						<Button type="button" color="light" size="xs" class="w-fit" onclick={clearAvatar}>
							Hapus foto
						</Button>
					{/if}
					<p class="text-xs text-slate-500 dark:text-slate-400">PNG, JPG, atau WEBP. Maks. 2MB.</p>
				</div>
			</div>

			<div>
				<Label for="name">Nama</Label>
				<Input id="name" bind:value={name} required />
			</div>

			<div>
				<Label for="email">Email</Label>
				<Input id="email" value={user.email} disabled />
			</div>

			<div>
				<Label for="role">Peran</Label>
				<Input id="role" value={user.role} disabled />
			</div>

			{#if error}
				<p class="text-sm text-red-600 dark:text-red-400">{error}</p>
			{/if}

			<Button type="submit" color="primary" disabled={saving}>
				{saving ? 'Menyimpan...' : 'Simpan'}
			</Button>
		</form>
	</Card>
{/if}

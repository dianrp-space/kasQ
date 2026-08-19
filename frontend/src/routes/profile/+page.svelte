<script lang="ts">
	import { api } from '$lib/api';
	import type { User } from '$lib/types';
	import { getContext, onMount } from 'svelte';
	import { toast } from '$lib/toast.svelte';
	import { Alert, Avatar, Button, Input, Label, Spinner } from 'flowbite-svelte';
	import PasswordInput from '$lib/components/PasswordInput.svelte';
	import { CameraPhotoOutline, TrashBinOutline } from 'flowbite-svelte-icons';

	let user = $state<User | null>(null);
	let name = $state('');
	let avatarFile = $state<File | null>(null);
	let avatarPreview = $state<string | null>(null);
	let removeAvatar = $state(false);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let passwordSaving = $state(false);
	let passwordError = $state('');
	let passwordSuccess = $state('');

	let blobPreviewUrl: string | null = null;
	let avatarInput = $state<HTMLInputElement | undefined>();

	const refreshUser = getContext<() => Promise<void>>('refreshUser');

	const initials = $derived(
		(name || user?.name || '?')
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('') || '?'
	);

	const roleLabel = $derived(user?.role === 'admin' ? 'Admin' : 'Ops');
	const nameDirty = $derived(name.trim() !== (user?.name ?? ''));
	const avatarDirty = $derived(!!avatarFile || removeAvatar);
	const profileDirty = $derived(nameDirty || avatarDirty);
	const passwordMismatch = $derived(confirmPassword.length > 0 && newPassword !== confirmPassword);
	const passwordTooShort = $derived(newPassword.length > 0 && newPassword.length < 6);
	const showAvatar = $derived(!!avatarPreview && !removeAvatar);

	async function load(bustAvatar = false) {
		loading = true;
		error = '';
		try {
			user = await api.me();
			name = user.name;
			removeAvatar = false;
			avatarFile = null;
			if (blobPreviewUrl) {
				URL.revokeObjectURL(blobPreviewUrl);
				blobPreviewUrl = null;
			}
			avatarPreview = null;
			if (user.has_avatar) {
				avatarPreview = api.getMyAvatar(bustAvatar ? Date.now() : undefined);
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
		if (blobPreviewUrl) {
			URL.revokeObjectURL(blobPreviewUrl);
			blobPreviewUrl = null;
		}
		avatarFile = file;
		removeAvatar = false;
		blobPreviewUrl = file ? URL.createObjectURL(file) : null;
		avatarPreview = blobPreviewUrl;
	}

	async function save(e: Event) {
		e.preventDefault();
		if (!user || !profileDirty) return;
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
			await load(true);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal simpan profil';
		} finally {
			saving = false;
		}
	}

	function clearAvatar() {
		if (blobPreviewUrl) {
			URL.revokeObjectURL(blobPreviewUrl);
			blobPreviewUrl = null;
		}
		avatarFile = null;
		avatarPreview = null;
		removeAvatar = true;
		if (avatarInput) avatarInput.value = '';
	}

	async function changePassword(e: Event) {
		e.preventDefault();
		passwordError = '';
		passwordSuccess = '';
		if (newPassword !== confirmPassword) {
			passwordError = 'Konfirmasi password tidak cocok';
			return;
		}
		passwordSaving = true;
		try {
			const res = await api.changePassword(currentPassword, newPassword);
			passwordSuccess = res.message;
			currentPassword = '';
			newPassword = '';
			confirmPassword = '';
			toast.success('Password diubah');
		} catch (err) {
			passwordError = err instanceof Error ? err.message : 'Gagal ubah password';
		} finally {
			passwordSaving = false;
		}
	}

	onMount(() => {
		load();
		return () => {
			if (blobPreviewUrl) URL.revokeObjectURL(blobPreviewUrl);
		};
	});
</script>

<svelte:head><title>Profil — KasQ</title></svelte:head>

<div class="mb-5 sm:mb-6">
	<h1 class="text-xl font-semibold tracking-tight text-slate-900 dark:text-slate-50 sm:text-2xl">Profil</h1>
	<p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Nama, foto, dan password akun.</p>
</div>

{#if loading}
	<div class="profile-stack" aria-busy="true">
		<div class="profile-skel">
			<div class="flex items-center gap-4">
				<div class="h-20 w-20 rounded-full bg-slate-100 dark:bg-slate-800"></div>
				<div class="flex-1 space-y-2">
					<div class="h-4 w-40 rounded bg-slate-100 dark:bg-slate-800"></div>
					<div class="h-3 w-56 rounded bg-slate-100 dark:bg-slate-800"></div>
				</div>
			</div>
			<div class="mt-6 h-10 rounded-lg bg-slate-100 dark:bg-slate-800"></div>
			<div class="mt-3 h-10 rounded-lg bg-slate-100 dark:bg-slate-800"></div>
		</div>
		<div class="profile-skel h-56"></div>
	</div>
{:else if !user}
	<Alert color="red" class="max-w-2xl">{error || 'Profil tidak bisa dimuat.'}</Alert>
{:else}
	<div class="profile-stack">
		<section class="profile-panel">
			<form onsubmit={save} class="space-y-6">
				<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
					<div class="relative w-fit shrink-0">
						{#if showAvatar}
							<Avatar src={avatarPreview ?? undefined} alt={name} size="xl" />
						{:else}
							<Avatar size="xl" class="bg-primary-100 text-primary-800 dark:bg-primary-900/50 dark:text-primary-200">
								<span class="text-lg font-semibold">{initials}</span>
							</Avatar>
						{/if}
						<label class="profile-avatar-btn" title="Ganti foto">
							<CameraPhotoOutline class="h-4 w-4" />
							<span class="sr-only">Ganti foto profil</span>
							<input
								bind:this={avatarInput}
								id="avatar"
								type="file"
								accept="image/png,image/jpeg,image/webp"
								class="sr-only"
								onchange={onAvatarChange}
							/>
						</label>
					</div>
					<div class="min-w-0 flex-1">
						<p class="truncate text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-50">
							{name.trim() || user.name}
						</p>
						<p class="mt-0.5 truncate text-sm text-slate-500 dark:text-slate-400">{user.email}</p>
						<div class="mt-2 flex flex-wrap items-center gap-2">
							<span class="rounded-full bg-primary-50 px-2.5 py-0.5 text-xs font-medium text-primary-800 dark:bg-primary-900/40 dark:text-primary-200">
								{roleLabel}
							</span>
							{#if avatarDirty}
								<span class="text-xs text-slate-500 dark:text-slate-400">Foto belum disimpan</span>
							{/if}
						</div>
					</div>
				</div>

				<div class="grid gap-4 sm:grid-cols-2">
					<div class="auth-field sm:col-span-2">
						<Label for="name" class="text-sm font-medium text-slate-700 dark:text-slate-300">Nama</Label>
						<Input id="name" bind:value={name} required autocomplete="name" />
					</div>
					<div class="profile-meta-row">
						<p class="profile-meta-label">Email</p>
						<p class="profile-meta-value break-all">{user.email}</p>
						<p class="auth-hint">Email tidak bisa diubah dari sini.</p>
					</div>
					<div class="profile-meta-row">
						<p class="profile-meta-label">Peran</p>
						<p class="profile-meta-value">{roleLabel}</p>
						<p class="auth-hint">Ditetapkan oleh admin.</p>
					</div>
				</div>

				<div class="flex flex-wrap items-center gap-2">
					{#if showAvatar || user.has_avatar}
						<Button type="button" color="light" size="sm" onclick={clearAvatar}>
							<TrashBinOutline class="me-1.5 h-4 w-4" />
							Hapus foto
						</Button>
					{/if}
					<p class="text-xs text-slate-500 dark:text-slate-400">PNG, JPG, atau WEBP. Maks. 2MB.</p>
				</div>

				{#if error}
					<Alert color="red">{error}</Alert>
				{/if}

				<Button type="submit" class="auth-submit" disabled={saving || !profileDirty}>
					{#if saving}
						<Spinner size="4" class="me-2" />
					{/if}
					{saving ? 'Menyimpan' : 'Simpan profil'}
				</Button>
			</form>
		</section>

		<section class="profile-panel">
			<h2 class="text-base font-semibold tracking-tight text-slate-900 dark:text-slate-50">Password</h2>
			<p class="mt-1 mb-5 text-sm text-slate-500 dark:text-slate-400">Ganti password untuk akun ini.</p>
			<form onsubmit={changePassword} class="space-y-4">
				<div class="auth-field">
					<Label for="currentPassword" class="text-sm font-medium text-slate-700 dark:text-slate-300">Password saat ini</Label>
					<PasswordInput
						id="currentPassword"
						bind:value={currentPassword}
						required
						autocomplete="current-password"
					/>
				</div>
				<div class="auth-field">
					<Label for="newPassword" class="text-sm font-medium text-slate-700 dark:text-slate-300">Password baru</Label>
					<PasswordInput
						id="newPassword"
						bind:value={newPassword}
						required
						minlength={6}
						autocomplete="new-password"
					/>
					{#if passwordTooShort}
						<p class="auth-hint-error">Minimal 6 karakter</p>
					{:else}
						<p class="auth-hint">Minimal 6 karakter</p>
					{/if}
				</div>
				<div class="auth-field">
					<Label for="confirmPassword" class="text-sm font-medium text-slate-700 dark:text-slate-300">Konfirmasi password baru</Label>
					<PasswordInput
						id="confirmPassword"
						bind:value={confirmPassword}
						required
						minlength={6}
						autocomplete="new-password"
					/>
					{#if passwordMismatch}
						<p class="auth-hint-error">Konfirmasi belum cocok</p>
					{/if}
				</div>
				{#if passwordError}
					<Alert color="red">{passwordError}</Alert>
				{/if}
				{#if passwordSuccess}
					<Alert color="green">{passwordSuccess}</Alert>
				{/if}
				<Button type="submit" class="auth-submit" disabled={passwordSaving || passwordMismatch}>
					{#if passwordSaving}
						<Spinner size="4" class="me-2" />
					{/if}
					{passwordSaving ? 'Menyimpan' : 'Ubah password'}
				</Button>
			</form>
		</section>
	</div>
{/if}

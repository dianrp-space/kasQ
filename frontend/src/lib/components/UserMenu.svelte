<script lang="ts">
	import type { User } from '$lib/types';
	import { api } from '$lib/api';
	import { toast } from '$lib/toast.svelte';
	import { getContext } from 'svelte';
	import { Avatar, Dropdown, DropdownItem, Modal } from 'flowbite-svelte';
	import { CameraPhotoOutline, EyeOutline } from 'flowbite-svelte-icons';

	const maxAvatarSize = 2 * 1024 * 1024;

	let {
		user,
		menuId,
		placement = 'bottom-end'
	}: {
		user: User;
		menuId: string;
		placement?: 'bottom-end' | 'top' | 'top-start' | 'top-end';
	} = $props();

	let avatarUrl = $state<string | null>(null);
	let viewOpen = $state(false);
	let fileInput = $state<HTMLInputElement | undefined>();
	let uploading = $state(false);

	const refreshUser = getContext<() => Promise<void>>('refreshUser');
	const initials = $derived(
		user.name
			.split(/\s+/)
			.filter(Boolean)
			.slice(0, 2)
			.map((part) => part[0]?.toUpperCase() ?? '')
			.join('') || '?'
	);

	$effect(() => {
		avatarUrl = user.has_avatar ? api.getMyAvatar() : null;
	});

	function openFilePicker() {
		fileInput?.click();
	}

	async function onFileChange(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		const file = input.files?.[0] ?? null;
		input.value = '';
		if (!file) return;
		if (file.size > maxAvatarSize) {
			toast.error('Foto profil maksimal 2MB');
			return;
		}
		if (!/^image\/(png|jpeg|jpg|webp)$/i.test(file.type) && !/\.(png|jpe?g|webp)$/i.test(file.name)) {
			toast.error('Foto harus PNG, JPG, atau WEBP');
			return;
		}
		uploading = true;
		try {
			const form = new FormData();
			form.append('name', user.name);
			form.append('avatar', file);
			await api.updateMe(form);
			await refreshUser?.();
			avatarUrl = api.getMyAvatar(Date.now());
			toast.success('Foto profil diperbarui');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Gagal unggah foto');
		} finally {
			uploading = false;
		}
	}
</script>

<input
	bind:this={fileInput}
	type="file"
	accept="image/png,image/jpeg,image/webp"
	class="sr-only"
	onchange={onFileChange}
/>

<button
	id={menuId}
	type="button"
	class="user-menu-trigger shrink-0 rounded-full p-0.5 ring-2 ring-slate-200 transition hover:ring-primary-400 focus:outline-none focus:ring-primary-500 disabled:opacity-60 dark:ring-slate-600 dark:hover:ring-primary-500"
	aria-label="Foto profil {user.name}"
	title="Foto profil"
	disabled={uploading}
>
	{#if avatarUrl}
		<Avatar src={avatarUrl} alt={user.name} size="md" class="shadow-sm" />
	{:else}
		<Avatar
			size="md"
			class="bg-primary-600 text-sm font-semibold text-white shadow-sm dark:bg-primary-500"
		>
			{initials}
		</Avatar>
	{/if}
</button>

<Dropdown simple triggeredBy={`#${menuId}`} {placement} class="user-menu-dropdown w-48 py-1">
	<DropdownItem
		onclick={() => (viewOpen = true)}
		liClass="list-none"
		class="flex items-center gap-2.5 px-4 py-2.5"
	>
		<EyeOutline class="h-4 w-4 shrink-0 text-slate-500 dark:text-slate-400" />
		<span>Lihat foto</span>
	</DropdownItem>
	<DropdownItem
		onclick={openFilePicker}
		liClass="list-none"
		class="flex items-center gap-2.5 px-4 py-2.5"
	>
		<CameraPhotoOutline class="h-4 w-4 shrink-0 text-slate-500 dark:text-slate-400" />
		<span>Ubah foto</span>
	</DropdownItem>
</Dropdown>

<Modal bind:open={viewOpen} title="Foto profil" size="md" autoclose={false} class="z-50">
	<div class="flex flex-col items-center gap-4 py-2">
		{#if avatarUrl}
			<img src={avatarUrl} alt={user.name} class="max-h-[70vh] w-auto max-w-full rounded-xl object-contain" />
		{:else}
			<Avatar
				size="xl"
				class="h-32 w-32 bg-primary-600 text-3xl font-semibold text-white dark:bg-primary-500"
			>
				{initials}
			</Avatar>
			<p class="text-sm text-slate-500 dark:text-slate-400">Belum ada foto profil</p>
		{/if}
	</div>
</Modal>

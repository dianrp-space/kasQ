<script lang="ts">
	import type { User } from '$lib/types';
	import { api } from '$lib/api';
	import { Avatar, Dropdown, DropdownDivider, DropdownItem } from 'flowbite-svelte';
	import { ArrowRightToBracketOutline, UserCircleSolid } from 'flowbite-svelte-icons';

	let { user, onLogout }: { user: User; onLogout: () => void } = $props();

	let avatarUrl = $state<string | null>(null);
	const menuId = $derived(`user-menu-${user.id}`);
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
</script>

<button
	id={menuId}
	type="button"
	class="user-menu-trigger rounded-full p-0.5 ring-2 ring-slate-200 transition hover:ring-primary-400 focus:outline-none focus:ring-primary-500 dark:ring-slate-600 dark:hover:ring-primary-500"
	aria-label="Menu akun {user.name}"
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

<Dropdown simple triggeredBy={`#${menuId}`} placement="bottom-end" class="user-menu-dropdown w-48 py-1">
	<DropdownItem href="/profile" liClass="list-none" class="flex items-center gap-2.5 px-4 py-2.5">
		<UserCircleSolid class="h-4 w-4 shrink-0 text-slate-500 dark:text-slate-400" />
		<span>Profil</span>
	</DropdownItem>
	<DropdownDivider class="my-1" />
	<DropdownItem
		onclick={onLogout}
		liClass="list-none"
		class="flex w-full items-center gap-2.5 px-4 py-2.5 text-red-600 dark:text-red-400"
	>
		<ArrowRightToBracketOutline class="h-4 w-4 shrink-0" />
		<span>Logout</span>
	</DropdownItem>
</Dropdown>

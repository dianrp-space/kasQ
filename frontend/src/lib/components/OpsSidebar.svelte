<script lang="ts">
	import { page } from '$app/stores';
	import type { User } from '$lib/types';
	import {
		ArrowRightToBracketOutline,
		ChartPieOutline,
		HeartOutline,
		LinkOutline,
		PenOutline,
		UserCircleSolid
	} from 'flowbite-svelte-icons';
	import { opsNavLinks, isOpsNavActive } from '$lib/opsNav';
	import VersionLink from '$lib/components/VersionLink.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import UserMenu from '$lib/components/UserMenu.svelte';

	let { user, onLogout }: { user: User; onLogout: () => void } = $props();

	const icons = {
		'/dashboard': ChartPieOutline,
		'/transactions': PenOutline,
		'/integrations': LinkOutline,
		'/support': HeartOutline,
		'/profile': UserCircleSolid
	} as const;
</script>

<aside class="hidden h-full w-56 shrink-0 flex-col border-r border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900 md:flex">
	<nav class="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto p-4">
		<p class="mb-2 px-3 text-xs font-semibold uppercase tracking-wide text-slate-400 dark:text-slate-500">Menu</p>
		{#each opsNavLinks as link}
			{@const Icon = icons[link.href as keyof typeof icons]}
			<a
				href={link.href}
				class="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors
					{isOpsNavActive(link.href, $page.url.pathname)
					? 'bg-primary-50 text-primary-800 dark:bg-primary-900/40 dark:text-primary-300'
					: 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 dark:hover:text-white'}"
			>
				<Icon class="h-5 w-5 shrink-0" />
				{link.label}
			</a>
		{/each}
	</nav>
	<div class="border-t border-slate-200 p-3 dark:border-slate-700">
		<div class="mb-3 flex items-center gap-2">
			<UserMenu {user} menuId="sidebar-avatar" placement="top-start" />
			<div class="min-w-0 flex-1">
				<p class="truncate text-sm font-medium text-slate-800 dark:text-slate-100">{user.name}</p>
				<p class="truncate text-[11px] text-slate-400 dark:text-slate-500">{user.email}</p>
			</div>
			<ThemeToggle class="shrink-0" />
		</div>
		<button
			type="button"
			class="btn-logout flex w-full items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors"
			onclick={onLogout}
		>
			<ArrowRightToBracketOutline class="h-4 w-4 shrink-0" />
			Logout
		</button>
	</div>
	<VersionLink prefix="KasQ" class="block px-4 pb-4 pt-1 text-[11px] text-slate-400 dark:text-slate-500" />
</aside>

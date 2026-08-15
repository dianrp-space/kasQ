<script lang="ts">
	import { page } from '$app/stores';
	import {
		ChartPieOutline,
		HeartOutline,
		LinkOutline,
		PenOutline,
		UserCircleSolid
	} from 'flowbite-svelte-icons';
	import {
		opsBottomNavCenterHref,
		opsBottomNavLinks,
		isOpsNavActive
	} from '$lib/opsNav';

	const icons = {
		'/dashboard': ChartPieOutline,
		'/transactions': PenOutline,
		'/integrations': LinkOutline,
		'/support': HeartOutline,
		'/profile': UserCircleSolid
	} as const;
</script>

<nav
	class="ops-bottom-nav fixed inset-x-0 bottom-0 z-40 overflow-visible border-t border-slate-200 bg-white/95 backdrop-blur dark:border-slate-700 dark:bg-slate-900/95 md:hidden"
	aria-label="Navigasi utama"
>
	<div class="mx-auto grid h-16 max-w-lg grid-cols-5 items-end px-1 pb-2">
		{#each opsBottomNavLinks as link}
			{@const Icon = icons[link.href as keyof typeof icons]}
			{@const active = isOpsNavActive(link.href, $page.url.pathname)}
			{#if link.href === opsBottomNavCenterHref}
				<div class="flex flex-col items-center justify-end py-2">
					<a
						href={link.href}
						aria-label={link.shortLabel}
						aria-current={active ? 'page' : undefined}
						class="ops-bottom-nav-fab -mt-9 mb-1 flex h-14 w-14 shrink-0 items-center justify-center rounded-full shadow-lg transition-transform active:scale-95
							{active
							? 'bg-primary-700 text-white ring-4 ring-primary-100 dark:bg-primary-500 dark:ring-primary-900/60'
							: 'bg-primary-600 text-white ring-4 ring-white hover:bg-primary-700 dark:bg-primary-500 dark:ring-slate-900 dark:hover:bg-primary-400'}"
					>
						<Icon class="h-7 w-7" />
					</a>
					<span
						class="text-[11px] font-semibold leading-none
							{active ? 'text-primary-700 dark:text-primary-400' : 'text-slate-600 dark:text-slate-400'}"
					>
						{link.shortLabel}
					</span>
				</div>
			{:else}
				<a
					href={link.href}
					aria-label={link.shortLabel}
					aria-current={active ? 'page' : undefined}
					class="flex flex-col items-center justify-end gap-0.5 rounded-lg px-1 py-2 transition-colors
						{active
						? 'text-primary-700 dark:text-primary-400'
						: 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'}"
				>
					<Icon class="h-6 w-6 shrink-0" />
					<span class="text-[11px] font-medium leading-none">{link.shortLabel}</span>
				</a>
			{/if}
		{/each}
	</div>
</nav>

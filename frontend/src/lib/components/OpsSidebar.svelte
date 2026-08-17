<script lang="ts">
	import { page } from '$app/stores';
	import { ChartPieOutline, HeartOutline, LinkOutline, PenOutline, UserCircleSolid } from 'flowbite-svelte-icons';
	import { opsNavLinks, isOpsNavActive } from '$lib/opsNav';

	const icons = {
		'/dashboard': ChartPieOutline,
		'/transactions': PenOutline,
		'/integrations': LinkOutline,
		'/support': HeartOutline,
		'/profile': UserCircleSolid
	} as const;
</script>

<aside class="hidden w-56 shrink-0 border-r border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900 md:block">
	<nav class="flex flex-col gap-1 p-4">
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
</aside>

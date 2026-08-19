<script lang="ts">
	import { APP_VERSION } from '$lib/version';
	import { formatChangelogDate, formatChangelogInline } from '$lib/changelog';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head><title>Changelog — KasQ</title></svelte:head>

<div class="page-head">
	<h1>Changelog</h1>
	<p>Perubahan KasQ per versi. Versi yang dipakai sekarang v{APP_VERSION}.</p>
</div>

<div class="flex max-w-3xl flex-col gap-4">
	{#each data.releases as release (release.version)}
		{@const current = release.version === APP_VERSION}
		<section class="app-panel" id="v{release.version}">
			<div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
				<h2 class="text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-50">
					v{release.version}
				</h2>
				{#if release.date}
					<p class="text-sm text-slate-500 dark:text-slate-400">{formatChangelogDate(release.date)}</p>
				{/if}
				{#if current}
					<span class="rounded-full bg-primary-50 px-2 py-0.5 text-[11px] font-medium text-primary-800 dark:bg-primary-900/40 dark:text-primary-300">
						Versi ini
					</span>
				{/if}
			</div>
			{#if release.summary}
				<p class="mt-2 text-sm leading-relaxed text-slate-600 dark:text-slate-300">{release.summary}</p>
			{/if}
			{#each release.blocks as block, i (`${release.version}-${block.title}-${i}`)}
				<h3 class="mt-5 text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500">
					{block.title}
				</h3>
				{#if block.kind === 'list'}
					<ul class="mt-2 list-disc space-y-1.5 ps-5 text-sm leading-relaxed text-slate-700 dark:text-slate-300">
						{#each block.items as item}
							<li>{@html formatChangelogInline(item)}</li>
						{/each}
					</ul>
				{:else}
					<pre class="app-code mt-2">{block.code}</pre>
				{/if}
			{/each}
		</section>
	{/each}
</div>

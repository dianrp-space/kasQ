<script lang="ts">
	import defaultBrandImage from '$lib/assets/defaultBrand';
	import { appSettings, brandingUrl } from '$lib/appSettings.svelte';

	let {
		size = 'md',
		centered = false
	}: {
		size?: 'sm' | 'md' | 'lg';
		centered?: boolean;
	} = $props();

	const logoUrl = $derived(brandingUrl(appSettings.logo_url) ?? defaultBrandImage);
	const titleClass = $derived(
		size === 'lg' ? 'text-2xl' : size === 'sm' ? 'text-base' : 'text-lg'
	);
	const logoClass = $derived(
		size === 'lg' ? 'h-12' : size === 'sm' ? 'h-7' : 'h-9'
	);
	const taglineClass = $derived(
		size === 'lg'
			? 'text-sm'
			: size === 'sm'
				? 'text-[11px] leading-tight sm:text-xs md:text-sm'
				: 'text-sm'
	);
</script>

<div class="flex flex-col" class:items-center={centered}>
	<div class="flex items-center gap-2 sm:gap-3" class:justify-center={centered}>
		<img src={logoUrl} alt={appSettings.app_name} class="{logoClass} w-auto shrink-0 object-contain" />
		<div class="min-w-0" class:text-center={centered}>
			<h1 class="{titleClass} font-bold text-emerald-700 dark:text-emerald-400">{appSettings.app_name}</h1>
			{#if appSettings.app_tagline}
				<p class="{taglineClass} line-clamp-2 text-slate-500 dark:text-slate-400">{appSettings.app_tagline}</p>
			{/if}
		</div>
	</div>
</div>

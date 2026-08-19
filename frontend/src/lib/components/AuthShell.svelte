<script lang="ts">
	import type { Snippet } from 'svelte';
	import { loadAppSettings, appSettings } from '$lib/appSettings.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import VersionLink from '$lib/components/VersionLink.svelte';

	let {
		title,
		subtitle = '',
		children,
		footer
	}: {
		title: string;
		subtitle?: string;
		children: Snippet;
		footer?: Snippet;
	} = $props();

	$effect(() => {
		loadAppSettings();
	});
</script>

<div class="auth-shell">
	<div class="auth-grain" aria-hidden="true"></div>
	<div class="auth-frame">
		<aside class="auth-aside">
			<AppBrand size="lg" onDark asHeading={false} />
			<p class="auth-aside-lead">
				Catat pemasukan dan pengeluaran tim, lengkap dengan nota — dari web, WhatsApp, atau Telegram.
			</p>
			<ul class="auth-aside-list">
				<li>Saldo dan laporan per periode</li>
				<li>Nota foto tersimpan per transaksi</li>
				<li>Laporan kas bisa dibagikan lewat tautan</li>
			</ul>
			<VersionLink prefix={appSettings.app_name} class="auth-aside-meta" />
		</aside>

		<section class="auth-panel">
			<div class="auth-panel-top">
				<div class="md:hidden">
					<AppBrand size="sm" asHeading={false} />
					<VersionLink class="mt-1 block text-[10px] leading-none text-slate-400 dark:text-slate-500" />
				</div>
				<div class="ms-auto">
					<ThemeToggle class="border-slate-200/80 bg-white dark:border-slate-600 dark:bg-slate-800" />
				</div>
			</div>

			<div class="auth-panel-copy">
				<h1 class="auth-title">{title}</h1>
				{#if subtitle}
					<p class="auth-subtitle">{subtitle}</p>
				{/if}
			</div>

			<div class="auth-panel-body">
				{@render children()}
			</div>

			{#if footer}
				<div class="auth-footer">
					{@render footer()}
				</div>
			{/if}
		</section>
	</div>
</div>

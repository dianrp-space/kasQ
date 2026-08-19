<script lang="ts">
	import '../app.css';
	import defaultBrandImage from '$lib/assets/defaultBrand';
	import { api } from '$lib/api';
	import type { User } from '$lib/types';
	import Nav from '$lib/components/Nav.svelte';
	import OpsBottomNav from '$lib/components/OpsBottomNav.svelte';
	import OpsSidebar from '$lib/components/OpsSidebar.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import ToastHost from '$lib/components/ToastHost.svelte';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { appSettings, brandingUrl, loadAppSettings } from '$lib/appSettings.svelte';
	import { isOpsOnlyPath } from '$lib/roles';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { setContext } from 'svelte';

	let { children } = $props();
	let user = $state<User | null>(null);
	let loading = $state(true);

	const authPaths = ['/login', '/register', '/verify-email', '/forgot-password', '/reset-password'];
	const isPublic = $derived($page.url.pathname.startsWith('/report/'));
	const isAuthPage = $derived(authPaths.some((p) => $page.url.pathname.startsWith(p)));
	const isChangelog = $derived($page.url.pathname === '/changelog');

	const faviconHref = $derived(brandingUrl(appSettings.favicon_url) ?? defaultBrandImage);
	const pageTitle = $derived(`${appSettings.app_name}${appSettings.app_tagline ? ' — ' + appSettings.app_tagline.split('—')[0].trim() : ''}`);

	$effect(() => {
		loadAppSettings();
	});

	$effect(() => {
		if (isPublic || isAuthPage) {
			loading = false;
			return;
		}
		if (isChangelog) {
			api.me()
				.then((u) => {
					user = u;
					loading = false;
				})
				.catch(() => {
					user = null;
					loading = false;
				});
			return;
		}
		api.me()
			.then((u) => {
				user = u;
				loading = false;
				if (u.role === 'admin' && isOpsOnlyPath($page.url.pathname)) {
					goto('/admin');
				}
			})
			.catch(() => {
				goto('/login');
			});
	});

	async function refreshUser() {
		if (isPublic || isAuthPage) return;
		try {
			user = await api.me();
		} catch {
			if (isChangelog) {
				user = null;
				return;
			}
			goto('/login');
		}
	}

	setContext('refreshUser', refreshUser);

	$effect(() => {
		if (user?.role === 'admin' && isOpsOnlyPath($page.url.pathname)) {
			goto('/admin');
		}
	});

	async function logout() {
		await api.logout();
		goto('/login');
	}
</script>

<svelte:head>
	<link rel="icon" href={faviconHref} />
	<title>{pageTitle}</title>
</svelte:head>

{#if loading}
	<div class="flex min-h-screen items-center justify-center dark:bg-slate-900">
		<p class="text-slate-500 dark:text-slate-400">Memuat...</p>
	</div>
{:else if isPublic || isAuthPage}
	<div class="relative">
		{@render children()}
	</div>
	<ConfirmDialog />
	<ToastHost />
{:else if user}
	<div class="flex h-screen flex-col overflow-hidden dark:bg-slate-900">
		<Nav {user} onLogout={logout} />
		<div class="flex min-h-0 flex-1">
			{#if user.role === 'ops'}
				<OpsSidebar />
			{/if}
			<main class="min-h-0 flex-1 overflow-y-auto px-3 py-4 pb-[calc(5.5rem+env(safe-area-inset-bottom))] sm:px-4 md:py-6 md:pb-6">
				<div class="mx-auto max-w-6xl">
					{@render children()}
				</div>
			</main>
		</div>
		{#if user.role === 'ops'}
			<OpsBottomNav />
		{/if}
		<ConfirmDialog />
		<ToastHost />
	</div>
{:else if isChangelog}
	<div class="min-h-screen bg-slate-50 dark:bg-slate-900">
		<header class="border-b border-slate-200 bg-white px-4 py-3 dark:border-slate-700 dark:bg-slate-900">
			<div class="mx-auto flex max-w-3xl items-center justify-between gap-3">
				<a href="/login" class="min-w-0">
					<AppBrand size="sm" asHeading={false} />
				</a>
				<div class="flex items-center gap-3">
					<a href="/login" class="text-sm font-medium text-slate-500 hover:text-primary-700 dark:text-slate-400 dark:hover:text-primary-400">
						Masuk
					</a>
					<ThemeToggle class="shrink-0 bg-white shadow-sm dark:bg-slate-800" />
				</div>
			</div>
		</header>
		<main class="mx-auto max-w-3xl px-4 py-6">
			{@render children()}
		</main>
		<ConfirmDialog />
		<ToastHost />
	</div>
{/if}

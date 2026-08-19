<script lang="ts">
	import type { User } from '$lib/types';
	import AppBrand from '$lib/components/AppBrand.svelte';
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import UserMenu from '$lib/components/UserMenu.svelte';
	import VersionLink from '$lib/components/VersionLink.svelte';

	let { user, onLogout }: { user: User; onLogout: () => void } = $props();
</script>

<header class="sticky top-0 z-30 border-b border-slate-200 bg-white/95 backdrop-blur dark:border-slate-700 dark:bg-slate-900/95">
	<div class="flex items-center justify-between gap-2 px-3 py-2.5 sm:px-4 sm:py-3">
		<div class="flex min-w-0 shrink items-center gap-2">
			<a href={user.role === 'admin' ? '/admin' : '/dashboard'} class="min-w-0 shrink">
				<AppBrand size="sm" />
			</a>
			<VersionLink
				class="text-[10px] leading-none text-slate-400 dark:text-slate-500 {user.role === 'ops' ? 'md:hidden' : ''}"
			/>
		</div>
		<div class="flex items-center gap-1.5 sm:gap-2">
			<ThemeToggle />
			<UserMenu {user} {onLogout} />
		</div>
	</div>
</header>

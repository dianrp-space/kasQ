<script lang="ts">
	import { goto } from '$app/navigation';
	import { homePath } from '$lib/roles';
	import { onMount } from 'svelte';

	onMount(async () => {
		try {
			const me = await import('$lib/api').then((m) => m.api.me());
			goto(homePath(me.role));
		} catch {
			goto('/login');
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center dark:bg-slate-900">
	<p class="text-slate-500 dark:text-slate-400">Memuat KasQ...</p>
</div>

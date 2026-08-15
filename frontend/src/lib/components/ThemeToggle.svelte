<script lang="ts">
	import { DarkMode } from 'flowbite-svelte';

	let { class: className = '' }: { class?: string } = $props();
</script>

<DarkMode
	ariaLabel="Toggle tema gelap"
	size="md"
	class="border border-slate-200 dark:border-slate-600 {className}"
/>

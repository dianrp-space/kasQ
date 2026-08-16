<script lang="ts">
	import { Input, Label } from 'flowbite-svelte';

	let {
		value = $bindable(''),
		id = 'filter-month',
		class: className = 'w-full md:max-w-[180px]'
	}: {
		value?: string;
		id?: string;
		class?: string;
	} = $props();
</script>

<Label for={id} class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Bulan</Label>
<Input {id} type="month" bind:value class={className} />

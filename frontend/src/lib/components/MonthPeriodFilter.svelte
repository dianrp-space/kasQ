<script lang="ts">
	import { Input, Label } from 'flowbite-svelte';

	let {
		value = $bindable(''),
		id = 'filter-month',
		class: className = 'w-full md:max-w-[180px]',
		onchange
	}: {
		value?: string;
		id?: string;
		class?: string;
		onchange?: () => void;
	} = $props();

	function handleChange(e: Event) {
		const t = e.currentTarget;
		if (!(t instanceof HTMLInputElement)) return;
		if (t.value === value) return;
		value = t.value;
		onchange?.();
	}
</script>

<Label for={id} class="mb-1 block text-xs text-slate-500 dark:text-slate-400">Bulan</Label>
<Input {id} type="month" {value} class={className} oninput={handleChange} onchange={handleChange} />

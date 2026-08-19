<script lang="ts">
	import { Input } from 'flowbite-svelte';
	import { EyeOutline, EyeSlashOutline } from 'flowbite-svelte-icons';
	import type { HTMLInputAttributes } from 'svelte/elements';

	let {
		value = $bindable(''),
		id = '',
		placeholder = '',
		autocomplete,
		minlength,
		required = false,
		class: className = ''
	}: {
		value?: string;
		id?: string;
		placeholder?: string;
		autocomplete?: HTMLInputAttributes['autocomplete'];
		minlength?: number;
		required?: boolean;
		class?: string;
	} = $props();

	let show = $state(false);
</script>

<div class="relative">
	<Input
		{id}
		class="pr-11 {className}"
		type={show ? 'text' : 'password'}
		bind:value
		{placeholder}
		{autocomplete}
		{minlength}
		{required}
	/>
	<button
		type="button"
		class="absolute inset-y-0 end-0 flex w-11 items-center justify-center text-slate-400 transition-colors duration-200 ease-[cubic-bezier(0.32,0.72,0,1)] hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/60 dark:text-slate-500 dark:hover:text-slate-200"
		onclick={() => (show = !show)}
		aria-pressed={show}
		aria-label={show ? 'Sembunyikan password' : 'Tampilkan password'}
		title={show ? 'Sembunyikan password' : 'Tampilkan password'}
	>
		{#if show}
			<EyeSlashOutline class="h-5 w-5" />
		{:else}
			<EyeOutline class="h-5 w-5" />
		{/if}
	</button>
</div>

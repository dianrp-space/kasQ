<script lang="ts">
	import { Input } from 'flowbite-svelte';

	let {
		value = $bindable(''),
		id = '',
		placeholder = '',
		autocomplete = '',
		minlength,
		required = false,
		class: className = ''
	}: {
		value?: string;
		id?: string;
		placeholder?: string;
		autocomplete?: string;
		minlength?: number;
		required?: boolean;
		class?: string;
	} = $props();

	let show = $state(false);
</script>

<div class="relative">
	<Input
		{id}
		class="pr-10 {className}"
		type={show ? 'text' : 'password'}
		bind:value
		{placeholder}
		{autocomplete}
		{minlength}
		{required}
	/>
	<button
		type="button"
		class="absolute inset-y-0 right-2 flex items-center px-1 text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
		onclick={() => (show = !show)}
		title={show ? 'Sembunyikan password' : 'Tampilkan password'}
		aria-label={show ? 'Sembunyikan password' : 'Tampilkan password'}
	>
		{show ? '🙈' : '👁'}
	</button>
</div>

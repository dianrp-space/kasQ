<script lang="ts">
	import { Button } from 'flowbite-svelte';
	import { ArrowDownToBracketOutline } from 'flowbite-svelte-icons';

	let {
		disabled = false,
		busyFormat = null,
		onExport
	}: {
		disabled?: boolean;
		busyFormat?: 'xlsx' | 'pdf' | null;
		onExport: (format: 'xlsx' | 'pdf') => void;
	} = $props();

	const busy = $derived(busyFormat != null);
</script>

<div class="flex flex-wrap items-center gap-2">
	<Button color="light" size="sm" disabled={disabled || busy} onclick={() => onExport('xlsx')}>
		<ArrowDownToBracketOutline class="me-1.5 h-4 w-4" />
		{busyFormat === 'xlsx' ? 'Mengunduh...' : 'Excel'}
	</Button>
	<Button color="light" size="sm" disabled={disabled || busy} onclick={() => onExport('pdf')}>
		<ArrowDownToBracketOutline class="me-1.5 h-4 w-4" />
		{busyFormat === 'pdf' ? 'Mengunduh...' : 'PDF'}
	</Button>
</div>

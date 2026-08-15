<script lang="ts">
	import { Button, Modal } from 'flowbite-svelte';
	import { ArrowDownToBracketOutline } from 'flowbite-svelte-icons';

	let {
		open = $bindable(false),
		src = '',
		title = 'Preview Nota',
		onDownload
	}: {
		open: boolean;
		src: string;
		title?: string;
		onDownload?: () => void;
	} = $props();
</script>

<Modal bind:open {title} size="xl" class="z-50">
	<div class="overflow-auto bg-slate-100 p-4">
		{#if src}
			<img src={src} alt="Nota" class="mx-auto max-h-[75vh] w-auto max-w-full object-contain" />
		{/if}
	</div>
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			{#if onDownload}
				<Button color="light" onclick={onDownload}>
					<ArrowDownToBracketOutline class="me-2 h-4 w-4" />
					Download
				</Button>
			{/if}
			<Button color="alternative" onclick={() => (open = false)}>Tutup</Button>
		</div>
	{/snippet}
</Modal>

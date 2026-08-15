<script lang="ts">
	import { Button, Modal } from 'flowbite-svelte';
	import { confirmState, resolveConfirm } from '$lib/confirm.svelte';
</script>

<Modal bind:open={confirmState.open} title={confirmState.title} dismissable={false} size="sm">
	<p class="text-sm leading-relaxed text-slate-600">{confirmState.message}</p>
	{#snippet footer()}
		<div class="flex w-full justify-end gap-2">
			<Button color="alternative" onclick={() => resolveConfirm(false)}>
				{confirmState.cancelLabel}
			</Button>
			<Button color={confirmState.color} onclick={() => resolveConfirm(true)}>
				{confirmState.confirmLabel}
			</Button>
		</div>
	{/snippet}
</Modal>

<script lang="ts">
	import { Toast, ToastContainer } from 'flowbite-svelte';
	import { CheckCircleSolid, ExclamationCircleSolid, InfoCircleSolid } from 'flowbite-svelte-icons';
	import { dismiss, toastState, type ToastColor } from '$lib/toast.svelte';

	function toastColor(color: ToastColor) {
		if (color === 'green') return 'green';
		if (color === 'red') return 'red';
		if (color === 'yellow') return 'yellow';
		return 'blue';
	}
</script>

<ToastContainer position="top-right" class="z-[100] space-y-2">
	{#each toastState.items as item (item.id)}
		<Toast color={toastColor(item.color)} dismissable onclick={() => dismiss(item.id)}>
			{#snippet icon()}
				{#if item.color === 'green'}
					<CheckCircleSolid class="h-5 w-5" />
				{:else if item.color === 'red'}
					<ExclamationCircleSolid class="h-5 w-5" />
				{:else if item.color === 'yellow'}
					<ExclamationCircleSolid class="h-5 w-5" />
				{:else}
					<InfoCircleSolid class="h-5 w-5" />
				{/if}
			{/snippet}
			{item.message}
		</Toast>
	{/each}
</ToastContainer>

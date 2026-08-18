<script lang="ts">
	import { Button, Modal } from 'flowbite-svelte';
	import { ArrowDownToBracketOutline, AngleLeftOutline, AngleRightOutline } from 'flowbite-svelte-icons';

	let {
		open = $bindable(false),
		srcs = [],
		title = 'Preview Nota',
		onDownload,
		onDownloadAll
	}: {
		open: boolean;
		srcs: string[];
		title?: string;
		onDownload?: (index: number) => void;
		onDownloadAll?: () => void;
	} = $props();

	let index = $state(0);
	const total = $derived(srcs.length);
	const current = $derived(total > 0 ? srcs[Math.min(index, total - 1)] : '');

	$effect(() => {
		if (open) index = 0;
	});

	function prev() {
		if (total === 0) return;
		index = (index - 1 + total) % total;
	}

	function next() {
		if (total === 0) return;
		index = (index + 1) % total;
	}

	function onKey(e: KeyboardEvent) {
		if (!open || total < 2) return;
		if (e.key === 'ArrowLeft') prev();
		if (e.key === 'ArrowRight') next();
	}
</script>

<svelte:window onkeydown={onKey} />

<Modal bind:open {title} size="xl" class="z-50">
	<div class="relative overflow-auto bg-slate-100 p-4 dark:bg-slate-800">
		{#if current}
			<img src={current} alt="Nota {index + 1}" class="mx-auto max-h-[70vh] w-auto max-w-full object-contain" />
		{:else}
			<p class="py-10 text-center text-sm text-slate-400">Tidak ada foto</p>
		{/if}
		{#if total > 1}
			<button
				type="button"
				class="absolute left-3 top-1/2 -translate-y-1/2 rounded-full bg-white/90 p-2 text-slate-700 shadow dark:bg-slate-700 dark:text-slate-100"
				onclick={prev}
				aria-label="Sebelumnya"
			>
				<AngleLeftOutline class="h-5 w-5" />
			</button>
			<button
				type="button"
				class="absolute right-3 top-1/2 -translate-y-1/2 rounded-full bg-white/90 p-2 text-slate-700 shadow dark:bg-slate-700 dark:text-slate-100"
				onclick={next}
				aria-label="Berikutnya"
			>
				<AngleRightOutline class="h-5 w-5" />
			</button>
			<p class="mt-3 text-center text-xs text-slate-500 dark:text-slate-400">{index + 1} / {total}</p>
			<div class="mt-2 flex justify-center gap-1.5">
				{#each srcs as _, i}
					<button
						type="button"
						class="h-2 w-2 rounded-full {i === index ? 'bg-primary-600' : 'bg-slate-300 dark:bg-slate-500'}"
						onclick={() => (index = i)}
						aria-label="Nota {i + 1}"
					></button>
				{/each}
			</div>
		{/if}
	</div>
	{#snippet footer()}
		<div class="flex w-full flex-wrap justify-end gap-2">
			{#if onDownloadAll && total > 1}
				<Button color="light" onclick={onDownloadAll}>
					<ArrowDownToBracketOutline class="me-2 h-4 w-4" />
					Download ZIP ({total})
				</Button>
			{/if}
			{#if onDownload && current}
				<Button color="light" onclick={() => onDownload(index)}>
					<ArrowDownToBracketOutline class="me-2 h-4 w-4" />
					{total > 1 ? 'Download foto ini' : 'Download'}
				</Button>
			{/if}
			<Button color="alternative" onclick={() => (open = false)}>Tutup</Button>
		</div>
	{/snippet}
</Modal>

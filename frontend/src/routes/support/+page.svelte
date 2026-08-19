<script lang="ts">
	import { appSettings } from '$lib/appSettings.svelte';
	import { Button } from 'flowbite-svelte';
	import { ArrowUpRightFromSquareOutline } from 'flowbite-svelte-icons';

	const supportUrl = 'https://sawer.dianrp.com';
	const ownerUrl = 'https://dianrp.com';
</script>

<svelte:head><title>Support — KasQ</title></svelte:head>

<div class="page-head">
	<h1>Support</h1>
	<p>Dukung pengembangan {appSettings.app_name}, atau hubungi pembuatnya.</p>
</div>

<div class="flex max-w-2xl flex-col gap-5">
	<section class="app-panel">
		<p class="text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500">Sawer</p>
		<h2 class="mt-1 text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-50">
			Dukung {appSettings.app_name}
		</h2>
		<p class="mt-2 max-w-prose text-sm leading-relaxed text-slate-600 dark:text-slate-300">
			Kalau pencatatan kas tim ini membantu, dukungan lewat Sawer dipakai untuk perbaikan dan fitur berikutnya.
		</p>
		<Button href={supportUrl} target="_blank" rel="noopener noreferrer" class="auth-submit mt-5">
			Buka Sawer
			<ArrowUpRightFromSquareOutline class="ms-2 h-4 w-4" />
		</Button>
	</section>

	<section class="app-panel">
		<p class="text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500">dianrp.com</p>
		<h2 class="mt-1 text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-50">
			Website, aplikasi, otomasi
		</h2>
		<p class="mt-2 max-w-prose text-sm leading-relaxed text-slate-600 dark:text-slate-300">
			{appSettings.app_name} dibuat di dianrp.com. Konsultasi untuk UMKM sampai perusahaan — proses transparan, hasil yang bisa dipakai.
		</p>
		<Button
			href={ownerUrl}
			target="_blank"
			rel="noopener noreferrer"
			color="light"
			class="mt-5 w-full sm:w-auto"
		>
			Kunjungi dianrp.com
			<ArrowUpRightFromSquareOutline class="ms-2 h-4 w-4" />
		</Button>
	</section>
</div>

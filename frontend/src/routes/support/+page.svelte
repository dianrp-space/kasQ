<script lang="ts">
	import { appSettings } from '$lib/appSettings.svelte';
	import { Button, Card, Heading } from 'flowbite-svelte';
	import { ArrowUpRightFromSquareOutline, GlobeOutline, HeartOutline } from 'flowbite-svelte-icons';

	const supportUrl = 'https://sawer.dianrp.com';
	const ownerUrl = 'https://dianrp.com';
</script>

<svelte:head><title>Support — KasQ</title></svelte:head>

<Heading tag="h1" class="mb-4 text-2xl">Support</Heading>

<div class="flex max-w-2xl flex-col gap-4">
	<Card size="lg" shadow="sm" class="p-4 sm:p-6">
		<div class="flex flex-col items-center text-center sm:items-start sm:text-left">
			<div
				class="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/40 dark:text-primary-400"
			>
				<HeartOutline class="h-7 w-7" />
			</div>
			<Heading tag="h2" class="mb-2 text-lg">Terima kasih sudah pakai {appSettings.app_name}</Heading>
			<p class="mb-1 text-sm text-slate-600 dark:text-slate-300">
				Kalau aplikasi ini bermanfaat untuk pencatatan keuangan tim kamu, kamu bisa support pengembang
				agar {appSettings.app_name} terus dikembangkan dan diperbaiki.
			</p>
			<p class="mb-6 text-sm text-slate-500 dark:text-slate-400">
				Dukungan kamu sangat berarti — terima kasih!
			</p>
			<Button href={supportUrl} target="_blank" rel="noopener noreferrer" color="primary" class="w-full sm:w-auto">
				Support via Sawer
			</Button>
		</div>
	</Card>

	<Card size="lg" shadow="sm" class="p-4 sm:p-6">
		<div class="flex flex-col items-center text-center sm:items-start sm:text-left">
			<div
				class="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-300"
			>
				<GlobeOutline class="h-7 w-7" />
			</div>
			<p class="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-400 dark:text-slate-500">
				Website owner
			</p>
			<Heading tag="h2" class="mb-2 text-lg">Butuh solusi digital untuk bisnis kamu?</Heading>
			<p class="mb-1 text-sm text-slate-600 dark:text-slate-300">
				{appSettings.app_name} dibuat oleh tim di dianrp.com — spesialis website, aplikasi, dan otomasi
				yang bantu UMKM sampai perusahaan tumbuh lebih cepat tanpa ribet.
			</p>
			<p class="mb-6 text-sm text-slate-500 dark:text-slate-400">
				Konsultasi gratis, proses transparan, hasil yang bisa langsung dipakai. Yuk mampir dan lihat
				projek-projek kami!
			</p>
			<Button
				href={ownerUrl}
				target="_blank"
				rel="noopener noreferrer"
				color="light"
				class="w-full sm:w-auto"
			>
				Kunjungi dianrp.com
				<ArrowUpRightFromSquareOutline class="ms-2 h-4 w-4" />
			</Button>
		</div>
	</Card>
</div>

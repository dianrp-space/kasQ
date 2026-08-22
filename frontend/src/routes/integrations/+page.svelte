<script lang="ts">
	import { api } from '$lib/api';
	import type { Integration, Team, User } from '$lib/types';
	import { confirm } from '$lib/confirm.svelte';
	import { toast } from '$lib/toast.svelte';
	import { goto } from '$app/navigation';
	import { Alert, Badge, Button, Input, Label, Textarea } from 'flowbite-svelte';
	import PasswordInput from '$lib/components/PasswordInput.svelte';

	let user = $state<User | null>(null);
	let teams = $state<Team[]>([]);
	let selectedTeam = $state('');
	let integration = $state<Integration | null>(null);
	let teleToken = $state('');
	let teleChatId = $state('');
	let teleMode = $state<'system' | 'custom'>('system');
	let teleBotName = $state('');
	let teleBotUsername = $state('');
	let telePictureUrl = $state('');
	let waQR = $state('');
	let waStatus = $state('');
	let waPhone = $state('');
	let waName = $state('');
	let waPictureUrl = $state('');
	let waLoginTab = $state<'qr' | 'pair'>('qr');
	let waPairPhone = $state('');
	let waPairCode = $state('');
	let waQrTimeout = $state(20);
	let waPairExpires = $state(0);
	let waAllowedPhonesText = $state('');
	let error = $state('');
	let success = $state('');
	let reportSlug = $state('');
	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let teleAvatarObjectUrl: string | null = null;
	let waAvatarObjectUrl: string | null = null;

	function revokeTeleAvatar() {
		if (teleAvatarObjectUrl) {
			URL.revokeObjectURL(teleAvatarObjectUrl);
			teleAvatarObjectUrl = null;
		}
	}

	async function loadTeleAvatar(teamId: string, hasAvatar: boolean) {
		revokeTeleAvatar();
		telePictureUrl = '';
		if (!hasAvatar) return;
		const url = await api.getTeleBotAvatar(teamId);
		if (url) {
			teleAvatarObjectUrl = url;
			telePictureUrl = url;
		}
	}

	function revokeWaAvatar() {
		if (waAvatarObjectUrl) {
			URL.revokeObjectURL(waAvatarObjectUrl);
			waAvatarObjectUrl = null;
		}
	}

	async function loadWaAvatar(teamId: string) {
		revokeWaAvatar();
		waPictureUrl = '';
		const url = await api.getWABotAvatar(teamId);
		if (url) {
			waAvatarObjectUrl = url;
			waPictureUrl = url;
		}
	}

	const reportPreviewUrl = $derived(
		integration?.report_url
			? integration.report_url.replace(/\/[^/]+$/, '/' + (reportSlug.trim() || integration.report_token || ''))
			: ''
	);

	const waAlreadyOn = $derived(!!integration?.wa_enabled);
	const teleAlreadyOn = $derived(!!integration?.tele_enabled);
	const teleSettingsUnchanged = $derived.by(() => {
		if (!integration?.tele_enabled) return false;
		const savedMode = integration.tele_use_system_bot ? 'system' : 'custom';
		if (teleMode !== savedMode) return false;
		const savedChat =
			integration.tele_allowed_chat_id != null ? String(integration.tele_allowed_chat_id) : '';
		if (teleChatId.trim() !== savedChat) return false;
		if (teleMode === 'custom') {
			const savedToken = integration.tele_bot_token ?? '';
			if (teleToken.trim() && teleToken.trim() !== savedToken) return false;
		}
		return true;
	});
	const teleAktifkanDisabled = $derived(
		(teleMode === 'system' && !integration?.system_tele_bot_available) || teleSettingsUnchanged
	);

	async function load() {
		user = await api.me();
		if (user.role === 'admin') {
			goto('/admin');
			return;
		}
		teams = await api.getTeams();
		if (!selectedTeam && teams.length > 0) {
			selectedTeam = user.team_id || teams[0].id;
		}
		if (selectedTeam) await loadIntegration();
	}

	async function loadIntegration() {
		integration = await api.getIntegration(selectedTeam);
		waStatus = integration.wa_status;
		waPhone = integration.wa_phone ?? '';
		waName = integration.wa_name ?? '';
		waAllowedPhonesText = (integration.wa_allowed_phones ?? []).join('\n');
		waPictureUrl = '';
		if (integration.wa_enabled && integration.wa_status === 'connected') {
			await loadWaAvatar(selectedTeam);
		} else {
			revokeWaAvatar();
		}
		teleToken = integration.tele_bot_token ?? '';
		teleChatId =
			integration.tele_allowed_chat_id != null ? String(integration.tele_allowed_chat_id) : '';
		if (integration.tele_use_system_bot) {
			teleMode = 'system';
		} else if (integration.has_tele_token) {
			teleMode = 'custom';
		} else if (integration.system_tele_bot_available) {
			teleMode = 'system';
		} else {
			teleMode = 'custom';
		}
		teleBotName = integration.tele_bot_name ?? '';
		teleBotUsername = integration.tele_bot_username ?? '';
		if (integration.tele_enabled && integration.tele_bot_has_avatar) {
			await loadTeleAvatar(selectedTeam, true);
		} else {
			revokeTeleAvatar();
			telePictureUrl = '';
		}
		reportSlug = integration.report_token ?? integration.team_slug ?? '';
		if (integration.wa_enabled && waStatus === 'qr') {
			startQRPoll();
		}
		if (integration.wa_enabled && (waStatus === 'pair_code' || waStatus === 'connecting' || waStatus === 'awaiting_login')) {
			startQRPoll();
		}
	}

	function applyWAStatus(qr: Awaited<ReturnType<typeof api.getWAQR>>) {
		waStatus = qr.status;
		waQR = qr.qr;
		if (qr.phone) waPhone = qr.phone;
		waName = qr.wa_name ?? waName;
		if (qr.status === 'connected') {
			void loadWaAvatar(selectedTeam);
		}
		waPairCode = qr.pair_code ?? '';
		waQrTimeout = qr.qr_timeout_seconds ?? 20;
		waPairExpires = qr.pair_code_expires_seconds ?? 0;
		if (qr.login_mode === 'pair') waLoginTab = 'pair';
		else if (qr.login_mode === 'qr') waLoginTab = 'qr';
	}

	function startQRPoll() {
		stopQRPoll();
		pollInterval = setInterval(async () => {
			try {
				const qr = await api.getWAQR(selectedTeam);
				applyWAStatus(qr);
				if (qr.status === 'connected') {
					stopQRPoll();
					await loadIntegration();
				}
			} catch {}
		}, 3000);
	}

	async function beginQRLogin() {
		error = '';
		try {
			await api.startWAQRLogin(selectedTeam);
			startQRPoll();
			const qr = await api.getWAQR(selectedTeam);
			applyWAStatus(qr);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal mulai QR';
		}
	}

	async function beginPairLogin() {
		error = '';
		success = '';
		if (!waPairPhone.trim()) {
			error = 'Nomor WhatsApp wajib diisi';
			return;
		}
		try {
			const result = await api.startWAPairLogin(selectedTeam, waPairPhone.trim());
			waPairCode = result.pair_code;
			waPairExpires = result.expires_seconds;
			waStatus = result.status;
			success = 'Kode pairing dibuat — masukkan di HP dalam 60 detik';
			startQRPoll();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal buat kode pairing';
		}
	}

	function stopQRPoll() {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
	}

	async function saveWAAllowedPhones() {
		error = '';
		success = '';
		try {
			const phones = waAllowedPhonesText
				.split(/[\n,;]+/)
				.map((s) => s.trim())
				.filter(Boolean);
			const result = await api.updateWAAllowedPhones(selectedTeam, phones);
			waAllowedPhonesText = (result.phones ?? []).join('\n');
			success = phones.length
				? `Daftar ${result.phones.length} nomor WA disimpan — hanya nomor ini yang dibalas bot.`
				: 'Filter nomor dinonaktifkan — semua nomor boleh chat.';
			await loadIntegration();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal simpan daftar nomor';
		}
	}

	async function toggleWA(enabled: boolean) {
		if (enabled === waAlreadyOn) return;
		error = '';
		try {
			await api.updateWA(selectedTeam, enabled);
			if (enabled) {
				await loadIntegration();
				if (waLoginTab === 'qr') {
					await beginQRLogin();
				} else {
					startQRPoll();
				}
			} else {
				stopQRPoll();
				waQR = '';
				waPhone = '';
				waName = '';
				revokeWaAvatar();
				waPictureUrl = '';
				waPairCode = '';
				await loadIntegration();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal';
		}
	}

	async function saveTele(enabled: boolean) {
		if (enabled && teleSettingsUnchanged) return;
		if (!enabled && !teleAlreadyOn) return;
		error = '';
		success = '';
		try {
			const useSystem = teleMode === 'system';
			if (enabled && !teleChatId.trim()) {
				error = 'Chat ID wajib diisi. Kirim /start ke bot, lalu tempel angkanya di sini.';
				return;
			}
			if (enabled && !useSystem && !teleToken.trim() && !integration?.has_tele_token) {
				error = 'Bot Token wajib diisi untuk bot sendiri.';
				return;
			}

			const payload: {
				enabled: boolean;
				use_system_bot: boolean;
				bot_token?: string;
				chat_id?: number | null;
			} = { enabled, use_system_bot: useSystem };

			if (!useSystem && teleToken.trim()) {
				payload.bot_token = teleToken.trim();
			}
			if (teleChatId.trim()) {
				const n = parseInt(teleChatId.trim(), 10);
				if (Number.isNaN(n)) {
					error = 'Chat ID tidak valid. Harus angka, contoh 123456789.';
					return;
				}
				payload.chat_id = n;
			} else if (enabled) {
				payload.chat_id = null;
			}

			await api.updateTele(selectedTeam, payload);
			success = enabled ? 'Telegram diaktifkan' : 'Telegram dinonaktifkan';
			if (!enabled) {
				revokeTeleAvatar();
				teleBotName = '';
				teleBotUsername = '';
				telePictureUrl = '';
			}
			await loadIntegration();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal';
		}
	}

	async function saveReportSlug() {
		error = '';
		success = '';
		const slug = reportSlug.trim();
		if (!slug) {
			error = 'Slug link laporan wajib diisi';
			return;
		}
		try {
			const result = await api.updateReportToken(selectedTeam, slug);
			success = 'Link laporan disimpan';
			reportSlug = result.token;
			await loadIntegration();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal simpan link';
		}
	}

	async function resetReportSlug() {
		const ok = await confirm({
			title: 'Reset link laporan?',
			message: 'Reset link ke default (slug nama tim/kas)? Link lama tidak akan valid lagi.',
			confirmLabel: 'Reset',
			color: 'red'
		});
		if (!ok) return;
		error = '';
		success = '';
		try {
			const result = await api.resetReportToken(selectedTeam);
			success = 'Link direset: ' + result.report_url;
			reportSlug = result.token;
			toast.success('Link laporan direset');
			await loadIntegration();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal reset link';
		}
	}

	function copyLink() {
		if (reportPreviewUrl) {
			navigator.clipboard.writeText(reportPreviewUrl);
			success = 'Link disalin';
			toast.success('Link laporan disalin');
		}
	}

	$effect(() => {
		load();
		return () => {
			stopQRPoll();
			revokeTeleAvatar();
			revokeWaAvatar();
		};
	});
	$effect(() => {
		if (selectedTeam) loadIntegration();
	});
</script>

<svelte:head><title>Integrasi — KasQ</title></svelte:head>

<div class="page-head">
	<h1>Integrasi</h1>
	<p>Hubungkan WhatsApp, Telegram, dan tautan laporan publik.</p>
</div>

{#if error}<Alert color="red" class="mb-4">{error}</Alert>{/if}
{#if success}<Alert color="green" class="mb-4">{success}</Alert>{/if}

<div class="grid gap-5 lg:grid-cols-2">
	<section class="app-panel">
		<div class="mb-4 flex items-center justify-between gap-3">
			<h2 class="text-base font-semibold tracking-tight text-slate-900 dark:text-slate-50">WhatsApp</h2>
			<Badge color={integration?.wa_enabled ? 'green' : 'gray'}>
				{integration?.wa_enabled ? waStatus : 'OFF'}
			</Badge>
		</div>
		<p class="mb-4 text-sm leading-relaxed text-slate-500 dark:text-slate-400">
			Hubungkan akun WA via QR atau kode pairing. Boleh nomor bot khusus, atau nomor pribadi: chat ke
			<strong class="font-medium text-slate-700 dark:text-slate-200">diri sendiri</strong>
			lalu whitelist nomor itu agar chat dari orang lain tidak dibalas.
		</p>
		{#if integration?.wa_enabled && waStatus === 'connected' && (waPhone || waName)}
			<div class="mb-4 flex items-center gap-3 rounded-xl border border-emerald-200/80 bg-emerald-50 px-3 py-3 text-sm text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-100">
				{#if waPictureUrl}
					<img
						src={waPictureUrl}
						alt=""
						class="h-12 w-12 shrink-0 rounded-full border border-emerald-200 bg-white object-cover"
					/>
				{:else}
					<div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-emerald-200 text-lg font-semibold text-emerald-800">
						{(waName || waPhone || '?').slice(0, 1).toUpperCase()}
					</div>
				{/if}
				<div class="min-w-0">
					<p class="font-medium">Akun terhubung</p>
					{#if waName}
						<p class="truncate">{waName}</p>
					{/if}
					{#if waPhone}
						<p class="font-mono text-emerald-800 dark:text-emerald-300">{waPhone}</p>
					{/if}
				</div>
			</div>
		{/if}
		{#if integration?.wa_enabled && waStatus !== 'connected'}
			<div class="app-seg mb-4">
				<button type="button" aria-pressed={waLoginTab === 'qr'} onclick={() => (waLoginTab = 'qr')}>
					QR Code
				</button>
				<button type="button" aria-pressed={waLoginTab === 'pair'} onclick={() => (waLoginTab = 'pair')}>
					Kode pairing
				</button>
			</div>

			{#if waLoginTab === 'qr'}
				{#if waStatus === 'awaiting_login' || waStatus === 'disconnected'}
					<Button color="light" class="mb-3" onclick={beginQRLogin}>Tampilkan QR</Button>
				{/if}
				{#if waQR && waStatus !== 'connected'}
					<div class="rounded-xl border border-slate-200 bg-white p-4 text-center dark:border-slate-700 dark:bg-slate-900">
						<p class="mb-2 text-sm text-slate-500 dark:text-slate-400">Scan di WhatsApp → Perangkat tertaut → Tautkan perangkat</p>
						<p class="mb-3 text-xs text-amber-800 dark:text-amber-200">
							QR berganti otomatis setiap <strong>{waQrTimeout}</strong> detik (QR pertama ~60 detik).
							Sesi pairing ~2,5 menit sebelum perlu muat ulang.
						</p>
						{#if waQR.startsWith('data:image')}
							<img src={waQR} alt="WhatsApp QR" class="mx-auto h-64 w-64" />
						{:else}
							<p class="text-sm text-slate-400">Menunggu QR…</p>
						{/if}
					</div>
				{/if}
			{:else}
				<div class="space-y-3">
					<div class="auth-field">
						<Label for="waPairPhone" class="text-sm font-medium text-slate-700 dark:text-slate-300">Nomor WhatsApp</Label>
						<Input id="waPairPhone" class="font-mono" placeholder="62812xxxxxxx" bind:value={waPairPhone} />
						<p class="auth-hint">Format internasional, tanpa + dan tanpa 0 di depan.</p>
					</div>
					<Button color="light" onclick={beginPairLogin}>Dapatkan kode</Button>
					{#if waPairCode}
						<div class="rounded-xl border border-sky-200 bg-sky-50 p-4 text-center dark:border-sky-800 dark:bg-sky-950/40">
							<p class="mb-1 text-sm text-slate-600 dark:text-slate-300">Masukkan kode di HP</p>
							<p class="text-2xl font-semibold tracking-[0.2em] text-sky-900 dark:text-sky-100">{waPairCode}</p>
							<p class="mt-2 text-xs text-slate-500 dark:text-slate-400">
								WhatsApp → Perangkat tertaut → Tautkan perangkat → Tautkan dengan nomor telepon
							</p>
							{#if waPairExpires > 0}
								<p class="mt-2 text-xs text-amber-800 dark:text-amber-200">Kode kedaluwarsa dalam ~{waPairExpires} detik</p>
							{:else}
								<p class="mt-2 text-xs text-red-600 dark:text-red-400">Kode kedaluwarsa — klik Dapatkan kode lagi</p>
							{/if}
						</div>
					{/if}
				</div>
			{/if}
		{/if}
		<div class="mb-4 rounded-xl border border-slate-200 p-4 dark:border-slate-700">
			<h3 class="mb-1 text-sm font-semibold text-slate-900 dark:text-slate-50">Nomor yang boleh chat</h3>
			<p class="mb-3 text-sm leading-relaxed text-slate-500 dark:text-slate-400">
				Kosongkan daftar = semua nomor dibalas. Isi daftar = hanya nomor terdaftar yang dibalas.
				Kalau bot memakai nomor pribadi, masukkan nomor itu supaya hanya self-chat yang dijawab.
			</p>
			<div class="auth-field">
				<Label for="waAllowedPhones" class="text-sm font-medium text-slate-700 dark:text-slate-300">Nomor WA</Label>
				<Textarea
					id="waAllowedPhones"
					class="mt-1 font-mono text-sm"
					rows={4}
					placeholder={'628111111111\n628222222222'}
					bind:value={waAllowedPhonesText}
				/>
				<p class="auth-hint">Satu per baris. Boleh 62812… atau 0812…. Maks. 50 nomor.</p>
			</div>
			<Button color="light" class="mt-3" onclick={saveWAAllowedPhones}>Simpan daftar nomor</Button>
		</div>
		<div class="flex flex-col gap-2 sm:flex-row">
			<Button onclick={() => toggleWA(true)} disabled={waAlreadyOn}>Aktifkan WA</Button>
			<Button color="light" onclick={() => toggleWA(false)} disabled={!waAlreadyOn}>Nonaktifkan</Button>
		</div>
	</section>

	<section class="app-panel">
		<div class="mb-4 flex items-center justify-between gap-3">
			<h2 class="text-base font-semibold tracking-tight text-slate-900 dark:text-slate-50">Telegram</h2>
			<Badge color={integration?.tele_enabled ? 'blue' : 'gray'}>
				{integration?.tele_enabled ? 'ON' : 'OFF'}
			</Badge>
		</div>
		{#if integration?.tele_enabled && (teleBotName || teleBotUsername || telePictureUrl)}
			<div class="mb-4 flex items-center gap-3 rounded-xl border border-sky-200/80 bg-sky-50 p-3 text-sm text-sky-900 dark:border-sky-800 dark:bg-sky-950/50 dark:text-sky-100">
				{#if telePictureUrl}
					<img
						src={telePictureUrl}
						alt=""
						class="h-12 w-12 shrink-0 rounded-full border border-sky-200 bg-white object-cover"
					/>
				{:else}
					<div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-sky-200 text-lg font-semibold text-sky-800 dark:bg-sky-800 dark:text-sky-100">
						{(teleBotName || teleBotUsername || '?').slice(0, 1).toUpperCase()}
					</div>
				{/if}
				<div class="min-w-0">
					<p class="font-medium">{integration.tele_use_system_bot ? 'Bot KasQ aktif' : 'Bot sendiri aktif'}</p>
					{#if teleBotName}
						<p class="truncate">{teleBotName}</p>
					{/if}
					{#if teleBotUsername}
						<p class="font-mono text-sky-800 dark:text-sky-300">@{teleBotUsername}</p>
					{/if}
				</div>
			</div>
		{/if}
		<div class="app-seg app-seg-tele mb-4" role="group" aria-label="Pilih jenis bot Telegram">
			<button type="button" aria-pressed={teleMode === 'system'} onclick={() => (teleMode = 'system')}>
				Bot KasQ
				<span class="seg-sub">Bot resmi aplikasi</span>
			</button>
			<button type="button" aria-pressed={teleMode === 'custom'} onclick={() => (teleMode = 'custom')}>
				Bot sendiri
				<span class="seg-sub">Token dari BotFather</span>
			</button>
		</div>
		<div class="space-y-3">
			{#if teleMode === 'system'}
				{#if integration?.system_tele_bot_available}
					<div class="rounded-xl border border-slate-200 bg-slate-50 p-3 text-sm dark:border-slate-700 dark:bg-slate-800/80">
						<p class="font-medium text-slate-800 dark:text-slate-100">
							{integration.system_tele_bot_name || 'Bot KasQ'}
							{#if integration.system_tele_bot_username}
								<span class="font-mono text-sky-700 dark:text-sky-400">@{integration.system_tele_bot_username}</span>
							{/if}
						</p>
						<ol class="mt-2 list-decimal space-y-1 pl-4 text-xs text-slate-600 dark:text-slate-400">
							<li>
								{#if integration.system_tele_bot_username}
									Buka
									<a
										class="font-medium text-sky-700 underline dark:text-sky-400"
										href="https://t.me/{integration.system_tele_bot_username}"
										target="_blank"
										rel="noreferrer"
									>
										@{integration.system_tele_bot_username}
									</a>
									di Telegram
								{:else}
									Buka bot KasQ di Telegram
								{/if}
							</li>
							<li>Kirim <strong>/start</strong> — bot membalas Chat ID</li>
							<li>Tempel Chat ID di bawah, lalu aktifkan</li>
						</ol>
					</div>
					<div class="auth-field">
						<Label for="teleChatId" class="text-sm font-medium text-slate-700 dark:text-slate-300">Chat ID</Label>
						<Input
							id="teleChatId"
							class="font-mono text-sm"
							placeholder="Contoh: 123456789"
							bind:value={teleChatId}
						/>
						<p class="auth-hint">Wajib. Hanya chat pribadi dengan bot KasQ. Satu Chat ID untuk satu tim/kas.</p>
					</div>
				{:else}
					<p class="rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
						Bot KasQ belum dikonfigurasi di server. Gunakan tab Bot sendiri, atau minta admin mengisi
						<code class="font-mono">TELEGRAM_BOT_TOKEN</code> di environment.
					</p>
				{/if}
			{:else}
				<div class="auth-field">
					<Label for="teleToken" class="text-sm font-medium text-slate-700 dark:text-slate-300">Bot token</Label>
					<PasswordInput
						id="teleToken"
						placeholder="Token dari @BotFather"
						bind:value={teleToken}
						autocomplete="off"
					/>
				</div>
				<div class="auth-field">
					<Label for="teleChatIdCustom" class="text-sm font-medium text-slate-700 dark:text-slate-300">Chat ID grup atau pribadi</Label>
					<Input
						id="teleChatIdCustom"
						class="font-mono text-sm"
						placeholder="Contoh: -1001234567890"
						bind:value={teleChatId}
					/>
					<p class="auth-hint">Wajib. Grup biasanya ID negatif. Jika salah, bot membalas Chat ID saat dikirimi pesan.</p>
				</div>
				<p class="text-xs leading-relaxed text-slate-500 dark:text-slate-400">
					Di grup: gunakan <strong>/saldo</strong> (disarankan) atau <strong>!saldo</strong>.
					Jika <strong>!saldo</strong> tidak jalan, matikan Group Privacy di @BotFather → Bot Settings.
				</p>
			{/if}
			<div class="flex flex-col gap-2 sm:flex-row">
				<Button onclick={() => saveTele(true)} disabled={teleAktifkanDisabled}>
					{teleAlreadyOn && !teleSettingsUnchanged ? 'Simpan perubahan' : 'Aktifkan Tele'}
				</Button>
				<Button color="light" onclick={() => saveTele(false)} disabled={!teleAlreadyOn}>Nonaktifkan</Button>
			</div>
		</div>
	</section>
</div>

<div class="mt-5 grid gap-5 lg:grid-cols-2">
	<section class="app-panel">
		<h2 class="mb-1 text-base font-semibold tracking-tight text-slate-900 dark:text-slate-50">Link laporan</h2>
		<p class="mb-4 text-sm text-slate-500 dark:text-slate-400">
			Tautan publik tanpa login. Atur slug sendiri atau pakai default slug tim/kas.
		</p>
		{#if integration}
			<div class="space-y-3">
				<div class="auth-field">
					<Label for="reportSlug" class="text-sm font-medium text-slate-700 dark:text-slate-300">Slug URL</Label>
					<div class="flex flex-wrap items-center gap-2">
						<span class="text-xs text-slate-500 dark:text-slate-400">/report/</span>
						<Input
							id="reportSlug"
							class="flex-1 font-mono text-sm"
							placeholder={integration.team_slug ?? 'kas-batam'}
							bind:value={reportSlug}
						/>
					</div>
					{#if integration.team_slug}
						<p class="auth-hint">
							Default:
							<button type="button" class="font-medium text-primary-700 underline dark:text-primary-400" onclick={() => (reportSlug = integration?.team_slug ?? '')}>{integration.team_slug}</button>
							{#if integration.team_name}
								<span> (dari {integration.team_name})</span>
							{/if}
						</p>
					{/if}
				</div>
				{#if integration.report_url || reportPreviewUrl}
					<div class="flex flex-wrap items-center gap-2">
						<Input class="flex-1 font-mono text-xs" readonly value={integration.report_url ?? reportPreviewUrl} />
						<Button color="light" onclick={copyLink}>Salin</Button>
					</div>
				{/if}
				<div class="flex flex-wrap gap-2">
					<Button onclick={saveReportSlug}>Simpan slug</Button>
					<Button color="light" onclick={resetReportSlug}>Reset ke default</Button>
				</div>
			</div>
		{/if}
	</section>

	<section class="app-panel">
		<h2 class="mb-2 text-base font-semibold tracking-tight text-slate-900 dark:text-slate-50">Command bot</h2>
		<pre class="app-code">/start (Telegram — minta Chat ID)
/saldo atau !saldo
/link atau !link
out#100826#Deskripsi#12000</pre>
		<p class="mt-3 text-xs text-slate-500 dark:text-slate-400">Hari boleh dikosongkan (otomatis dari tanggal). Bot KasQ: kirim /start untuk Chat ID.</p>
	</section>
</div>

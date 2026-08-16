<script lang="ts">
	import defaultBrandImage from '$lib/assets/defaultBrand';
	import { api } from '$lib/api';
	import { loadAppSettings } from '$lib/appSettings.svelte';
	import type { Team, User } from '$lib/types';
	import { formatRupiah } from '$lib/utils';
	import { confirm } from '$lib/confirm.svelte';
	import { toast } from '$lib/toast.svelte';
	import { goto } from '$app/navigation';
	import {
		Alert,
		Badge,
		Button,
		Card,
		Checkbox,
		Heading,
		Input,
		Label,
		Select,
		TabItem,
		Tabs,
		Textarea
	} from 'flowbite-svelte';
	import PasswordInput from '$lib/components/PasswordInput.svelte';

	let teams = $state<Team[]>([]);
	let users = $state<User[]>([]);
	let tab = $state('teams');
	let error = $state('');
	let success = $state('');

	let teamName = $state('');
	let teamSlug = $state('');
	let teamBalance = $state(0);
	let editingTeamId = $state('');

	let userName = $state('');
	let userEmail = $state('');
	let userPassword = $state('');
	let userRole = $state<'admin' | 'ops'>('ops');
	let userTeamId = $state('');
	let editingUserId = $state('');

	let appName = $state('');
	let appTagline = $state('');
	let logoFile = $state<File | null>(null);
	let faviconFile = $state<File | null>(null);
	let removeLogo = $state(false);
	let removeFavicon = $state(false);
	let currentLogoUrl = $state<string | undefined>();
	let currentFaviconUrl = $state<string | undefined>();
	let settingsLoading = $state(false);

	const fileClass =
		'block w-full cursor-pointer rounded-lg border border-gray-300 bg-gray-50 text-sm text-gray-900 focus:outline-none';

	$effect(() => {
		api.me().then((u) => {
			if (u.role !== 'admin') goto('/dashboard');
		});
	});

	async function loadTeamsAndUsers() {
		[teams, users] = await Promise.all([api.getTeams(), api.getUsers()]);
	}

	async function loadSettingsForm() {
		const s = await api.getAppSettings();
		appName = s.app_name;
		appTagline = s.app_tagline;
		currentLogoUrl = s.logo_url;
		currentFaviconUrl = s.favicon_url;
		removeLogo = false;
		removeFavicon = false;
		logoFile = null;
		faviconFile = null;
	}

	async function load() {
		await Promise.all([loadTeamsAndUsers(), loadSettingsForm()]);
	}

	async function saveTeam(e: Event) {
		e.preventDefault();
		error = '';
		success = '';
		try {
			if (editingTeamId) {
				await api.updateTeam(editingTeamId, {
					name: teamName,
					slug: teamSlug,
					initial_balance: teamBalance
				});
			} else {
				await api.createTeam({
					name: teamName,
					slug: teamSlug || undefined,
					initial_balance: teamBalance
				});
			}
			success = 'Tim/Kas disimpan';
			toast.success('Tim/Kas disimpan');
			resetTeamForm();
			await loadTeamsAndUsers();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal';
			toast.error(error);
		}
	}

	async function saveUser(e: Event) {
		e.preventDefault();
		error = '';
		success = '';
		try {
			const data = {
				name: userName,
				email: userEmail,
				role: userRole,
				team_id: userTeamId || undefined,
				password: userPassword || undefined
			};
			if (editingUserId) {
				await api.updateUser(editingUserId, data as Parameters<typeof api.updateUser>[1]);
			} else {
				if (!userPassword) throw new Error('Password wajib untuk user baru');
				await api.createUser({ ...data, password: userPassword } as Parameters<typeof api.createUser>[0]);
			}
			success = 'User disimpan';
			toast.success('User disimpan');
			resetUserForm();
			await loadTeamsAndUsers();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal';
			toast.error(error);
		}
	}

	async function saveSettings(e: Event) {
		e.preventDefault();
		settingsLoading = true;
		error = '';
		success = '';
		try {
			const form = new FormData();
			form.append('app_name', appName);
			form.append('app_tagline', appTagline);
			if (logoFile) form.append('logo', logoFile);
			if (faviconFile) form.append('favicon', faviconFile);
			if (removeLogo) form.append('remove_logo', 'true');
			if (removeFavicon) form.append('remove_favicon', 'true');

			const updated = await api.updateAppSettings(form);
			currentLogoUrl = updated.logo_url;
			currentFaviconUrl = updated.favicon_url;
			removeLogo = false;
			removeFavicon = false;
			logoFile = null;
			faviconFile = null;
			await loadAppSettings();
			success = 'Pengaturan aplikasi disimpan';
			toast.success('Pengaturan disimpan');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Gagal simpan pengaturan';
			toast.error(error);
		} finally {
			settingsLoading = false;
		}
	}

	function editTeam(t: Team) {
		editingTeamId = t.id;
		teamName = t.name;
		teamSlug = t.slug;
		teamBalance = t.initial_balance;
		tab = 'teams';
	}

	function editUser(u: User) {
		editingUserId = u.id;
		userName = u.name;
		userEmail = u.email;
		userRole = u.role;
		userTeamId = u.team_id || '';
		userPassword = '';
		tab = 'users';
	}

	function resetTeamForm() {
		editingTeamId = '';
		teamName = '';
		teamSlug = '';
		teamBalance = 0;
	}

	function resetUserForm() {
		editingUserId = '';
		userName = '';
		userEmail = '';
		userPassword = '';
		userRole = 'ops';
		userTeamId = '';
	}

	async function deleteTeam(id: string) {
		const ok = await confirm({
			title: 'Hapus tim/kas?',
			message: 'Hapus tim/kas ini? Semua transaksi tim/kas juga akan terhapus.',
			confirmLabel: 'Hapus',
			color: 'red'
		});
		if (!ok) return;
		await api.deleteTeam(id);
		success = 'Tim/Kas dihapus';
		toast.success('Tim/Kas dihapus');
		await loadTeamsAndUsers();
	}

	async function deleteUser(id: string) {
		const ok = await confirm({
			title: 'Hapus user?',
			message: 'Hapus user ini? Tindakan ini tidak bisa dibatalkan.',
			confirmLabel: 'Hapus',
			color: 'red'
		});
		if (!ok) return;
		await api.deleteUser(id);
		toast.success('User dihapus');
		await loadTeamsAndUsers();
	}

	function previewUrl(file: File | null, fallback?: string) {
		if (file) return URL.createObjectURL(file);
		if (fallback) {
			const base = import.meta.env.PUBLIC_API_URL || '';
			return `${base}${fallback}`;
		}
		return undefined;
	}

	$effect(() => {
		load();
	});
</script>

<svelte:head><title>Admin — KasQ</title></svelte:head>

<Heading tag="h1" class="mb-6 text-2xl">Admin Panel</Heading>

{#if error}<Alert color="red" class="mb-3">{error}</Alert>{/if}
{#if success}<Alert color="green" class="mb-3">{success}</Alert>{/if}

<Tabs bind:selected={tab} tabStyle="underline" class="mb-6">
	<TabItem key="teams" title="Kelola Tim/Kas">
		<div class="grid gap-6 pt-4 lg:grid-cols-2">
			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<Heading tag="h2" class="mb-4 text-base">{editingTeamId ? 'Edit Tim/Kas' : 'Tambah Tim/Kas'}</Heading>
				<form onsubmit={saveTeam} class="space-y-3">
					<div>
						<Label for="teamName">Nama Tim/Kas</Label>
						<Input id="teamName" placeholder="Contoh: Kas Batam" bind:value={teamName} required />
					</div>
					<div>
						<Label for="teamSlug">Slug (opsional)</Label>
						<Input id="teamSlug" placeholder="kas-batam" bind:value={teamSlug} />
					</div>
					<div>
						<Label for="teamBalance">Saldo Awal (Rp)</Label>
						<Input id="teamBalance" type="number" min="0" bind:value={teamBalance} />
					</div>
					<div class="flex gap-2">
						<Button type="submit">Simpan</Button>
						{#if editingTeamId}
							<Button type="button" color="light" onclick={resetTeamForm}>Batal</Button>
						{/if}
					</div>
				</form>
			</Card>
			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<Heading tag="h2" class="mb-4 text-base">Daftar Tim/Kas</Heading>
				{#if teams.length === 0}
					<p class="text-sm text-slate-500">Belum ada tim/kas. Tambah tim/kas untuk assign ke user ops.</p>
				{/if}
				{#each teams as team}
					<div class="flex items-center justify-between border-b border-slate-100 py-2">
						<div>
							<p class="font-medium">{team.name}</p>
							<p class="text-xs text-slate-500">
								{team.slug} · Saldo awal: {formatRupiah(team.initial_balance)}
							</p>
						</div>
						<div class="flex gap-1">
							<Button size="xs" color="light" onclick={() => editTeam(team)}>Edit</Button>
							<Button size="xs" color="red" outline onclick={() => deleteTeam(team.id)}>Hapus</Button>
						</div>
					</div>
				{/each}
			</Card>
		</div>
	</TabItem>

	<TabItem key="users" title="User Management">
		<div class="grid gap-6 pt-4 lg:grid-cols-2">
			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<Heading tag="h2" class="mb-4 text-base">{editingUserId ? 'Edit User' : 'Tambah User'}</Heading>
				<form onsubmit={saveUser} class="space-y-3">
					<Input placeholder="Nama" bind:value={userName} required />
					<Input type="email" placeholder="Email" bind:value={userEmail} required />
					<PasswordInput
						placeholder={editingUserId ? 'Password baru (kosongkan jika tidak ubah)' : 'Password'}
						bind:value={userPassword}
						autocomplete="new-password"
					/>
					<Select bind:value={userRole}>
						<option value="ops">Ops</option>
						<option value="admin">Admin</option>
					</Select>
					<Select bind:value={userTeamId} required={userRole === 'ops'}>
						<option value="">{userRole === 'ops' ? '— Pilih tim/kas —' : '— Tanpa tim/kas (admin) —'}</option>
						{#each teams as team}
							<option value={team.id}>{team.name}</option>
						{/each}
					</Select>
					<div class="flex gap-2">
						<Button type="submit">Simpan</Button>
						{#if editingUserId}
							<Button type="button" color="light" onclick={resetUserForm}>Batal</Button>
						{/if}
					</div>
				</form>
			</Card>
			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<Heading tag="h2" class="mb-4 text-base">Daftar User</Heading>
				{#each users as u}
					<div class="flex items-center justify-between border-b border-slate-100 py-2">
						<div>
							<p class="font-medium">{u.name}</p>
							<p class="text-xs text-slate-500">
								{u.email} · <Badge color={u.role === 'admin' ? 'purple' : 'blue'}>{u.role}</Badge>
								{#if u.role === 'ops' && !u.team_id}
									· <Badge color="yellow">belum ada tim/kas</Badge>
								{/if}
							</p>
						</div>
						<div class="flex gap-1">
							<Button size="xs" color="light" onclick={() => editUser(u)}>Edit</Button>
							<Button size="xs" color="red" outline onclick={() => deleteUser(u.id)}>Hapus</Button>
						</div>
					</div>
				{/each}
			</Card>
		</div>
	</TabItem>

	<TabItem key="settings" title="Pengaturan Aplikasi">
		<div class="grid gap-6 pt-4 lg:grid-cols-2">
			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<Heading tag="h2" class="mb-4 text-base">Branding Aplikasi</Heading>
				<form onsubmit={saveSettings} class="space-y-4">
					<div>
						<Label for="appName">Nama Aplikasi</Label>
						<Input id="appName" bind:value={appName} required placeholder="KasQ" />
					</div>
					<div>
						<Label for="appTagline">Tagline / Subtitle</Label>
						<Textarea id="appTagline" rows={2} bind:value={appTagline} placeholder="Kas Ku — Pencatatan Keuangan Tim/Kas" />
					</div>
					<div>
						<Label for="logo">Logo</Label>
						<input
							id="logo"
							type="file"
							class={fileClass}
							accept="image/png,image/jpeg,image/svg+xml,image/webp"
							onchange={(e) => {
								logoFile = (e.target as HTMLInputElement).files?.[0] ?? null;
								removeLogo = false;
							}}
						/>
						<p class="mt-1 text-xs text-slate-500">PNG, JPG, SVG, atau WEBP. Maks. 2MB.</p>
						{#if currentLogoUrl && !removeLogo && !logoFile}
							<Checkbox bind:checked={removeLogo} class="mt-2">Hapus logo saat ini</Checkbox>
						{/if}
					</div>
					<div>
						<Label for="favicon">Favicon</Label>
						<input
							id="favicon"
							type="file"
							class={fileClass}
							accept="image/png,image/jpeg,image/svg+xml,image/x-icon,.ico"
							onchange={(e) => {
								faviconFile = (e.target as HTMLInputElement).files?.[0] ?? null;
								removeFavicon = false;
							}}
						/>
						<p class="mt-1 text-xs text-slate-500">ICO, PNG, JPG, atau SVG. Maks. 512KB.</p>
						{#if currentFaviconUrl && !removeFavicon && !faviconFile}
							<Checkbox bind:checked={removeFavicon} class="mt-2">Hapus favicon saat ini</Checkbox>
						{/if}
					</div>
					<Button type="submit" disabled={settingsLoading}>
						{settingsLoading ? 'Menyimpan...' : 'Simpan Pengaturan'}
					</Button>
				</form>
			</Card>
			<Card size="xl" shadow="sm" class="p-3 sm:p-4">
				<Heading tag="h2" class="mb-4 text-base">Preview</Heading>
				<div class="rounded-xl border border-slate-200 bg-slate-50 p-6 dark:border-slate-700 dark:bg-slate-800">
					<div class="flex items-center gap-3">
						<img
							src={previewUrl(logoFile, removeLogo ? undefined : currentLogoUrl) ?? defaultBrandImage}
							alt="Logo preview"
							class="h-10 w-auto object-contain"
						/>
						<div>
							<p class="font-semibold text-slate-900">{appName || 'KasQ'}</p>
							<p class="text-sm text-slate-500">{appTagline || 'Tagline aplikasi'}</p>
						</div>
					</div>
					<div class="mt-6 flex items-center gap-2">
						<span class="text-xs text-slate-500">Favicon:</span>
						<img
							src={previewUrl(faviconFile, removeFavicon ? undefined : currentFaviconUrl) ?? defaultBrandImage}
							alt="Favicon preview"
							class="h-6 w-6 object-contain"
						/>
					</div>
				</div>
				<p class="mt-4 text-xs text-slate-500">
					Perubahan nama, logo, dan favicon akan tampil di header, halaman login, dan tab browser.
				</p>
			</Card>
		</div>
	</TabItem>
</Tabs>

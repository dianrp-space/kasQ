<script lang="ts">
	import { Alert } from 'flowbite-svelte';
	import { InfoCircleSolid } from 'flowbite-svelte-icons';
</script>

<Alert color="yellow">
	{#snippet icon()}
		<InfoCircleSolid class="h-5 w-5" />
	{/snippet}
	<span class="font-medium">Akun belum ditugaskan ke tim/kas</span>
	<p class="mt-1">
		Kamu sudah bisa login, tapi belum bisa input transaksi. Minta admin menetapkan tim/kas kamu di
		<strong>Admin Panel → User</strong>, lalu refresh halaman ini.
	</p>
</Alert>

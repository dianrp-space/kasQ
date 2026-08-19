<script lang="ts">
	import { APP_VERSION } from '$lib/version';

	let {
		prefix = '',
		class: className = ''
	}: {
		prefix?: string;
		class?: string;
	} = $props();
</script>

<a
	href="/changelog"
	class="inline-block tabular-nums transition-opacity hover:opacity-80 {className}"
	title="Lihat changelog"
>
	{prefix}{prefix ? ' ' : ''}v{APP_VERSION}
</a>

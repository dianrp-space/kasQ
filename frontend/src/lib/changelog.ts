export type ChangelogListBlock = {
	kind: 'list';
	title: string;
	items: string[];
};

export type ChangelogCodeBlock = {
	kind: 'code';
	title: string;
	code: string;
};

export type ChangelogBlock = ChangelogListBlock | ChangelogCodeBlock;

export type ChangelogRelease = {
	version: string;
	date: string | null;
	summary: string;
	blocks: ChangelogBlock[];
};

export function parseChangelog(md: string): ChangelogRelease[] {
	const text = md.replace(/\r\n/g, '\n');
	return text
		.split(/^## \[/m)
		.slice(1)
		.map(parseRelease)
		.filter((r) => r.version);
}

function parseRelease(chunk: string): ChangelogRelease {
	const nl = chunk.indexOf('\n');
	const header = nl === -1 ? chunk : chunk.slice(0, nl);
	const body = nl === -1 ? '' : chunk.slice(nl + 1);
	const hm = /^([^\]]+)\](?:\s*[—–-]\s*(\d{4}-\d{2}-\d{2}))?/.exec(header.trim());
	const version = hm?.[1]?.trim() ?? '';
	const date = hm?.[2] ?? null;

	const blocks: ChangelogBlock[] = [];
	let summary = '';
	let currentTitle = '';
	let items: string[] = [];
	let inCode = false;
	const code: string[] = [];

	function flushList() {
		if (currentTitle && items.length) {
			blocks.push({ kind: 'list', title: currentTitle, items: [...items] });
		}
		items = [];
	}

	for (const line of body.split('\n')) {
		if (line.startsWith('```')) {
			if (!inCode) {
				flushList();
				inCode = true;
				code.length = 0;
			} else {
				blocks.push({ kind: 'code', title: currentTitle, code: code.join('\n').trimEnd() });
				inCode = false;
			}
			continue;
		}
		if (inCode) {
			code.push(line);
			continue;
		}
		const h3 = /^###\s+(.+)$/.exec(line);
		if (h3) {
			flushList();
			currentTitle = h3[1].trim();
			continue;
		}
		const bullet = /^[-*]\s+(.+)$/.exec(line);
		if (bullet) {
			items.push(bullet[1].trim());
			continue;
		}
		const trimmed = line.trim();
		if (trimmed && !currentTitle) {
			summary = summary ? `${summary} ${trimmed}` : trimmed;
		}
	}
	flushList();
	return { version, date, summary, blocks };
}

export function formatChangelogDate(iso: string | null): string {
	if (!iso) return '';
	return new Date(`${iso}T00:00:00`).toLocaleDateString('id-ID', {
		day: 'numeric',
		month: 'long',
		year: 'numeric'
	});
}

export function formatChangelogInline(text: string): string {
	const escaped = text
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;');
	return escaped
		.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
		.replace(
			/`([^`]+)`/g,
			'<code class="rounded bg-slate-100 px-1 py-0.5 font-mono text-[0.8125rem] dark:bg-slate-800">$1</code>'
		)
		.replace(
			/\[([^\]]+)\]\((https?:[^)]+)\)/g,
			'<a href="$2" class="text-primary-700 underline decoration-primary-700/30 underline-offset-2 dark:text-primary-400" target="_blank" rel="noopener noreferrer">$1</a>'
		);
}

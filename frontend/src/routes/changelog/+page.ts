import changelogMd from 'virtual:changelog';
import { parseChangelog } from '$lib/changelog';

export function load() {
	return {
		releases: parseChangelog(changelogMd)
	};
}

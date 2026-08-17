export const opsNavLinks = [
	{ href: '/dashboard', label: 'Dashboard', shortLabel: 'Dashboard', icon: '📊' },
	{ href: '/transactions', label: 'Input Transaksi', shortLabel: 'Input', icon: '✏️' },
	{ href: '/integrations', label: 'Integrasi', shortLabel: 'Integrasi', icon: '🔗' },
	{ href: '/support', label: 'Support', shortLabel: 'Support', icon: '💚' },
	{ href: '/profile', label: 'Profil', shortLabel: 'Profil', icon: '👤' }
] as const;

/** Bottom dock: Input di tengah (menonjol), Profil di kanan */
export const opsBottomNavLinks = [
	opsNavLinks[0],
	opsNavLinks[2],
	opsNavLinks[1],
	opsNavLinks[3],
	opsNavLinks[4]
] as const;

export const opsBottomNavCenterHref = '/transactions';

export function isOpsNavActive(href: string, pathname: string) {
	return pathname === href || pathname.startsWith(href + '/');
}

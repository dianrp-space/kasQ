export function homePath(role: 'admin' | 'ops') {
	return role === 'admin' ? '/admin' : '/dashboard';
}

export const opsOnlyPaths = ['/dashboard', '/transactions', '/integrations'];

export function isOpsOnlyPath(pathname: string) {
	return opsOnlyPaths.some((p) => pathname === p || pathname.startsWith(p + '/'));
}

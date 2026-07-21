export function normalizePermissionCode(code: string): string {
	return code.replace(/:/g, '.');
}

export function hasPermission(userPerms: string[], requiredPerm: string): boolean {
	const normalizedRequired = normalizePermissionCode(requiredPerm);
	return userPerms.some(p => normalizePermissionCode(p) === normalizedRequired);
}

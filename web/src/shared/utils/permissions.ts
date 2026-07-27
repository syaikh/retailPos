export function hasPermission(userPerms: string[], requiredPerm: string): boolean {
	return userPerms.some(p => p === requiredPerm);
}

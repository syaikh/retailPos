/**
 * Barrel entry point so `import { ... } from '$shared/utils/theme'` resolves.
 * The actual runes store lives in theme.svelte.ts.
 */

export { themeStore, initTheme, applyTheme, currentTheme } from './theme.svelte';
export type { Theme } from './theme.svelte';

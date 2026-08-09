/**
 * Directory entry point so `import { labels } from '$shared/i18n'` resolves.
 * The actual runes store lives in index.svelte.ts.
 */

export { i18n, labels, currentLocale, setLocale, toggleLocale, t } from './index.svelte';
export type { Locale } from './index.svelte';
export type { LabelKey, Labels } from './id';

/**
 * Reactive theme store (Svelte 5 runes).
 * Mirrors the I18nStore pattern — $state-backed so derived values
 * in components recompute when the theme changes.
 */

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'pos.theme';

function loadInitialTheme(): Theme {
  if (typeof localStorage === 'undefined') return 'dark';
  const saved = localStorage.getItem(STORAGE_KEY);
  return saved === 'light' ? 'light' : 'dark';
}

class ThemeStore {
  current = $state<Theme>(loadInitialTheme());

  setTheme(theme: Theme): void {
    this.current = theme;
    if (typeof document !== 'undefined') {
      const root = document.documentElement;
      if (theme === 'dark') {
        root.classList.add('dark');
      } else {
        root.classList.remove('dark');
      }
    }
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, theme);
    }
  }
}

export const themeStore = new ThemeStore();

/** Apply the saved theme to the DOM on app init. */
export function initTheme(): void {
  themeStore.setTheme(themeStore.current);
}

/** Apply a specific theme (used by auth prefs + toggle). */
export function applyTheme(theme: Theme): void {
  themeStore.setTheme(theme);
}

/** Current reactive theme value. */
export function currentTheme(): Theme {
  return themeStore.current;
}

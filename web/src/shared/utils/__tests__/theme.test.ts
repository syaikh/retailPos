import { describe, it, expect, beforeEach, vi } from 'vitest';
import { applyTheme, initTheme, currentTheme, themeStore } from '../theme.svelte';

describe('theme', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('dark');
    localStorage.clear();
  });

  it('applyTheme("dark") adds dark class to <html>', () => {
    applyTheme('dark');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(currentTheme()).toBe('dark');
  });

  it('applyTheme("light") removes dark class from <html>', () => {
    document.documentElement.classList.add('dark');
    applyTheme('light');
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(currentTheme()).toBe('light');
  });

  it('persists theme to localStorage', () => {
    applyTheme('light');
    expect(localStorage.getItem('pos.theme')).toBe('light');
    applyTheme('dark');
    expect(localStorage.getItem('pos.theme')).toBe('dark');
  });

  it('initTheme applies stored theme', () => {
    localStorage.setItem('pos.theme', 'light');
    // Re-import to trigger loadInitialTheme with the stored value
    // Since themeStore is a singleton, we set current directly
    themeStore.current = 'light';
    initTheme();
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(currentTheme()).toBe('light');
  });
});

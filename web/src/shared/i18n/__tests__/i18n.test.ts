import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { i18n, labels, currentLocale, setLocale, toggleLocale } from '../index.svelte';
import { id } from '../id';
import { en } from '../en';

describe('i18n store', () => {
  const originalStorage = globalThis.localStorage;

  beforeEach(() => {
    globalThis.localStorage = originalStorage;
    i18n.setLocale('id');
  });

  afterEach(() => {
    globalThis.localStorage = originalStorage;
  });

  it('defaults to Indonesian and exposes reactive labels', () => {
    expect(currentLocale()).toBe('id');
    expect(labels.save).toBe('Simpan');
    expect(labels.cancel).toBe('Batal');
  });

  it('switches to English via setLocale and persists to localStorage', () => {
    setLocale('en');
    expect(currentLocale()).toBe('en');
    expect(labels.save).toBe('Save');
    expect(globalThis.localStorage.getItem('pos.locale')).toBe('en');
  });

  it('toggles locale back and forth', () => {
    setLocale('id');
    toggleLocale();
    expect(currentLocale()).toBe('en');
    toggleLocale();
    expect(currentLocale()).toBe('id');
  });

  it('labels is reactive after locale change', () => {
    setLocale('id');
    expect(labels.save).toBe('Simpan');
    setLocale('en');
    expect(labels.save).toBe('Save');
    setLocale('id');
    expect(labels.save).toBe('Simpan');
  });

  it('en mirrors id keys exactly', () => {
    const idKeys = Object.keys(id).sort();
    const enKeys = Object.keys(en).sort();
    expect(enKeys).toEqual(idKeys);
  });
});

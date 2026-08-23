/**
 * Reactive i18n store (Svelte 5 runes).
 * Usage in .svelte components: import { i18n, labels } from '$shared/i18n';
 * then reference {labels.save} or {i18n.labels.save} in templates — both are
 * reactive because labels is a Proxy backed by the store's $state.
 * Locale is persisted to localStorage across the app.
 */

import { id } from './id';
import { en } from './en';
import type { Labels } from './id';

export type Locale = 'id' | 'en';

const dictionaries: Record<Locale, Labels> = { id, en };
const STORAGE_KEY = 'pos.locale';

function loadInitialLocale(): Locale {
  if (typeof localStorage === 'undefined') return 'id';
  const saved = localStorage.getItem(STORAGE_KEY);
  return saved === 'en' ? 'en' : 'id';
}

class I18nStore {
  locale = $state<Locale>(loadInitialLocale());

  get labels(): Labels {
    return dictionaries[this.locale];
  }

  setLocale(locale: Locale) {
    this.locale = locale;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, locale);
    }
  }

  toggleLocale() {
    this.setLocale(this.locale === 'id' ? 'en' : 'id');
  }
}

export const i18n = new I18nStore();

/**
 * Reactive label dictionary. Reading labels.x tracks the underlying $state,
 * so templates using {labels.save} re-render when the locale changes.
 */
export const labels = new Proxy(i18n, {
  get(store, prop) {
    if (typeof prop === 'string' && prop in store.labels) {
      return store.labels[prop as keyof Labels];
    }
    return (store as unknown as Record<string, unknown>)[prop as string];
  },
}) as unknown as Labels;

/**
 * Map well-known payment method codes to localized label keys. Custom methods
 * (added by the store) fall back to their stored name.
 */
const PAYMENT_METHOD_LABEL_KEYS: Record<string, keyof Labels> = {
  CASH: 'cash',
  CARD: 'card',
  E_WALLET: 'eWallet',
};

/**
 * Localized display name for a payment method. Reads `i18n.labels`, so calling
 * this in a template re-renders when the UI language changes.
 */
export function paymentMethodLabel(code: string, fallback?: string): string {
  const key = PAYMENT_METHOD_LABEL_KEYS[code.toUpperCase()];
  if (key) return i18n.labels[key];
  return fallback ?? code;
}

export const currentLocale = () => i18n.locale;
export const setLocale = (locale: Locale) => i18n.setLocale(locale);
export const toggleLocale = () => i18n.toggleLocale();

/**
 * Intl locale string for date formatting, following the selected UI language.
 * 'en' → 'en-US' (English month/day names); otherwise 'id-ID' (Indonesian).
 */
export function getDateLocale(): 'en-US' | 'id-ID' {
  return i18n.locale === 'en' ? 'en-US' : 'id-ID';
}

/**
 * Locale-aware date formatting. Use this instead of `date.toLocaleString('id-ID', ...)`
 * whenever the output should follow the UI language (e.g. month/day abbreviations).
 */
export function formatLocaleDate(date: Date, options?: Intl.DateTimeFormatOptions): string {
  return date.toLocaleString(getDateLocale(), options);
}

/**
 * Translate a label key with `{placeholder}` interpolation.
 * Example: t('importRows', { count: 5 }) -> "Impor 5 Baris".
 */
export function t(key: keyof Labels, params?: Record<string, string | number>): string {
  let str = i18n.labels[key];
  if (params) {
    for (const [name, value] of Object.entries(params)) {
      str = str.replaceAll(`{${name}}`, String(value));
    }
  }
  return str;
}

import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { setLocale } from '$shared/i18n';
import { formatDate, formatDateTime } from './format';

describe('consignment/lib/format', () => {
  beforeAll(() => setLocale('en'));
  afterAll(() => setLocale('id'));

  it('formatDate returns dash for empty input', () => {
    expect(formatDate('')).toBe('-');
    expect(formatDate(undefined)).toBe('-');
  });

  it('formatDate renders the Jakarta date', () => {
    // 2026-06-15T17:00:00Z + 7h = 2026-06-16 WIB
    expect(formatDate('2026-06-15T17:00:00Z')).toBe('Jun 16, 2026');
  });

  it('formatDateTime returns dash for empty input', () => {
    expect(formatDateTime('')).toBe('-');
    expect(formatDateTime(undefined)).toBe('-');
  });

  it('formatDateTime renders the Jakarta date and time', () => {
    // 2026-06-15T00:00:00Z + 7h = 2026-06-15 07:00 WIB
    expect(formatDateTime('2026-06-15T00:00:00Z')).toBe('Jun 15, 2026, 07:00 AM');
  });

  it('delegates invalid non-empty input to the em dash returned by the Jakarta helper', () => {
    expect(formatDate('not-a-date')).toBe('—');
    expect(formatDateTime('not-a-date')).toBe('—');
  });
});

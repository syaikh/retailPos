import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { getCached, setCache, invalidateCache } from '../cache';

const mockNow = 1_700_000_000_000;

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(mockNow);
});

afterEach(() => {
  vi.useRealTimers();
});

describe('getCached', () => {
  it('returns null for unknown key', () => {
    expect(getCached('nonexistent')).toBeNull();
  });

  it('returns cached value before expiry', () => {
    setCache('foo', { data: 42 }, 60_000);
    expect(getCached('foo')).toEqual({ data: 42 });
  });

  it('returns null after expiry', () => {
    setCache('foo', { data: 42 }, 60_000);
    vi.advanceTimersByTime(61_000);
    expect(getCached('foo')).toBeNull();
  });

  it('returns null for expired entry even if it exists', () => {
    setCache('foo', 'still-here', 10_000);
    vi.advanceTimersByTime(10_001);
    expect(getCached('foo')).toBeNull();
  });
});

describe('setCache', () => {
  it('stores a value retrievable by getCached', () => {
    setCache('bar', 'hello', 30_000);
    expect(getCached('bar')).toBe('hello');
  });

  it('overwrites existing key', () => {
    setCache('bar', 'first', 60_000);
    setCache('bar', 'second', 60_000);
    expect(getCached('bar')).toBe('second');
  });
});

describe('invalidateCache', () => {
  it('removes a cached entry', () => {
    setCache('baz', 99, 60_000);
    invalidateCache('baz');
    expect(getCached('baz')).toBeNull();
  });

  it('does not throw for unknown key', () => {
    expect(() => invalidateCache('nope')).not.toThrow();
  });
});

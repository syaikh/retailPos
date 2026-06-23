import { describe, it, expect, beforeEach } from 'vitest';
import { getAuthToken, setAccessToken, removeAccessToken } from '../session';

describe('session', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('getAuthToken returns null when no token', () => {
    expect(getAuthToken()).toBeNull();
  });

  it('setAccessToken stores token in sessionStorage', () => {
    setAccessToken('test-token-123');
    expect(sessionStorage.getItem('access_token')).toBe('test-token-123');
  });

  it('getAuthToken returns stored token', () => {
    setAccessToken('test-token-456');
    expect(getAuthToken()).toBe('test-token-456');
  });

  it('removeAccessToken clears the token', () => {
    setAccessToken('test-token-789');
    removeAccessToken();
    expect(getAuthToken()).toBeNull();
  });

  it('getAuthToken returns null when window is undefined', () => {
    const origWindow = globalThis.window;
    (globalThis as any).window = undefined;
    expect(getAuthToken()).toBeNull();
    (globalThis as any).window = origWindow;
  });
});

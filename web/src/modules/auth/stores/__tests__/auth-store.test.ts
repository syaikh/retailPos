import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore } from '../auth-store.svelte';

describe('auth-store', () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it('exports useAuthStore function', () => {
    expect(typeof useAuthStore).toBe('function');
  });

  it('returns expected API shape', () => {
    const store = useAuthStore();
    expect(store).toHaveProperty('user');
    expect(store).toHaveProperty('isAuthenticated');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('setUser');
    expect(store).toHaveProperty('clearUser');
    expect(store).toHaveProperty('getToken');
  });

  it('clearUser removes token and resets state', () => {
    sessionStorage.setItem('access_token', 'some-token');
    const store = useAuthStore();
    store.clearUser();
    expect(sessionStorage.getItem('access_token')).toBeNull();
    expect(store.isAuthenticated).toBe(false);
    expect(store.user).toBeNull();
  });

  it('getToken returns token from sessionStorage', () => {
    sessionStorage.setItem('access_token', 'token-abc');
    const store = useAuthStore();
    expect(store.getToken()).toBe('token-abc');
  });

  it('getToken returns null after clearUser', () => {
    sessionStorage.setItem('access_token', 'token-xyz');
    const store = useAuthStore();
    store.clearUser();
    expect(store.getToken()).toBeNull();
  });

  it('setUser sets user and marks as authenticated', () => {
    const store = useAuthStore();
    const user = { id: 1, username: 'test', email: 'test@test.com' } as any;
    store.setUser(user);
    expect(store.user).toEqual(user);
    expect(store.isAuthenticated).toBe(true);
    expect(store.loading).toBe(false);
  });

  it('setter for user updates reactive state', () => {
    const store = useAuthStore();
    const user = { id: 2, username: 'setter-test' } as any;
    store.user = user;
    expect(store.user).toEqual(user);
  });

  it('setter for isAuthenticated updates reactive state', () => {
    const store = useAuthStore();
    store.isAuthenticated = true;
    expect(store.isAuthenticated).toBe(true);
    store.isAuthenticated = false;
    expect(store.isAuthenticated).toBe(false);
  });
});

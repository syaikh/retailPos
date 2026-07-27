import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockPost = vi.fn();
vi.mock('axios', () => ({
  default: {
    create: () => ({
      post: mockPost,
      interceptors: {
        response: { use: vi.fn(), eject: vi.fn() },
        request: { use: vi.fn(), eject: vi.fn() },
      },
    }),
    isAxiosError: (err: unknown) =>
      typeof err === 'object' && err !== null && 'isAxiosError' in err,
  },
}));

function makeAxiosError(status: number) {
  const err = new Error(`HTTP ${status}`) as any;
  err.isAxiosError = true;
  err.response = { status };
  return err;
}

describe('auth-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  describe('login', () => {
    it('stores access token on success', async () => {
      mockPost.mockResolvedValueOnce({
        status: 200,
        data: { access_token: 'new-token', user: { id: 1, username: 'admin', email: 'admin@test.com' } },
      });

      const { login } = await import('../auth-service');
      const result = await login('admin', 'password');

      expect(result).not.toBe(false);
      if (result) {
        expect(result.access_token).toBe('new-token');
        expect(result.user.username).toBe('admin');
      }
      expect(sessionStorage.getItem('access_token')).toBe('new-token');
    });

    it('merges permission claims from token when present', async () => {
      mockPost.mockResolvedValueOnce({
        status: 200,
        data: {
          access_token: 'eyJhbGciOiJIUzI1NiJ9.eyJwZXJtaXNzaW9ucyI6WyJhZG1pbiIsInVzZXIiXX0.abc',
          user: { id: 1, username: 'admin' },
        },
      });

      const { login } = await import('../auth-service');
      const result = await login('admin', 'password');

      expect(result).not.toBe(false);
      if (result) {
        expect(result.user.permissions).toEqual(['admin', 'user']);
      }
    });

    it('returns false on API failure', async () => {
      mockPost.mockRejectedValueOnce(new Error('Network error'));

      const { login } = await import('../auth-service');
      const result = await login('wrong', 'wrong');

      expect(result).toBe(false);
    });
  });

  describe('logout', () => {
    it('clears session and redirects', async () => {
      mockPost.mockResolvedValueOnce({ status: 200 });
      sessionStorage.setItem('access_token', 'will-be-cleared');

      const { logout } = await import('../auth-service');
      await logout();

      expect(sessionStorage.getItem('access_token')).toBeNull();
    });
  });

  describe('refreshAccessToken', () => {
    it('calls doRefresh and returns new token', async () => {
      mockPost.mockResolvedValueOnce({ data: { access_token: 'refreshed-token' } });

      const { refreshAccessToken } = await import('../auth-service');
      const result = await refreshAccessToken();

      expect(result).toBe('refreshed-token');
      expect(sessionStorage.getItem('access_token')).toBe('refreshed-token');
    });

    it('returns null on refresh failure', async () => {
      mockPost.mockRejectedValueOnce(new Error('Refresh failed'));

      const { refreshAccessToken } = await import('../auth-service');
      const result = await refreshAccessToken();

      expect(result).toBeNull();
    });
  });

  describe('proactive refresh', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it('startProactiveRefresh calls doRefresh on interval', async () => {
      sessionStorage.setItem('access_token', 'some-token');
      mockPost.mockResolvedValue({ data: { access_token: 'refreshed' } });

      const mod = await import('../auth-service');
      mod.startProactiveRefresh();

      await vi.advanceTimersByTimeAsync(13 * 60 * 1000);

      expect(mockPost).toHaveBeenCalledWith('/refresh');
      mod.stopProactiveRefresh();
    });

    it('stops refresh timer when no token present', async () => {
      const mod = await import('../auth-service');
      mod.startProactiveRefresh();

      await vi.advanceTimersByTimeAsync(13 * 60 * 1000);

      expect(mockPost).not.toHaveBeenCalled();
    });

    it('stopProactiveRefresh clears the timer', async () => {
      sessionStorage.setItem('access_token', 'some-token');
      mockPost.mockResolvedValue({ data: { access_token: 'refreshed' } });

      const mod = await import('../auth-service');
      mod.startProactiveRefresh();
      mod.stopProactiveRefresh();

      await vi.advanceTimersByTimeAsync(13 * 60 * 1000);

      expect(mockPost).not.toHaveBeenCalled();
    });
  });

  describe('setupAxiosInterceptors', () => {
    it('registers response interceptor', async () => {
      const { setupAxiosInterceptors } = await import('../auth-service');
      const client = { interceptors: { response: { use: vi.fn() } } } as any;

      setupAxiosInterceptors(client);

      expect(client.interceptors.response.use).toHaveBeenCalled();
    });
  });

  describe('checkAuth', () => {
    it('returns true when /validate succeeds', async () => {
      mockPost.mockResolvedValueOnce({ status: 200 });

      const { checkAuth } = await import('../auth-service');
      const result = await checkAuth();

      expect(result).toBe(true);
    });

    it('auto-refreshes on 401 and retries validation', async () => {
      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockResolvedValueOnce({ data: { access_token: 'new-token' } })
        .mockResolvedValueOnce({ status: 200 });

      const { checkAuth } = await import('../auth-service');
      const result = await checkAuth();

      expect(result).toBe(true);
      expect(mockPost).toHaveBeenCalledWith('/refresh');
    });

    it('returns false when refresh fails after 401', async () => {
      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockRejectedValueOnce(new Error('Refresh failed'));

      const { checkAuth } = await import('../auth-service');
      const result = await checkAuth();

      expect(result).toBe(false);
    });

    it('returns false on non-401 axios error', async () => {
      mockPost.mockRejectedValueOnce(makeAxiosError(500));

      const { checkAuth } = await import('../auth-service');
      const result = await checkAuth();

      expect(result).toBe(false);
    });

    it('returns false on non-axios error', async () => {
      mockPost.mockRejectedValueOnce(new Error('Generic error'));

      const { checkAuth } = await import('../auth-service');
      const result = await checkAuth();

      expect(result).toBe(false);
    });
  });

  describe('restoreSession', () => {
    it('returns user when validate succeeds', async () => {
      sessionStorage.setItem('access_token', 'valid-token');
      mockPost.mockResolvedValueOnce({
        data: { user: { id: 1, username: 'admin' }, permissions: ['read'] },
      });

      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.user?.username).toBe('admin');
        expect(result.user?.permissions).toEqual(['read']);
      }
    });

    it('returns success false when no token in sessionStorage', async () => {
      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it('returns success false when validate returns no user', async () => {
      sessionStorage.setItem('access_token', 'valid-token');
      mockPost.mockResolvedValueOnce({ data: {} });

      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it('auto-refreshes on 401 and retries', async () => {
      sessionStorage.setItem('access_token', 'stale-token');

      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockResolvedValueOnce({ data: { access_token: 'new-token' } })
        .mockResolvedValueOnce({
          data: { user: { id: 1, username: 'refreshed' }, permissions: ['write'] },
        });

      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.user?.username).toBe('refreshed');
        expect(result.user?.permissions).toEqual(['write']);
      }
    });

    it('returns success false when refresh fails after 401 on validate', async () => {
      sessionStorage.setItem('access_token', 'stale-token');

      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockRejectedValueOnce(new Error('fail'));

      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it('returns success false when retry validate still fails after refresh', async () => {
      sessionStorage.setItem('access_token', 'stale-token');

      mockPost
        .mockRejectedValueOnce(makeAxiosError(401))
        .mockResolvedValueOnce({ data: { access_token: 'new-token' } })
        .mockRejectedValueOnce(makeAxiosError(401));

      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });

    it('returns success false on non-401 error', async () => {
      sessionStorage.setItem('access_token', 'some-token');
      mockPost.mockRejectedValueOnce(makeAxiosError(500));

      const { restoreSession } = await import('../auth-service');
      const result = await restoreSession();

      expect(result.success).toBe(false);
    });
  });
});

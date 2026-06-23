import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockPost = vi.fn();
vi.mock('axios', () => ({
  default: {
    create: () => ({
      post: mockPost,
    }),
  },
}));

describe('auth-service login', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  it('login stores access token on success', async () => {
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

  it('login returns false on API failure', async () => {
    mockPost.mockRejectedValueOnce(new Error('Network error'));

    const { login } = await import('../auth-service');
    const result = await login('wrong', 'wrong');

    expect(result).toBe(false);
  });

  it('logout clears sessionStorage', async () => {
    mockPost.mockResolvedValueOnce({ status: 200 });
    sessionStorage.setItem('access_token', 'will-be-cleared');

    const { logout } = await import('../auth-service');
    await logout();

    expect(sessionStorage.getItem('access_token')).toBeNull();
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import client, { getAuthToken, setAuthToken } from './client';
import { login, register, logout } from './auth';

describe('auth API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    setAuthToken(null);
  });

  it('login sends credentials and stores access token', async () => {
    const mockSession = {
      access_token: 'fake-jwt-token',
      refresh_token: 'fake-refresh-token',
      token_type: 'Bearer',
      access_expires_at: '2026-11-01T15:00:00Z',
      refresh_expires_at: '2026-11-08T15:00:00Z',
    };

    vi.spyOn(client, 'post').mockResolvedValueOnce({ data: mockSession });

    const credentials = { email: 'test@example.com', password: 'password123' };
    const result = await login(credentials);

    expect(client.post).toHaveBeenCalledWith('/auth/login', credentials);
    expect(result).toEqual(mockSession);
    expect(getAuthToken()).toBe('fake-jwt-token');
  });

  it('register sends account registration payload', async () => {
    const mockAccount = {
      id: 123,
      email: 'test@example.com',
    };

    vi.spyOn(client, 'post').mockResolvedValueOnce({ data: mockAccount });

    const payload = { email: 'test@example.com', password: 'password123' };
    const result = await register(payload);

    expect(client.post).toHaveBeenCalledWith('/auth/register', payload);
    expect(result).toEqual(mockAccount);
  });

  it('logout posts to /auth/logout and clears access token', async () => {
    setAuthToken('initial-token');
    vi.spyOn(client, 'post').mockResolvedValueOnce({ data: undefined });

    await logout();

    expect(client.post).toHaveBeenCalledWith('/auth/logout', {});
    expect(getAuthToken()).toBeNull();
  });
});

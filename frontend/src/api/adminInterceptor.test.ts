import { AxiosError, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import client from './client';
import './adminInterceptor';
import { clearSession, getAccessToken, refreshAdminSession } from '../auth/adminSession';
import type { AdminSession } from '../auth/adminSession';

vi.mock('../auth/adminSession', () => ({
  getAccessToken: vi.fn(),
  clearSession: vi.fn(),
  refreshAdminSession: vi.fn(),
}));

const okResponse = (config: InternalAxiosRequestConfig, data: unknown): AxiosResponse => ({
  data,
  status: 200,
  statusText: 'OK',
  headers: {},
  config,
});

const unauthorized = (config: InternalAxiosRequestConfig): Promise<never> => {
  const response = okResponse(config, { error: 'authentication required' });
  response.status = 401;
  response.statusText = 'Unauthorized';
  return Promise.reject(new AxiosError('Request failed', 'ERR_BAD_REQUEST', config, undefined, response));
};

beforeEach(() => {
  vi.clearAllMocks();
  client.defaults.adapter = async (config) => okResponse(config as InternalAxiosRequestConfig, {});
});

describe('adminInterceptor', () => {
  it('adjunta el access token cuando hay sesión', async () => {
    vi.mocked(getAccessToken).mockReturnValue('access-token');
    let authorization: string | undefined;
    client.defaults.adapter = async (config) => {
      authorization = config.headers.get('Authorization') as string | undefined;
      return okResponse(config as InternalAxiosRequestConfig, {});
    };

    await client.get('/players/ranking');

    expect(authorization).toBe('Bearer access-token');
  });

  it('ante un 401 refresca y reintenta una vez con el token nuevo', async () => {
    vi.mocked(getAccessToken).mockReturnValue('viejo');
    vi.mocked(refreshAdminSession).mockResolvedValue({
      email: 'admin@nosepudo.ar',
      accessToken: 'nuevo',
      refreshToken: 'refresh',
      accessExpiresAt: '',
      refreshExpiresAt: '',
    } as AdminSession);

    let calls = 0;
    const seen: Array<string | undefined> = [];
    client.defaults.adapter = async (config) => {
      calls += 1;
      seen.push(config.headers.get('Authorization') as string | undefined);
      if (calls === 1) {
        return unauthorized(config as InternalAxiosRequestConfig);
      }
      return okResponse(config as InternalAxiosRequestConfig, { items: [] });
    };

    const response = await client.get('/sync/runs');

    expect(response.status).toBe(200);
    expect(calls).toBe(2);
    expect(seen[0]).toBe('Bearer viejo');
    expect(seen[1]).toBe('Bearer nuevo');
    expect(refreshAdminSession).toHaveBeenCalledTimes(1);
  });

  it('no intenta refrescar en los endpoints de autenticación', async () => {
    vi.mocked(getAccessToken).mockReturnValue('viejo');
    client.defaults.adapter = async (config) => unauthorized(config as InternalAxiosRequestConfig);

    await expect(client.post('/auth/login', { email: 'a', password: 'b' })).rejects.toBeInstanceOf(
      AxiosError,
    );

    expect(refreshAdminSession).not.toHaveBeenCalled();
    expect(clearSession).not.toHaveBeenCalled();
  });

  it('si el reintento vuelve a dar 401, limpia la sesión', async () => {
    vi.mocked(getAccessToken).mockReturnValue('viejo');
    vi.mocked(refreshAdminSession).mockResolvedValue({
      email: 'admin@nosepudo.ar',
      accessToken: 'nuevo',
      refreshToken: 'refresh',
      accessExpiresAt: '',
      refreshExpiresAt: '',
    } as AdminSession);
    client.defaults.adapter = async (config) => unauthorized(config as InternalAxiosRequestConfig);

    await expect(client.get('/sync/runs')).rejects.toBeInstanceOf(AxiosError);

    expect(refreshAdminSession).toHaveBeenCalledTimes(1);
    expect(clearSession).toHaveBeenCalledTimes(1);
  });

  it('respeta un Authorization explícito, como el del refresh token', async () => {
    vi.mocked(getAccessToken).mockReturnValue('access-token');
    let authorization: string | undefined;
    client.defaults.adapter = async (config) => {
      authorization = config.headers.get('Authorization') as string | undefined;
      return okResponse(config as InternalAxiosRequestConfig, {});
    };

    await client.post('/auth/refresh', undefined, {
      headers: { Authorization: 'Bearer refresh-token' },
    });

    expect(authorization).toBe('Bearer refresh-token');
  });
});

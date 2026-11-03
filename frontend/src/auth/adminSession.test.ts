import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { login, logout, refresh, SessionDTO } from '../api/admin';
import {
  AdminSession,
  clearSession,
  decodeAccessToken,
  isSuperuserSession,
  loadSession,
  loginAdmin,
  logoutAdmin,
  refreshAdminSession,
  saveSession,
  useAdminSession,
} from './adminSession';

vi.mock('../api/admin', () => ({
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn(),
}));

const encodeBase64Url = (value: string): string =>
  btoa(value).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');

const makeToken = (payload: Record<string, unknown>): string =>
  `header.${encodeBase64Url(JSON.stringify(payload))}.signature`;

const superuserToken = makeToken({ priv: 'superuser', sub: '42', exp: 1893456000 });
const commonUserToken = makeToken({ priv: 'user', sub: '7', exp: 1893456000 });

const sessionDTO = (access: string): SessionDTO => ({
  access_token: access,
  refresh_token: 'refresh-token',
  token_type: 'Bearer',
  access_expires_at: '2030-01-01T00:00:00Z',
  refresh_expires_at: '2030-01-08T00:00:00Z',
});

const storedSession = (access: string): AdminSession => ({
  email: 'admin@nosepudo.ar',
  accessToken: access,
  refreshToken: 'refresh-token',
  accessExpiresAt: '2030-01-01T00:00:00Z',
  refreshExpiresAt: '2030-01-08T00:00:00Z',
});

beforeEach(() => {
  window.sessionStorage.clear();
  vi.clearAllMocks();
});

describe('adminSession', () => {
  it('decodifica el privilegio y el sujeto del access token', () => {
    const claims = decodeAccessToken(superuserToken);

    expect(claims?.privilege).toBe('superuser');
    expect(claims?.subject).toBe('42');
    expect(claims?.expiresAt).toBe(1893456000);
  });

  it('devuelve null para un token inválido', () => {
    expect(decodeAccessToken('no-es-un-jwt')).toBeNull();
  });

  it('reconoce solo al superusuario', () => {
    expect(isSuperuserSession(storedSession(superuserToken))).toBe(true);
    expect(isSuperuserSession(storedSession(commonUserToken))).toBe(false);
  });

  it('guarda y recupera la sesión de sessionStorage', () => {
    saveSession(storedSession(superuserToken));

    expect(loadSession()?.email).toBe('admin@nosepudo.ar');
    expect(loadSession()?.accessToken).toBe(superuserToken);

    clearSession();
    expect(loadSession()).toBeNull();
  });

  it('loginAdmin guarda la sesión que devuelve el backend', async () => {
    vi.mocked(login).mockResolvedValue(sessionDTO(superuserToken));

    const session = await loginAdmin('admin@nosepudo.ar', 'secreto');

    expect(login).toHaveBeenCalledWith('admin@nosepudo.ar', 'secreto');
    expect(session.accessToken).toBe(superuserToken);
    expect(loadSession()?.refreshToken).toBe('refresh-token');
  });

  it('logoutAdmin llama al backend y limpia la sesión', async () => {
    saveSession(storedSession(superuserToken));
    vi.mocked(logout).mockResolvedValue();

    await logoutAdmin();

    expect(logout).toHaveBeenCalledWith(superuserToken);
    expect(loadSession()).toBeNull();
  });

  it('refreshAdminSession comparte un único refresh entre llamadas concurrentes', async () => {
    saveSession(storedSession('viejo-access'));
    const nextToken = makeToken({ priv: 'superuser', sub: '42', exp: 1893456000 });
    let resolveRefresh: (value: SessionDTO) => void = () => undefined;
    vi.mocked(refresh).mockImplementation(
      () =>
        new Promise<SessionDTO>((resolve) => {
          resolveRefresh = resolve;
        }),
    );

    const first = refreshAdminSession();
    const second = refreshAdminSession();

    expect(refresh).toHaveBeenCalledTimes(1);

    resolveRefresh(sessionDTO(nextToken));

    const [firstSession, secondSession] = await Promise.all([first, second]);

    expect(firstSession.accessToken).toBe(nextToken);
    expect(secondSession.accessToken).toBe(nextToken);
    expect(loadSession()?.accessToken).toBe(nextToken);
  });

  it('useAdminSession reacciona a los cambios de sesión', () => {
    const { result } = renderHook(() => useAdminSession());

    expect(result.current.session).toBeNull();
    expect(result.current.isSuperuser).toBe(false);

    act(() => saveSession(storedSession(superuserToken)));

    expect(result.current.session?.email).toBe('admin@nosepudo.ar');
    expect(result.current.isSuperuser).toBe(true);

    act(() => clearSession());

    expect(result.current.session).toBeNull();
    expect(result.current.isSuperuser).toBe(false);
  });
});

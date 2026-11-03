import { useCallback, useEffect, useState } from 'react';
import {
  login as apiLogin,
  logout as apiLogout,
  refresh as apiRefresh,
  SessionDTO,
} from '../api/admin';

const STORAGE_KEY = 'nsp.admin.session';
const SESSION_EVENT = 'nsp:admin-session';

export interface AdminSession {
  email: string;
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
}

interface AccessClaims {
  privilege: string | null;
  subject: string | null;
  expiresAt: number | null;
}

const decodeBase64Url = (value: string): string => {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/');
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=');
  return atob(padded);
};

export const decodeAccessToken = (token: string): AccessClaims | null => {
  const payload = token.split('.')[1];
  if (!payload) {
    return null;
  }

  try {
    const parsed = JSON.parse(decodeBase64Url(payload)) as Record<string, unknown>;
    return {
      privilege: typeof parsed.priv === 'string' ? parsed.priv : null,
      subject: typeof parsed.sub === 'string' ? parsed.sub : null,
      expiresAt: typeof parsed.exp === 'number' ? parsed.exp : null,
    };
  } catch {
    return null;
  }
};

const isAdminSession = (value: unknown): value is AdminSession => {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const record = value as Record<string, unknown>;
  return (
    typeof record.email === 'string' &&
    typeof record.accessToken === 'string' &&
    typeof record.refreshToken === 'string' &&
    typeof record.accessExpiresAt === 'string' &&
    typeof record.refreshExpiresAt === 'string'
  );
};

const notifySessionChange = () => {
  window.dispatchEvent(new Event(SESSION_EVENT));
};

export const loadSession = (): AdminSession | null => {
  const raw = window.sessionStorage.getItem(STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    const parsed: unknown = JSON.parse(raw);
    return isAdminSession(parsed) ? parsed : null;
  } catch {
    return null;
  }
};

export const saveSession = (session: AdminSession) => {
  window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(session));
  notifySessionChange();
};

export const clearSession = () => {
  window.sessionStorage.removeItem(STORAGE_KEY);
  notifySessionChange();
};

export const getAccessToken = (): string | null => loadSession()?.accessToken ?? null;

export const isSuperuserSession = (session: AdminSession): boolean =>
  decodeAccessToken(session.accessToken)?.privilege === 'superuser';

const sessionFromDTO = (email: string, dto: SessionDTO): AdminSession => ({
  email,
  accessToken: dto.access_token,
  refreshToken: dto.refresh_token,
  accessExpiresAt: dto.access_expires_at,
  refreshExpiresAt: dto.refresh_expires_at,
});

let refreshInFlight: Promise<AdminSession> | null = null;

export const refreshAdminSession = (): Promise<AdminSession> => {
  if (refreshInFlight) {
    return refreshInFlight;
  }

  const current = loadSession();
  if (!current) {
    return Promise.reject(new Error('no hay sesión de administración'));
  }

  refreshInFlight = apiRefresh(current.refreshToken)
    .then((dto) => {
      const next = sessionFromDTO(current.email, dto);
      saveSession(next);
      return next;
    })
    .finally(() => {
      refreshInFlight = null;
    });

  return refreshInFlight;
};

export const loginAdmin = async (email: string, password: string): Promise<AdminSession> => {
  const dto = await apiLogin(email, password);
  const session = sessionFromDTO(email, dto);
  saveSession(session);
  return session;
};

export const logoutAdmin = async (): Promise<void> => {
  const current = loadSession();
  try {
    if (current) {
      await apiLogout(current.accessToken);
    }
  } finally {
    clearSession();
  }
};

export const useAdminSession = () => {
  const [session, setSession] = useState<AdminSession | null>(() => loadSession());

  useEffect(() => {
    const sync = () => setSession(loadSession());
    window.addEventListener(SESSION_EVENT, sync);
    return () => window.removeEventListener(SESSION_EVENT, sync);
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const next = await loginAdmin(email, password);
    setSession(next);
  }, []);

  const logout = useCallback(async () => {
    await logoutAdmin();
    setSession(null);
  }, []);

  return {
    session,
    isSuperuser: session ? isSuperuserSession(session) : false,
    login,
    logout,
  };
};

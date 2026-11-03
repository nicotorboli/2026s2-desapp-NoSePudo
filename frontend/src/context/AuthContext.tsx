import React, { createContext, useContext, useEffect, useState, useMemo, useCallback } from 'react';
import { login as apiLogin, register as apiRegister, logout as apiLogout, LoginRequestDTO, RegisterRequestDTO } from '../api/auth';
import { getAuthToken, setAuthToken } from '../api/client';

export interface AuthUser {
  id: number;
  email: string;
  privilege?: string;
}

interface AuthContextType {
  user: AuthUser | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (credentials: LoginRequestDTO) => Promise<void>;
  registerAndLogin: (account: RegisterRequestDTO) => Promise<void>;
  logout: () => Promise<void>;
}

const defaultAuthContext: AuthContextType = {
  user: null,
  token: null,
  isAuthenticated: false,
  login: async () => {},
  registerAndLogin: async () => {},
  logout: async () => {},
};

const AuthContext = createContext<AuthContextType>(defaultAuthContext);

const USER_EMAIL_KEY = 'nsp_user_email';

interface JwtPayload {
  sub?: string;
  priv?: string;
  exp?: number;
}

const parseJwt = (token: string): JwtPayload | null => {
  try {
    const base64Url = token.split('.')[1];
    if (!base64Url) return null;
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
    const jsonPayload = decodeURIComponent(
      window
        .atob(padded)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );
    return JSON.parse(jsonPayload);
  } catch {
    return null;
  }
};

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [token, setToken] = useState<string | null>(getAuthToken());
  const [user, setUser] = useState<AuthUser | null>(null);

  const initUserFromToken = useCallback((jwtToken: string, fallbackEmail?: string) => {
    const payload = parseJwt(jwtToken);
    if (!payload || !payload.sub) {
      setAuthToken(null);
      setToken(null);
      setUser(null);
      return;
    }

    if (payload.exp && payload.exp * 1000 < Date.now()) {
      setAuthToken(null);
      setToken(null);
      setUser(null);
      localStorage.removeItem(USER_EMAIL_KEY);
      return;
    }

    const storedEmail = fallbackEmail || localStorage.getItem(USER_EMAIL_KEY) || `Usuario #${payload.sub}`;
    if (fallbackEmail) {
      localStorage.setItem(USER_EMAIL_KEY, fallbackEmail);
    }

    setUser({
      id: Number(payload.sub),
      email: storedEmail,
      privilege: payload.priv,
    });
  }, []);

  useEffect(() => {
    const initialToken = getAuthToken();
    if (initialToken) {
      initUserFromToken(initialToken);
    }
  }, [initUserFromToken]);

  const login = async (credentials: LoginRequestDTO): Promise<void> => {
    const session = await apiLogin(credentials);
    setToken(session.access_token);
    initUserFromToken(session.access_token, credentials.email);
  };

  const registerAndLogin = async (account: RegisterRequestDTO): Promise<void> => {
    await apiRegister(account);
    await login({ email: account.email, password: account.password });
  };

  const logout = async (): Promise<void> => {
    try {
      await apiLogout();
    } catch {
      setAuthToken(null);
    } finally {
      setToken(null);
      setUser(null);
      localStorage.removeItem(USER_EMAIL_KEY);
    }
  };

  const value = useMemo(
    () => ({
      user,
      token,
      isAuthenticated: Boolean(token && user),
      login,
      registerAndLogin,
      logout,
    }),
    [user, token]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextType => {
  return useContext(AuthContext);
};

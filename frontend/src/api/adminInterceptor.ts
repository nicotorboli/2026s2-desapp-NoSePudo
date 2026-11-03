import type { AxiosError, InternalAxiosRequestConfig } from 'axios';
import client from './client';
import { clearSession, getAccessToken, refreshAdminSession } from '../auth/adminSession';

interface RetryableConfig extends InternalAxiosRequestConfig {
  _retried?: boolean;
}

const AUTH_PATHS = ['/auth/login', '/auth/refresh', '/auth/logout'];

const isAuthPath = (url?: string): boolean => {
  if (!url) {
    return false;
  }
  return AUTH_PATHS.some((path) => url.includes(path));
};

client.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token && !config.headers.has('Authorization')) {
    config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  async (error: unknown) => {
    const axiosError = error as AxiosError;
    const config = axiosError.config as RetryableConfig | undefined;

    if (!config || axiosError.response?.status !== 401 || isAuthPath(config.url)) {
      return Promise.reject(error);
    }

    if (config._retried) {
      clearSession();
      return Promise.reject(error);
    }

    config._retried = true;

    try {
      const session = await refreshAdminSession();
      config.headers.set('Authorization', `Bearer ${session.accessToken}`);
    } catch {
      clearSession();
      return Promise.reject(error);
    }

    return client.request(config);
  },
);

import client, { setAuthToken } from './client';

export interface LoginRequestDTO {
  email: string;
  password: string;
}

export interface RegisterRequestDTO {
  email: string;
  password: string;
}

export interface SessionResponseDTO {
  access_token: string;
  refresh_token: string;
  token_type: string;
  access_expires_at: string;
  refresh_expires_at: string;
}

export interface AccountResponseDTO {
  id: number;
  email: string;
  created_at?: string;
}

const post = async <T, B>(url: string, body: B): Promise<T> => {
  const response = await client.post<T>(url, body);
  return response.data;
};

export const login = async (credentials: LoginRequestDTO): Promise<SessionResponseDTO> => {
  const session = await post<SessionResponseDTO, LoginRequestDTO>('/auth/login', credentials);
  setAuthToken(session.access_token, session.refresh_token);
  return session;
};

export const register = async (account: RegisterRequestDTO): Promise<AccountResponseDTO> => {
  return post<AccountResponseDTO, RegisterRequestDTO>('/auth/register', account);
};

export const logout = async (): Promise<void> => {
  try {
    await post<void, Record<string, never>>('/auth/logout', {});
  } finally {
    setAuthToken(null);
  }
};

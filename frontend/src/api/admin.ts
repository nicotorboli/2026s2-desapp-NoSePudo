import client from './client';

export type SyncType = 'players' | 'statistics';
export type SyncTrigger = 'cron' | 'manual';
export type SyncRunStatus = 'running' | 'succeeded' | 'partial' | 'failed' | 'skipped';

export interface SyncRunProgressDTO {
  current: number;
  total: number;
  message: string;
}

export interface SyncRunDTO {
  id: number;
  type: SyncType;
  trigger: SyncTrigger;
  status: SyncRunStatus;
  startedAt: string;
  finishedAt: string | null;
  totalProcessed: number;
  totalUpdated: number;
  totalDeactivated: number;
  totalMatched: number;
  totalUnmatched: number;
  totalInserted: number;
  message: string;
  error: string;
  triggeredBy: number | null;
  progress: SyncRunProgressDTO | null;
}

interface SyncRunPageDTO {
  items: SyncRunDTO[];
  offset: number;
  limit: number;
  total: number;
}

export interface SyncScheduleDTO {
  type: SyncType;
  enabled: boolean;
  intervalMinutes: number;
  nextRunAt: string;
  updatedAt: string;
  updatedBy: number | null;
}

interface UpdateSyncScheduleBody {
  enabled: boolean;
  intervalMinutes: number;
}

interface SyncRunListParams {
  type?: SyncType;
  offset?: number;
  limit?: number;
}

export interface ScoringStrategyDTO {
  id: number;
  name: string;
  alpha: number;
  weights: Record<string, Record<string, number>>;
}

interface ActiveStrategyDTO {
  id: number;
  name: string;
}

interface LeagueCoefficientDTO {
  leagueCode: string;
  coefficient: number;
  effectiveFrom: string | null;
}

export interface SessionDTO {
  access_token: string;
  refresh_token: string;
  token_type: string;
  access_expires_at: string;
  refresh_expires_at: string;
}

const get = async <T>(url: string, params?: Record<string, unknown>): Promise<T> => {
  const response = await client.get<T>(url, { params });
  return response.data;
};

const post = async <T>(url: string, data?: unknown): Promise<T> => {
  const response = await client.post<T>(url, data);
  return response.data;
};

const put = async <T>(url: string, data?: unknown): Promise<T> => {
  const response = await client.put<T>(url, data);
  return response.data;
};

export const login = async (email: string, password: string): Promise<SessionDTO> => {
  return post<SessionDTO>('/auth/login', { email, password });
};

export const refresh = async (refreshToken: string): Promise<SessionDTO> => {
  const response = await client.post<SessionDTO>('/auth/refresh', undefined, {
    headers: { Authorization: `Bearer ${refreshToken}` },
  });
  return response.data;
};

export const logout = async (accessToken: string): Promise<void> => {
  await client.post('/auth/logout', undefined, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
};

export const createSyncRun = async (type: SyncType): Promise<SyncRunDTO> => {
  return post<SyncRunDTO>(`/sync/${type}/runs`);
};

export const getSyncRun = async (id: number): Promise<SyncRunDTO> => {
  return get<SyncRunDTO>(`/sync/runs/${id}`);
};

export const listSyncRuns = async (params?: SyncRunListParams): Promise<SyncRunPageDTO> => {
  return get<SyncRunPageDTO>('/sync/runs', params as Record<string, unknown> | undefined);
};

export const listSyncSchedules = async (): Promise<SyncScheduleDTO[]> => {
  return get<SyncScheduleDTO[]>('/sync/schedules');
};

export const updateSyncSchedule = async (
  type: SyncType,
  body: UpdateSyncScheduleBody,
): Promise<SyncScheduleDTO> => {
  return put<SyncScheduleDTO>(`/sync/schedules/${type}`, body);
};

export const listStrategies = async (): Promise<ScoringStrategyDTO[]> => {
  return get<ScoringStrategyDTO[]>('/scoring/strategies');
};

const getActiveStrategy = async (): Promise<ActiveStrategyDTO> => {
  return get<ActiveStrategyDTO>('/scoring/active-strategy');
};

export const getActiveStrategyOptional = async (): Promise<ActiveStrategyDTO | null> => {
  try {
    return await getActiveStrategy();
  } catch (error) {
    if (isNotFound(error)) {
      return null;
    }
    throw error;
  }
};

export const activateStrategy = async (id: number): Promise<ActiveStrategyDTO> => {
  return put<ActiveStrategyDTO>('/scoring/active-strategy', { id });
};

export const getLeagueCoefficient = async (league: string): Promise<LeagueCoefficientDTO> => {
  return get<LeagueCoefficientDTO>(`/scoring/league-coefficients/${league}`);
};

const isNotFound = (error: unknown): boolean => {
  if (typeof error !== 'object' || error === null) {
    return false;
  }
  const response = (error as { response?: { status?: number } }).response;
  return response?.status === 404;
};

export const setLeagueCoefficient = async (
  league: string,
  coefficient: number,
): Promise<LeagueCoefficientDTO> => {
  return put<LeagueCoefficientDTO>(`/scoring/league-coefficients/${league}`, { coefficient });
};

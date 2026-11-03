import client from './client';

export interface PlayerDetailDTO {
  id: number;
  externalId: number;
  name: string;
  club: string;
  league: string;
  leagueCode: string;
  position: string;
  dateOfBirth: string | null;
  nationality: string | null;
  shirtNumber: number | null;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface PlayerStatisticsDTO {
  createdAt: string;
  goals: number;
  assists: number;
  shots: number;
  keyPasses: number;
  dribbles: number;
  tackles: number;
  interceptions: number;
  fouls: number;
  clears: number;
  blocks: number;
  wasDribbled: number;
  rating: number;
}

export interface PlayerFilterParams {
  page?: number;
  limit?: number;
  league?: string;
  club?: string;
  position?: string;
  search?: string;
  includeInactive?: boolean;
}

export type RankingSort = 'name' | 'score_asc' | 'score_desc';

export interface PlayerScoreDTO {
  computedAt: string;
  strategyId: number;
  strategyName: string;
  rawScore: number;
  percentile: number;
  score: number;
  leagueCoefficient: number;
  playerId: number;
}

export interface PlayerRankingItemDTO {
  id: number;
  name: string;
  club: string;
  league: string;
  position: string;
  score: number | null;
}

export interface PlayerRankingPageDTO {
  items: PlayerRankingItemDTO[];
  offset: number;
  limit: number;
  total: number;
}

interface PlayerRankingParams {
  offset?: number;
  limit?: number;
  sort?: RankingSort;
  league?: string;
  club?: string;
  position?: string;
  search?: string;
  includeInactive?: boolean;
}

// Private helper method per Constitution Principle VI
const get = async <T>(url: string, params?: Record<string, unknown>): Promise<T> => {
  const response = await client.get<T>(url, { params });
  return response.data;
};

export const getPlayerById = async (id: number): Promise<PlayerDetailDTO> => {
  return get<PlayerDetailDTO>(`/players/${id}`);
};

export const getPlayerStatistics = async (id: number): Promise<PlayerStatisticsDTO[]> => {
  return get<PlayerStatisticsDTO[]>(`/players/${id}/statistics`);
};

export const getPlayerScore = async (id: number): Promise<PlayerScoreDTO> => {
  return get<PlayerScoreDTO>(`/players/${id}/score`);
};

export const getPlayerScores = async (id: number): Promise<PlayerScoreDTO[]> => {
  return get<PlayerScoreDTO[]>(`/players/${id}/scores`);
};

export const getPlayersRanking = async (params?: PlayerRankingParams): Promise<PlayerRankingPageDTO> => {
  return get<PlayerRankingPageDTO>('/players/ranking', params as Record<string, unknown>);
};

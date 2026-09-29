import client from './client';

export interface PlayerListItemDTO {
  id: number;
  name: string;
  club: string;
  league: string;
  position: string;
}

export interface PaginatedPlayersDTO {
  items: PlayerListItemDTO[];
  page: number;
  limit: number;
  total: number;
  totalPages: number;
}

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

export interface PlayerFilterParams {
  page?: number;
  limit?: number;
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

export const getPlayers = async (params?: PlayerFilterParams): Promise<PaginatedPlayersDTO> => {
  return get<PaginatedPlayersDTO>('/players', params as Record<string, unknown>);
};

export const getPlayerById = async (id: number): Promise<PlayerDetailDTO> => {
  return get<PlayerDetailDTO>(`/players/${id}`);
};

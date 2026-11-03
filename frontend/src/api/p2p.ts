import client from './client';

export type P2PSessionStatus = 'PENDING' | 'COMPLETED' | 'CANCELLED' | 'EXPIRED';

export interface P2PUserSummaryDTO {
  id: number;
  email: string;
}

export interface P2PPlayerSummaryDTO {
  id: number;
  name: string;
  clubName: string;
}

export interface P2PSessionDTO {
  id: number;
  token: string;
  sessionUrl: string;
  seller: P2PUserSummaryDTO;
  buyer: P2PUserSummaryDTO | null;
  player: P2PPlayerSummaryDTO;
  quantity: number;
  pricePerToken: number;
  totalAmount: number;
  status: P2PSessionStatus;
  createdAt: string;
  expiresAt: string;
  completedAt?: string | null;
}

export interface CreateP2PSessionRequestDTO {
  playerId: number;
  quantity: number;
  pricePerToken: number;
}

export interface P2PTradeResponseDTO {
  tradeId: number;
  sessionToken: string;
  buyerId: number;
  sellerId: number;
  playerId: number;
  quantity: number;
  pricePerToken: number;
  totalAmount: number;
  executedAt: string;
}

export const createP2PSession = async (data: CreateP2PSessionRequestDTO): Promise<P2PSessionDTO> => {
  const response = await client.post<P2PSessionDTO>('/p2p/sessions', data);
  return response.data;
};

export const getP2PSession = async (token: string): Promise<P2PSessionDTO> => {
  const response = await client.get<P2PSessionDTO>(`/p2p/sessions/${token}`);
  return response.data;
};

export const acceptP2PSession = async (token: string): Promise<P2PTradeResponseDTO> => {
  const response = await client.post<P2PTradeResponseDTO>(`/p2p/sessions/${token}/accept`, {});
  return response.data;
};

export const cancelP2PSession = async (token: string): Promise<void> => {
  await client.post<void>(`/p2p/sessions/${token}/cancel`, {});
};

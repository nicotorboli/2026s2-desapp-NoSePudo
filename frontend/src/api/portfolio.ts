import client from './client';
import { OrderSide } from './orders';

export interface PortfolioPositionDTO {
  playerId: number;
  playerName: string;
  clubName: string;
  position: string;
  quantity: number;
  availableQuantity?: number;
  reservedQuantity?: number;
  averagePrice: string | number;
  currentPrice: string | number;
  currentValuation: string | number;
  unrealizedResult: string | number;
}

export interface PortfolioResponseDTO {
  userId: number;
  availableCash: string | number;
  reservedCash: string | number;
  tokensValuation: string | number;
  totalValue: string | number;
  unrealizedResult: string | number;
  positions: PortfolioPositionDTO[];
}

export interface TransactionDTO {
  id: number;
  side: OrderSide;
  playerId: number;
  playerName: string;
  quantity: number;
  price: string | number;
  totalAmount: string | number;
  executedAt: string;
}

export interface TransactionPageDTO {
  items: TransactionDTO[];
  total: number;
  page: number;
  limit: number;
  totalPages: number;
}

const get = async <T>(url: string, params?: Record<string, unknown>): Promise<T> => {
  const response = await client.get<T>(url, { params });
  return response.data;
};

export const getUserPortfolio = async (userId: number): Promise<PortfolioResponseDTO> => {
  return get<PortfolioResponseDTO>(`/users/${userId}/portfolio`);
};

export const getUserTransactions = async (
  userId: number,
  page = 1,
  limit = 20
): Promise<TransactionPageDTO> => {
  return get<TransactionPageDTO>(`/users/${userId}/transactions`, { page, limit });
};

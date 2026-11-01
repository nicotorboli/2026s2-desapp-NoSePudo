import { describe, it, expect, vi, beforeEach } from 'vitest';
import client from './client';
import { getUserPortfolio, getUserTransactions } from './portfolio';

describe('portfolio API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('getUserPortfolio fetches portfolio by user id', async () => {
    const mockPortfolio = {
      userId: 5,
      availableCash: 950.0,
      reservedCash: 50.0,
      tokensValuation: 100.0,
      totalValue: 1100.0,
      unrealizedResult: 10.0,
      positions: [
        {
          playerId: 10,
          playerName: 'Lionel Messi',
          clubName: 'Inter Miami CF',
          position: 'Attacker',
          quantity: 10,
          averagePrice: 1.0,
          currentPrice: 1.5,
          currentValuation: 15.0,
          unrealizedResult: 5.0,
        },
      ],
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockPortfolio });

    const result = await getUserPortfolio(5);

    expect(client.get).toHaveBeenCalledWith('/users/5/portfolio', { params: undefined });
    expect(result).toEqual(mockPortfolio);
  });

  it('getUserTransactions fetches paginated transactions', async () => {
    const mockPage = {
      items: [
        {
          id: 1,
          side: 'BUY' as const,
          playerId: 10,
          playerName: 'Lionel Messi',
          quantity: 5,
          price: 1.0,
          totalAmount: 5.0,
          executedAt: '2026-11-01T15:00:00Z',
        },
      ],
      total: 1,
      page: 1,
      limit: 10,
      totalPages: 1,
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockPage });

    const result = await getUserTransactions(5, 1, 10);

    expect(client.get).toHaveBeenCalledWith('/users/5/transactions', { params: { page: 1, limit: 10 } });
    expect(result).toEqual(mockPage);
  });
});

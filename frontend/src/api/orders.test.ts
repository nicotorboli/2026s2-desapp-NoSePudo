import { describe, it, expect, vi, beforeEach } from 'vitest';
import client from './client';
import { placeBuyOrder, placeSellOrder, getOpenOrders, cancelOrder } from './orders';

describe('orders API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('placeBuyOrder sends buy order payload', async () => {
    const mockResponse = {
      id: 1,
      side: 'BUY' as const,
      playerId: 10,
      price: 1.5,
      quantity: 5,
      executedQuantity: 5,
      remainingQuantity: 0,
      timeInForce: 'FOK' as const,
      status: 'FILLED' as const,
      createdAt: '2026-11-01T15:00:00Z',
      trades: [],
    };

    vi.spyOn(client, 'post').mockResolvedValueOnce({ data: mockResponse });

    const payload = { playerId: 10, quantity: 5, timeInForce: 'FOK' as const };
    const result = await placeBuyOrder(payload);

    expect(client.post).toHaveBeenCalledWith('/orders/buy', payload);
    expect(result).toEqual(mockResponse);
  });

  it('placeSellOrder sends sell order payload', async () => {
    const mockResponse = {
      id: 2,
      side: 'SELL' as const,
      playerId: 10,
      price: 1.5,
      quantity: 3,
      executedQuantity: 3,
      remainingQuantity: 0,
      timeInForce: 'GTC' as const,
      status: 'FILLED' as const,
      createdAt: '2026-11-01T15:00:00Z',
      trades: [],
    };

    vi.spyOn(client, 'post').mockResolvedValueOnce({ data: mockResponse });

    const payload = { playerId: 10, quantity: 3, price: 1.5, timeInForce: 'GTC' as const };
    const result = await placeSellOrder(payload);

    expect(client.post).toHaveBeenCalledWith('/orders/sell', {
      ...payload,
      price: '1.5000',
    });
    expect(result).toEqual(mockResponse);
  });

  it('getOpenOrders fetches open orders list', async () => {
    const mockOrders = [
      {
        id: 3,
        side: 'BUY' as const,
        playerId: 10,
        price: 1.2,
        quantity: 10,
        executedQuantity: 0,
        remainingQuantity: 10,
        timeInForce: 'GTC' as const,
        status: 'PENDING' as const,
        createdAt: '2026-11-01T15:00:00Z',
      },
    ];

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockOrders });

    const result = await getOpenOrders(10);

    expect(client.get).toHaveBeenCalledWith('/orders', { params: { playerId: 10 } });
    expect(result).toEqual(mockOrders);
  });

  it('cancelOrder sends DELETE request for order id', async () => {
    const mockResponse = {
      id: 3,
      status: 'CANCELLED' as const,
      releasedCredits: 12.0,
    };

    vi.spyOn(client, 'delete').mockResolvedValueOnce({ data: mockResponse });

    const result = await cancelOrder(3);

    expect(client.delete).toHaveBeenCalledWith('/orders/3');
    expect(result).toEqual(mockResponse);
  });
});

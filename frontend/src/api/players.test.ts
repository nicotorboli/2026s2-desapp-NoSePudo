import { describe, it, expect, vi, beforeEach } from 'vitest';
import client from './client';
import {
  getPlayerById,
  getPlayerPrice,
  getPlayerQuotes,
  getPlayerScore,
  getPlayerScores,
  getPlayersRanking,
} from './players';

describe('players API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('getPlayersRanking fetches the ranking with parameters', async () => {
    const mockData = {
      items: [
        {
          id: 1,
          name: 'Bukayo Saka',
          club: 'Arsenal FC',
          league: 'Premier League',
          position: 'Attacker',
          score: 0.87,
          price: '51.0000',
        },
      ],
      offset: 0,
      limit: 100,
      total: 1,
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockData });

    const params = { offset: 0, limit: 100, sort: 'score_desc' as const };
    const result = await getPlayersRanking(params);

    expect(client.get).toHaveBeenCalledWith('/players/ranking', { params });
    expect(result).toEqual(mockData);
  });

  it('getPlayerById fetches a single player detail by id', async () => {
    const mockDetail = {
      id: 7,
      externalId: 7821,
      name: 'Bukayo Saka',
      club: 'Arsenal FC',
      league: 'Premier League',
      leagueCode: 'PL',
      position: 'Attacker',
      dateOfBirth: '2001-09-05',
      nationality: 'England',
      shirtNumber: 7,
      price: '51.0000',
      active: true,
      createdAt: '2026-09-28T12:00:00Z',
      updatedAt: '2026-09-28T12:00:00Z',
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockDetail });

    const result = await getPlayerById(7);

    expect(client.get).toHaveBeenCalledWith('/players/7', { params: undefined });
    expect(result).toEqual(mockDetail);
  });

  it('getPlayerScore fetches the current score of a player', async () => {
    const mockScore = {
      computedAt: '2026-11-02T10:00:00Z',
      strategyId: 1,
      strategyName: 'rateada',
      rawScore: 0.7,
      percentile: 0.8,
      score: 0.4,
      leagueCoefficient: 0.5,
      playerId: 7,
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockScore });

    const result = await getPlayerScore(7);

    expect(client.get).toHaveBeenCalledWith('/players/7/score', { params: undefined });
    expect(result).toEqual(mockScore);
  });

  it('getPlayerScores fetches the score history of a player', async () => {
    const mockScores = [
      {
        computedAt: '2026-11-02T10:00:00Z',
        strategyId: 1,
        strategyName: 'rateada',
        rawScore: 0.7,
        percentile: 0.8,
        score: 0.4,
        leagueCoefficient: 0.5,
        playerId: 7,
      },
    ];

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockScores });

    const result = await getPlayerScores(7);

    expect(client.get).toHaveBeenCalledWith('/players/7/scores', { params: undefined });
    expect(result).toEqual(mockScores);
  });

  it('getPlayerQuotes fetches the quote history of a player', async () => {
    const mockQuotes = [
      {
        playerId: 7,
        createdAt: '2026-11-02T10:00:00Z',
        strategy: 'rateada',
        strategyVersion: '1',
        price: '51.0000',
      },
    ];

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockQuotes });

    const result = await getPlayerQuotes(7);

    expect(client.get).toHaveBeenCalledWith('/players/7/quotes', { params: undefined });
    expect(result).toEqual(mockQuotes);
  });

  it('getPlayerPrice fetches the current price without a date', async () => {
    const mockPrice = {
      playerId: 7,
      computedAt: '2026-11-02T10:00:00Z',
      strategyName: 'rateada',
      price: '51.0000',
      score: 0.5,
      leagueCoefficient: 1,
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockPrice });

    const result = await getPlayerPrice(7);

    expect(client.get).toHaveBeenCalledWith('/players/7/price', { params: undefined });
    expect(result).toEqual(mockPrice);
  });

  it('getPlayerPrice sends the date when provided', async () => {
    const mockPrice = {
      playerId: 7,
      computedAt: '2026-11-02T10:00:00Z',
      strategyName: 'rateada',
      price: '51.0000',
      score: 0.5,
      leagueCoefficient: 1,
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockPrice });

    await getPlayerPrice(7, '2026-11-02');

    expect(client.get).toHaveBeenCalledWith('/players/7/price', {
      params: { date: '2026-11-02' },
    });
  });
});

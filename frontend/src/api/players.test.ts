import { describe, it, expect, vi, beforeEach } from 'vitest';
import client from './client';
import { getPlayers, getPlayerById } from './players';

describe('players API', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('getPlayers fetches paginated players with filter parameters', async () => {
    const mockData = {
      items: [
        {
          id: 1,
          name: 'Bukayo Saka',
          club: 'Arsenal FC',
          league: 'Premier League',
          position: 'Attacker',
        },
      ],
      page: 1,
      limit: 20,
      total: 1,
      totalPages: 1,
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockData });

    const params = { page: 1, limit: 20, league: 'PL', search: 'Saka' };
    const result = await getPlayers(params);

    expect(client.get).toHaveBeenCalledWith('/players', { params });
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
      active: true,
      createdAt: '2026-09-28T12:00:00Z',
      updatedAt: '2026-09-28T12:00:00Z',
    };

    vi.spyOn(client, 'get').mockResolvedValueOnce({ data: mockDetail });

    const result = await getPlayerById(7);

    expect(client.get).toHaveBeenCalledWith('/players/7', { params: undefined });
    expect(result).toEqual(mockDetail);
  });
});

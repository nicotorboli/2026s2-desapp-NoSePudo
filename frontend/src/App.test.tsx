import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { afterEach, describe, expect, it, vi, beforeEach } from 'vitest';
import * as playersApi from './api/players';
import App from './App';

afterEach(cleanup);

describe('App', () => {
  const mockPlayers: playersApi.PaginatedPlayersDTO = {
    items: [
      {
        id: 101,
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

  const mockDetail: playersApi.PlayerDetailDTO = {
    id: 101,
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
    updatedAt: '2026-09-28T12:30:00Z',
  };

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders the catalog page by default', async () => {
    vi.spyOn(playersApi, 'getPlayers').mockResolvedValueOnce(mockPlayers);

    render(<App />);

    expect(screen.getByText('NoSePudo Market')).toBeDefined();
    expect(screen.getByText('Catálogo de Jugadores de Fútbol Europeo')).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });
  });

  it('navigates to player detail when a player is selected and returns to catalog on back click', async () => {
    vi.spyOn(playersApi, 'getPlayers').mockResolvedValue(mockPlayers);
    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValue(mockDetail);

    render(<App />);

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    fireEvent.click(screen.getByText('Bukayo Saka'));

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /volver al catálogo/i })).toBeDefined();
    });

    fireEvent.click(screen.getByRole('button', { name: /volver al catálogo/i }));

    await waitFor(() => {
      expect(screen.getByText('Catálogo de Jugadores de Fútbol Europeo')).toBeDefined();
    });
  });
});

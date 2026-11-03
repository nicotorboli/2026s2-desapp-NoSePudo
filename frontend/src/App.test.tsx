import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { afterEach, describe, expect, it, vi, beforeEach } from 'vitest';
import * as playersApi from './api/players';
import App from './App';

const mocks = vi.hoisted(() => ({
  useAuth: vi.fn(),
}));

vi.mock('./context/AuthContext', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./context/AuthContext')>();
  return { ...actual, useAuth: mocks.useAuth };
});

vi.mock('./components/PlayerTrade/PlayerTrade', () => ({
  PlayerTrade: () => <div data-testid="trade-widget" />,
}));

vi.mock('./pages/PortfolioPage/PortfolioPage', () => ({
  PortfolioPage: () => <div>Mi patrimonio</div>,
}));

afterEach(cleanup);

describe('App', () => {
  const mockPlayers: playersApi.PlayerRankingPageDTO = {
    items: [
      {
        id: 101,
        name: 'Bukayo Saka',
        club: 'Arsenal FC',
        league: 'Premier League',
        position: 'Attacker',
        score: 0.87,
      },
    ],
    offset: 0,
    limit: 25,
    total: 1,
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

  const anonymousAuth = {
    user: null,
    token: null,
    isAuthenticated: false,
    login: vi.fn(),
    registerAndLogin: vi.fn(),
    logout: vi.fn().mockResolvedValue(undefined),
  };

  beforeEach(() => {
    vi.restoreAllMocks();
    mocks.useAuth.mockReturnValue(anonymousAuth);
    vi.spyOn(playersApi, 'getPlayerStatistics').mockResolvedValue([]);
    vi.spyOn(playersApi, 'getPlayerScores').mockResolvedValue([]);
    vi.spyOn(playersApi, 'getPlayerScore').mockRejectedValue(new Error('not found'));
  });

  it('renders the catalog page by default', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValueOnce(mockPlayers);

    render(<App />);

    expect(screen.getByText('NoSePudo Market')).toBeDefined();
    expect(screen.getByText('Catálogo de Jugadores de Fútbol Europeo')).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });
  });

  it('navigates to player detail when a player is selected and returns to catalog on back click', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValue(mockPlayers);
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

  it('navigates to the ranking page from the navbar', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValue(mockPlayers);

    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /ranking/i }));

    await waitFor(() => {
      expect(screen.getByText('Ranking de jugadores')).toBeDefined();
    });
  });

  it('shows Mi Patrimonio only when authenticated and navigates to the portfolio', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValue(mockPlayers);
    mocks.useAuth.mockReturnValue({
      user: { id: 1, email: 'user@test.local' },
      token: 'valid-token',
      isAuthenticated: true,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout: vi.fn().mockResolvedValue(undefined),
    });

    render(<App />);

    fireEvent.click(screen.getByRole('button', { name: /mi patrimonio/i }));

    await waitFor(() => {
      expect(screen.getByText('Mi patrimonio')).toBeDefined();
    });
  });

  it('shows the Administración entry for a superuser', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValue(mockPlayers);
    mocks.useAuth.mockReturnValue({
      user: { id: 1, email: 'admin@test.local', privilege: 'superuser' },
      token: 'valid-token',
      isAuthenticated: true,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout: vi.fn().mockResolvedValue(undefined),
    });

    render(<App />);

    expect(screen.getByRole('button', { name: /administración/i })).toBeDefined();
  });
});

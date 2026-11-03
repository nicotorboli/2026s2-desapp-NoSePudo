import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { PlayerDetailPage } from './PlayerDetailPage';
import * as playersApi from '../../api/players';

vi.mock('../../components/PlayerTrade/PlayerTrade', () => ({
  PlayerTrade: () => <div data-testid="trade-widget" />,
}));

afterEach(cleanup);

describe('PlayerDetailPage', () => {
  const mockDetail: playersApi.PlayerDetailDTO = {
    id: 101,
    externalId: 7821,
    name: 'Bukayo Saka',
    club: 'Arsenal FC',
    league: 'Premier League',
    leagueCode: 'PL',
    position: 'Attacker',
    dateOfBirth: null,
    nationality: null,
    shirtNumber: null,
    price: null,
    active: true,
    createdAt: '2026-09-28T12:00:00Z',
    updatedAt: '2026-09-28T12:30:00Z',
  };

  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(playersApi, 'getPlayerStatistics').mockResolvedValue([]);
    vi.spyOn(playersApi, 'getPlayerScores').mockResolvedValue([]);
    vi.spyOn(playersApi, 'getPlayerQuotes').mockResolvedValue([]);
  });

  it('renders all player attributes and handles fallbacks', async () => {
    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValueOnce(mockDetail);
    vi.spyOn(playersApi, 'getPlayerScore').mockRejectedValueOnce(new Error('not found'));

    render(<PlayerDetailPage playerId={101} onBack={vi.fn()} />);

    expect(screen.getByRole('status')).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    expect(screen.getByText('Arsenal FC')).toBeDefined();
    expect(screen.getByText('Activo')).toBeDefined();
    expect(screen.getByText('Delantero')).toBeDefined();
    expect(screen.getByText('Sin asignar')).toBeDefined();
    expect(screen.getByText('Desconocida')).toBeDefined();
    expect(screen.getByText('—')).toBeDefined();
    expect(screen.getByText('Sin score calculado (N/R)')).toBeDefined();
  });

  it('renders error state on 404/failure', async () => {
    vi.spyOn(playersApi, 'getPlayerById').mockRejectedValueOnce(new Error('Not found'));

    render(<PlayerDetailPage playerId={999} onBack={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Jugador no encontrado')).toBeDefined();
    });
  });

  it('renders the player score and breakdown when available', async () => {
    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValueOnce(mockDetail);
    vi.spyOn(playersApi, 'getPlayerScore').mockResolvedValueOnce({
      computedAt: '2026-11-02T10:00:00Z',
      strategyId: 1,
      strategyName: 'rateada',
      rawScore: 0.7,
      percentile: 0.8,
      score: 0.4,
      leagueCoefficient: 0.5,
      playerId: 101,
    });

    render(<PlayerDetailPage playerId={101} onBack={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('40.00%')).toBeDefined();
    });

    expect(screen.getByText('80.00%')).toBeDefined();
    expect(screen.getByText('0.50')).toBeDefined();
    expect(screen.getByText('rateada')).toBeDefined();
  });

  it('calls onBack when back button is clicked', async () => {
    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValueOnce(mockDetail);
    const handleBack = vi.fn();

    render(<PlayerDetailPage playerId={101} onBack={handleBack} />);

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    fireEvent.click(screen.getByRole('button', { name: /volver al catálogo/i }));
    expect(handleBack).toHaveBeenCalledTimes(1);
  });

  it('renders the trade widget alongside the public detail', async () => {
    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValueOnce(mockDetail);

    render(<PlayerDetailPage playerId={101} onBack={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId('trade-widget')).toBeDefined();
    });
  });
});

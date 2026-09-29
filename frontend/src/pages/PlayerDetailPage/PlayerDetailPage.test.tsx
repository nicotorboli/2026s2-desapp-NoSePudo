import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { PlayerDetailPage } from './PlayerDetailPage';
import * as playersApi from '../../api/players';

afterEach(cleanup);

describe('PlayerDetailPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders all player attributes and handles fallbacks', async () => {
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
      active: true,
      createdAt: '2026-09-28T12:00:00Z',
      updatedAt: '2026-09-28T12:30:00Z',
    };

    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValueOnce(mockDetail);

    render(<PlayerDetailPage playerId={101} onBack={vi.fn()} />);

    expect(screen.getByRole('status')).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    expect(screen.getByText('Arsenal FC')).toBeDefined();
    expect(screen.getByText('Activo')).toBeDefined();
    expect(screen.getByText('Delantero')).toBeDefined();
    expect(screen.getByText('Sin asignar')).toBeDefined(); // Fallback for null shirtNumber
    expect(screen.getByText('Desconocida')).toBeDefined(); // Fallback for null nationality
    expect(screen.getByText('—')).toBeDefined(); // Fallback for null dateOfBirth
  });

  it('renders error state on 404/failure', async () => {
    vi.spyOn(playersApi, 'getPlayerById').mockRejectedValueOnce(new Error('Not found'));

    render(<PlayerDetailPage playerId={999} onBack={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('Jugador no encontrado')).toBeDefined();
    });
  });

  it('calls onBack when back button is clicked', async () => {
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

    vi.spyOn(playersApi, 'getPlayerById').mockResolvedValueOnce(mockDetail);
    const handleBack = vi.fn();

    render(<PlayerDetailPage playerId={101} onBack={handleBack} />);

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    fireEvent.click(screen.getByRole('button', { name: /volver al catálogo/i }));
    expect(handleBack).toHaveBeenCalledTimes(1);
  });
});

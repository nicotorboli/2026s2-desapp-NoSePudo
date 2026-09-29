import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { PlayerListPage } from './PlayerListPage';
import * as playersApi from '../../api/players';

afterEach(cleanup);

describe('PlayerListPage', () => {
  const mockPlayers: playersApi.PaginatedPlayersDTO = {
    items: [
      {
        id: 101,
        name: 'Bukayo Saka',
        club: 'Arsenal FC',
        league: 'Premier League',
        position: 'Attacker',
      },
      {
        id: 102,
        name: 'Martin Ødegaard',
        club: 'Arsenal FC',
        league: 'Premier League',
        position: 'Midfielder',
      },
    ],
    page: 1,
    limit: 20,
    total: 2,
    totalPages: 1,
  };

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders loading state initially and then displays players', async () => {
    vi.spyOn(playersApi, 'getPlayers').mockResolvedValueOnce(mockPlayers);

    render(<PlayerListPage onSelectPlayer={vi.fn()} />);

    expect(screen.getByRole('status')).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
      expect(screen.getByText('Martin Ødegaard')).toBeDefined();
    });

    expect(screen.getAllByText('Arsenal FC').length).toBe(2);
    expect(screen.getAllByText('Premier League').length).toBe(2);
  });

  it('displays empty state when no players are returned', async () => {
    vi.spyOn(playersApi, 'getPlayers').mockResolvedValueOnce({
      items: [],
      page: 1,
      limit: 20,
      total: 0,
      totalPages: 0,
    });

    render(<PlayerListPage onSelectPlayer={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('No se encontraron jugadores')).toBeDefined();
    });
  });

  it('displays error banner on API failure and allows retry', async () => {
    const spy = vi
      .spyOn(playersApi, 'getPlayers')
      .mockRejectedValueOnce(new Error('Network error'))
      .mockResolvedValueOnce(mockPlayers);

    render(<PlayerListPage onSelectPlayer={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('No se pudieron cargar los jugadores. Por favor, intenta de nuevo más tarde.')).toBeDefined();
    });

    const retryBtn = screen.getByRole('button', { name: /reintentar/i });
    fireEvent.click(retryBtn);

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    expect(spy).toHaveBeenCalledTimes(2);
  });

  it('calls onSelectPlayer when a row is clicked', async () => {
    vi.spyOn(playersApi, 'getPlayers').mockResolvedValueOnce(mockPlayers);
    const handleSelect = vi.fn();

    render(<PlayerListPage onSelectPlayer={handleSelect} />);

    await waitFor(() => {
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    fireEvent.click(screen.getByText('Bukayo Saka'));
    expect(handleSelect).toHaveBeenCalledWith(101);
  });
});

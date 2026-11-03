import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { RankingPage } from './RankingPage';
import * as playersApi from '../../api/players';

afterEach(cleanup);

describe('RankingPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders the ranked players with their scores and positions', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValueOnce({
      items: [
        {
          id: 1,
          name: 'Erling Haaland',
          club: 'Manchester City',
          league: 'Premier League',
          position: 'Attacker',
          score: 0.95,
          price: null,
        },
        {
          id: 2,
          name: 'Bukayo Saka',
          club: 'Arsenal FC',
          league: 'Premier League',
          position: 'Attacker',
          score: 0.87,
          price: null,
        },
      ],
      offset: 0,
      limit: 100,
      total: 2,
    });

    render(<RankingPage onSelectPlayer={vi.fn()} />);

    expect(screen.getByRole('status')).toBeDefined();

    await waitFor(() => {
      expect(screen.getByText('Erling Haaland')).toBeDefined();
      expect(screen.getByText('Bukayo Saka')).toBeDefined();
    });

    expect(screen.getByText('95.00%')).toBeDefined();
    expect(screen.getByText('#1')).toBeDefined();
    expect(screen.getByText('#2')).toBeDefined();
  });

  it('calls onSelectPlayer when a row is clicked', async () => {
    vi.spyOn(playersApi, 'getPlayersRanking').mockResolvedValueOnce({
      items: [
        {
          id: 1,
          name: 'Erling Haaland',
          club: 'Manchester City',
          league: 'Premier League',
          position: 'Attacker',
          score: 0.95,
          price: null,
        },
      ],
      offset: 0,
      limit: 100,
      total: 1,
    });

    const handleSelect = vi.fn();
    render(<RankingPage onSelectPlayer={handleSelect} />);

    await waitFor(() => {
      expect(screen.getByText('Erling Haaland')).toBeDefined();
    });

    fireEvent.click(screen.getByText('Erling Haaland'));
    expect(handleSelect).toHaveBeenCalledWith(1);
  });
});

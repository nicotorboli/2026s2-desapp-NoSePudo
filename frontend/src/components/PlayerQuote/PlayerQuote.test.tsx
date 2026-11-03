import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { PlayerQuote } from './PlayerQuote';
import * as playersApi from '../../api/players';

afterEach(cleanup);

const aQuote = (createdAt: string, price: string): playersApi.PlayerQuoteDTO => ({
  playerId: 7,
  createdAt,
  strategy: 'rateada',
  strategyVersion: '1',
  price,
});

describe('PlayerQuote', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('renders the current quote and the history table', async () => {
    vi.spyOn(playersApi, 'getPlayerQuotes').mockResolvedValue([
      aQuote('2026-11-03T10:00:00Z', '76.0000'),
      aQuote('2026-11-01T10:00:00Z', '51.0000'),
    ]);

    render(<PlayerQuote playerId={7} />);

    await waitFor(() => {
      expect(screen.getAllByText('76 Créditos').length).toBeGreaterThan(0);
    });

    expect(screen.getByText('Historial de cotizaciones')).toBeDefined();
    expect(screen.getByText('51 Créditos')).toBeDefined();
    expect(screen.getByRole('columnheader', { name: /fecha/i })).toBeDefined();
    expect(screen.getByRole('columnheader', { name: /precio/i })).toBeDefined();
  });

  it('renders an empty state when there are no quotes', async () => {
    vi.spyOn(playersApi, 'getPlayerQuotes').mockResolvedValue([]);

    render(<PlayerQuote playerId={7} />);

    await waitFor(() => {
      expect(screen.getByText('Sin cotización')).toBeDefined();
    });
    expect(screen.getByText('Sin cotizaciones registradas')).toBeDefined();
  });

  it('searches a quote by date and shows its price', async () => {
    vi.spyOn(playersApi, 'getPlayerQuotes').mockResolvedValue([
      aQuote('2026-11-03T10:00:00Z', '76.0000'),
    ]);
    const priceSpy = vi.spyOn(playersApi, 'getPlayerPrice').mockResolvedValue({
      playerId: 7,
      computedAt: '2026-10-01T10:00:00Z',
      strategyName: 'rateada',
      price: '41.0000',
      score: 0.4,
      leagueCoefficient: 1,
    });

    render(<PlayerQuote playerId={7} />);
    await waitFor(() => {
      expect(screen.getAllByText('76 Créditos').length).toBeGreaterThan(0);
    });

    fireEvent.change(screen.getByLabelText(/buscar cotización por fecha/i), {
      target: { value: '2026-10-01' },
    });
    fireEvent.click(screen.getByRole('button', { name: /consultar/i }));

    await waitFor(() => {
      expect(screen.getByText('41 Créditos')).toBeDefined();
    });
    expect(priceSpy).toHaveBeenCalledWith(7, '2026-10-01');
  });

  it('shows an error when there is no quote for the date', async () => {
    vi.spyOn(playersApi, 'getPlayerQuotes').mockResolvedValue([]);
    vi.spyOn(playersApi, 'getPlayerPrice').mockRejectedValue(new Error('not found'));

    render(<PlayerQuote playerId={7} />);
    await waitFor(() => {
      expect(screen.getByText('Sin cotización')).toBeDefined();
    });

    fireEvent.change(screen.getByLabelText(/buscar cotización por fecha/i), {
      target: { value: '2020-01-01' },
    });
    fireEvent.click(screen.getByRole('button', { name: /consultar/i }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeDefined();
    });
    expect(screen.getByText('No hay cotización para esa fecha.')).toBeDefined();
  });
});

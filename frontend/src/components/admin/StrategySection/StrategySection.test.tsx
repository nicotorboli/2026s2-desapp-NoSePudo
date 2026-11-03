import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  activateStrategy,
  getActiveStrategyOptional,
  listStrategies,
  ScoringStrategyDTO,
} from '../../../api/admin';
import { StrategySection } from './StrategySection';

vi.mock('../../../api/admin', () => ({
  listStrategies: vi.fn(),
  getActiveStrategyOptional: vi.fn(),
  activateStrategy: vi.fn(),
}));

const strategies: ScoringStrategyDTO[] = [
  {
    id: 1,
    name: 'rateada',
    alpha: 0.5,
    weights: {
      Attacker: { goals: 0.35, fouls: 0.05 },
      Goalkeeper: { clears: 0.45 },
    },
  },
  {
    id: 2,
    name: 'estadistica',
    alpha: 0.2,
    weights: { Defender: { tackles: 0.5 } },
  },
];

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(listStrategies).mockResolvedValue(strategies);
  vi.mocked(getActiveStrategyOptional).mockResolvedValue({ id: 1, name: 'rateada' });
  vi.mocked(activateStrategy).mockResolvedValue({ id: 2, name: 'estadistica' });
});

afterEach(cleanup);

describe('StrategySection', () => {
  it('muestra la estrategia activa, el alpha y los pesos traducidos', async () => {
    render(<StrategySection />);

    await waitFor(() => {
      expect(screen.getByText('rateada')).toBeDefined();
    });

    expect(screen.getByText('estadistica')).toBeDefined();
    expect(screen.getByText('Activa')).toBeDefined();
    expect(screen.getByText(/α = 0.500/)).toBeDefined();
    expect(screen.getByText('Delantero')).toBeDefined();
    expect(screen.getByText('Goles')).toBeDefined();
    expect(screen.getByText('0.350')).toBeDefined();
  });

  it('activa otra estrategia después de confirmar', async () => {
    render(<StrategySection />);

    await waitFor(() => {
      expect(screen.getByText('estadistica')).toBeDefined();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Activar' }));

    const dialog = screen.getByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Activar' }));

    await waitFor(() => {
      expect(activateStrategy).toHaveBeenCalledWith(2);
    });

    expect(
      await screen.findByText('Estrategia "estadistica" activada. Se recalcularon los scores.'),
    ).toBeDefined();
  });
});

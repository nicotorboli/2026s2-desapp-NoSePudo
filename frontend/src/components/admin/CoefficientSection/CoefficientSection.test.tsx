import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { getLeagueCoefficient, setLeagueCoefficient } from '../../../api/admin';
import { CoefficientSection } from './CoefficientSection';

vi.mock('../../../api/admin', () => ({
  getLeagueCoefficient: vi.fn(),
  setLeagueCoefficient: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(getLeagueCoefficient).mockImplementation(async (code) => {
    if (code === 'PL') {
      return {
        leagueCode: 'PL',
        coefficient: 1.5,
        effectiveFrom: '2026-11-01T00:00:00Z',
      };
    }
    return { leagueCode: code, coefficient: 1, effectiveFrom: null };
  });
});

afterEach(cleanup);

describe('CoefficientSection', () => {
  it('muestra 1.0 como valor por defecto cuando la liga no tiene coeficiente guardado', async () => {
    render(<CoefficientSection />);

    await waitFor(() => {
      expect(screen.getByText('Premier League (Inglaterra)')).toBeDefined();
    });

    expect(screen.getByText('1.5000')).toBeDefined();
    expect(screen.getAllByText('1.0000 (por defecto)')).toHaveLength(4);
  });

  it('guarda el coeficiente editado y refresca la fila', async () => {
    vi.mocked(setLeagueCoefficient).mockResolvedValue({
      leagueCode: 'BL1',
      coefficient: 2.5,
      effectiveFrom: '2026-11-02T10:00:00Z',
    });

    render(<CoefficientSection />);

    const input = await screen.findByLabelText('Nuevo coeficiente de Bundesliga (Alemania)');
    fireEvent.change(input, { target: { value: '2.5' } });
    fireEvent.click(screen.getByLabelText('Guardar coeficiente de Bundesliga (Alemania)'));

    await waitFor(() => {
      expect(setLeagueCoefficient).toHaveBeenCalledWith('BL1', 2.5);
    });

    expect(await screen.findByText('2.5000')).toBeDefined();
  });

  it('no llama al backend si el coeficiente no es positivo', async () => {
    render(<CoefficientSection />);

    const input = await screen.findByLabelText('Nuevo coeficiente de Serie A (Italia)');
    fireEvent.change(input, { target: { value: '0' } });
    fireEvent.click(screen.getByLabelText('Guardar coeficiente de Serie A (Italia)'));

    expect(await screen.findByText('El coeficiente debe ser un número mayor que cero.')).toBeDefined();
    expect(setLeagueCoefficient).not.toHaveBeenCalled();
  });
});

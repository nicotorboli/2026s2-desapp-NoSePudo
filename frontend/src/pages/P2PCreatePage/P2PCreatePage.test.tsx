import { describe, it, expect, vi, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor, fireEvent } from '@testing-library/react';
import { P2PCreatePage } from './P2PCreatePage';
import * as portfolioApi from '../../api/portfolio';
import * as p2pApi from '../../api/p2p';

const authState = vi.hoisted(() => ({
  user: { id: 1, email: 'vendedor@test.com' },
  isAuthenticated: true,
}));

vi.mock('../../context/AuthContext', () => ({
  useAuth: () => authState,
}));

const portfolio: portfolioApi.PortfolioResponseDTO = {
  userId: 1,
  availableCash: '0',
  reservedCash: '0',
  tokensValuation: '5.0000',
  totalValue: '5.0000',
  unrealizedResult: '0.0000',
  positions: [
    {
      playerId: 7,
      playerName: 'Jugador',
      clubName: 'Club',
      position: 'Attacker',
      quantity: 5,
      availableQuantity: 5,
      reservedQuantity: 0,
      averagePrice: '1.0000',
      currentPrice: '1.0000',
      currentValuation: '5.0000',
      unrealizedResult: '0.0000',
    },
  ],
};

const createdSession: p2pApi.P2PSessionDTO = {
  id: 1,
  token: 'token-de-prueba',
  sessionUrl: '/?token=token-de-prueba',
  seller: { id: 1, email: 'vendedor@test.com' },
  buyer: null,
  player: { id: 7, name: 'Jugador', clubName: 'Club' },
  quantity: 1,
  pricePerToken: '1.5000',
  totalAmount: '1.5000',
  status: 'PENDING',
  createdAt: '2026-11-16T12:00:00Z',
  expiresAt: '2026-11-16T12:05:00Z',
  completedAt: null,
};

const renderPage = () =>
  render(<P2PCreatePage onBack={vi.fn()} onOpenSession={vi.fn()} onOpenAuth={vi.fn()} />);

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe('P2PCreatePage', () => {
  it('submits the decimal price exactly as typed', async () => {
    vi.spyOn(portfolioApi, 'getUserPortfolio').mockResolvedValue(portfolio);
    const create = vi.spyOn(p2pApi, 'createP2PSession').mockResolvedValue(createdSession);

    renderPage();

    const price = await screen.findByLabelText(/Precio por Token/i);
    fireEvent.change(price, { target: { value: '1,5' } });

    expect(screen.getByText('1,5 créditos')).toBeTruthy();

    fireEvent.click(screen.getByRole('button', { name: /Crear Oferta y Generar Link/i }));

    await waitFor(() =>
      expect(create).toHaveBeenCalledWith({ playerId: 7, quantity: 1, pricePerToken: '1.5' })
    );
    await screen.findByText(/Enlace de Transacción Generado/i);
  });

  it('rejects a non numeric price without calling the API', async () => {
    vi.spyOn(portfolioApi, 'getUserPortfolio').mockResolvedValue(portfolio);
    const create = vi.spyOn(p2pApi, 'createP2PSession').mockResolvedValue(createdSession);

    renderPage();

    const price = await screen.findByLabelText(/Precio por Token/i);
    fireEvent.change(price, { target: { value: 'abc' } });
    fireEvent.click(screen.getByRole('button', { name: /Crear Oferta y Generar Link/i }));

    await screen.findByText(/completa todos los campos con valores válidos/i);
    expect(create).not.toHaveBeenCalled();
  });
});

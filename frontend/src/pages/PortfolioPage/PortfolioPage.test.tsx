import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import { PortfolioPage } from './PortfolioPage';
import * as AuthContextModule from '../../context/AuthContext';
import * as portfolioApi from '../../api/portfolio';
import * as ordersApi from '../../api/orders';

describe('PortfolioPage', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  afterEach(cleanup);

  it('renders restricted access message if unauthenticated', () => {
    vi.spyOn(AuthContextModule, 'useAuth').mockReturnValue({
      user: null,
      token: null,
      isAuthenticated: false,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout: vi.fn(),
    });

    render(<PortfolioPage onSelectPlayer={vi.fn()} onBackToCatalog={vi.fn()} />);

    expect(screen.getByText('Acceso restringido')).toBeDefined();
    expect(screen.getByText(/debes iniciar sesión/i)).toBeDefined();
  });

  it('renders summary cards and positions when authenticated', async () => {
    vi.spyOn(AuthContextModule, 'useAuth').mockReturnValue({
      user: { id: 1, email: 'test@example.com' },
      token: 'valid-token',
      isAuthenticated: true,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout: vi.fn(),
    });

    const mockPortfolio: portfolioApi.PortfolioResponseDTO = {
      userId: 1,
      availableCash: 950.0,
      reservedCash: 50.0,
      tokensValuation: 100.0,
      totalValue: 1100.0,
      unrealizedResult: 20.0,
      positions: [
        {
          playerId: 10,
          playerName: 'Lionel Messi',
          clubName: 'Inter Miami CF',
          position: 'Attacker',
          quantity: 10,
          averagePrice: 1.0,
          currentPrice: 1.5,
          currentValuation: 15.0,
          unrealizedResult: 5.0,
        },
      ],
    };

    vi.spyOn(portfolioApi, 'getUserPortfolio').mockResolvedValue(mockPortfolio);
    vi.spyOn(ordersApi, 'getOpenOrders').mockResolvedValue([]);
    vi.spyOn(portfolioApi, 'getUserTransactions').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      limit: 10,
      totalPages: 0,
    });

    render(<PortfolioPage onSelectPlayer={vi.fn()} onBackToCatalog={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText('950 Créditos')).toBeDefined();
      expect(screen.getByText('50 Créditos')).toBeDefined();
      expect(screen.getByText('Lionel Messi')).toBeDefined();
    });
  });
});

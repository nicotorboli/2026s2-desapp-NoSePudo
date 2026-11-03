import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AdminPage } from './AdminPage';

const mocks = vi.hoisted(() => ({
  useAuth: vi.fn(),
}));

vi.mock('../../context/AuthContext', () => ({
  useAuth: mocks.useAuth,
}));

vi.mock('../../components/admin/SyncSection/SyncSection', () => ({
  SyncSection: () => <div>Sincronizaciones sección</div>,
}));

vi.mock('../../components/admin/StrategySection/StrategySection', () => ({
  StrategySection: () => <div>Estrategias sección</div>,
}));

vi.mock('../../components/admin/CoefficientSection/CoefficientSection', () => ({
  CoefficientSection: () => <div>Coeficientes sección</div>,
}));

const logout = vi.fn().mockResolvedValue(undefined);

const superuser = { id: 1, email: 'admin@nosepudo.ar', privilege: 'superuser' };
const commonUser = { id: 2, email: 'user@nosepudo.ar', privilege: 'user' };

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(cleanup);

describe('AdminPage', () => {
  it('muestra acceso restringido y pide login cuando no hay sesión', () => {
    mocks.useAuth.mockReturnValue({
      user: null,
      token: null,
      isAuthenticated: false,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout,
    });
    const onOpenAuth = vi.fn();

    render(<AdminPage onOpenAuth={onOpenAuth} />);

    expect(screen.getByText('Acceso restringido')).toBeDefined();
    fireEvent.click(screen.getByRole('button', { name: /iniciar sesión/i }));
    expect(onOpenAuth).toHaveBeenCalledTimes(1);
  });

  it('muestra acceso restringido para un usuario común', () => {
    mocks.useAuth.mockReturnValue({
      user: commonUser,
      token: 'valid-token',
      isAuthenticated: true,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout,
    });

    render(<AdminPage />);

    expect(screen.getByText('Acceso restringido')).toBeDefined();
    expect(screen.getByText(/user@nosepudo\.ar/)).toBeDefined();
  });

  it('muestra el panel y las secciones para un superusuario', () => {
    mocks.useAuth.mockReturnValue({
      user: superuser,
      token: 'valid-token',
      isAuthenticated: true,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout,
    });

    render(<AdminPage />);

    expect(screen.getByText('Administración')).toBeDefined();
    expect(screen.getByText('admin@nosepudo.ar')).toBeDefined();
    expect(screen.getByText('Sincronizaciones sección')).toBeDefined();
    expect(screen.getByText('Estrategias sección')).toBeDefined();
    expect(screen.getByText('Coeficientes sección')).toBeDefined();
    expect(screen.queryByText('Catálogo')).toBeNull();
  });

  it('cierra sesión desde el panel', () => {
    mocks.useAuth.mockReturnValue({
      user: superuser,
      token: 'valid-token',
      isAuthenticated: true,
      login: vi.fn(),
      registerAndLogin: vi.fn(),
      logout,
    });

    render(<AdminPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }));

    expect(logout).toHaveBeenCalledTimes(1);
  });
});

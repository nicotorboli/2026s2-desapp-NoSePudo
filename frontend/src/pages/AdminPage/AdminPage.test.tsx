import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AdminPage } from './AdminPage';

const mocks = vi.hoisted(() => ({
  useAdminSession: vi.fn(),
}));

vi.mock('../../auth/adminSession', () => ({
  useAdminSession: mocks.useAdminSession,
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
const login = vi.fn().mockResolvedValue(undefined);

const superuserSession = {
  email: 'admin@nosepudo.ar',
  accessToken: 'access',
  refreshToken: 'refresh',
  accessExpiresAt: '2030-01-01T00:00:00Z',
  refreshExpiresAt: '2030-01-08T00:00:00Z',
};

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(cleanup);

describe('AdminPage', () => {
  it('muestra el login cuando no hay sesión', () => {
    mocks.useAdminSession.mockReturnValue({ session: null, isSuperuser: false, login, logout });

    render(<AdminPage />);

    expect(screen.getByText('Administración')).toBeDefined();
    expect(screen.getByLabelText('Email')).toBeDefined();
    expect(screen.getByLabelText('Contraseña')).toBeDefined();
    expect(screen.getByRole('button', { name: 'Ingresar' })).toBeDefined();
  });

  it('muestra acceso restringido para un usuario común', () => {
    mocks.useAdminSession.mockReturnValue({
      session: { ...superuserSession, email: 'user@nosepudo.ar' },
      isSuperuser: false,
      login,
      logout,
    });

    render(<AdminPage />);

    expect(screen.getByText('Acceso restringido')).toBeDefined();
    expect(screen.getByText(/user@nosepudo\.ar/)).toBeDefined();
  });

  it('muestra el panel y las secciones para un superusuario', () => {
    mocks.useAdminSession.mockReturnValue({
      session: superuserSession,
      isSuperuser: true,
      login,
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
    mocks.useAdminSession.mockReturnValue({
      session: superuserSession,
      isSuperuser: true,
      login,
      logout,
    });

    render(<AdminPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Cerrar sesión' }));

    expect(logout).toHaveBeenCalledTimes(1);
  });
});

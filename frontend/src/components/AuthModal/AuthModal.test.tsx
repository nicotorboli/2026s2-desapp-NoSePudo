import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, render, screen, fireEvent, waitFor } from '@testing-library/react';
import { AuthModal } from './AuthModal';
import * as AuthContextModule from '../../context/AuthContext';

describe('AuthModal', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    cleanup();
  });

  afterEach(cleanup);

  it('does not render when isOpen is false', () => {
    const { container } = render(<AuthModal isOpen={false} onClose={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders login form by default when isOpen is true', () => {
    render(<AuthModal isOpen={true} onClose={vi.fn()} />);

    expect(screen.getByRole('dialog')).toBeDefined();
    expect(screen.getByRole('heading', { name: 'Iniciar Sesión' })).toBeDefined();
    expect(screen.getByLabelText(/correo electrónico/i)).toBeDefined();
    expect(screen.getByLabelText(/contraseña/i)).toBeDefined();
  });

  it('switches to register tab when clicking Registrarse', () => {
    render(<AuthModal isOpen={true} onClose={vi.fn()} />);

    const registerTabBtn = screen.getAllByRole('button', { name: 'Registrarse' })[0];
    fireEvent.click(registerTabBtn);

    expect(screen.getByRole('heading', { name: 'Crear Cuenta' })).toBeDefined();
    expect(screen.getByLabelText(/confirmar contraseña/i)).toBeDefined();
    expect(screen.getByText(/1,000 créditos/i)).toBeDefined();
  });

  it('calls onClose when clicking close button', () => {
    const onCloseMock = vi.fn();
    render(<AuthModal isOpen={true} onClose={onCloseMock} />);

    const closeBtn = screen.getAllByRole('button', { name: /cerrar modal/i })[0];
    fireEvent.click(closeBtn);

    expect(onCloseMock).toHaveBeenCalledOnce();
  });

  it('submits login when login form is filled and submitted', async () => {
    const mockLogin = vi.fn().mockResolvedValue(undefined);
    const mockAuthContext = {
      user: null,
      token: null,
      isAuthenticated: false,
      login: mockLogin,
      registerAndLogin: vi.fn(),
      logout: vi.fn(),
    };

    vi.spyOn(AuthContextModule, 'useAuth').mockReturnValue(mockAuthContext);

    const onCloseMock = vi.fn();
    render(<AuthModal isOpen={true} onClose={onCloseMock} />);

    fireEvent.change(screen.getByLabelText(/correo electrónico/i), {
      target: { value: 'user@test.com' },
    });
    fireEvent.change(screen.getByLabelText(/contraseña/i), {
      target: { value: 'password123' },
    });

    fireEvent.click(screen.getAllByRole('button', { name: 'Iniciar Sesión' })[0]);

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        email: 'user@test.com',
        password: 'password123',
      });
      expect(onCloseMock).toHaveBeenCalled();
    });
  });
});

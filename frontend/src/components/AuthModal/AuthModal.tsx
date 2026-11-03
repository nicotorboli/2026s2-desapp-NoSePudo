import React, { useState, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import './AuthModal.css';

interface AuthModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialTab?: 'login' | 'register';
}

export const AuthModal: React.FC<AuthModalProps> = ({
  isOpen,
  onClose,
  initialTab = 'login',
}) => {
  const { login, registerAndLogin } = useAuth();
  const [tab, setTab] = useState<'login' | 'register'>(initialTab);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      if (initialTab) {
        setTab(initialTab);
      }
      setError(null);
    }
  }, [isOpen, initialTab]);

  if (!isOpen) {
    return null;
  }

  const resetForm = () => {
    setEmail('');
    setPassword('');
    setConfirmPassword('');
    setError(null);
  };

  const handleTabChange = (newTab: 'login' | 'register') => {
    setTab(newTab);
    setError(null);
    setConfirmPassword('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!email || !password) {
      setError('Por favor complete todos los campos.');
      return;
    }

    if (tab === 'register') {
      if (password.length < 8) {
        setError('La contraseña debe tener al menos 8 caracteres.');
        return;
      }
      if (password.length > 72) {
        setError('La contraseña no puede superar los 72 caracteres.');
        return;
      }
      if (password !== confirmPassword) {
        setError('Las contraseñas no coinciden.');
        return;
      }
    }

    setIsSubmitting(true);
    try {
      if (tab === 'login') {
        await login({ email, password });
      } else {
        await registerAndLogin({ email, password });
      }
      resetForm();
      onClose();
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosErr = err as { response?: { status?: number; data?: { error?: string } } };
        if (axiosErr.response?.status === 401) {
          setError('Credenciales inválidas. Verifique su email y contraseña.');
        } else if (axiosErr.response?.status === 409) {
          setError('Ya existe una cuenta registrada con este correo electrónico.');
        } else if (axiosErr.response?.data?.error) {
          setError(axiosErr.response.data.error);
        } else {
          setError('Error al procesar la solicitud. Intente nuevamente.');
        }
      } else {
        setError('Error de conexión con el servidor.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="auth-modal" role="dialog" aria-modal="true" aria-labelledby="auth-modal-title">
      <div className="auth-modal__overlay" onClick={onClose} />
      <div className="auth-modal__dialog">
        <button
          type="button"
          className="auth-modal__close-btn"
          onClick={onClose}
          aria-label="Cerrar modal"
        >
          ×
        </button>

        <h2 id="auth-modal-title" className="auth-modal__title">
          {tab === 'login' ? 'Iniciar Sesión' : 'Crear Cuenta'}
        </h2>

        <div className="auth-modal__tabs">
          <button
            type="button"
            className={`auth-modal__tab ${tab === 'login' ? 'auth-modal__tab--active' : ''}`}
            onClick={() => handleTabChange('login')}
          >
            Ingresar
          </button>
          <button
            type="button"
            className={`auth-modal__tab ${tab === 'register' ? 'auth-modal__tab--active' : ''}`}
            onClick={() => handleTabChange('register')}
          >
            Registrarse
          </button>
        </div>

        {tab === 'register' && (
          <div className="auth-modal__alert auth-modal__alert--info">
            Al registrarte recibís automáticamente <strong>1,000 créditos</strong> de bienvenida para operar en el mercado.
          </div>
        )}

        {error && (
          <div className="auth-modal__alert auth-modal__alert--error" role="alert">
            <div>{error}</div>
            {tab === 'login' && error.includes('Credenciales') && (
              <button
                type="button"
                className="auth-modal__switch-link auth-modal__switch-link--inline"
                onClick={() => {
                  setTab('register');
                  setError(null);
                }}
              >
                ¿No tenés cuenta todavía? Creala acá
              </button>
            )}
          </div>
        )}

        <form className="auth-modal__form" onSubmit={handleSubmit}>
          <div className="auth-modal__field">
            <label className="auth-modal__label" htmlFor="auth-email">
              Correo Electrónico
            </label>
            <input
              id="auth-email"
              type="email"
              className="auth-modal__input"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="ejemplo@correo.com"
              required
              autoFocus
            />
          </div>

          <div className="auth-modal__field">
            <label className="auth-modal__label" htmlFor="auth-password">
              Contraseña
            </label>
            <input
              id="auth-password"
              type="password"
              className="auth-modal__input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Mínimo 8 caracteres"
              required
            />
            {tab === 'register' && (
              <span className="auth-modal__hint">Entre 8 y 72 caracteres</span>
            )}
          </div>

          {tab === 'register' && (
            <div className="auth-modal__field">
              <label className="auth-modal__label" htmlFor="auth-confirm-password">
                Confirmar Contraseña
              </label>
              <input
                id="auth-confirm-password"
                type="password"
                className="auth-modal__input"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Repita la contraseña"
                required
              />
            </div>
          )}

          <button
            type="submit"
            className="auth-modal__submit-btn"
            disabled={isSubmitting}
          >
            {isSubmitting
              ? 'Procesando...'
              : tab === 'login'
              ? 'Iniciar Sesión'
              : 'Registrarse y Obtener 1,000 CR'}
          </button>
        </form>

        <div className="auth-modal__footer">
          {tab === 'login' ? (
            <p>
              ¿No tenés cuenta?{' '}
              <button
                type="button"
                className="auth-modal__switch-link"
                onClick={() => handleTabChange('register')}
              >
                Registrate acá
              </button>
            </p>
          ) : (
            <p>
              ¿Ya tenés cuenta?{' '}
              <button
                type="button"
                className="auth-modal__switch-link"
                onClick={() => handleTabChange('login')}
              >
                Iniciá sesión
              </button>
            </p>
          )}
        </div>
      </div>
    </div>
  );
};

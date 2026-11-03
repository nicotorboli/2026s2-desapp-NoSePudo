import React, { useState } from 'react';
import './AdminLogin.css';

interface AdminLoginProps {
  onLogin: (email: string, password: string) => Promise<void>;
}

export const AdminLogin: React.FC<AdminLoginProps> = ({ onLogin }) => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      await onLogin(email, password);
    } catch {
      setError('No se pudo iniciar sesión. Verificá las credenciales e intentá de nuevo.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="admin-login">
      <form className="admin-login__card" onSubmit={handleSubmit}>
        <h1 className="admin-login__title">Administración</h1>
        <p className="admin-login__subtitle">
          Ingresá con una cuenta de superusuario para administrar el catálogo.
        </p>

        <label className="admin-login__label" htmlFor="admin-email">
          Email
        </label>
        <input
          id="admin-email"
          type="email"
          className="admin-login__input"
          autoComplete="username"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          required
        />

        <label className="admin-login__label" htmlFor="admin-password">
          Contraseña
        </label>
        <input
          id="admin-password"
          type="password"
          className="admin-login__input"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          required
        />

        {error && (
          <p className="admin-login__error" role="alert">
            {error}
          </p>
        )}

        <button
          type="submit"
          className="admin-login__button"
          disabled={isSubmitting || !email || !password}
        >
          {isSubmitting ? 'Ingresando...' : 'Ingresar'}
        </button>
      </form>
    </main>
  );
};

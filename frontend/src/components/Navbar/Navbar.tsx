import React from 'react';
import { useAuth } from '../../context/AuthContext';
import './Navbar.css';

type NavView = 'catalog' | 'portfolio' | 'ranking' | 'admin' | 'detail';

interface NavbarProps {
  currentView?: NavView;
  onNavigate?: (view: NavView) => void;
  onOpenAuth?: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  currentView = 'catalog',
  onNavigate,
  onOpenAuth,
}) => {
  const { user, isAuthenticated, logout } = useAuth();
  const isAdmin = user?.privilege === 'superuser';

  const go = (view: NavView) => {
    if (onNavigate) {
      onNavigate(view);
    }
  };

  const handleLogout = async () => {
    await logout();
    go('catalog');
  };

  const navClass = (view: NavView) =>
    `navbar__nav-button ${currentView === view ? 'navbar__nav-button--active' : ''}`;

  return (
    <header className="navbar">
      <div className="navbar__container">
        <div className="navbar__left">
          <button
            type="button"
            className="navbar__brand-button"
            onClick={() => go('catalog')}
            aria-label="Ir al catálogo de jugadores"
          >
            <span className="navbar__title">NoSePudo Market</span>
          </button>

          <nav className="navbar__nav-links">
            <button type="button" className={navClass('catalog')} onClick={() => go('catalog')}>
              Catálogo
            </button>
            <button type="button" className={navClass('ranking')} onClick={() => go('ranking')}>
              Ranking
            </button>
            {isAuthenticated && (
              <button
                type="button"
                className={navClass('portfolio')}
                onClick={() => go('portfolio')}
              >
                Mi Patrimonio
              </button>
            )}
            {isAdmin && (
              <button
                type="button"
                className={navClass('admin')}
                onClick={() => go('admin')}
              >
                Administración
              </button>
            )}
          </nav>
        </div>

        <div className="navbar__right">
          {isAuthenticated && user ? (
            <>
              <div className="navbar__user-info">
                <span className="navbar__user-email">{user.email}</span>
              </div>
              <button type="button" className="navbar__logout-button" onClick={handleLogout}>
                Cerrar Sesión
              </button>
            </>
          ) : (
            onOpenAuth && (
              <button type="button" className="navbar__auth-button" onClick={onOpenAuth}>
                Iniciar Sesión
              </button>
            )
          )}
        </div>
      </div>
    </header>
  );
};

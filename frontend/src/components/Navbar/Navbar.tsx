import React from 'react';
import { useAuth } from '../../context/AuthContext';
import './Navbar.css';

interface NavbarProps {
  currentView?: 'catalog' | 'portfolio';
  onNavigateHome?: () => void;
  onNavigateView?: (view: 'catalog' | 'portfolio') => void;
  onOpenAuth?: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  currentView = 'catalog',
  onNavigateHome,
  onNavigateView,
  onOpenAuth,
}) => {
  const { user, isAuthenticated, logout } = useAuth();

  const handleBrandClick = () => {
    if (onNavigateView) {
      onNavigateView('catalog');
    }
    if (onNavigateHome) {
      onNavigateHome();
    }
  };

  const handleLogout = async () => {
    await logout();
    if (onNavigateView) {
      onNavigateView('catalog');
    }
    if (onNavigateHome) {
      onNavigateHome();
    }
  };

  return (
    <header className="navbar">
      <div className="navbar__container">
        <div className="navbar__left">
          <button
            type="button"
            className="navbar__brand-button"
            onClick={handleBrandClick}
            aria-label="Ir al catálogo de jugadores"
          >
            <span className="navbar__title">NoSePudo Market</span>
          </button>

          {onNavigateView && (
            <nav className="navbar__nav-links">
              <button
                type="button"
                className={`navbar__nav-button ${currentView === 'catalog' ? 'navbar__nav-button--active' : ''}`}
                onClick={() => {
                  onNavigateView('catalog');
                  if (onNavigateHome) onNavigateHome();
                }}
              >
                Catálogo
              </button>
              {isAuthenticated && (
                <button
                  type="button"
                  className={`navbar__nav-button ${currentView === 'portfolio' ? 'navbar__nav-button--active' : ''}`}
                  onClick={() => onNavigateView('portfolio')}
                >
                  Mi Patrimonio
                </button>
              )}
            </nav>
          )}
        </div>

        <div className="navbar__right">
          {isAuthenticated && user ? (
            <>
              <div className="navbar__user-info">
                <span className="navbar__user-email">{user.email}</span>
              </div>
              <button
                type="button"
                className="navbar__logout-button"
                onClick={handleLogout}
              >
                Cerrar Sesión
              </button>
            </>
          ) : (
            onOpenAuth && (
              <button
                type="button"
                className="navbar__auth-button"
                onClick={onOpenAuth}
              >
                Iniciar Sesión
              </button>
            )
          )}
        </div>
      </div>
    </header>
  );
};

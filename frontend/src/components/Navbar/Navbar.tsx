import React from 'react';
import './Navbar.css';

interface NavbarProps {
  onNavigateHome?: () => void;
  onNavigateRanking?: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ onNavigateHome, onNavigateRanking }) => {
  return (
    <header className="navbar">
      <div className="navbar__container">
        <button
          type="button"
          className="navbar__brand-button"
          onClick={onNavigateHome}
          aria-label="Ir al catálogo de jugadores"
        >
          <span className="navbar__logo">⚽</span>
          <span className="navbar__title">NoSePudo Market</span>
        </button>
        <nav className="navbar__nav" aria-label="Navegación principal">
          <button type="button" className="navbar__link" onClick={onNavigateHome}>
            Catálogo
          </button>
          <button type="button" className="navbar__link" onClick={onNavigateRanking}>
            Ranking
          </button>
        </nav>
      </div>
    </header>
  );
};

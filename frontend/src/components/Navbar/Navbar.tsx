import React from 'react';
import './Navbar.css';

interface NavbarProps {
  onNavigateHome?: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ onNavigateHome }) => {
  return (
    <header className="navbar">
      <div className="navbar__container">
        <button
          type="button"
          className="navbar__brand-button"
          onClick={onNavigateHome}
          aria-label="Go to players catalog"
        >
          <span className="navbar__logo">⚽</span>
          <span className="navbar__title">NoSePudo Football</span>
        </button>
      </div>
    </header>
  );
};

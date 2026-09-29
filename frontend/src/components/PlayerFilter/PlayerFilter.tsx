import React from 'react';
import { PlayerFilterParams } from '../../api/players';
import './PlayerFilter.css';

interface PlayerFilterProps {
  filters: PlayerFilterParams;
  onChange: (filters: PlayerFilterParams) => void;
  onReset: () => void;
}

export const PlayerFilter: React.FC<PlayerFilterProps> = ({ filters, onChange, onReset }) => {
  const handleInputChange = (field: keyof PlayerFilterParams, value: string | boolean) => {
    onChange({
      ...filters,
      [field]: value,
      page: 1, // Reset to first page on filter change
    });
  };

  return (
    <div className="player-filter">
      <div className="player-filter__group">
        <label htmlFor="filter-search" className="player-filter__label">
          Buscar por nombre
        </label>
        <input
          id="filter-search"
          type="text"
          className="player-filter__input"
          placeholder="ej. Bukayo Saka"
          value={filters.search || ''}
          onChange={(e) => handleInputChange('search', e.target.value)}
        />
      </div>

      <div className="player-filter__group">
        <label htmlFor="filter-league" className="player-filter__label">
          Liga
        </label>
        <select
          id="filter-league"
          className="player-filter__select"
          value={filters.league || ''}
          onChange={(e) => handleInputChange('league', e.target.value)}
        >
          <option value="">Todas las ligas</option>
          <option value="PL">Premier League (Inglaterra)</option>
          <option value="BL1">Bundesliga (Alemania)</option>
          <option value="PD">La Liga (España)</option>
          <option value="SA">Serie A (Italia)</option>
          <option value="FL1">Ligue 1 (Francia)</option>
        </select>
      </div>

      <div className="player-filter__group">
        <label htmlFor="filter-club" className="player-filter__label">
          Nombre del club
        </label>
        <input
          id="filter-club"
          type="text"
          className="player-filter__input"
          placeholder="ej. Arsenal FC"
          value={filters.club || ''}
          onChange={(e) => handleInputChange('club', e.target.value)}
        />
      </div>

      <div className="player-filter__group">
        <label htmlFor="filter-position" className="player-filter__label">
          Posición
        </label>
        <select
          id="filter-position"
          className="player-filter__select"
          value={filters.position || ''}
          onChange={(e) => handleInputChange('position', e.target.value)}
        >
          <option value="">Todas las posiciones</option>
          <option value="Goalkeeper">Arquero</option>
          <option value="Defender">Defensor</option>
          <option value="Midfielder">Mediocampista</option>
          <option value="Attacker">Delantero</option>
        </select>
      </div>

      <div className="player-filter__actions">
        <button
          type="button"
          className="player-filter__button player-filter__button--reset"
          onClick={onReset}
        >
          Limpiar filtros
        </button>
      </div>
    </div>
  );
};

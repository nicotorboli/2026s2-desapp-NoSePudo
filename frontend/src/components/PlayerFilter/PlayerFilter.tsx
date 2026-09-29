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
          Search by Name
        </label>
        <input
          id="filter-search"
          type="text"
          className="player-filter__input"
          placeholder="e.g. Bukayo Saka"
          value={filters.search || ''}
          onChange={(e) => handleInputChange('search', e.target.value)}
        />
      </div>

      <div className="player-filter__group">
        <label htmlFor="filter-league" className="player-filter__label">
          League
        </label>
        <select
          id="filter-league"
          className="player-filter__select"
          value={filters.league || ''}
          onChange={(e) => handleInputChange('league', e.target.value)}
        >
          <option value="">All Leagues</option>
          <option value="PL">Premier League (England)</option>
          <option value="BL1">Bundesliga (Germany)</option>
          <option value="PD">La Liga (Spain)</option>
          <option value="SA">Serie A (Italy)</option>
          <option value="FL1">Ligue 1 (France)</option>
        </select>
      </div>

      <div className="player-filter__group">
        <label htmlFor="filter-club" className="player-filter__label">
          Club Name
        </label>
        <input
          id="filter-club"
          type="text"
          className="player-filter__input"
          placeholder="e.g. Arsenal FC"
          value={filters.club || ''}
          onChange={(e) => handleInputChange('club', e.target.value)}
        />
      </div>

      <div className="player-filter__group">
        <label htmlFor="filter-position" className="player-filter__label">
          Position
        </label>
        <select
          id="filter-position"
          className="player-filter__select"
          value={filters.position || ''}
          onChange={(e) => handleInputChange('position', e.target.value)}
        >
          <option value="">All Positions</option>
          <option value="Goalkeeper">Goalkeeper</option>
          <option value="Defender">Defender</option>
          <option value="Midfielder">Midfielder</option>
          <option value="Attacker">Attacker</option>
        </select>
      </div>

      <div className="player-filter__actions">
        <button
          type="button"
          className="player-filter__button player-filter__button--reset"
          onClick={onReset}
        >
          Clear Filters
        </button>
      </div>
    </div>
  );
};

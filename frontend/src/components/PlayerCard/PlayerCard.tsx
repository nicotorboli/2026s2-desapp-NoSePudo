import React from 'react';
import { PlayerDetailDTO } from '../../api/players';
import './PlayerCard.css';

interface PlayerCardProps {
  player: PlayerDetailDTO;
}

export const PlayerCard: React.FC<PlayerCardProps> = ({ player }) => {
  const formatShirtNumber = (num: number | null) => {
    if (num === null || num === undefined) {
      return 'Unassigned';
    }
    return `#${num}`;
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) {
      return '—';
    }
    return dateStr;
  };

  const formatDateTime = (dateStr: string) => {
    if (!dateStr) return '—';
    try {
      const d = new Date(dateStr);
      return d.toLocaleString();
    } catch {
      return dateStr;
    }
  };

  return (
    <article className="player-card">
      <div className="player-card__header">
        <div className="player-card__identity">
          <h2 className="player-card__name">{player.name}</h2>
          <span className="player-card__club">{player.club}</span>
        </div>
        <span
          className={`player-card__badge ${
            player.active ? 'player-card__badge--active' : 'player-card__badge--inactive'
          }`}
        >
          {player.active ? 'Active' : 'Inactive'}
        </span>
      </div>

      <div className="player-card__grid">
        <div className="player-card__field">
          <span className="player-card__label">Position</span>
          <span className="player-card__value">{player.position}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Shirt Number</span>
          <span className="player-card__value">{formatShirtNumber(player.shirtNumber)}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">League</span>
          <span className="player-card__value">
            {player.league} ({player.leagueCode})
          </span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Nationality</span>
          <span className="player-card__value">{player.nationality || 'Unknown'}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Date of Birth</span>
          <span className="player-card__value">{formatDate(player.dateOfBirth)}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">External ID</span>
          <span className="player-card__value">{player.externalId}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Created At</span>
          <span className="player-card__value player-card__value--meta">
            {formatDateTime(player.createdAt)}
          </span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Last Updated</span>
          <span className="player-card__value player-card__value--meta">
            {formatDateTime(player.updatedAt)}
          </span>
        </div>
      </div>
    </article>
  );
};

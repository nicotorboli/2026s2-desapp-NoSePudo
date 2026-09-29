import React from 'react';
import { PlayerDetailDTO } from '../../api/players';
import { translatePosition } from '../../utils/translations';
import './PlayerCard.css';

interface PlayerCardProps {
  player: PlayerDetailDTO;
}

export const PlayerCard: React.FC<PlayerCardProps> = ({ player }) => {
  const formatShirtNumber = (num: number | null) => {
    if (num === null || num === undefined) {
      return 'Sin asignar';
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
      return d.toLocaleString('es-ES');
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
          {player.active ? 'Activo' : 'Inactivo'}
        </span>
      </div>

      <div className="player-card__grid">
        <div className="player-card__field">
          <span className="player-card__label">Posición</span>
          <span className="player-card__value">{translatePosition(player.position)}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Número de camiseta</span>
          <span className="player-card__value">{formatShirtNumber(player.shirtNumber)}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Liga</span>
          <span className="player-card__value">
            {player.league} ({player.leagueCode})
          </span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Nacionalidad</span>
          <span className="player-card__value">{player.nationality || 'Desconocida'}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Fecha de nacimiento</span>
          <span className="player-card__value">{formatDate(player.dateOfBirth)}</span>
        </div>

        <div className="player-card__field">
          <span className="player-card__label">Última actualización</span>
          <span className="player-card__value player-card__value--meta">
            {formatDateTime(player.updatedAt)}
          </span>
        </div>
      </div>
    </article>
  );
};
